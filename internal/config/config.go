package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
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
	NoReport        bool // Flag to disable terminal report output

	// CLI-specific settings
	Service     string // Service to use: "sonarr", "radarr", or "auto"
	SeriesIDs   []int  // Specific series IDs to process (empty means all)
	ShowVersion bool   // Show version and exit

	// Broken symlink handling
	AddMissingMovies bool // Whether to add movies/series to collection when found from broken symlinks
	QualityProfileID int  // Quality profile ID to use when adding movies (default: 12)
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

// LoadConfig loads configuration from environment variables and command line flags with sensible defaults
func LoadConfig() (*Config, error) {
	return LoadConfigWithFlags(nil, nil, nil, nil, nil, nil, nil, nil)
}

// LoadConfigWithFlags loads configuration with optional flag overrides (used for testing)
func LoadConfigWithFlags(dryRun, noReport, showVersion *bool, logLevel, service, sonarrURL, sonarrAPIKey *string, seriesIDs *string) (*Config, error) {
	// Create a new FlagSet for isolated flag parsing (prevents test conflicts)
	fs := flag.NewFlagSet("refresharr", flag.ContinueOnError)

	// Parse command line flags only if not provided
	if dryRun == nil || noReport == nil || showVersion == nil || logLevel == nil || service == nil || sonarrURL == nil || sonarrAPIKey == nil || seriesIDs == nil {
		var (
			dryRunFlag      = fs.Bool("dry-run", false, "Run in dry-run mode (no changes will be made)")
			noReportFlag    = fs.Bool("no-report", false, "Disable terminal report output (report will still be saved to file)")
			showVersionFlag = fs.Bool("version", false, "Show version information and exit")
			logLevelFlag    = fs.String("log-level", "", "Set log level (DEBUG, INFO, WARN, ERROR)")
			serviceFlag     = fs.String("service", "auto", "Service to use: sonarr, radarr, or auto (default: auto)")
			sonarrURLFlag   = fs.String("sonarr-url", "", "Sonarr URL (overrides SONARR_URL env var)")
			sonarrAPIFlag   = fs.String("sonarr-api-key", "", "Sonarr API key (overrides SONARR_API_KEY env var)")
			seriesIDsFlag   = fs.String("series-ids", "", "Comma-separated list of specific series IDs to process (empty means all)")
		)

		// Set custom usage function
		fs.Usage = func() {
			fmt.Fprintf(os.Stderr, "RefreshArr - Missing File Cleanup Service\n\n")
			fmt.Fprintf(os.Stderr, "Usage: %s [command] [options]\n\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "Commands:\n")
			fmt.Fprintf(os.Stderr, "  (default)     Clean up missing file references in *arr databases\n")
			fmt.Fprintf(os.Stderr, "  fix-imports   Fix stuck Sonarr imports (already imported issues)\n\n")
			fmt.Fprintf(os.Stderr, "Options:\n")
			fs.PrintDefaults()
			fmt.Fprintf(os.Stderr, "\nEnvironment Variables:\n")
			fmt.Fprintf(os.Stderr, "  SONARR_URL      Sonarr base URL (default: http://127.0.0.1:8989)\n")
			fmt.Fprintf(os.Stderr, "  SONARR_API_KEY  Sonarr API key (required)\n")
			fmt.Fprintf(os.Stderr, "  RADARR_URL      Radarr base URL (default: http://127.0.0.1:7878)\n")
			fmt.Fprintf(os.Stderr, "  RADARR_API_KEY  Radarr API key (required for Radarr)\n")
			fmt.Fprintf(os.Stderr, "  REQUEST_TIMEOUT HTTP request timeout (default: 30s)\n")
			fmt.Fprintf(os.Stderr, "  REQUEST_DELAY   Delay between API requests (default: 500ms)\n")
			fmt.Fprintf(os.Stderr, "  CONCURRENT_LIMIT Max concurrent requests (default: 5)\n")
			fmt.Fprintf(os.Stderr, "  LOG_LEVEL       Log level (default: INFO)\n")
			fmt.Fprintf(os.Stderr, "  DRY_RUN         Run in dry-run mode (default: false)\n")
			fmt.Fprintf(os.Stderr, "  ADD_MISSING_MOVIES  Add movies/series to collection when found from broken symlinks (default: false)\n")
			fmt.Fprintf(os.Stderr, "  QUALITY_PROFILE_ID  Quality profile ID for new movies (default: 12)\n")
			fmt.Fprintf(os.Stderr, "\nExamples:\n")
			fmt.Fprintf(os.Stderr, "  %s --dry-run\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "  %s --service sonarr --series-ids '123,456,789'\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "  %s --sonarr-url 'http://192.168.1.100:8989' --sonarr-api-key 'your-key'\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "  %s --log-level DEBUG\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "  %s fix-imports --dry-run\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "  %s fix-imports --sonarr-url 'http://192.168.1.100:8989' --sonarr-api-key 'your-key'\n", os.Args[0])
		}

		// Parse flags (only if we're not in test mode)
		// In tests, we skip flag parsing since test flags conflict with our flags
		args := os.Args[1:]
		if len(args) > 0 && !strings.Contains(args[0], "test") {
			err := fs.Parse(args)
			if err != nil {
				return nil, fmt.Errorf("error parsing flags: %w", err)
			}
		}

		// Use parsed values if not provided
		if dryRun == nil {
			dryRun = dryRunFlag
		}
		if noReport == nil {
			noReport = noReportFlag
		}
		if showVersion == nil {
			showVersion = showVersionFlag
		}
		if logLevel == nil {
			logLevel = logLevelFlag
		}
		if service == nil {
			service = serviceFlag
		}
		if sonarrURL == nil {
			sonarrURL = sonarrURLFlag
		}
		if sonarrAPIKey == nil {
			sonarrAPIKey = sonarrAPIFlag
		}
		if seriesIDs == nil {
			seriesIDs = seriesIDsFlag
		}

		// Handle new flags (they'll be processed later in the config loading)
	}

	// Load .env file if it exists (ignore errors - .env file is optional)
	_ = godotenv.Load()

	config := &Config{
		// Default values
		RequestTimeout:   30 * time.Second,
		RequestDelay:     500 * time.Millisecond,
		ConcurrentLimit:  5,
		AddMissingMovies: false, // Default to disabled
		QualityProfileID: 12,    // Default quality profile ID
	}

	// Set values from flags or defaults
	// For DryRun, check flag first, then environment variable
	if dryRun != nil && *dryRun {
		config.DryRun = true
	} else if dryRunEnv := os.Getenv("DRY_RUN"); dryRunEnv != "" {
		config.DryRun = dryRunEnv == "true" || dryRunEnv == "1"
	} else {
		config.DryRun = false
	}
	config.NoReport = noReport != nil && *noReport
	config.ShowVersion = showVersion != nil && *showVersion

	// Set service (default to "auto")
	if service != nil && *service != "" {
		config.Service = *service
	} else {
		config.Service = "auto"
	}

	// Parse series IDs if provided
	if seriesIDs != nil && *seriesIDs != "" {
		ids, err := parseSeriesIDs(*seriesIDs)
		if err != nil {
			return nil, fmt.Errorf("error parsing series IDs: %w", err)
		}
		config.SeriesIDs = ids
	}

	// Load configuration from environment variables with CLI flag overrides

	// Sonarr configuration
	config.Sonarr.APIKey = os.Getenv("SONARR_API_KEY")
	if config.Sonarr.APIKey != "" {
		// Only set default URL if API key is provided
		config.Sonarr.URL = getEnvOrDefault("SONARR_URL", "http://127.0.0.1:8989")
	} else {
		// Use URL from environment if provided, but no default
		config.Sonarr.URL = os.Getenv("SONARR_URL")
	}

	// Override with CLI flags if provided
	if sonarrURL != nil && *sonarrURL != "" {
		config.Sonarr.URL = *sonarrURL
	}
	if sonarrAPIKey != nil && *sonarrAPIKey != "" {
		config.Sonarr.APIKey = *sonarrAPIKey
	}

	// Radarr configuration
	config.Radarr.APIKey = os.Getenv("RADARR_API_KEY")
	if config.Radarr.APIKey != "" {
		// Only set default URL if API key is provided
		config.Radarr.URL = getEnvOrDefault("RADARR_URL", "http://127.0.0.1:7878")
	} else {
		// Use URL from environment if provided, but no default
		config.Radarr.URL = os.Getenv("RADARR_URL")
	}

	// Request configuration
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
		if limit, err := strconv.Atoi(limitStr); err == nil {
			config.ConcurrentLimit = limit
		}
	}

	// Log level configuration
	if logLevel != nil && *logLevel != "" {
		config.LogLevel = *logLevel
	} else if envLogLevel := os.Getenv("LOG_LEVEL"); envLogLevel != "" {
		config.LogLevel = envLogLevel
	} else {
		config.LogLevel = "INFO"
	}

	// Configure broken symlink handling
	config.AddMissingMovies = getEnvBool("ADD_MISSING_MOVIES", false)
	if qualityProfileStr := os.Getenv("QUALITY_PROFILE_ID"); qualityProfileStr != "" {
		if qualityID, err := strconv.Atoi(qualityProfileStr); err == nil {
			config.QualityProfileID = qualityID
		} else {
			config.QualityProfileID = 12 // Default
		}
	} else {
		config.QualityProfileID = 12 // Default
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// Validate validates the configuration settings
func (c *Config) Validate() error {
	// First, check if at least one service is configured
	sonarrConfigured := c.Sonarr.APIKey != ""
	radarrConfigured := c.Radarr.APIKey != ""

	if !sonarrConfigured && !radarrConfigured {
		return fmt.Errorf("at least one service must be configured (Sonarr or Radarr)")
	}

	// Validate Sonarr configuration
	if sonarrConfigured && c.Sonarr.URL == "" {
		return fmt.Errorf("Sonarr URL is required when Sonarr API key is provided")
	}
	if c.Sonarr.URL != "" && c.Sonarr.APIKey == "" {
		return fmt.Errorf("SONARR_API_KEY is required when SONARR_URL is provided")
	}

	// Validate Radarr configuration
	if radarrConfigured && c.Radarr.URL == "" {
		return fmt.Errorf("Radarr URL is required when Radarr API key is provided")
	}
	if c.Radarr.URL != "" && c.Radarr.APIKey == "" {
		return fmt.Errorf("RADARR_API_KEY is required when RADARR_URL is provided")
	}

	// Validate request timeout
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request timeout must be greater than 0")
	}

	// Validate concurrent limit
	if c.ConcurrentLimit <= 0 {
		return fmt.Errorf("concurrent limit must be greater than 0")
	}

	return nil
}

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

// parseSeriesIDs parses a comma-separated string of series IDs into a slice of integers
func parseSeriesIDs(seriesIDsStr string) ([]int, error) {
	if seriesIDsStr == "" {
		return nil, nil
	}

	parts := strings.Split(seriesIDsStr, ",")
	seriesIDs := make([]int, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		id, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid series ID '%s': %w", part, err)
		}

		seriesIDs = append(seriesIDs, id)
	}

	return seriesIDs, nil
}
