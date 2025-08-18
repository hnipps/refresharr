package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the application
type Config struct {
	Sonarr SonarrConfig
	Radarr RadarrConfig

	// Global settings
	RequestTimeout  time.Duration
	RequestDelay    time.Duration
	ConcurrentLimit int
	LogLevel        string
	DryRun          bool
}

// SonarrConfig holds Sonarr-specific configuration
type SonarrConfig struct {
	URL    string
	APIKey string
}

// RadarrConfig holds Radarr-specific configuration (for future use)
type RadarrConfig struct {
	URL    string
	APIKey string
}

// LoadConfig loads configuration from environment variables with sensible defaults
func LoadConfig() (*Config, error) {
	// Load .env file if it exists (ignore errors - .env file is optional)
	_ = godotenv.Load()

	config := &Config{
		// Default values
		RequestTimeout:  30 * time.Second,
		RequestDelay:    500 * time.Millisecond,
		ConcurrentLimit: 5,
		LogLevel:        "INFO",
		DryRun:          false,
	}

	// Load Sonarr configuration
	config.Sonarr.URL = getEnvOrDefault("SONARR_URL", "http://127.0.0.1:8989")
	config.Sonarr.APIKey = getEnvOrDefault("SONARR_API_KEY", "")

	// Load Radarr configuration (for future use)
	config.Radarr.URL = getEnvOrDefault("RADARR_URL", "http://127.0.0.1:7878")
	config.Radarr.APIKey = getEnvOrDefault("RADARR_API_KEY", "")

	// Load global settings
	if timeoutStr := os.Getenv("REQUEST_TIMEOUT"); timeoutStr != "" {
		if timeout, err := time.ParseDuration(timeoutStr); err == nil {
			config.RequestTimeout = timeout
		}
	}

	if delayStr := os.Getenv("REQUEST_DELAY"); delayStr != "" {
		if delay, err := time.ParseDuration(delayStr); err == nil {
			config.RequestDelay = delay
		}
	}

	if limitStr := os.Getenv("CONCURRENT_LIMIT"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			config.ConcurrentLimit = limit
		}
	}

	config.LogLevel = getEnvOrDefault("LOG_LEVEL", "INFO")
	config.DryRun = getEnvBool("DRY_RUN", false)

	// Validate required configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// For now, we only validate Sonarr since that's what we're implementing first
	if c.Sonarr.URL == "" {
		return fmt.Errorf("SONARR_URL is required")
	}

	if c.Sonarr.APIKey == "" {
		return fmt.Errorf("SONARR_API_KEY is required")
	}

	if c.RequestTimeout <= 0 {
		return fmt.Errorf("REQUEST_TIMEOUT must be positive")
	}

	if c.ConcurrentLimit <= 0 {
		return fmt.Errorf("CONCURRENT_LIMIT must be positive")
	}

	return nil
}

// getEnvOrDefault returns the environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvBool returns the environment variable as a boolean or a default value
func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	}
	return defaultValue
}
