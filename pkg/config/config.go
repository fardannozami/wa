package config

import "os"

type Config struct {
	Port        string
	DatabaseURL string
	JWTSecret   string
	SessionDir  string

	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string

	RedisURL string

	RateLimitPerMinute int
	MinDelaySeconds    int
	MaxDelaySeconds    int
	LogLevel           string
}

func Load() *Config {
	return &Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/wa_saas?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		SessionDir:  getEnv("SESSION_DIR", "./sessions"),

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),

		RedisURL: getEnv("REDIS_URL", "redis://localhost:6379"),

		RateLimitPerMinute: 20,
		MinDelaySeconds:    2,
		MaxDelaySeconds:    10,
		LogLevel:           getEnv("LOG_LEVEL", "info"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
