package main

import (
	"context"
	"os"
	"time"

	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/asnakech/asnakech-servers/internal/database"
	"github.com/asnakech/asnakech-servers/internal/logging"
	"github.com/asnakech/asnakech-servers/internal/server"
)

// @title           Asnakech School API
// @version         0.1.0
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
	cfg, err := config.Load()
	if err != nil {
		_, _ = os.Stderr.WriteString("config error: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger := logging.Setup(cfg.Env, cfg.LogLevel)

	deps := server.Dependencies{}
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		pool, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
		cancel()
		if err != nil {
			logger.Fatal().Err(err).Msg("postgres connection failed")
		}
		deps.DB = pool
		logger.Info().Msg("postgres pool ready")
	} else {
		logger.Warn().Msg("DATABASE_URL not set — database features disabled")
	}

	srv := server.New(cfg, logger, deps)
	if err := srv.Run(); err != nil {
		logger.Fatal().Err(err).Msg("server error")
	}
}
