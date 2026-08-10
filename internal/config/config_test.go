package config_test

import (
	"testing"
	"time"

	"github.com/asnakech/asnakech-servers/internal/config"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("PORT", "8080")
	t.Setenv("ENV", "development")
	t.Setenv("APP_VERSION", "0.1.0")
	t.Setenv("LOG_LEVEL", "info")
	t.Setenv("CORS_ALLOWED_ORIGINS", "*")
	t.Setenv("CORS_ALLOW_CREDENTIALS", "false")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("REDIS_URL", "")
	t.Setenv("JWT_SECRET", "")
	t.Setenv("JWT_ACCESS_TTL", "15m")
	t.Setenv("JWT_REFRESH_TTL", "168h")
	t.Setenv("S3_ENDPOINT", "")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ServerPort != "8080" {
		t.Fatalf("port %s", cfg.ServerPort)
	}
	if cfg.LogLevel != "info" {
		t.Fatalf("log level %s", cfg.LogLevel)
	}
	if cfg.JWTAccessTTL != 15*time.Minute {
		t.Fatalf("access ttl %s", cfg.JWTAccessTTL)
	}
}

func TestLoadRequiresJWTInProduction(t *testing.T) {
	t.Setenv("ENV", "production")
	t.Setenv("JWT_SECRET", "")

	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when JWT_SECRET missing in production")
	}
}
