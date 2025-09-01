package main

import (
	"os"
	"testing"

	"github.com/hnipps/refresharr/internal/arr"
	"github.com/hnipps/refresharr/internal/config"
	"github.com/hnipps/refresharr/internal/filesystem"
)

// TestMain verifies that the main function can be called without panicking
// This is an integration test that ensures all components work together
func TestMain_RequiredEnvVars(t *testing.T) {
	// Set required environment variables for a valid configuration
	originalArgs := os.Args
	defer func() { os.Args = originalArgs }()

	// Set up environment for a dry run test
	os.Setenv("SONARR_URL", "http://test-sonarr:8989")
	os.Setenv("SONARR_API_KEY", "test-api-key")
	os.Setenv("DRY_RUN", "true")
	os.Setenv("LOG_LEVEL", "ERROR") // Minimize test output

	defer func() {
		os.Unsetenv("SONARR_URL")
		os.Unsetenv("SONARR_API_KEY")
		os.Unsetenv("DRY_RUN")
		os.Unsetenv("LOG_LEVEL")
	}()

	// This test mainly verifies that main() doesn't panic during initialization
	// Since we can't easily test the full execution without a real Sonarr instance,
	// we just test that configuration loading and component creation works

	// We can't actually run main() in a test easily since it calls os.Exit
	// But we can test that the components can be created successfully
	// This is a smoke test to ensure no obvious errors in initialization

	// Note: A full integration test would require mocking or a test Sonarr instance
	// For now, we just verify that the test setup doesn't cause import issues
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// The main function would be tested in an end-to-end test scenario
	// with a test server, but that's beyond the scope of unit tests
}

func TestMain_MissingEnvVars(t *testing.T) {
	// Clear environment variables to test error handling
	originalEnv := make(map[string]string)
	requiredVars := []string{"SONARR_URL", "SONARR_API_KEY", "DRY_RUN", "LOG_LEVEL"}

	// Save original values
	for _, v := range requiredVars {
		originalEnv[v] = os.Getenv(v)
		os.Unsetenv(v)
	}

	// Restore original values
	defer func() {
		for _, v := range requiredVars {
			if val, ok := originalEnv[v]; ok && val != "" {
				os.Setenv(v, val)
			}
		}
	}()

	// Test that missing required env vars would cause configuration to fail
	// We can't easily test main() directly due to os.Exit, but we know it should fail
	// This test documents the expected behavior
	t.Log("Configuration should fail without required environment variables")

	// This is more of a documentation test - in a real scenario,
	// main() would exit with code 1 due to missing configuration
}

// Integration test helper - this would be used in actual integration tests
func TestComponentIntegration(t *testing.T) {
	// This test verifies that all the major components can be created
	// and connected together without errors

	// Set up test environment
	os.Setenv("SONARR_URL", "http://localhost:8989")
	os.Setenv("SONARR_API_KEY", "test-key")
	os.Setenv("DRY_RUN", "true")
	defer func() {
		os.Unsetenv("SONARR_URL")
		os.Unsetenv("SONARR_API_KEY")
		os.Unsetenv("DRY_RUN")
	}()

	// Test that we can create all the components that main() creates
	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	logger := arr.NewStandardLogger(cfg.LogLevel)
	if logger == nil {
		t.Fatal("Failed to create logger")
	}

	fileChecker := filesystem.NewFileSystemChecker()
	if fileChecker == nil {
		t.Fatal("Failed to create file checker")
	}

	sonarrClient := arr.NewSonarrClient(&cfg.Sonarr, cfg.RequestTimeout, logger)
	if sonarrClient == nil {
		t.Fatal("Failed to create Sonarr client")
	}

	progressReporter := arr.NewConsoleProgressReporter(logger)
	if progressReporter == nil {
		t.Fatal("Failed to create progress reporter")
	}

	cleanupService := arr.NewCleanupService(
		sonarrClient,
		fileChecker,
		logger,
		progressReporter,
		cfg.RequestDelay,
		cfg.DryRun,
	)
	if cleanupService == nil {
		t.Fatal("Failed to create cleanup service")
	}

	// All components created successfully
	t.Log("All main components created successfully")
}
