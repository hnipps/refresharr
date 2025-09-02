package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/hnipps/refresharr/pkg/models"
)

// Generator handles the generation and output of missing files reports
type Generator struct {
	logger Logger
}

// Logger defines the interface for logging operations
type Logger interface {
	Debug(msg string, args ...interface{})
	Info(msg string, args ...interface{})
	Warn(msg string, args ...interface{})
	Error(msg string, args ...interface{})
}

// NewGenerator creates a new report generator
func NewGenerator(logger Logger) *Generator {
	return &Generator{
		logger: logger,
	}
}

// GenerateReport creates a missing files report and optionally saves it to disk and prints it
func (g *Generator) GenerateReport(report *models.MissingFilesReport, printToTerminal bool) error {
	if report == nil {
		return fmt.Errorf("report is nil")
	}

	// Always save report to disk
	if err := g.saveReportToDisk(report); err != nil {
		return fmt.Errorf("failed to save report to disk: %w", err)
	}

	// Print to terminal if requested
	if printToTerminal {
		g.printReportToTerminal(report)
	}

	return nil
}

// saveReportToDisk saves the report as JSON to the reports directory
func (g *Generator) saveReportToDisk(report *models.MissingFilesReport) error {
	// Create reports directory if it doesn't exist
	reportsDir := "reports"
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return fmt.Errorf("failed to create reports directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102-150405")
	filename := fmt.Sprintf("%s-missing-files-report-%s.json", report.ServiceType, timestamp)
	if report.RunType == "dry-run" {
		filename = fmt.Sprintf("%s-missing-files-report-dryrun-%s.json", report.ServiceType, timestamp)
	}

	filepath := filepath.Join(reportsDir, filename)

	// Marshal report to JSON with pretty printing
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report to JSON: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write report file: %w", err)
	}

	g.logger.Info("📄 Report saved to: %s", filepath)
	return nil
}

// printReportToTerminal prints the report in human-readable format to the terminal
func (g *Generator) printReportToTerminal(report *models.MissingFilesReport) {
	g.logger.Info("")
	g.logger.Info("📊 MISSING FILES REPORT")
	g.logger.Info("==========================================")
	g.logger.Info("Generated: %s", report.GeneratedAt)
	g.logger.Info("Service: %s", report.ServiceType)
	g.logger.Info("Run Type: %s", report.RunType)
	g.logger.Info("Total Missing Files: %d", report.TotalMissing)
	g.logger.Info("")

	if report.TotalMissing == 0 {
		g.logger.Info("🎉 No missing files found!")
		return
	}

	g.logger.Info("Missing Files:")
	g.logger.Info("==========================================")

	for i, entry := range report.MissingFiles {
		g.logger.Info("%d. %s", i+1, entry.MediaName)

		if entry.MediaType == "series" && entry.Season != nil && entry.Episode != nil {
			episodeName := entry.EpisodeName
			if episodeName == "" {
				episodeName = "Unknown Episode"
			}
			g.logger.Info("   Episode: S%02dE%02d - %s", *entry.Season, *entry.Episode, episodeName)
		}

		g.logger.Info("   Missing File: %s", entry.FilePath)
		g.logger.Info("   File ID: %d", entry.FileID)
		g.logger.Info("   Processed: %s", entry.ProcessedAt)

		if i < len(report.MissingFiles)-1 {
			g.logger.Info("")
		}
	}

	g.logger.Info("==========================================")
}
