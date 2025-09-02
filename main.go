package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/hnipps/refresharr/internal/arr"
	"github.com/hnipps/refresharr/internal/config"
	"github.com/hnipps/refresharr/internal/filesystem"
	"github.com/hnipps/refresharr/internal/report"
	"github.com/hnipps/refresharr/pkg/models"
)

// Version information - set at build time
var version = "dev"

func main() {
	ctx := context.Background()

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
		)

		// Run cleanup (with series filtering if applicable)
		var result *models.CleanupResult
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
