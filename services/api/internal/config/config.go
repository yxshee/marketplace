package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds runtime configuration values for the API service.
type Config struct {
	Port                string
	Environment         string
	JWTSecret           string
	JWTIssuer           string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	SuperAdminEmails    string
	SupportEmails       string
	FinanceEmails       string
	CatalogModEmails    string
	DefaultCommission   int32
	StripeMode          string
	StripeSecretKey     string
	StripeWebhookSecret string
}

func getenvOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func getenvIntOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getenvDurationSeconds(key string, fallbackSeconds int) time.Duration {
	return time.Duration(getenvIntOrDefault(key, fallbackSeconds)) * time.Second
}

// Load reads config from env vars with safe defaults for local development.
func Load() Config {
	return Config{
		Port:                getenvOrDefault("API_PORT", "8080"),
		Environment:         getenvOrDefault("API_ENV", "development"),
		JWTSecret:           getenvOrDefault("API_JWT_SECRET", "local-dev-jwt-secret-change-me"),
		JWTIssuer:           getenvOrDefault("API_JWT_ISSUER", "marketplace-api"),
		AccessTokenTTL:      getenvDurationSeconds("API_ACCESS_TOKEN_TTL_SECONDS", 900),
		RefreshTokenTTL:     getenvDurationSeconds("API_REFRESH_TOKEN_TTL_SECONDS", 1209600),
		SuperAdminEmails:    getenvOrDefault("API_SUPER_ADMIN_EMAILS", ""),
		SupportEmails:       getenvOrDefault("API_SUPPORT_EMAILS", ""),
		FinanceEmails:       getenvOrDefault("API_FINANCE_EMAILS", ""),
		CatalogModEmails:    getenvOrDefault("API_CATALOG_MOD_EMAILS", ""),
		DefaultCommission:   int32(getenvIntOrDefault("API_DEFAULT_COMMISSION_BPS", 1000)),
		StripeMode:          getenvOrDefault("API_STRIPE_MODE", "mock"),
		StripeSecretKey:     getenvOrDefault("API_STRIPE_SECRET_KEY", ""),
		StripeWebhookSecret: getenvOrDefault("API_STRIPE_WEBHOOK_SECRET", "whsec_dev_local"),
	}
}
