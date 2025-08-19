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
	config.Sonarr.URL = getEnvOrDefault("SONARR_URL", "")
	config.Sonarr.APIKey = getEnvOrDefault("SONARR_API_KEY", "")

	// Set default Sonarr URL if API key is provided but URL is not
	if config.Sonarr.APIKey != "" && config.Sonarr.URL == "" {
		config.Sonarr.URL = "http://127.0.0.1:8989"
	}

	// Load Radarr configuration
	config.Radarr.URL = getEnvOrDefault("RADARR_URL", "")
	config.Radarr.APIKey = getEnvOrDefault("RADARR_API_KEY", "")

	// Set default Radarr URL if API key is provided but URL is not
	if config.Radarr.APIKey != "" && config.Radarr.URL == "" {
		config.Radarr.URL = "http://127.0.0.1:7878"
	}

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
	// Check if at least one service is configured
	hasSonarr := c.Sonarr.URL != "" && c.Sonarr.APIKey != ""
	hasRadarr := c.Radarr.URL != "" && c.Radarr.APIKey != ""

	if !hasSonarr && !hasRadarr {
		return fmt.Errorf("at least one service must be configured (Sonarr or Radarr)")
	}

	// Validate Sonarr configuration if provided
	if c.Sonarr.URL != "" && c.Sonarr.APIKey == "" {
		return fmt.Errorf("SONARR_API_KEY is required when SONARR_URL is provided")
	}

	if c.Sonarr.APIKey != "" && c.Sonarr.URL == "" {
		return fmt.Errorf("SONARR_URL is required when SONARR_API_KEY is provided")
	}

	// Validate Radarr configuration if provided
	if c.Radarr.URL != "" && c.Radarr.APIKey == "" {
		return fmt.Errorf("RADARR_API_KEY is required when RADARR_URL is provided")
	}

	if c.Radarr.APIKey != "" && c.Radarr.URL == "" {
		return fmt.Errorf("RADARR_URL is required when RADARR_API_KEY is provided")
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
