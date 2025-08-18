package main

import (
	"context"
	"log"
	"os"

	"github.com/hnipps/refresharr/internal/arr"
	"github.com/hnipps/refresharr/internal/config"
	"github.com/hnipps/refresharr/internal/filesystem"
)

func main() {
	ctx := context.Background()

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create logger
	logger := arr.NewStandardLogger(cfg.LogLevel)
	logger.Info("Starting RefreshArr - Missing File Cleanup Service")

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

	// Run cleanup
	result, err := cleanupService.CleanupMissingFiles(ctx)
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
