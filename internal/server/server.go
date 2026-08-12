package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asnakech/asnakech-servers/internal/auth"
	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/asnakech/asnakech-servers/internal/handlers"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/platform/ready"
	"github.com/asnakech/asnakech-servers/internal/rbac"
	"github.com/asnakech/asnakech-servers/internal/repository/postgres"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Dependencies are optional infrastructure handles wired at startup.
type Dependencies struct {
	DB *pgxpool.Pool
}

// Server wraps the HTTP stack for the Asnakech API.
type Server struct {
	cfg    *config.Config
	log    zerolog.Logger
	deps   Dependencies
	router *gin.Engine
	http   *http.Server
}

func New(cfg *config.Config, logger zerolog.Logger, deps Dependencies) *Server {
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.ZerologRequest(logger))
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowCredentials: cfg.CORSCredentials,
	}))

	s := &Server{
		cfg:    cfg,
		log:    logger,
		deps:   deps,
		router: router,
		http: &http.Server{
			Addr:         ":" + cfg.ServerPort,
			Handler:      router,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	checker := &ready.Checker{
		Pool:        s.deps.DB,
		DatabaseURL: s.cfg.DatabaseURL,
		RedisURL:    s.cfg.RedisURL,
		S3Endpoint:  s.cfg.S3Endpoint,
	}

	healthHandler := handlers.NewHealthHandler(s.cfg.AppVersion, checker)
	welcomeHandler := handlers.NewWelcomeHandler(s.cfg.AppVersion)

	s.router.GET("/health", healthHandler.HealthCheck)
	s.router.GET("/ready", healthHandler.ReadyCheck)
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/", welcomeHandler.Welcome)

		if s.deps.DB != nil {
			roleRepo := postgres.NewRoleRepository(s.deps.DB)
			userRepo := postgres.NewUserRepository(s.deps.DB)
			refreshRepo := postgres.NewRefreshTokenRepository(s.deps.DB)
			resetRepo := postgres.NewPasswordResetTokenRepository(s.deps.DB)
			verifyRepo := postgres.NewEmailVerificationTokenRepository(s.deps.DB)

			roleService := service.NewRoleService(roleRepo)
			roleHandler := handlers.NewRoleHandler(roleService)
			v1.GET("/roles", roleHandler.ListRoles)

			jwtSecret := s.cfg.JWTSecret
			if jwtSecret == "" {
				jwtSecret = "dev-only-change-me"
				s.log.Warn().Msg("JWT_SECRET empty — using insecure development default")
			}

			tokenManager := auth.NewTokenManager(jwtSecret, s.cfg.JWTAccessTTL, s.cfg.JWTRefreshTTL)
			authService := service.NewAuthService(
				userRepo,
				roleRepo,
				refreshRepo,
				resetRepo,
				verifyRepo,
				tokenManager,
				s.cfg.IsDevelopment(),
			)
			authHandler := handlers.NewAuthHandler(authService)
			userService := service.NewUserService(userRepo, roleRepo)
			userHandler := handlers.NewUserHandler(userService)

			authLimiter := middleware.NewIPRateLimiter(2, 10)
			authGroup := v1.Group("/auth")
			authGroup.Use(authLimiter.Limit())
			{
				authGroup.POST("/register", authHandler.Register)
				authGroup.POST("/login", authHandler.Login)
				authGroup.POST("/refresh", authHandler.Refresh)
				authGroup.POST("/logout", authHandler.Logout)
				authGroup.POST("/forgot-password", authHandler.ForgotPassword)
				authGroup.POST("/reset-password", authHandler.ResetPassword)
				authGroup.POST("/verify-email", authHandler.VerifyEmail)
				authGroup.GET("/me", middleware.BearerAuth(tokenManager), authHandler.Me)
			}

			usersGroup := v1.Group("/users")
			usersGroup.Use(middleware.BearerAuth(tokenManager))
			{
				usersGroup.GET("/me", userHandler.GetMyProfile)
				usersGroup.PATCH("/me", middleware.RequirePermission(rbac.PermProfileWrite), userHandler.UpdateMyProfile)
				usersGroup.PUT("/me/avatar", middleware.RequirePermission(rbac.PermProfileWrite), userHandler.SetMyAvatar)
				usersGroup.GET("/me/avatar/upload-intent", middleware.RequirePermission(rbac.PermProfileWrite), userHandler.AvatarUploadIntent)
			}

			adminGroup := v1.Group("/admin")
			adminGroup.Use(middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermUsersManage))
			{
				adminGroup.GET("/users", userHandler.AdminListUsers)
				adminGroup.GET("/users/:id", userHandler.AdminGetUser)
				adminGroup.PATCH("/users/:id", userHandler.AdminUpdateUser)
				adminGroup.DELETE("/users/:id", userHandler.AdminDeleteUser)
			}

			orgRepo := postgres.NewOrganizationRepository(s.deps.DB)
			orgMemberRepo := postgres.NewOrganizationMemberRepository(s.deps.DB)
			orgInviteRepo := postgres.NewOrganizationInviteRepository(s.deps.DB)
			orgService := service.NewOrganizationService(
				orgRepo,
				orgMemberRepo,
				orgInviteRepo,
				userRepo,
				s.cfg.IsDevelopment(),
			)
			orgHandler := handlers.NewOrganizationHandler(orgService)

			orgsGroup := v1.Group("/organizations")
			orgsGroup.Use(middleware.BearerAuth(tokenManager))
			{
				orgsGroup.POST("", middleware.RequirePermission(rbac.PermOrgsCreate), orgHandler.CreateOrganization)
				orgsGroup.GET("", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.ListMyOrganizations)
				orgsGroup.POST("/invites/accept", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.AcceptInvite)
				orgsGroup.GET("/:id", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.GetOrganization)
				orgsGroup.PATCH("/:id", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.UpdateOrganization)
				orgsGroup.DELETE("/:id", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.DeleteOrganization)
				orgsGroup.GET("/:id/members", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.ListMembers)
				orgsGroup.PATCH("/:id/members/:userId", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.UpdateMemberRole)
				orgsGroup.DELETE("/:id/members/:userId", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.RemoveMember)
				orgsGroup.POST("/:id/invites", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.CreateInvite)
				orgsGroup.GET("/:id/invites", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.ListInvites)
				orgsGroup.DELETE("/:id/invites/:inviteId", middleware.RequirePermission(rbac.PermOrgsRead), orgHandler.RevokeInvite)
			}

			categoryRepo := postgres.NewCategoryRepository(s.deps.DB)
			tagRepo := postgres.NewTagRepository(s.deps.DB)
			courseRepo := postgres.NewCourseRepository(s.deps.DB)
			courseService := service.NewCourseService(courseRepo, categoryRepo, tagRepo, orgRepo, orgMemberRepo)
			courseHandler := handlers.NewCourseHandler(courseService)

			moduleRepo := postgres.NewModuleRepository(s.deps.DB)
			lessonRepo := postgres.NewLessonRepository(s.deps.DB)
			blockRepo := postgres.NewContentBlockRepository(s.deps.DB)
			enrollmentRepo := postgres.NewEnrollmentRepository(s.deps.DB)
			inviteCodeRepo := postgres.NewEnrollmentInviteCodeRepository(s.deps.DB)
			enrollmentService := service.NewEnrollmentService(courseRepo, enrollmentRepo, inviteCodeRepo)
			enrollmentHandler := handlers.NewEnrollmentHandler(enrollmentService)
			curriculumService := service.NewCurriculumService(courseRepo, moduleRepo, lessonRepo, blockRepo, enrollmentService)
			curriculumHandler := handlers.NewCurriculumHandler(curriculumService)

			lessonProgressRepo := postgres.NewLessonProgressRepository(s.deps.DB)
			courseProgressRepo := postgres.NewCourseProgressRepository(s.deps.DB)
			progressService := service.NewProgressService(
				moduleRepo, lessonRepo, enrollmentRepo, lessonProgressRepo, courseProgressRepo,
			)
			progressHandler := handlers.NewProgressHandler(progressService)

			v1.GET("/categories", courseHandler.ListCategories)
			v1.POST("/categories",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermCoursesManage),
				courseHandler.CreateCategory,
			)

			v1.GET("/me/enrollments",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermCoursesRead),
				enrollmentHandler.ListMyEnrollments,
			)
			v1.GET("/me/progress",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermCoursesRead),
				progressHandler.ListMyProgress,
			)

			coursesGroup := v1.Group("/courses")
			{
				coursesGroup.GET("", middleware.OptionalBearerAuth(tokenManager), courseHandler.ListCourses)
				coursesGroup.GET("/:id", middleware.OptionalBearerAuth(tokenManager), courseHandler.GetCourse)
				coursesGroup.POST("", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), courseHandler.CreateCourse)
				coursesGroup.PATCH("/:id", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), courseHandler.UpdateCourse)
				coursesGroup.PUT("/:id/tags", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), courseHandler.SetCourseTags)
				coursesGroup.POST("/:id/publish", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), courseHandler.PublishCourse)
				coursesGroup.POST("/:id/archive", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), courseHandler.ArchiveCourse)
				coursesGroup.DELETE("/:id", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), courseHandler.DeleteCourse)

				coursesGroup.GET("/:id/curriculum", middleware.OptionalBearerAuth(tokenManager), curriculumHandler.GetCurriculum)
				coursesGroup.POST("/:id/modules", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.CreateModule)
				coursesGroup.PUT("/:id/modules/reorder", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.ReorderModules)

				coursesGroup.POST("/:id/enroll", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), enrollmentHandler.Enroll)
				coursesGroup.DELETE("/:id/enroll", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), enrollmentHandler.Unenroll)
				coursesGroup.GET("/:id/enrollments", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), enrollmentHandler.ListCourseEnrollments)
				coursesGroup.GET("/:id/access", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), enrollmentHandler.GetCourseAccess)
				coursesGroup.PATCH("/:id/enrollment-settings", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), enrollmentHandler.UpdateEnrollmentSettings)
				coursesGroup.POST("/:id/invite-codes", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), enrollmentHandler.CreateInviteCode)
				coursesGroup.GET("/:id/invite-codes", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), enrollmentHandler.ListInviteCodes)
				coursesGroup.DELETE("/:id/invite-codes/:codeId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), enrollmentHandler.RevokeInviteCode)

				coursesGroup.GET("/:id/progress", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), progressHandler.GetCourseProgress)
			}

			modulesGroup := v1.Group("/modules")
			{
				modulesGroup.PATCH("/:moduleId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.UpdateModule)
				modulesGroup.DELETE("/:moduleId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.DeleteModule)
				modulesGroup.POST("/:moduleId/lessons", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.CreateLesson)
				modulesGroup.PUT("/:moduleId/lessons/reorder", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.ReorderLessons)
			}

			lessonsGroup := v1.Group("/lessons")
			{
				lessonsGroup.PATCH("/:lessonId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.UpdateLesson)
				lessonsGroup.POST("/:lessonId/publish", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.PublishLesson)
				lessonsGroup.POST("/:lessonId/unpublish", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.UnpublishLesson)
				lessonsGroup.DELETE("/:lessonId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.DeleteLesson)
				lessonsGroup.POST("/:lessonId/blocks", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.CreateBlock)
				lessonsGroup.PUT("/:lessonId/blocks/reorder", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.ReorderBlocks)
				lessonsGroup.PUT("/:lessonId/progress", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), progressHandler.UpsertLessonProgress)
				lessonsGroup.GET("/:lessonId/progress", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), progressHandler.GetLessonProgress)
			}

			blocksGroup := v1.Group("/blocks")
			{
				blocksGroup.PATCH("/:blockId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.UpdateBlock)
				blocksGroup.DELETE("/:blockId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), curriculumHandler.DeleteBlock)
			}
		}
	}
}

// Run starts the HTTP server and blocks until a shutdown signal is received.
func (s *Server) Run() error {
	errCh := make(chan error, 1)

	go func() {
		s.log.Info().
			Str("addr", "0.0.0.0:"+s.cfg.ServerPort).
			Str("env", s.cfg.Env).
			Str("version", s.cfg.AppVersion).
			Bool("postgres", s.deps.DB != nil).
			Msg("server starting")

		if err := s.http.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("listen: %w", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		return err
	case sig := <-quit:
		s.log.Info().Str("signal", sig.String()).Msg("shutdown signal received")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.http.Shutdown(ctx); err != nil {
		return fmt.Errorf("forced shutdown: %w", err)
	}

	if s.deps.DB != nil {
		s.deps.DB.Close()
		s.log.Info().Msg("postgres pool closed")
	}

	s.log.Info().Msg("server stopped")
	return nil
}
