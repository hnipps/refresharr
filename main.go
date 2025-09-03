package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/hnipps/refresharr/internal/arr"
	"github.com/hnipps/refresharr/internal/config"
	"github.com/hnipps/refresharr/internal/filesystem"
	"github.com/hnipps/refresharr/internal/plex"
	"github.com/hnipps/refresharr/internal/report"
	"github.com/hnipps/refresharr/pkg/models"
)

// Version information - set at build time
var version = "dev"

func main() {
	ctx := context.Background()

	// Determine command - check if first argument is a known command
	args := os.Args[1:]
	var command string
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		// First arg doesn't start with "-", could be a command
		switch args[0] {
		case "fix-imports":
			command = "fix-imports"
			// Remove command from args for flag parsing
			os.Args = append([]string{os.Args[0]}, args[1:]...)
		case "compare-plex":
			command = "compare-plex"
			// Remove command from args for flag parsing
			os.Args = append([]string{os.Args[0]}, args[1:]...)
		default:
			command = "cleanup" // Default command
		}
	} else {
		command = "cleanup" // Default command
	}

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Handle version flag
	if cfg.ShowVersion {
		fmt.Printf("RefreshArr version %s\n", version)
		fmt.Println("Missing File Cleanup Service for Sonarr and Radarr")
		os.Exit(0)
	}

	// Route to appropriate command handler
	switch command {
	case "fix-imports":
		runFixImportsCommand(ctx, cfg)
	case "compare-plex":
		runComparePlexCommand(ctx, cfg)
	case "cleanup":
		runCleanupCommand(ctx, cfg)
	default:
		log.Fatalf("Unknown command: %s", command)
	}
}

// runFixImportsCommand handles the fix-imports command
func runFixImportsCommand(ctx context.Context, cfg *config.Config) {
	// Create logger
	logger := arr.NewStandardLogger(cfg.LogLevel)
	logger.Info("Starting RefreshArr %s - Sonarr Import Fixer", version)

	// Only Sonarr is supported for import fixing
	if cfg.Sonarr.URL == "" || cfg.Sonarr.APIKey == "" {
		logger.Error("Sonarr must be configured to use the fix-imports command")
		logger.Error("Please set SONARR_URL and SONARR_API_KEY environment variables or use CLI flags")
		os.Exit(1)
	}

	// Create Sonarr client
	client := arr.NewSonarrClient(&cfg.Sonarr, cfg.RequestTimeout, logger)

	// Test connection
	if err := client.TestConnection(ctx); err != nil {
		logger.Error("Failed to connect to Sonarr: %s", err.Error())
		os.Exit(1)
	}

	// Create import fixer
	importFixer := arr.NewImportFixer(client, logger, cfg.DryRun)

	// Run the import fixer
	result, err := importFixer.FixImports(ctx, true) // removeFromClient = true by default
	if err != nil {
		logger.Error("Import fixer failed: %s", err.Error())
		os.Exit(1)
	}

	// Report results
	if result.DryRun && result.TotalStuckItems > 0 {
		logger.Info("🔍 Found %d stuck import(s) that would be fixed", result.TotalStuckItems)
		logger.Info("Run without --dry-run to actually fix these imports")
	} else if result.FixedItems > 0 {
		logger.Info("🎉 Successfully imported %d out of %d stuck imports!", result.FixedItems, result.TotalStuckItems)
		if len(result.Errors) > 0 {
			failedCount := result.TotalStuckItems - result.FixedItems
			logger.Info("📝 %d items failed to import and were left in queue for manual resolution:", failedCount)
			for _, errMsg := range result.Errors {
				logger.Info("  %s", errMsg)
			}
			logger.Info("Please check these items in Sonarr's Activity → Queue tab and resolve manually.")
		}
	} else if result.TotalStuckItems > 0 {
		logger.Info("⚠️  No items were successfully imported - all %d items remain in queue for manual resolution", result.TotalStuckItems)
		logger.Info("Please check these items in Sonarr's Activity → Queue tab and resolve manually.")
	} else if result.TotalStuckItems == 0 {
		logger.Info("✨ No stuck imports found - your queue is clean!")
	}
}

// runCleanupCommand handles the default cleanup command
func runCleanupCommand(ctx context.Context, cfg *config.Config) {
	// Create logger
	logger := arr.NewStandardLogger(cfg.LogLevel)
	logger.Info("Starting RefreshArr %s - Missing File Cleanup Service", version)

	// Create file system checker
	fileChecker := filesystem.NewFileSystemChecker()

	// Create progress reporter
	progressReporter := arr.NewConsoleProgressReporter(logger)

	// Determine which service(s) to run based on configuration
	services := determineServices(cfg, logger)
	if len(services) == 0 {
		logger.Error("No services configured or available")
		os.Exit(1)
	}

	allSuccessful := true
	allResults := make([]*models.CleanupResult, 0, len(services))

	// Process each configured service
	for _, serviceInfo := range services {
		logger.Info("Processing %s service...", serviceInfo.Name)

		// Create cleanup service with concurrency support
		cleanupService := arr.NewCleanupServiceWithConcurrency(
			serviceInfo.Client,
			fileChecker,
			logger,
			progressReporter,
			cfg.RequestDelay,
			cfg.ConcurrentLimit,
			cfg.DryRun,
			cfg.QualityProfileID,
			cfg.AddMissingMovies,
		)

		// Run cleanup (with series filtering if applicable)
		var result *models.CleanupResult
		var err error
		if serviceInfo.Name == "sonarr" && len(cfg.SeriesIDs) > 0 {
			// Filter to specific series for Sonarr
			result, err = cleanupService.CleanupMissingFilesForSeries(ctx, cfg.SeriesIDs)
		} else {
			// Clean all missing files
			result, err = cleanupService.CleanupMissingFiles(ctx)
		}

		if err != nil {
			logger.Error("Cleanup failed for %s: %s", serviceInfo.Name, err.Error())
			allSuccessful = false
			continue
		}

		allResults = append(allResults, result)

		if !result.Success {
			logger.Warn("%s cleanup completed with errors", serviceInfo.Name)
			for _, msg := range result.Messages {
				logger.Warn("  %s", msg)
			}
			allSuccessful = false
		} else {
			logger.Info("🎉 %s cleanup completed successfully!", serviceInfo.Name)
		}
	}

	// Generate combined report if we have results and reports are enabled
	if len(allResults) > 0 && !cfg.NoReport {
		reportGenerator := report.NewGenerator(logger)

		for i, result := range allResults {
			if result.Report != nil {
				serviceName := services[i].Name
				logger.Info("Report for %s:", serviceName)
				if err := reportGenerator.GenerateReport(result.Report, true); err != nil {
					logger.Warn("Failed to generate report for %s: %s", serviceName, err.Error())
				}
			}
		}
	}

	if !allSuccessful {
		logger.Warn("Some cleanup operations completed with errors")
		os.Exit(1)
	}

	logger.Info("🎉 All cleanup operations completed successfully!")
}

// ServiceInfo holds information about a configured service
type ServiceInfo struct {
	Name   string
	Client arr.Client
}

// determineServices decides which services to run based on configuration
func determineServices(cfg *config.Config, logger arr.Logger) []ServiceInfo {
	var services []ServiceInfo

	switch cfg.Service {
	case "sonarr":
		if cfg.Sonarr.URL != "" && cfg.Sonarr.APIKey != "" {
			client := arr.NewSonarrClient(&cfg.Sonarr, cfg.RequestTimeout, logger)
			services = append(services, ServiceInfo{Name: "sonarr", Client: client})
		} else {
			logger.Error("Sonarr service requested but not properly configured")
		}

	case "radarr":
		if cfg.Radarr.URL != "" && cfg.Radarr.APIKey != "" {
			client := arr.NewRadarrClient(&cfg.Radarr, cfg.RequestTimeout, logger)
			services = append(services, ServiceInfo{Name: "radarr", Client: client})
		} else {
			logger.Error("Radarr service requested but not properly configured")
		}

	case "auto":
		// Add Sonarr if configured
		if cfg.Sonarr.URL != "" && cfg.Sonarr.APIKey != "" {
			client := arr.NewSonarrClient(&cfg.Sonarr, cfg.RequestTimeout, logger)
			services = append(services, ServiceInfo{Name: "sonarr", Client: client})
		}

		// Add Radarr if configured
		if cfg.Radarr.URL != "" && cfg.Radarr.APIKey != "" {
			client := arr.NewRadarrClient(&cfg.Radarr, cfg.RequestTimeout, logger)
			services = append(services, ServiceInfo{Name: "radarr", Client: client})
		}
	}

	return services
}

// runComparePlexCommand handles the compare-plex command
func runComparePlexCommand(ctx context.Context, cfg *config.Config) {
	// Create logger
	logger := arr.NewStandardLogger(cfg.LogLevel)
	logger.Info("Starting RefreshArr %s - Plex Comparison Tool", version)

	// Check if TMDB ID is provided as argument
	// Since we removed the command from os.Args, the TMDB ID should be at position 0
	args := os.Args[1:]
	if len(args) < 1 {
		logger.Error("TMDB ID is required as argument")
		logger.Error("Usage: refresharr compare-plex <tmdb-id>")
		logger.Error("Example: refresharr compare-plex 12345")
		os.Exit(1)
	}

	// Parse TMDB ID
	tmdbIDStr := args[0]
	tmdbID, err := strconv.Atoi(tmdbIDStr)
	if err != nil {
		logger.Error("Invalid TMDB ID '%s': must be a number", tmdbIDStr)
		os.Exit(1)
	}

	// Validate Radarr configuration
	if cfg.Radarr.URL == "" || cfg.Radarr.APIKey == "" {
		logger.Error("Radarr must be configured to use the compare-plex command")
		logger.Error("Please set RADARR_URL and RADARR_API_KEY environment variables")
		os.Exit(1)
	}

	// Validate Plex configuration
	if cfg.Plex.URL == "" || cfg.Plex.Token == "" {
		logger.Error("Plex must be configured to use the compare-plex command")
		logger.Error("Please set PLEX_URL and PLEX_TOKEN environment variables")
		os.Exit(1)
	}

	// Create Radarr client
	radarrClient := arr.NewRadarrClient(&cfg.Radarr, cfg.RequestTimeout, logger)

	// Test Radarr connection
	if err := radarrClient.TestConnection(ctx); err != nil {
		logger.Error("Failed to connect to Radarr: %s", err.Error())
		os.Exit(1)
	}

	// Create Plex client
	plexClient := plex.NewPlexClient(&cfg.Plex, cfg.RequestTimeout, logger)

	// Test Plex connection
	if err := plexClient.TestConnection(ctx); err != nil {
		logger.Error("Failed to connect to Plex: %s", err.Error())
		os.Exit(1)
	}

	// Get movie from Radarr by TMDB ID
	logger.Info("🔍 Looking up movie with TMDB ID %d in Radarr...", tmdbID)
	radarrMovie, err := radarrClient.GetMovieByTMDBID(ctx, tmdbID)
	if err != nil {
		logger.Error("❌ Movie with TMDB ID %d does not exist in Radarr", tmdbID)
		os.Exit(1)
	}

	logger.Info("✅ Found movie in Radarr: %s (%d)", radarrMovie.Title, radarrMovie.Year)

	// Check Radarr file status
	radarrHasFile := radarrMovie.HasFile
	var radarrFilePath string
	if radarrHasFile && radarrMovie.MovieFileID != nil {
		movieFile, err := radarrClient.GetMovieFile(ctx, *radarrMovie.MovieFileID)
		if err != nil {
			logger.Warn("⚠️  Could not get movie file details from Radarr: %s", err.Error())
			radarrFilePath = "Unknown"
		} else {
			radarrFilePath = movieFile.Path
		}
	}

	logger.Info("📁 Radarr file status: HasFile=%t", radarrHasFile)
	if radarrHasFile {
		logger.Info("📄 Radarr file path: %s", radarrFilePath)
	}

	// Get movie from Plex by TMDB ID
	logger.Info("🔍 Looking up movie with TMDB ID %d in Plex...", tmdbID)
	plexMovie, err := plexClient.GetMovieByTMDBID(ctx, tmdbID)
	if err != nil {
		logger.Warn("⚠️  Movie with TMDB ID %d not found in Plex: %s", tmdbID, err.Error())

		// Generate comparison report
		logger.Info("\n📊 COMPARISON REPORT")
		logger.Info("==================")
		logger.Info("Movie: %s (%d)", radarrMovie.Title, radarrMovie.Year)
		logger.Info("TMDB ID: %d", tmdbID)
		logger.Info("Radarr Status: %s", getFileStatusText(radarrHasFile))
		logger.Info("Plex Status: Not Found")
		logger.Info("Match Status: ❌ MISMATCH - Movie not in Plex library")

		if radarrHasFile {
			logger.Info("⚠️  Radarr shows file available but movie not found in Plex")
			logger.Info("💡 Suggestion: Check if Plex library is scanning the correct directories")
		}
		return
	}

	logger.Info("✅ Found movie in Plex: %s (%d)", plexMovie.Title, plexMovie.Year)

	// Check Plex availability status
	plexAvailable := plexMovie.Available
	logger.Info("📁 Plex availability status: Available=%t", plexAvailable)

	// Generate comparison report
	logger.Info("\n📊 COMPARISON REPORT")
	logger.Info("==================")
	logger.Info("Movie: %s (%d)", radarrMovie.Title, radarrMovie.Year)
	logger.Info("TMDB ID: %d", tmdbID)
	logger.Info("Radarr Status: %s", getFileStatusText(radarrHasFile))
	logger.Info("Plex Status: %s", getAvailabilityStatusText(plexAvailable))

	// Determine match status
	if radarrHasFile == plexAvailable {
		logger.Info("Match Status: ✅ MATCH - Both services agree")
		if radarrHasFile {
			logger.Info("🎉 Movie is available in both Radarr and Plex")
		} else {
			logger.Info("📭 Movie is not available in either service")
		}
	} else {
		logger.Info("Match Status: ❌ MISMATCH - Services disagree")
		if radarrHasFile && !plexAvailable {
			logger.Info("⚠️  Radarr shows file available but Plex shows unavailable")
			logger.Info("💡 Suggestion: Check if Plex needs to refresh its library")
			if radarrFilePath != "" {
				logger.Info("📄 Check file at: %s", radarrFilePath)
			}
		} else if !radarrHasFile && plexAvailable {
			logger.Info("⚠️  Plex shows movie available but Radarr shows no file")
			logger.Info("💡 Suggestion: Check if Radarr needs to scan for existing files")
		}
	}
}

// getFileStatusText returns a human-readable file status
func getFileStatusText(hasFile bool) string {
	if hasFile {
		return "File Available"
	}
	return "No File"
}

// getAvailabilityStatusText returns a human-readable availability status
func getAvailabilityStatusText(available bool) string {
	if available {
		return "Available"
	}
	return "Not Available"
}
