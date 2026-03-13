package config

import (
	"log"
	"os"
)

// Config holds all configuration for the application.
type Config struct {
	// Server
	Port           string
	AllowedOrigins string

	// Supabase
	SupabaseURL        string
	SupabaseAnonKey    string
	SupabaseServiceKey string

	// Redis
	RedisURL string
}

// Load reads configuration from environment variables.
func Load() *Config {
	cfg := &Config{
		Port:               getEnv("PORT", "8080"),
		AllowedOrigins:     getEnv("ALLOWED_ORIGINS", "http://localhost:5173"),
		SupabaseURL:        requireEnv("SUPABASE_URL"),
		SupabaseAnonKey:    requireEnv("SUPABASE_ANON_KEY"),
		SupabaseServiceKey: requireEnv("SUPABASE_SERVICE_KEY"),
		RedisURL:           getEnv("REDIS_URL", "redis://localhost:6379"),
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Printf("WARNING: environment variable %s is not set", key)
	}
	return v
}
