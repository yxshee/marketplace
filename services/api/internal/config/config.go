package config

import "os"

// Config holds runtime configuration values for the API service.
type Config struct {
	Port        string
	Environment string
}

func getenvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

// Load reads config from env vars with safe defaults for local development.
func Load() Config {
	return Config{
		Port:        getenvOrDefault("API_PORT", "8080"),
		Environment: getenvOrDefault("API_ENV", "development"),
	}
}
