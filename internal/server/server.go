package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/asnakech/asnakech-servers/internal/handlers"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/platform/ready"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
)

// Server wraps the HTTP stack for the Asnakech API.
type Server struct {
	cfg    *config.Config
	log    zerolog.Logger
	router *gin.Engine
	http   *http.Server
}

func New(cfg *config.Config, logger zerolog.Logger) *Server {
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
		DatabaseURL: s.cfg.DatabaseURL,
		RedisURL:    s.cfg.RedisURL,
		S3Endpoint:  s.cfg.S3Endpoint,
	}

	healthHandler := handlers.NewHealthHandler(s.cfg.AppVersion, checker)
	welcomeHandler := handlers.NewWelcomeHandler(s.cfg.AppVersion)

	s.router.GET("/health", healthHandler.HealthCheck)
	s.router.GET("/ready", healthHandler.ReadyCheck)

	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/", welcomeHandler.Welcome)
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

	s.log.Info().Msg("server stopped")
	return nil
}
