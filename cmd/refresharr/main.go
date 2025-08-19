package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/hnipps/refresharr/internal/arr"
	"github.com/hnipps/refresharr/internal/config"
	"github.com/hnipps/refresharr/internal/filesystem"
	"github.com/hnipps/refresharr/pkg/models"
)

var version = "dev"

func main() {
	// Command line flags
	var (
		showVersion   = flag.Bool("version", false, "Show version information")
		showHelp      = flag.Bool("help", false, "Show help information")
		dryRun        = flag.Bool("dry-run", false, "Perform a dry run (no changes will be made)")
		logLevel      = flag.String("log-level", "", "Log level (DEBUG, INFO, WARN, ERROR)")
		service       = flag.String("service", "", "Service to use: sonarr, radarr, or both (default: auto-detect)")
		sonarrURL     = flag.String("sonarr-url", "", "Sonarr URL (overrides SONARR_URL env var)")
		sonarrAPIKey  = flag.String("sonarr-api-key", "", "Sonarr API key (overrides SONARR_API_KEY env var)")
		radarrURL     = flag.String("radarr-url", "", "Radarr URL (overrides RADARR_URL env var)")
		radarrAPIKey  = flag.String("radarr-api-key", "", "Radarr API key (overrides RADARR_API_KEY env var)")
		seriesIDsFlag = flag.String("series-ids", "", "Comma-separated list of series IDs to process (default: all)")
		movieIDsFlag  = flag.String("movie-ids", "", "Comma-separated list of movie IDs to process (default: all)")
	)

	flag.Parse()

	if *showVersion {
		fmt.Printf("RefreshArr version %s\n", version)
		fmt.Println("A modular Go service for cleaning up missing file references in *arr applications")
		return
	}

	if *showHelp {
		showUsage()
		return
	}

	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override configuration with command line flags
	if *dryRun {
		cfg.DryRun = true
	}
	if *logLevel != "" {
		cfg.LogLevel = *logLevel
	}
	if *sonarrURL != "" {
		cfg.Sonarr.URL = *sonarrURL
	}
	if *sonarrAPIKey != "" {
		cfg.Sonarr.APIKey = *sonarrAPIKey
	}
	if *radarrURL != "" {
		cfg.Radarr.URL = *radarrURL
	}
	if *radarrAPIKey != "" {
		cfg.Radarr.APIKey = *radarrAPIKey
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	// Create logger
	logger := arr.NewStandardLogger(cfg.LogLevel)
	logger.Info("Starting RefreshArr v%s - Missing File Cleanup Service", version)

	if cfg.DryRun {
		logger.Info("🏃 DRY RUN MODE ENABLED - No changes will be made")
	}

	// Create file system checker
	fileChecker := filesystem.NewFileSystemChecker()

	// Create progress reporter
	progressReporter := arr.NewConsoleProgressReporter(logger)

	// Determine which services to run
	services := determineServices(cfg, *service)
	if len(services) == 0 {
		logger.Error("No services configured or available")
		os.Exit(1)
	}

	// Run cleanup for each service
	var allResults []*models.CleanupResult
	for _, serviceName := range services {
		var client arr.Client
		var result *models.CleanupResult

		// Create appropriate client
		switch serviceName {
		case "sonarr":
			client = arr.NewSonarrClient(&cfg.Sonarr, cfg.RequestTimeout, logger)
		case "radarr":
			client = arr.NewRadarrClient(&cfg.Radarr, cfg.RequestTimeout, logger)
		default:
			logger.Error("Unknown service: %s", serviceName)
			continue
		}

		// Create cleanup service
		cleanupService := arr.NewCleanupService(
			client,
			fileChecker,
			logger,
			progressReporter,
			cfg.RequestDelay,
			cfg.DryRun,
		)

		// Run cleanup based on service type and provided IDs
		if serviceName == "sonarr" {
			// Parse series IDs if provided
			var seriesIDs []int
			if *seriesIDsFlag != "" {
				seriesIDs = parseIDs(*seriesIDsFlag, "series", logger)
			}

			if len(seriesIDs) > 0 {
				logger.Info("Processing specific series IDs: %v", seriesIDs)
				result, err = cleanupService.CleanupMissingFilesForSeries(ctx, seriesIDs)
			} else {
				result, err = cleanupService.CleanupMissingFiles(ctx)
			}
		} else if serviceName == "radarr" {
			// Parse movie IDs if provided
			var movieIDs []int
			if *movieIDsFlag != "" {
				movieIDs = parseIDs(*movieIDsFlag, "movie", logger)
			}

			if len(movieIDs) > 0 {
				logger.Info("Processing specific movie IDs: %v", movieIDs)
				result, err = cleanupService.CleanupMissingFilesForMovies(ctx, movieIDs)
			} else {
				result, err = cleanupService.CleanupMissingFiles(ctx)
			}
		}

		if err != nil {
			logger.Error("Cleanup failed for %s: %s", serviceName, err.Error())
			os.Exit(1)
		}

		allResults = append(allResults, result)
	}

	// Check overall success
	overallSuccess := true
	for _, result := range allResults {
		if !result.Success {
			overallSuccess = false
			for _, msg := range result.Messages {
				logger.Warn("  %s", msg)
			}
		}
	}

	if !overallSuccess {
		logger.Warn("Cleanup completed with errors")
		os.Exit(1)
	}

	logger.Info("🎉 Cleanup completed successfully!")
}

// determineServices determines which services to run based on configuration and flags
func determineServices(cfg *config.Config, serviceFlag string) []string {
	var services []string

	switch serviceFlag {
	case "sonarr":
		if cfg.Sonarr.URL != "" && cfg.Sonarr.APIKey != "" {
			services = append(services, "sonarr")
		}
	case "radarr":
		if cfg.Radarr.URL != "" && cfg.Radarr.APIKey != "" {
			services = append(services, "radarr")
		}
	case "both":
		if cfg.Sonarr.URL != "" && cfg.Sonarr.APIKey != "" {
			services = append(services, "sonarr")
		}
		if cfg.Radarr.URL != "" && cfg.Radarr.APIKey != "" {
			services = append(services, "radarr")
		}
	default:
		// Auto-detect based on configuration
		if cfg.Sonarr.URL != "" && cfg.Sonarr.APIKey != "" {
			services = append(services, "sonarr")
		}
		if cfg.Radarr.URL != "" && cfg.Radarr.APIKey != "" {
			services = append(services, "radarr")
		}
	}

	return services
}

// parseIDs parses a comma-separated list of IDs
func parseIDs(idsFlag, itemType string, logger arr.Logger) []int {
	var ids []int
	idStrings := strings.Split(idsFlag, ",")
	for _, idStr := range idStrings {
		idStr = strings.TrimSpace(idStr)
		if idStr == "" {
			continue
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Error("Invalid %s ID: %s", itemType, idStr)
			os.Exit(1)
		}
		ids = append(ids, id)
	}
	return ids
}

func showUsage() {
	fmt.Println("RefreshArr - Missing File Cleanup Service for *arr Applications")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Printf("  %s [OPTIONS]\n", os.Args[0])
	fmt.Println()
	fmt.Println("DESCRIPTION:")
	fmt.Println("  RefreshArr cleans up missing file references in Sonarr and Radarr by:")
	fmt.Println("  - Checking all episodes/movies that claim to have files")
	fmt.Println("  - Verifying if the files actually exist on disk")
	fmt.Println("  - Removing database records for missing files")
	fmt.Println("  - Triggering a refresh to update status")
	fmt.Println()
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("  SONARR_URL           Sonarr base URL (default: http://127.0.0.1:8989)")
	fmt.Println("  SONARR_API_KEY       Sonarr API key")
	fmt.Println("  RADARR_URL           Radarr base URL (default: http://127.0.0.1:7878)")
	fmt.Println("  RADARR_API_KEY       Radarr API key")
	fmt.Println("  REQUEST_TIMEOUT      HTTP request timeout (default: 30s)")
	fmt.Println("  REQUEST_DELAY        Delay between requests (default: 500ms)")
	fmt.Println("  CONCURRENT_LIMIT     Max concurrent operations (default: 5)")
	fmt.Println("  LOG_LEVEL           Log level: DEBUG, INFO, WARN, ERROR (default: INFO)")
	fmt.Println("  DRY_RUN             Set to true for dry run mode (default: false)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Auto-detect and run for all configured services")
	fmt.Println("  refresharr")
	fmt.Println()
	fmt.Println("  # Run only for Sonarr")
	fmt.Println("  refresharr --service sonarr")
	fmt.Println()
	fmt.Println("  # Run only for Radarr")
	fmt.Println("  refresharr --service radarr")
	fmt.Println()
	fmt.Println("  # Run for both services")
	fmt.Println("  refresharr --service both")
	fmt.Println()
	fmt.Println("  # Dry run to see what would be cleaned up")
	fmt.Println("  refresharr --dry-run")
	fmt.Println()
	fmt.Println("  # Process specific series in Sonarr")
	fmt.Println("  refresharr --service sonarr --series-ids \"123,456,789\"")
	fmt.Println()
	fmt.Println("  # Process specific movies in Radarr")
	fmt.Println("  refresharr --service radarr --movie-ids \"123,456,789\"")
	fmt.Println()
	fmt.Println("  # Custom instances with debug logging")
	fmt.Println("  refresharr --sonarr-url \"http://192.168.1.100:8989\" --sonarr-api-key \"your-key\" --log-level DEBUG")
	fmt.Println()
}
