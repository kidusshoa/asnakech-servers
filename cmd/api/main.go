package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/asnakech/asnakech-servers/internal/handlers"
	"github.com/gin-gonic/gin"
)

// @title           Asnakech School API
// @version         1.0
// @description     API for Asnakech School Platform
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.asnakech.com/support
// @contact.email  support@asnakech.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /api/v1
func main() {
	// Load configuration
	cfg := config.Load()

	// Set Gin mode
	if cfg.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	// Initialize Gin router
	r := gin.Default()

	// Add middlewares
	setupMiddlewares(r)

	// Setup routes
	setupRoutes(r)

	// Create server with timeouts
	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		log.Printf("Server is running on http://0.0.0.0:%s\n", cfg.ServerPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func setupMiddlewares(r *gin.Engine) {
	// Add CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Add request ID middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("X-Request-ID", c.GetHeader("X-Request-ID"))
		c.Next()
	})

	// Add request logging
	r.Use(gin.Logger())
}

func setupRoutes(r *gin.Engine) {
	// Initialize handlers
	healthHandler := handlers.NewHealthHandler()

	// Health check endpoint
	r.GET("/health", healthHandler.HealthCheck)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		
		v1.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Welcome to Asnakech School API",
				"version": "1.0.0",
			})
		})
	}
}
