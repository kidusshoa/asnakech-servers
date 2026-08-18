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
	"github.com/asnakech/asnakech-servers/internal/live"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/notify"
	"github.com/asnakech/asnakech-servers/internal/payment"
	"github.com/asnakech/asnakech-servers/internal/platform/metrics"
	"github.com/asnakech/asnakech-servers/internal/platform/ready"
	"github.com/asnakech/asnakech-servers/internal/rbac"
	"github.com/asnakech/asnakech-servers/internal/repository/postgres"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/asnakech/asnakech-servers/internal/storage"
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
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		logger.Warn().Err(err).Msg("trusted proxy config ignored")
	}
	router.Use(gin.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Locale())
	router.Use(middleware.SecurityHeaders(middleware.SecurityHeadersConfig{
		HSTS: cfg.SecurityHSTS,
	}))
	if cfg.MetricsEnabled {
		metrics.Default.SetVersion(cfg.AppVersion)
		router.Use(middleware.HTTPMetrics(metrics.Default))
	}
	router.Use(middleware.ZerologRequest(logger))
	router.Use(middleware.CORS(middleware.CORSConfig{
		AllowedOrigins:   cfg.CORSOrigins,
		AllowCredentials: cfg.CORSCredentials,
	}))
	if cfg.RateLimitGlobalRPS > 0 {
		globalLimiter := middleware.NewIPRateLimiter(cfg.RateLimitGlobalRPS, cfg.RateLimitGlobalBurst)
		router.Use(globalLimiter.LimitExcept(
			"/health", "/ready", "/metrics", "/swagger",
		))
	}

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
	if s.cfg.MetricsEnabled {
		metricsHandler := handlers.NewMetricsHandler(metrics.Default)
		s.router.GET("/metrics", metricsHandler.Prometheus)
	}
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

			objectStore, err := storage.NewS3Store(s.cfg)
			if err != nil {
				s.log.Error().Err(err).Msg("s3 client init failed — media uploads disabled")
				objectStore = storage.NoopStore{}
			}
			mediaRepo := postgres.NewMediaRepository(s.deps.DB)
			courseRepoEarly := postgres.NewCourseRepository(s.deps.DB)
			mediaService := service.NewMediaService(
				mediaRepo, courseRepoEarly, objectStore, storage.SkipScanner{}, s.cfg.MediaPresignTTL,
			)
			mediaHandler := handlers.NewMediaHandler(mediaService)

			userService := service.NewUserService(userRepo, roleRepo)
			userHandler := handlers.NewUserHandler(userService, mediaService)

			analyticsRepo := postgres.NewAnalyticsRepository(s.deps.DB)
			analyticsService := service.NewAnalyticsService(courseRepoEarly, analyticsRepo)
			analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)

			searchRepo := postgres.NewSearchRepository(s.deps.DB)
			parentLinkRepo := postgres.NewParentLinkRepository(s.deps.DB)
			discoveryService := service.NewDiscoveryService(searchRepo, s.cfg)
			parentService := service.NewParentService(userRepo, parentLinkRepo)
			discoveryHandler := handlers.NewDiscoveryHandler(discoveryService, parentService)

			authLimiter := middleware.NewIPRateLimiter(s.cfg.RateLimitAuthRPS, s.cfg.RateLimitAuthBurst)
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

			v1.GET("/me/media", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermProfileRead), mediaHandler.ListMyMedia)
			mediaGroup := v1.Group("/media")
			mediaGroup.Use(middleware.BearerAuth(tokenManager))
			{
				mediaGroup.POST("/uploads", middleware.RequirePermission(rbac.PermProfileWrite), mediaHandler.CreateUpload)
				mediaGroup.POST("/:id/complete", middleware.RequirePermission(rbac.PermProfileWrite), mediaHandler.CompleteUpload)
				mediaGroup.GET("/:id", middleware.RequirePermission(rbac.PermProfileRead), mediaHandler.GetMedia)
				mediaGroup.DELETE("/:id", middleware.RequirePermission(rbac.PermProfileWrite), mediaHandler.DeleteMedia)
				mediaGroup.POST("/:id/scan-result", middleware.RequirePermission(rbac.PermUsersManage), mediaHandler.ApplyScanResult)
			}

			adminGroup := v1.Group("/admin")
			adminGroup.Use(middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermUsersManage))
			{
				adminGroup.GET("/users", userHandler.AdminListUsers)
				adminGroup.GET("/users/:id", userHandler.AdminGetUser)
				adminGroup.PATCH("/users/:id", userHandler.AdminUpdateUser)
				adminGroup.DELETE("/users/:id", userHandler.AdminDeleteUser)
				adminGroup.GET("/overview", analyticsHandler.AdminOverview)
				adminGroup.GET("/reports/enrollments", analyticsHandler.EnrollmentReport)
				adminGroup.GET("/reports/revenue", analyticsHandler.RevenueReport)
				adminGroup.GET("/reports/users", analyticsHandler.UserGrowthReport)
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

			quizRepo := postgres.NewQuizRepository(s.deps.DB)
			questionRepo := postgres.NewQuizQuestionRepository(s.deps.DB)
			attemptRepo := postgres.NewQuizAttemptRepository(s.deps.DB)
			answerRepo := postgres.NewQuizAttemptAnswerRepository(s.deps.DB)
			assignmentRepo := postgres.NewAssignmentRepository(s.deps.DB)
			submissionRepo := postgres.NewAssignmentSubmissionRepository(s.deps.DB)
			assessmentService := service.NewAssessmentService(
				courseRepo, enrollmentRepo, quizRepo, questionRepo, attemptRepo, answerRepo, assignmentRepo, submissionRepo,
			)
			assessmentHandler := handlers.NewAssessmentHandler(assessmentService)

			liveSessionRepo := postgres.NewLiveSessionRepository(s.deps.DB)
			attendanceRepo := postgres.NewSessionAttendanceRepository(s.deps.DB)
			liveProviders := live.NewRegistry(s.cfg)
			liveService := service.NewLiveService(courseRepo, enrollmentRepo, liveSessionRepo, attendanceRepo, liveProviders)
			liveHandler := handlers.NewLiveHandler(liveService)

			announcementRepo := postgres.NewAnnouncementRepository(s.deps.DB)
			threadRepo := postgres.NewDiscussionThreadRepository(s.deps.DB)
			postRepo := postgres.NewDiscussionPostRepository(s.deps.DB)
			dmConvRepo := postgres.NewDMConversationRepository(s.deps.DB)
			dmMsgRepo := postgres.NewDMMessageRepository(s.deps.DB)
			notificationRepo := postgres.NewNotificationRepository(s.deps.DB)
			outbox := notify.NewOutboxWriter(notificationRepo)
			commsService := service.NewCommunicationService(
				courseRepo, enrollmentRepo, userRepo,
				announcementRepo, threadRepo, postRepo, dmConvRepo, dmMsgRepo, notificationRepo, outbox,
			)
			commsHandler := handlers.NewCommunicationHandler(commsService)

			certRepo := postgres.NewCertificateRepository(s.deps.DB)
			certService := service.NewCertificateService(
				courseRepo, enrollmentRepo, userRepo, courseProgressRepo, certRepo,
				quizRepo, assignmentRepo, attemptRepo, submissionRepo,
			)
			certHandler := handlers.NewCertificateHandler(certService)

			orderRepo := postgres.NewOrderRepository(s.deps.DB)
			couponRepo := postgres.NewCouponRepository(s.deps.DB)
			webhookRepo := postgres.NewPaymentWebhookRepository(s.deps.DB)
			paymentProviders := payment.NewRegistry(s.cfg)
			paymentService := service.NewPaymentService(
				courseRepo, orderRepo, couponRepo, webhookRepo, enrollmentService, paymentProviders,
			)
			paymentHandler := handlers.NewPaymentHandler(paymentService)

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
			v1.GET("/me/calendar",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermCoursesRead),
				liveHandler.ListCalendar,
			)
			v1.GET("/me/conversations",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermProfileRead),
				commsHandler.ListConversations,
			)
			v1.GET("/me/notifications",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermProfileRead),
				commsHandler.ListNotifications,
			)
			v1.POST("/me/notifications/read-all",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermProfileRead),
				commsHandler.MarkAllNotificationsRead,
			)
			v1.GET("/me/certificates",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermCoursesRead),
				certHandler.ListMyCertificates,
			)
			v1.GET("/me/orders",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermCoursesRead),
				paymentHandler.ListMyOrders,
			)
			v1.GET("/me/recommendations",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermCoursesRead),
				discoveryHandler.Recommendations,
			)
			v1.GET("/me/children",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermProfileRead),
				discoveryHandler.ListChildren,
			)
			v1.POST("/me/children/link",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermProfileRead),
				discoveryHandler.LinkChild,
			)
			v1.DELETE("/me/children/:studentId",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermProfileRead),
				discoveryHandler.UnlinkChild,
			)
			v1.GET("/me/transcript",
				middleware.BearerAuth(tokenManager),
				middleware.RequirePermission(rbac.PermCoursesRead),
				certHandler.MyTranscript,
			)
			v1.GET("/certificates/verify/:code", certHandler.VerifyCertificate)

			v1.GET("/search", discoveryHandler.Search)
			v1.GET("/features", discoveryHandler.FeatureFlags)
			v1.GET("/locales", discoveryHandler.Locales)
			v1.GET("/i18n/messages", discoveryHandler.Messages)

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

				coursesGroup.POST("/:id/quizzes", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.CreateQuiz)
				coursesGroup.GET("/:id/quizzes", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.ListQuizzes)
				coursesGroup.POST("/:id/assignments", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.CreateAssignment)
				coursesGroup.GET("/:id/assignments", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.ListAssignments)
				coursesGroup.GET("/:id/gradebook", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.Gradebook)

				coursesGroup.POST("/:id/sessions", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), liveHandler.CreateSession)
				coursesGroup.GET("/:id/sessions", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), liveHandler.ListCourseSessions)

				coursesGroup.POST("/:id/announcements", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), commsHandler.CreateAnnouncement)
				coursesGroup.GET("/:id/announcements", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), commsHandler.ListAnnouncements)
				coursesGroup.POST("/:id/threads", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), commsHandler.CreateThread)
				coursesGroup.GET("/:id/threads", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), commsHandler.ListThreads)

				coursesGroup.POST("/:id/checkout", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), paymentHandler.CreateCheckout)
				coursesGroup.GET("/:id/orders", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), paymentHandler.ListCourseOrders)
				coursesGroup.GET("/:id/analytics", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), analyticsHandler.CourseAnalytics)

				coursesGroup.POST("/:id/certificate", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), certHandler.IssueCertificate)
				coursesGroup.GET("/:id/certificates", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), certHandler.ListCourseCertificates)
				coursesGroup.GET("/:id/transcript/:userId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), certHandler.UserCourseTranscript)
			}

			ordersGroup := v1.Group("/orders")
			ordersGroup.Use(middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead))
			{
				ordersGroup.GET("/:orderId", paymentHandler.GetOrder)
				ordersGroup.POST("/:orderId/confirm", paymentHandler.ConfirmOrder)
				ordersGroup.POST("/:orderId/refund", middleware.RequirePermission(rbac.PermCoursesWrite), paymentHandler.RefundOrder)
			}

			couponsGroup := v1.Group("/coupons")
			couponsGroup.Use(middleware.BearerAuth(tokenManager))
			{
				couponsGroup.POST("", middleware.RequirePermission(rbac.PermCoursesManage), paymentHandler.CreateCoupon)
				couponsGroup.GET("", middleware.RequirePermission(rbac.PermCoursesRead), paymentHandler.ListCoupons)
				couponsGroup.POST("/:couponId/revoke", middleware.RequirePermission(rbac.PermCoursesManage), paymentHandler.RevokeCoupon)
			}

			v1.POST("/webhooks/payments/:provider", paymentHandler.PaymentWebhook)

			certificatesGroup := v1.Group("/certificates")
			certificatesGroup.Use(middleware.BearerAuth(tokenManager))
			{
				certificatesGroup.GET("/:certificateId", middleware.RequirePermission(rbac.PermCoursesRead), certHandler.GetCertificate)
				certificatesGroup.GET("/:certificateId/download", middleware.RequirePermission(rbac.PermCoursesRead), certHandler.DownloadCertificate)
				certificatesGroup.POST("/:certificateId/revoke", middleware.RequirePermission(rbac.PermCoursesWrite), certHandler.RevokeCertificate)
			}

			announcementsGroup := v1.Group("/announcements")
			announcementsGroup.Use(middleware.BearerAuth(tokenManager))
			{
				announcementsGroup.GET("/:announcementId", middleware.RequirePermission(rbac.PermCoursesRead), commsHandler.GetAnnouncement)
				announcementsGroup.PATCH("/:announcementId", middleware.RequirePermission(rbac.PermCoursesWrite), commsHandler.UpdateAnnouncement)
				announcementsGroup.POST("/:announcementId/publish", middleware.RequirePermission(rbac.PermCoursesWrite), commsHandler.PublishAnnouncement)
				announcementsGroup.DELETE("/:announcementId", middleware.RequirePermission(rbac.PermCoursesWrite), commsHandler.DeleteAnnouncement)
			}

			threadsGroup := v1.Group("/threads")
			threadsGroup.Use(middleware.BearerAuth(tokenManager))
			{
				threadsGroup.GET("/:threadId", middleware.RequirePermission(rbac.PermCoursesRead), commsHandler.GetThread)
				threadsGroup.POST("/:threadId/lock", middleware.RequirePermission(rbac.PermCoursesWrite), commsHandler.LockThread)
				threadsGroup.POST("/:threadId/posts", middleware.RequirePermission(rbac.PermCoursesRead), commsHandler.CreatePost)
				threadsGroup.GET("/:threadId/posts", middleware.RequirePermission(rbac.PermCoursesRead), commsHandler.ListPosts)
			}

			postsGroup := v1.Group("/posts")
			postsGroup.Use(middleware.BearerAuth(tokenManager))
			{
				postsGroup.PATCH("/:postId", middleware.RequirePermission(rbac.PermCoursesRead), commsHandler.UpdatePost)
				postsGroup.DELETE("/:postId", middleware.RequirePermission(rbac.PermCoursesRead), commsHandler.DeletePost)
			}

			conversationsGroup := v1.Group("/conversations")
			conversationsGroup.Use(middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermProfileRead))
			{
				conversationsGroup.POST("", commsHandler.StartConversation)
				conversationsGroup.GET("/:id/messages", commsHandler.ListMessages)
				conversationsGroup.POST("/:id/messages", commsHandler.SendMessage)
				conversationsGroup.POST("/:id/read", commsHandler.MarkConversationRead)
			}

			notificationsGroup := v1.Group("/notifications")
			notificationsGroup.Use(middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermProfileRead))
			{
				notificationsGroup.POST("/:id/read", commsHandler.MarkNotificationRead)
			}

			sessionsGroup := v1.Group("/sessions")
			sessionsGroup.Use(middleware.BearerAuth(tokenManager))
			{
				sessionsGroup.GET("/:sessionId", middleware.RequirePermission(rbac.PermCoursesRead), liveHandler.GetSession)
				sessionsGroup.PATCH("/:sessionId", middleware.RequirePermission(rbac.PermCoursesWrite), liveHandler.UpdateSession)
				sessionsGroup.POST("/:sessionId/publish", middleware.RequirePermission(rbac.PermCoursesWrite), liveHandler.PublishSession)
				sessionsGroup.POST("/:sessionId/complete", middleware.RequirePermission(rbac.PermCoursesWrite), liveHandler.CompleteSession)
				sessionsGroup.POST("/:sessionId/cancel", middleware.RequirePermission(rbac.PermCoursesWrite), liveHandler.CancelSession)
				sessionsGroup.POST("/:sessionId/generate-link", middleware.RequirePermission(rbac.PermCoursesWrite), liveHandler.GenerateLink)
				sessionsGroup.GET("/:sessionId/join", middleware.RequirePermission(rbac.PermCoursesRead), liveHandler.JoinSession)
				sessionsGroup.GET("/:sessionId/attendance", middleware.RequirePermission(rbac.PermCoursesWrite), liveHandler.ListAttendance)
				sessionsGroup.PUT("/:sessionId/attendance/:userId", middleware.RequirePermission(rbac.PermCoursesWrite), liveHandler.MarkAttendance)
				sessionsGroup.POST("/:sessionId/attendance/check-in", middleware.RequirePermission(rbac.PermCoursesRead), liveHandler.CheckIn)
			}

			quizzesGroup := v1.Group("/quizzes")
			{
				quizzesGroup.GET("/:quizId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.GetQuiz)
				quizzesGroup.PATCH("/:quizId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.UpdateQuiz)
				quizzesGroup.POST("/:quizId/publish", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.PublishQuiz)
				quizzesGroup.POST("/:quizId/questions", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.AddQuestion)
				quizzesGroup.PUT("/:quizId/questions/reorder", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.ReorderQuestions)
				quizzesGroup.POST("/:quizId/attempts", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.StartAttempt)
				quizzesGroup.GET("/:quizId/attempts", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.ListMyAttempts)
			}

			questionsGroup := v1.Group("/questions")
			{
				questionsGroup.PATCH("/:questionId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.UpdateQuestion)
				questionsGroup.DELETE("/:questionId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.DeleteQuestion)
			}

			attemptsGroup := v1.Group("/attempts")
			{
				attemptsGroup.GET("/:attemptId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.GetAttempt)
				attemptsGroup.PUT("/:attemptId/answers", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.SaveAnswers)
				attemptsGroup.POST("/:attemptId/submit", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.SubmitAttempt)
			}

			assignmentsGroup := v1.Group("/assignments")
			{
				assignmentsGroup.GET("/:assignmentId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.GetAssignment)
				assignmentsGroup.PATCH("/:assignmentId", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.UpdateAssignment)
				assignmentsGroup.POST("/:assignmentId/publish", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.PublishAssignment)
				assignmentsGroup.PUT("/:assignmentId/submission", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.UpsertSubmission)
				assignmentsGroup.GET("/:assignmentId/submission", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesRead), assessmentHandler.GetMySubmission)
				assignmentsGroup.GET("/:assignmentId/submissions", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.ListSubmissions)
			}

			submissionsGroup := v1.Group("/submissions")
			{
				submissionsGroup.POST("/:submissionId/grade", middleware.BearerAuth(tokenManager), middleware.RequirePermission(rbac.PermCoursesWrite), assessmentHandler.GradeSubmission)
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
