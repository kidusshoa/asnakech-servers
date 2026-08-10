package main

import (
	"context"
	"os"
	"time"

	"github.com/asnakech/asnakech-servers/internal/config"
	"github.com/asnakech/asnakech-servers/internal/database"
	"github.com/asnakech/asnakech-servers/internal/logging"
	"github.com/asnakech/asnakech-servers/internal/server"

	_ "github.com/asnakech/asnakech-servers/docs/swagger"
)

// @title           Asnakech School API
// @version         0.1.0
// @description     Backend API for the Asnakech online education platform.
// @description     All JSON responses use a shared envelope: `{ success, data, error, meta }`.
// @description     Every response includes an `X-Request-ID` header.
// @termsOfService  http://www.asnakech.com/terms

// @contact.name   API Support
// @contact.url    http://www.asnakech.com/support
// @contact.email  support@asnakech.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description JWT access token. Format: Bearer {token}

//go:generate go run github.com/swaggo/swag/cmd/swag@v1.16.4 init -g main.go -d .,../../internal/handlers,../../internal/response,../../internal/platform/ready -o ../../docs/swagger --parseInternal
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
