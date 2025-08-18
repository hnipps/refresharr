package arr

import (
	"github.com/hnipps/refresharr/pkg/models"
)

// ConsoleProgressReporter implements the ProgressReporter interface for console output
type ConsoleProgressReporter struct {
	logger Logger
}

// NewConsoleProgressReporter creates a new ConsoleProgressReporter
func NewConsoleProgressReporter(logger Logger) ProgressReporter {
	return &ConsoleProgressReporter{
		logger: logger,
	}
}

// StartSeries reports the start of processing a series
func (r *ConsoleProgressReporter) StartSeries(seriesID int, seriesName string, current, total int) {
	r.logger.Info("")
	r.logger.Info("Processing series %d/%d (ID: %d)", current, total, seriesID)
	r.logger.Info("Series: %s", seriesName)
}

// StartEpisode reports the start of processing an episode
func (r *ConsoleProgressReporter) StartEpisode(episodeID int, seasonNum, episodeNum int) {
	r.logger.Info("  Checking S%dE%d (Episode ID: %d)", seasonNum, episodeNum, episodeID)
}

// ReportMissingFile reports that a file is missing
func (r *ConsoleProgressReporter) ReportMissingFile(filePath string) {
	r.logger.Warn("    ❌ MISSING: %s", filePath)
}

// ReportDeletedRecord reports that a record was deleted
func (r *ConsoleProgressReporter) ReportDeletedRecord(fileID int) {
	r.logger.Info("    ✅ Successfully deleted episode file record (ID: %d)", fileID)
}

// ReportError reports an error during processing
func (r *ConsoleProgressReporter) ReportError(err error) {
	r.logger.Error("    ❌ Error: %s", err.Error())
}

// Finish reports the final cleanup statistics
func (r *ConsoleProgressReporter) Finish(stats models.CleanupStats) {
	r.logger.Info("")
	r.logger.Info("================================================")
	r.logger.Info("Cleanup Summary:")
	r.logger.Info("  Total items checked: %d", stats.TotalItemsChecked)
	r.logger.Info("  Missing files found: %d", stats.MissingFiles)
	r.logger.Info("  Records deleted: %d", stats.DeletedRecords)
	if stats.Errors > 0 {
		r.logger.Warn("  Errors encountered: %d", stats.Errors)
	}
	r.logger.Info("")

	if stats.DeletedRecords > 0 {
		r.logger.Info("🔄 Triggering refresh to update status...")
	} else {
		r.logger.Info("ℹ️  No missing files found - nothing to clean up.")
	}
}
