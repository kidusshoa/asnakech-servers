package config

import (
	"os"
	"strconv"
)

type Config struct {
	ServerPort string
	Env        string
}

func Load() *Config {
	port := getEnv("PORT", "8080")
	env := getEnv("ENV", "development")

	return &Config{
		ServerPort: port,
		Env:        env,
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
