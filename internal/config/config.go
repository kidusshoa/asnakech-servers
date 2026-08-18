package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config holds process configuration loaded from the environment.
type Config struct {
	ServerPort      string
	Env             string
	AppVersion      string
	LogLevel        string
	CORSOrigins     []string
	CORSCredentials bool

	DatabaseURL string
	RedisURL    string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	S3Endpoint      string
	S3Region        string
	S3Bucket        string
	S3AccessKey     string
	S3SecretKey     string
	S3UsePathStyle  bool
	S3PublicBaseURL string
	MediaPresignTTL time.Duration

	LiveDefaultProvider string
	LiveJitsiBaseURL    string

	PaymentDefaultProvider string
	PaymentWebhookSecret   string
	StripeSecretKey        string
	StripeWebhookSecret    string
	ChapaSecretKey         string
	ChapaWebhookSecret     string

	FeatureFlags []string

	TrustedProxies       []string
	RateLimitGlobalRPS   float64
	RateLimitGlobalBurst int
	RateLimitAuthRPS     float64
	RateLimitAuthBurst   int
	MetricsEnabled       bool
	SecurityHSTS         bool
}

// Load reads optional .env, then environment variables into Config.
// Missing .env is ignored (normal for production containers).
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		ServerPort:      getEnv("PORT", "8080"),
		Env:             getEnv("ENV", "development"),
		AppVersion:      getEnv("APP_VERSION", "0.1.0"),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		CORSOrigins:     splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "*")),
		CORSCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", false),

		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", ""),

		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTAccessTTL:  getEnvAsDuration("JWT_ACCESS_TTL", 15*time.Minute),
		JWTRefreshTTL: getEnvAsDuration("JWT_REFRESH_TTL", 168*time.Hour),

		S3Endpoint:      getEnv("S3_ENDPOINT", ""),
		S3Region:        getEnv("S3_REGION", "us-east-1"),
		S3Bucket:        getEnv("S3_BUCKET", "asnakech"),
		S3AccessKey:     getEnv("S3_ACCESS_KEY", ""),
		S3SecretKey:     getEnv("S3_SECRET_KEY", ""),
		S3UsePathStyle:  getEnvAsBool("S3_USE_PATH_STYLE", true),
		S3PublicBaseURL: getEnv("S3_PUBLIC_BASE_URL", ""),
		MediaPresignTTL: getEnvAsDuration("MEDIA_PRESIGN_TTL", 15*time.Minute),

		LiveDefaultProvider: getEnv("LIVE_DEFAULT_PROVIDER", "custom"),
		LiveJitsiBaseURL:    getEnv("LIVE_JITSI_BASE_URL", "https://meet.jit.si"),

		PaymentDefaultProvider: getEnv("PAYMENT_DEFAULT_PROVIDER", "manual"),
		PaymentWebhookSecret:   getEnv("PAYMENT_WEBHOOK_SECRET", ""),
		StripeSecretKey:        getEnv("STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret:    getEnv("STRIPE_WEBHOOK_SECRET", ""),
		ChapaSecretKey:         getEnv("CHAPA_SECRET_KEY", ""),
		ChapaWebhookSecret:     getEnv("CHAPA_WEBHOOK_SECRET", ""),

		FeatureFlags: splitFeatureFlags(getEnv("FEATURE_FLAGS", "")),

		TrustedProxies:       splitOptionalCSV(getEnv("TRUSTED_PROXIES", "")),
		RateLimitGlobalRPS:   getEnvAsFloat("RATE_LIMIT_GLOBAL_RPS", 100),
		RateLimitGlobalBurst: getEnvAsInt("RATE_LIMIT_GLOBAL_BURST", 300),
		RateLimitAuthRPS:     getEnvAsFloat("RATE_LIMIT_AUTH_RPS", 2),
		RateLimitAuthBurst:   getEnvAsInt("RATE_LIMIT_AUTH_BURST", 10),
		MetricsEnabled:       getEnvAsBool("METRICS_ENABLED", true),
		SecurityHSTS:         getEnvAsBool("SECURITY_HSTS", false),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func (c *Config) validate() error {
	if c.ServerPort == "" {
		return fmt.Errorf("PORT must not be empty")
	}
	if c.Env == "production" && c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required when ENV=production")
	}
	if c.Env == "production" && c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required when ENV=production")
	}
	return nil
}

// IsDevelopment reports whether the process runs in a non-production env.
func (c *Config) IsDevelopment() bool {
	return c.Env != "production"
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvAsFloat(key string, defaultValue float64) float64 {
	if value, exists := os.LookupEnv(key); exists {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		parsed, err := strconv.ParseBool(value)
		if err == nil {
			return parsed
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value, exists := os.LookupEnv(key); exists {
		if d, err := time.ParseDuration(value); err == nil {
			return d
		}
	}
	return defaultValue
}

func splitFeatureFlags(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}

func splitOptionalCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
