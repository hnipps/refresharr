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
		sonarrURL     = flag.String("sonarr-url", "", "Sonarr URL (overrides SONARR_URL env var)")
		sonarrAPIKey  = flag.String("sonarr-api-key", "", "Sonarr API key (overrides SONARR_API_KEY env var)")
		seriesIDsFlag = flag.String("series-ids", "", "Comma-separated list of series IDs to process (default: all)")
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

	// Create Sonarr client
	sonarrClient := arr.NewSonarrClient(&cfg.Sonarr, cfg.RequestTimeout, logger)

	// Create progress reporter
	progressReporter := arr.NewConsoleProgressReporter(logger)

	// Create cleanup service
	cleanupService := arr.NewCleanupService(
		sonarrClient,
		fileChecker,
		logger,
		progressReporter,
		cfg.RequestDelay,
		cfg.DryRun,
	)

	// Parse series IDs if provided
	var seriesIDs []int
	if *seriesIDsFlag != "" {
		idStrings := strings.Split(*seriesIDsFlag, ",")
		for _, idStr := range idStrings {
			idStr = strings.TrimSpace(idStr)
			if idStr == "" {
				continue
			}

			id, err := strconv.Atoi(idStr)
			if err != nil {
				logger.Error("Invalid series ID: %s", idStr)
				os.Exit(1)
			}
			seriesIDs = append(seriesIDs, id)
		}

		if len(seriesIDs) > 0 {
			logger.Info("Processing specific series IDs: %v", seriesIDs)
		}
	}

	// Run cleanup
	var result *models.CleanupResult
	if len(seriesIDs) > 0 {
		result, err = cleanupService.CleanupMissingFilesForSeries(ctx, seriesIDs)
	} else {
		result, err = cleanupService.CleanupMissingFiles(ctx)
	}

	if err != nil {
		logger.Error("Cleanup failed: %s", err.Error())
		os.Exit(1)
	}

	if !result.Success {
		logger.Warn("Cleanup completed with errors")
		for _, msg := range result.Messages {
			logger.Warn("  %s", msg)
		}
		os.Exit(1)
	}

	logger.Info("🎉 Cleanup completed successfully!")
}

func showUsage() {
	fmt.Println("RefreshArr - Missing File Cleanup Service for *arr Applications")
	fmt.Println()
	fmt.Println("USAGE:")
	fmt.Printf("  %s [OPTIONS]\n", os.Args[0])
	fmt.Println()
	fmt.Println("DESCRIPTION:")
	fmt.Println("  RefreshArr cleans up missing file references in Sonarr by:")
	fmt.Println("  - Checking all episodes that claim to have files")
	fmt.Println("  - Verifying if the files actually exist on disk")
	fmt.Println("  - Removing database records for missing files")
	fmt.Println("  - Triggering a refresh to update status")
	fmt.Println()
	fmt.Println("OPTIONS:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("ENVIRONMENT VARIABLES:")
	fmt.Println("  SONARR_URL           Sonarr base URL (default: http://127.0.0.1:8989)")
	fmt.Println("  SONARR_API_KEY       Sonarr API key (required)")
	fmt.Println("  REQUEST_TIMEOUT      HTTP request timeout (default: 30s)")
	fmt.Println("  REQUEST_DELAY        Delay between requests (default: 500ms)")
	fmt.Println("  CONCURRENT_LIMIT     Max concurrent operations (default: 5)")
	fmt.Println("  LOG_LEVEL           Log level: DEBUG, INFO, WARN, ERROR (default: INFO)")
	fmt.Println("  DRY_RUN             Set to true for dry run mode (default: false)")
	fmt.Println()
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Basic usage (requires SONARR_API_KEY env var)")
	fmt.Println("  refresharr")
	fmt.Println()
	fmt.Println("  # Dry run to see what would be cleaned up")
	fmt.Println("  refresharr --dry-run")
	fmt.Println()
	fmt.Println("  # Process specific series")
	fmt.Println("  refresharr --series-ids \"123,456,789\"")
	fmt.Println()
	fmt.Println("  # Custom Sonarr instance with debug logging")
	fmt.Println("  refresharr --sonarr-url \"http://192.168.1.100:8989\" --sonarr-api-key \"your-key\" --log-level DEBUG")
	fmt.Println()
}
