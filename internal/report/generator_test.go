package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hnipps/refresharr/pkg/models"
)

// mockLogger implements the Logger interface for testing
type mockLogger struct {
	logs []string
}

func (m *mockLogger) Debug(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	m.logs = append(m.logs, "DEBUG: "+formatted)
}

func (m *mockLogger) Info(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	m.logs = append(m.logs, "INFO: "+formatted)
}

func (m *mockLogger) Warn(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	m.logs = append(m.logs, "WARN: "+formatted)
}

func (m *mockLogger) Error(msg string, args ...interface{}) {
	formatted := fmt.Sprintf(msg, args...)
	m.logs = append(m.logs, "ERROR: "+formatted)
}

func TestNewGenerator(t *testing.T) {
	logger := &mockLogger{}
	generator := NewGenerator(logger)

	if generator == nil {
		t.Fatal("NewGenerator() returned nil")
	}

	if generator.logger != logger {
		t.Error("NewGenerator() did not set logger correctly")
	}
}

func TestGenerateReport_NilReport(t *testing.T) {
	logger := &mockLogger{}
	generator := NewGenerator(logger)

	err := generator.GenerateReport(nil, true)
	if err == nil {
		t.Error("GenerateReport() should return error for nil report")
	}

	if !strings.Contains(err.Error(), "report is nil") {
		t.Errorf("Expected error message about nil report, got: %s", err.Error())
	}
}

func TestGenerateReport_EmptyReport(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	logger := &mockLogger{}
	generator := NewGenerator(logger)

	report := &models.MissingFilesReport{
		GeneratedAt:  "2023-12-01T10:00:00Z",
		RunType:      "dry-run",
		ServiceType:  "sonarr",
		TotalMissing: 0,
		MissingFiles: []models.MissingFileEntry{},
	}

	err := generator.GenerateReport(report, true)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	// Check that reports directory was created
	if _, err := os.Stat("reports"); os.IsNotExist(err) {
		t.Error("Reports directory was not created")
	}

	// Check that JSON file was created
	files, err := filepath.Glob("reports/sonarr-missing-files-report-dryrun-*.json")
	if err != nil {
		t.Fatalf("Failed to glob report files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 report file, found %d", len(files))
	}

	// Verify file content
	if len(files) > 0 {
		content, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("Failed to read report file: %v", err)
		}

		var savedReport models.MissingFilesReport
		if err := json.Unmarshal(content, &savedReport); err != nil {
			t.Fatalf("Failed to unmarshal report: %v", err)
		}

		if savedReport.TotalMissing != 0 {
			t.Errorf("Expected TotalMissing 0, got %d", savedReport.TotalMissing)
		}

		if savedReport.ServiceType != "sonarr" {
			t.Errorf("Expected ServiceType 'sonarr', got '%s'", savedReport.ServiceType)
		}
	}

	// Check terminal output
	infoLogs := 0
	for _, log := range logger.logs {
		if strings.Contains(log, "INFO:") {
			infoLogs++
		}
	}

	if infoLogs == 0 {
		t.Error("Expected INFO logs for terminal output, got none")
	}
}

func TestGenerateReport_WithMissingFiles(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	logger := &mockLogger{}
	generator := NewGenerator(logger)

	season := 1
	episode := 5
	report := &models.MissingFilesReport{
		GeneratedAt:  "2023-12-01T10:00:00Z",
		RunType:      "real-run",
		ServiceType:  "sonarr",
		TotalMissing: 2,
		MissingFiles: []models.MissingFileEntry{
			{
				MediaType:   "series",
				MediaName:   "Test Series",
				EpisodeName: "Test Episode",
				Season:      &season,
				Episode:     &episode,
				FilePath:    "/media/tv/test.mkv",
				FileID:      123,
				ProcessedAt: "2023-12-01T10:00:00Z",
			},
			{
				MediaType:   "movie",
				MediaName:   "Test Movie",
				FilePath:    "/media/movies/test.mp4",
				FileID:      456,
				ProcessedAt: "2023-12-01T10:00:00Z",
			},
		},
	}

	err := generator.GenerateReport(report, true)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	// Check that JSON file was created for real-run
	files, err := filepath.Glob("reports/sonarr-missing-files-report-*.json")
	if err != nil {
		t.Fatalf("Failed to glob report files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 report file, found %d", len(files))
	}

	// Verify file content contains the missing files
	if len(files) > 0 {
		content, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("Failed to read report file: %v", err)
		}

		var savedReport models.MissingFilesReport
		if err := json.Unmarshal(content, &savedReport); err != nil {
			t.Fatalf("Failed to unmarshal report: %v", err)
		}

		if savedReport.TotalMissing != 2 {
			t.Errorf("Expected TotalMissing 2, got %d", savedReport.TotalMissing)
		}

		if len(savedReport.MissingFiles) != 2 {
			t.Errorf("Expected 2 missing file entries, got %d", len(savedReport.MissingFiles))
		}

		// Check first entry (series)
		if len(savedReport.MissingFiles) > 0 {
			entry := savedReport.MissingFiles[0]
			if entry.MediaType != "series" {
				t.Errorf("Expected MediaType 'series', got '%s'", entry.MediaType)
			}
			if entry.MediaName != "Test Series" {
				t.Errorf("Expected MediaName 'Test Series', got '%s'", entry.MediaName)
			}
			if entry.Season == nil || *entry.Season != 1 {
				t.Errorf("Expected Season 1, got %v", entry.Season)
			}
			if entry.Episode == nil || *entry.Episode != 5 {
				t.Errorf("Expected Episode 5, got %v", entry.Episode)
			}
		}
	}

	// Check that terminal output includes missing files info
	hasReportHeader := false
	hasSeriesInfo := false
	hasMovieInfo := false

	for _, log := range logger.logs {
		if strings.Contains(log, "MISSING FILES REPORT") {
			hasReportHeader = true
		}
		if strings.Contains(log, "Test Series") {
			hasSeriesInfo = true
		}
		if strings.Contains(log, "Test Movie") {
			hasMovieInfo = true
		}
	}

	if !hasReportHeader {
		t.Error("Expected report header in terminal output")
	}
	if !hasSeriesInfo {
		t.Error("Expected series info in terminal output")
	}
	if !hasMovieInfo {
		t.Error("Expected movie info in terminal output")
	}
}

func TestGenerateReport_NoTerminalOutput(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	originalWd, _ := os.Getwd()
	defer os.Chdir(originalWd)
	os.Chdir(tempDir)

	logger := &mockLogger{}
	generator := NewGenerator(logger)

	report := &models.MissingFilesReport{
		GeneratedAt:  "2023-12-01T10:00:00Z",
		RunType:      "dry-run",
		ServiceType:  "sonarr",
		TotalMissing: 0,
		MissingFiles: []models.MissingFileEntry{},
	}

	// Generate report without terminal output
	err := generator.GenerateReport(report, false)
	if err != nil {
		t.Fatalf("GenerateReport() failed: %v", err)
	}

	// Check that file was still created
	files, err := filepath.Glob("reports/sonarr-missing-files-report-dryrun-*.json")
	if err != nil {
		t.Fatalf("Failed to glob report files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 report file, found %d", len(files))
	}

	// Check that terminal output is minimal (only file save message)
	reportInfoLogs := 0
	for _, log := range logger.logs {
		if strings.Contains(log, "MISSING FILES REPORT") {
			reportInfoLogs++
		}
	}

	if reportInfoLogs > 0 {
		t.Error("Expected no report output to terminal, but found report content")
	}

	// Should still have the file save message
	hasSaveMessage := false
	for _, log := range logger.logs {
		if strings.Contains(log, "Report saved to:") {
			hasSaveMessage = true
		}
	}

	if !hasSaveMessage {
		t.Error("Expected file save message even with no terminal output")
	}
}
