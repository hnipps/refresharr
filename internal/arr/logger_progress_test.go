package arr

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/hnipps/refresharr/pkg/models"
)

func TestNewStandardLogger(t *testing.T) {
	logger := NewStandardLogger("INFO")
	if logger == nil {
		t.Error("NewStandardLogger() returned nil")
	}

	// Verify it implements the Logger interface
	logger.Debug("test debug")
	logger.Info("test info")
	logger.Warn("test warn")
	logger.Error("test error")
}

func TestStandardLogger_LogLevels(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	originalLogger := log.Default()
	defer log.SetOutput(originalLogger.Writer())
	log.SetOutput(&buf)
	log.SetFlags(0) // Remove timestamp for testing

	tests := []struct {
		name     string
		level    string
		messages []string
		expected []string
	}{
		{
			name:  "DEBUG level",
			level: "DEBUG",
			messages: []string{
				"debug message",
				"info message",
				"warn message",
				"error message",
			},
			expected: []string{
				"[DEBUG] debug message",
				"[INFO] info message",
				"[WARN] warn message",
				"[ERROR] error message",
			},
		},
		{
			name:  "INFO level",
			level: "INFO",
			messages: []string{
				"debug message",
				"info message",
				"warn message",
				"error message",
			},
			expected: []string{
				"[INFO] info message",
				"[WARN] warn message",
				"[ERROR] error message",
			},
		},
		{
			name:  "WARN level",
			level: "WARN",
			messages: []string{
				"debug message",
				"info message",
				"warn message",
				"error message",
			},
			expected: []string{
				"[WARN] warn message",
				"[ERROR] error message",
			},
		},
		{
			name:  "ERROR level",
			level: "ERROR",
			messages: []string{
				"debug message",
				"info message",
				"warn message",
				"error message",
			},
			expected: []string{
				"[ERROR] error message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			logger := NewStandardLogger(tt.level)

			// Log messages
			logger.Debug(tt.messages[0])
			logger.Info(tt.messages[1])
			logger.Warn(tt.messages[2])
			logger.Error(tt.messages[3])

			output := buf.String()
			lines := strings.Split(strings.TrimSpace(output), "\n")

			// Filter empty lines
			var actualLines []string
			for _, line := range lines {
				if line != "" {
					actualLines = append(actualLines, line)
				}
			}

			if len(actualLines) != len(tt.expected) {
				t.Errorf("Expected %d log lines, got %d", len(tt.expected), len(actualLines))
				t.Errorf("Expected: %v", tt.expected)
				t.Errorf("Actual: %v", actualLines)
				return
			}

			for i, expected := range tt.expected {
				if actualLines[i] != expected {
					t.Errorf("Line %d: expected '%s', got '%s'", i, expected, actualLines[i])
				}
			}
		})
	}
}

func TestStandardLogger_Formatting(t *testing.T) {
	var buf bytes.Buffer
	originalLogger := log.Default()
	defer log.SetOutput(originalLogger.Writer())
	log.SetOutput(&buf)
	log.SetFlags(0)

	logger := NewStandardLogger("DEBUG")

	// Test formatting with arguments
	logger.Info("Hello %s, you have %d messages", "World", 5)

	output := strings.TrimSpace(buf.String())
	expected := "[INFO] Hello World, you have 5 messages"

	if output != expected {
		t.Errorf("Expected '%s', got '%s'", expected, output)
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected LogLevel
	}{
		{"DEBUG", LogLevelDebug},
		{"debug", LogLevelDebug},
		{"Info", LogLevelInfo},
		{"INFO", LogLevelInfo},
		{"WARN", LogLevelWarn},
		{"warn", LogLevelWarn},
		{"WARNING", LogLevelWarn},
		{"ERROR", LogLevelError},
		{"error", LogLevelError},
		{"INVALID", LogLevelInfo}, // Should default to INFO
		{"", LogLevelInfo},        // Should default to INFO
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			if result != tt.expected {
				t.Errorf("parseLogLevel(%s) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNewConsoleProgressReporter(t *testing.T) {
	logger := &mockLogger{}
	reporter := NewConsoleProgressReporter(logger)
	if reporter == nil {
		t.Error("NewConsoleProgressReporter() returned nil")
	}
}

func TestConsoleProgressReporter_StartSeries(t *testing.T) {
	logger := &mockLogger{}
	reporter := NewConsoleProgressReporter(logger)

	reporter.StartSeries(123, "Test Series", 2, 5)

	// Check that appropriate log messages were called
	if len(logger.infoMessages) < 2 {
		t.Errorf("Expected at least 2 info messages, got %d", len(logger.infoMessages))
	}

	// Check for series name and progress
	foundSeriesMessage := false
	foundProgressMessage := false
	for _, msg := range logger.infoMessages {
		if strings.Contains(msg, "Test Series") {
			foundSeriesMessage = true
		}
		if strings.Contains(msg, "2/5") && strings.Contains(msg, "123") {
			foundProgressMessage = true
		}
	}

	if !foundSeriesMessage {
		t.Error("Expected series name message not found")
	}
	if !foundProgressMessage {
		t.Error("Expected progress message not found")
	}
}

func TestConsoleProgressReporter_StartEpisode(t *testing.T) {
	logger := &mockLogger{}
	reporter := NewConsoleProgressReporter(logger)

	reporter.StartEpisode(456, 2, 10)

	// Should log episode information
	if len(logger.infoMessages) != 1 {
		t.Errorf("Expected 1 info message, got %d", len(logger.infoMessages))
	}

	message := logger.infoMessages[0]
	if !strings.Contains(message, "S2E10") || !strings.Contains(message, "456") {
		t.Errorf("Expected episode info in message, got: %s", message)
	}
}

func TestConsoleProgressReporter_ReportMissingFile(t *testing.T) {
	logger := &mockLogger{}
	reporter := NewConsoleProgressReporter(logger)

	testPath := "/path/to/missing/file.mkv"
	reporter.ReportMissingFile(testPath)

	// Should log a warning
	if len(logger.warnMessages) != 1 {
		t.Errorf("Expected 1 warn message, got %d", len(logger.warnMessages))
	}

	message := logger.warnMessages[0]
	if !strings.Contains(message, testPath) || !strings.Contains(message, "MISSING") {
		t.Errorf("Expected missing file info in message, got: %s", message)
	}
}

func TestConsoleProgressReporter_ReportDeletedRecord(t *testing.T) {
	logger := &mockLogger{}
	reporter := NewConsoleProgressReporter(logger)

	reporter.ReportDeletedRecord(789)

	// Should log success info
	if len(logger.infoMessages) != 1 {
		t.Errorf("Expected 1 info message, got %d", len(logger.infoMessages))
	}

	message := logger.infoMessages[0]
	if !strings.Contains(message, "789") || !strings.Contains(message, "deleted") {
		t.Errorf("Expected deleted record info in message, got: %s", message)
	}
}

func TestConsoleProgressReporter_ReportError(t *testing.T) {
	logger := &mockLogger{}
	reporter := NewConsoleProgressReporter(logger)

	testError := "test error message"
	reporter.ReportError(errors.New(testError))

	// Should log an error
	if len(logger.errorMessages) != 1 {
		t.Errorf("Expected 1 error message, got %d", len(logger.errorMessages))
	}

	message := logger.errorMessages[0]
	if !strings.Contains(message, testError) {
		t.Errorf("Expected error message in log, got: %s", message)
	}
}

func TestConsoleProgressReporter_Finish(t *testing.T) {
	logger := &mockLogger{}
	reporter := NewConsoleProgressReporter(logger)

	stats := models.CleanupStats{
		TotalItemsChecked: 50,
		MissingFiles:      5,
		DeletedRecords:    3,
		Errors:            1,
	}

	reporter.Finish(stats)

	// Should log multiple info messages with stats
	if len(logger.infoMessages) < 5 {
		t.Errorf("Expected at least 5 info messages, got %d", len(logger.infoMessages))
	}

	// Check that stats are mentioned
	allMessages := strings.Join(logger.infoMessages, " ")
	if !strings.Contains(allMessages, "50") {
		t.Error("Expected total items checked to be mentioned")
	}
	if !strings.Contains(allMessages, "5") {
		t.Error("Expected missing files count to be mentioned")
	}
	if !strings.Contains(allMessages, "3") {
		t.Error("Expected deleted records count to be mentioned")
	}

	// Should also log warning for errors
	if len(logger.warnMessages) != 1 {
		t.Errorf("Expected 1 warn message for errors, got %d", len(logger.warnMessages))
	}
	if !strings.Contains(logger.warnMessages[0], "1") {
		t.Error("Expected error count to be mentioned in warning")
	}
}

func TestConsoleProgressReporter_FinishNoRecordsDeleted(t *testing.T) {
	logger := &mockLogger{}
	reporter := NewConsoleProgressReporter(logger)

	stats := models.CleanupStats{
		TotalItemsChecked: 50,
		MissingFiles:      0,
		DeletedRecords:    0,
		Errors:            0,
	}

	reporter.Finish(stats)

	// Should log info messages
	if len(logger.infoMessages) < 5 {
		t.Errorf("Expected at least 5 info messages, got %d", len(logger.infoMessages))
	}

	// Should mention that nothing needed cleanup
	allMessages := strings.Join(logger.infoMessages, " ")
	if !strings.Contains(allMessages, "nothing to clean") {
		t.Error("Expected message about nothing to clean")
	}

	// Should not log warnings for errors
	if len(logger.warnMessages) != 0 {
		t.Errorf("Expected 0 warn messages, got %d", len(logger.warnMessages))
	}
}

func TestConsoleProgressReporter_FinishMissingFilesButNoRecordsDeleted(t *testing.T) {
	logger := &mockLogger{}
	reporter := NewConsoleProgressReporter(logger)

	// This is the scenario my fix addresses: missing files found but no records deleted
	// (e.g., due to dry-run mode, broken symlinks, or deletion errors)
	stats := models.CleanupStats{
		TotalItemsChecked: 50,
		MissingFiles:      38, // Files were detected as missing
		DeletedRecords:    0,  // But no records were actually deleted
		Errors:            0,
	}

	reporter.Finish(stats)

	// Should log info messages
	if len(logger.infoMessages) < 5 {
		t.Errorf("Expected at least 5 info messages, got %d", len(logger.infoMessages))
	}

	// Should mention that missing files were found but no records deleted
	allMessages := strings.Join(logger.infoMessages, " ")
	if !strings.Contains(allMessages, "38") {
		t.Error("Expected missing files count to be mentioned")
	}
	if !strings.Contains(allMessages, "Missing files found but no records deleted") {
		t.Error("Expected message about missing files found but no records deleted")
	}

	// Should NOT say "nothing to clean" since missing files were found
	if strings.Contains(allMessages, "nothing to clean") {
		t.Error("Should not say 'nothing to clean' when missing files were found")
	}

	// Should not log warnings for errors since errors = 0
	if len(logger.warnMessages) != 0 {
		t.Errorf("Expected 0 warn messages, got %d", len(logger.warnMessages))
	}
}
