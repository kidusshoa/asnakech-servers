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
