package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the MCP server
type Config struct {
	// Server configuration
	Port          int
	Host          string
	MaxUploadSize int64

	// Storage configuration
	TempDir string

	// Database configuration
	DatabaseURL string

	// Cloudflare R2 configuration
	R2AccessKeyID     string
	R2SecretAccessKey string
	R2BucketName      string
	R2Endpoint        string

	// Mistral API configuration
	MistralAPIKey    string
	MistralVibeAPIKey string

	// Voyage AI configuration
	VoyageAIAPIKey string

	// Cloudflare configuration
	CloudflareAccountID string
	CloudflareTeamDomain string
	AuthStrategy       string

	// Environment
	Environment string
}

// FromEnv loads configuration from environment variables
func FromEnv() *Config {
	return &Config{
		// Server configuration
		Port:          getIntEnv("PORT", 8080),
		Host:          getEnv("HOST", "0.0.0.0"),
		MaxUploadSize: getInt64Env("MAX_UPLOAD_SIZE", 500*1024*1024), // 500MB default

		// Storage configuration
		TempDir: getEnv("TEMP_DIR", "/tmp/godocs"),

		// Database configuration
		DatabaseURL: getEnv("NEON_CONNECTION_STRING", ""),

		// Cloudflare R2 configuration
		R2AccessKeyID:     getEnv("R2_ACCESS_KEY_ID", ""),
		R2SecretAccessKey: getEnv("R2_SECRET_ACCESS_KEY", ""),
		R2BucketName:      getEnv("R2_BUCKET_NAME", "godocs"),
		R2Endpoint:        getEnv("R2_ENDPOINT", ""),

		// Mistral API configuration
		MistralAPIKey:     getEnv("MISTRAL_API_KEY", ""),
		MistralVibeAPIKey: getEnv("MISTRAL_VIBE_API_KEY", ""),

		// Voyage AI configuration
		VoyageAIAPIKey: getEnv("VOYAGEAI_API_KEY", ""),

		// Cloudflare configuration
		CloudflareAccountID: getEnv("CLOUDFLARE_ACCOUNT_ID", ""),
		CloudflareTeamDomain: getEnv("CLOUDFLARE_TEAM_DOMAIN", ""),
		AuthStrategy:        getEnv("AUTH_STRATEGY", "cloudflare_access"),

		// Environment
		Environment: getEnv("DOPPLER_ENVIRONMENT", "dev"),
	}
}

// Helper functions

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if int64Val, err := strconv.ParseInt(value, 10, 64); err == nil {
			return int64Val
		}
	}
	return defaultValue
}
