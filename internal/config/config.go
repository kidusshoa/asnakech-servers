package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	ServerPort       string
	Env              string
	AppVersion       string
	CORSOrigins      []string
	CORSCredentials  bool
}

func Load() *Config {
	return &Config{
		ServerPort:      getEnv("PORT", "8080"),
		Env:             getEnv("ENV", "development"),
		AppVersion:      getEnv("APP_VERSION", "0.1.0"),
		CORSOrigins:     splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "*")),
		CORSCredentials: getEnvAsBool("CORS_ALLOW_CREDENTIALS", false),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
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
