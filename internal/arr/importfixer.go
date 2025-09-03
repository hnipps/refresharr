package arr

import (
	"context"
	"fmt"
	"strings"

	"github.com/hnipps/refresharr/pkg/models"
)

// ImportFixer handles fixing stuck import issues in Sonarr
type ImportFixer struct {
	client Client
	logger Logger
	dryRun bool
}

// NewImportFixer creates a new ImportFixer instance
func NewImportFixer(client Client, logger Logger, dryRun bool) *ImportFixer {
	return &ImportFixer{
		client: client,
		logger: logger,
		dryRun: dryRun,
	}
}

// AnalyzeStuckImports finds all items in the queue with "already imported" issues
func (f *ImportFixer) AnalyzeStuckImports(ctx context.Context) ([]models.QueueItem, error) {
	f.logger.Info("Fetching download queue...")

	queue, err := f.client.GetQueue(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch queue: %w", err)
	}

	if len(queue) == 0 {
		f.logger.Info("No items in queue")
		return []models.QueueItem{}, nil
	}

	f.logger.Info("Found %d items in queue", len(queue))

	var stuckItems []models.QueueItem
	for _, item := range queue {
		if f.isAlreadyImportedIssue(item) {
			stuckItems = append(stuckItems, item)
		}
	}

	f.logger.Info("Found %d items with 'already imported' issues", len(stuckItems))

	// Log details about stuck items
	for _, item := range stuckItems {
		seriesTitle := "Unknown Series"
		if item.Series != nil {
			seriesTitle = item.Series.Title
		}

		sizeMB := float64(item.Size) / (1024 * 1024)
		f.logger.Info("  ID: %d | %s - %s (%.2f MB)", item.ID, seriesTitle, item.Title, sizeMB)

		// Show additional queue item details if available
		if item.DownloadID != "" {
			f.logger.Info("    DownloadID: %s", item.DownloadID)
		}
		if item.OutputPath != "" {
			f.logger.Info("    OutputPath: %s", item.OutputPath)
		}
		if item.Protocol != "" {
			f.logger.Info("    Protocol: %s", item.Protocol)
		}
		if item.DownloadClient != "" {
			f.logger.Info("    DownloadClient: %s", item.DownloadClient)
		}

		// Show status messages if available
		for i, msg := range item.StatusMessages {
			if i >= 3 { // Limit to first 3 messages
				break
			}
			f.logger.Info("    → %s", msg.Title)
		}
	}

	return stuckItems, nil
}

// isAlreadyImportedIssue checks if a queue item has the "already imported" issue
func (f *ImportFixer) isAlreadyImportedIssue(item models.QueueItem) bool {
	// Check if it's waiting to import (completed status)
	status := strings.ToLower(item.Status)
	if status != "completed" {
		return false
	}

	// Check status messages for the specific issue
	for _, message := range item.StatusMessages {
		msgText := strings.ToLower(message.Title)
		if f.containsImportIssueKeywords(msgText) {
			return true
		}
	}

	// Also check error message
	errorMsg := strings.ToLower(item.ErrorMessage)
	if f.containsImportIssueKeywords(errorMsg) {
		return true
	}

	return false
}

// containsImportIssueKeywords checks if a message contains import issue keywords
func (f *ImportFixer) containsImportIssueKeywords(message string) bool {
	keywords := []string{
		"already imported",
		"episode file already imported",
		"one or more episodes expected",
		"missing from the release",
	}

	for _, keyword := range keywords {
		if strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}

// FixImports analyzes and fixes all stuck import issues
func (f *ImportFixer) FixImports(ctx context.Context, removeFromClient bool) (*models.ImportFixResult, error) {
	stuckItems, err := f.AnalyzeStuckImports(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze stuck imports: %w", err)
	}

	result := &models.ImportFixResult{
		TotalStuckItems: len(stuckItems),
		FixedItems:      0,
		Errors:          []string{},
		Success:         true,
		DryRun:          f.dryRun,
	}

	if len(stuckItems) == 0 {
		f.logger.Info("No stuck imports found to fix!")
		return result, nil
	}

	if f.dryRun {
		f.logger.Info("[DRY RUN] Would attempt to import %d stuck import(s)", len(stuckItems))
		f.logger.Info("Items that fail to import will be left in queue for manual resolution")
		f.logger.Info("Run without --dry-run to actually process these items")
		return result, nil
	}

	f.logger.Info("Processing %d stuck imports - attempting to import without removing from queue...", len(stuckItems))

	// First, try to trigger a download client scan to refresh stuck imports
	f.logger.Info("Triggering download client scan to refresh stuck imports...")
	if err := f.client.TriggerDownloadClientScan(ctx); err != nil {
		f.logger.Warn("Failed to trigger download client scan: %s (continuing anyway)", err.Error())
	}

	for _, item := range stuckItems {
		seriesTitle := "Unknown Series"
		if item.Series != nil {
			seriesTitle = item.Series.Title
		}

		f.logger.Info("Processing: %s - %s (ID: %d)", seriesTitle, item.Title, item.ID)

		// Attempt manual import
		imported := f.attemptManualImport(ctx, item)

		if imported {
			f.logger.Info("  ✓ Successfully imported via manual import")
			result.FixedItems++
		} else {
			// Log failure but do NOT remove from queue - leave for manual resolution
			errMsg := fmt.Sprintf("Failed to import queue item %d (%s - %s). Item left in queue for manual resolution.", item.ID, seriesTitle, item.Title)
			f.logger.Warn("  ⚠ %s", errMsg)
			result.Errors = append(result.Errors, errMsg)
			// Note: We don't set Success = false here since this is expected behavior
		}
	}

	f.logger.Info("Import results: %d/%d successfully imported, %d left in queue for manual resolution",
		result.FixedItems, result.TotalStuckItems, result.TotalStuckItems-result.FixedItems)

	if len(result.Errors) > 0 {
		f.logger.Info("Items requiring manual attention:")
		for _, errMsg := range result.Errors {
			f.logger.Info("  • %s", errMsg)
		}
	}
	return result, nil
}

// TestConnection tests the connection to the service
func (f *ImportFixer) TestConnection(ctx context.Context) error {
	return f.client.TestConnection(ctx)
}

// attemptManualImport tries to manually import a stuck queue item
func (f *ImportFixer) attemptManualImport(ctx context.Context, item models.QueueItem) bool {
	if item.Series == nil {
		f.logger.Debug("  → No series information available for manual import")
		return false
	}

	seriesTitle := item.Series.Title
	f.logger.Debug("  → Attempting manual import for: %s", seriesTitle)

	// Strategy 1: Try using OutputPath if available
	if item.OutputPath != "" {
		f.logger.Debug("  → Trying OutputPath: %s", item.OutputPath)
		if f.tryManualImportByPath(ctx, item.OutputPath, item) {
			f.logger.Info("  → Successfully imported using OutputPath")
			return true
		}
	}

	// Strategy 2: Try using DownloadID if available
	if item.DownloadID != "" {
		f.logger.Debug("  → Trying DownloadID: %s", item.DownloadID)
		if f.tryManualImportByDownloadID(ctx, item.DownloadID, item) {
			f.logger.Info("  → Successfully imported using DownloadID")
			return true
		}
	}

	// Strategy 3: Try using Series ID approach (scan for files matching the series)
	f.logger.Debug("  → Trying SeriesID approach for series: %s (ID: %d)", seriesTitle, item.Series.ID)
	if f.tryManualImportBySeriesID(ctx, item) {
		f.logger.Info("  → Successfully imported using SeriesID approach")
		return true
	}

	f.logger.Debug("  → All manual import strategies failed")
	return false
}

// tryManualImportByPath attempts manual import using a specific folder path
func (f *ImportFixer) tryManualImportByPath(ctx context.Context, folderPath string, item models.QueueItem) bool {
	f.logger.Debug("    → Scanning folder for importable files: %s", folderPath)

	// Get files available for manual import from this folder
	manualImportItems, err := f.client.GetManualImport(ctx, folderPath)
	if err != nil {
		f.logger.Debug("    → Failed to get manual import items for folder %s: %s", folderPath, err.Error())
		return false
	}

	if len(manualImportItems) == 0 {
		f.logger.Debug("    → No importable files found in folder %s", folderPath)
		return false
	}

	f.logger.Debug("    → Found %d potential files for import", len(manualImportItems))

	// Filter files that match our queue item and series
	matchedFiles := f.filterMatchingFiles(manualImportItems, item)
	if len(matchedFiles) == 0 {
		f.logger.Debug("    → No files matched the queue item criteria")
		return false
	}

	f.logger.Debug("    → %d files matched queue item criteria", len(matchedFiles))

	// Execute manual import for matched files
	return f.executeManualImport(ctx, matchedFiles, item)
}

// tryManualImportByDownloadID attempts manual import using download ID
func (f *ImportFixer) tryManualImportByDownloadID(ctx context.Context, downloadID string, item models.QueueItem) bool {
	f.logger.Debug("    → Attempting manual import using downloadID: %s", downloadID)

	// Use the enhanced GetManualImportWithParams method with downloadID
	manualImportItems, err := f.client.GetManualImportWithParams(ctx, "", downloadID, 0, true)
	if err != nil {
		f.logger.Debug("    → Failed to get manual import items by downloadID: %s", err.Error())
	} else if len(manualImportItems) > 0 {
		f.logger.Debug("    → Found %d files using downloadID", len(manualImportItems))
		matchedFiles := f.filterMatchingFiles(manualImportItems, item)
		if len(matchedFiles) > 0 {
			return f.executeManualImport(ctx, matchedFiles, item)
		}
	}

	// Fallback: Try to find common download folders and search there
	commonDownloadPaths := []string{
		"/downloads/complete",
		"/downloads",
		"/mnt/downloads",
		"/data/downloads",
	}

	for _, basePath := range commonDownloadPaths {
		f.logger.Debug("    → Trying common download path: %s", basePath)
		if f.tryManualImportByPath(ctx, basePath, item) {
			return true
		}
	}

	f.logger.Debug("    → DownloadID approach failed - no files found")
	return false
}

// tryManualImportBySeriesID attempts manual import by scanning for series-related files
func (f *ImportFixer) tryManualImportBySeriesID(ctx context.Context, item models.QueueItem) bool {
	if item.Series == nil {
		return false
	}

	f.logger.Debug("    → Attempting import using series information")

	// Try series-specific paths first
	seriesPaths := []string{
		fmt.Sprintf("/downloads/complete/%s", item.Series.Title),
		fmt.Sprintf("/downloads/%s", item.Series.Title),
		fmt.Sprintf("/mnt/downloads/%s", item.Series.Title),
	}

	for _, seriesPath := range seriesPaths {
		f.logger.Debug("    → Trying series-specific path: %s", seriesPath)
		if f.tryManualImportByPath(ctx, seriesPath, item) {
			return true
		}
	}

	// If series-specific paths don't work, try generic download paths
	// but filter more strictly by series ID
	f.logger.Debug("    → Trying generic paths with series filtering")
	return f.tryGenericPathsWithSeriesFiltering(ctx, item)
}

// tryGenericPathsWithSeriesFiltering tries common download paths with strict series filtering
func (f *ImportFixer) tryGenericPathsWithSeriesFiltering(ctx context.Context, item models.QueueItem) bool {
	// First try the enhanced method with series ID filtering
	f.logger.Debug("    → Trying enhanced series ID filtering")
	manualImportItems, err := f.client.GetManualImportWithParams(ctx, "", "", item.Series.ID, true)
	if err != nil {
		f.logger.Debug("    → Enhanced series ID filtering failed: %s", err.Error())
	} else if len(manualImportItems) > 0 {
		f.logger.Debug("    → Found %d files using series ID filtering", len(manualImportItems))
		matchedFiles := f.filterFilesBySeriesID(manualImportItems, item.Series.ID)
		if len(matchedFiles) > 0 {
			if f.executeManualImport(ctx, matchedFiles, item) {
				return true
			}
		}
	}

	// Fallback to path-based scanning
	commonPaths := []string{
		"/downloads/complete",
		"/downloads",
		"/mnt/downloads",
		"/data/downloads",
	}

	for _, path := range commonPaths {
		f.logger.Debug("    → Scanning %s for series %s files", path, item.Series.Title)

		manualImportItems, err := f.client.GetManualImport(ctx, path)
		if err != nil {
			f.logger.Debug("    → Failed to scan %s: %s", path, err.Error())
			continue
		}

		// Apply strict filtering for this series only
		matchedFiles := f.filterFilesBySeriesID(manualImportItems, item.Series.ID)
		if len(matchedFiles) > 0 {
			f.logger.Debug("    → Found %d files for series in %s", len(matchedFiles), path)
			if f.executeManualImport(ctx, matchedFiles, item) {
				return true
			}
		}
	}

	return false
}

// filterMatchingFiles filters manual import items to those matching the queue item
func (f *ImportFixer) filterMatchingFiles(items []models.ManualImportItem, queueItem models.QueueItem) []models.ManualImportItem {
	var matched []models.ManualImportItem

	for _, item := range items {
		// Check if file matches our series
		if item.Series != nil && queueItem.Series != nil {
			if item.Series.ID == queueItem.Series.ID {
				f.logger.Debug("      → Matched file: %s (Series: %s)", item.Name, item.Series.Title)
				matched = append(matched, item)
			}
		} else if queueItem.DownloadID != "" && item.DownloadID == queueItem.DownloadID {
			// If no series match, try download ID match
			f.logger.Debug("      → Matched file by downloadID: %s", item.Name)
			matched = append(matched, item)
		}
	}

	return matched
}

// filterFilesBySeriesID filters files strictly by series ID
func (f *ImportFixer) filterFilesBySeriesID(items []models.ManualImportItem, seriesID int) []models.ManualImportItem {
	var matched []models.ManualImportItem

	for _, item := range items {
		if item.Series != nil && item.Series.ID == seriesID {
			matched = append(matched, item)
		}
	}

	return matched
}

// executeManualImport executes the manual import for the given files
func (f *ImportFixer) executeManualImport(ctx context.Context, files []models.ManualImportItem, queueItem models.QueueItem) bool {
	if len(files) == 0 {
		return false
	}

	f.logger.Debug("    → Executing manual import for %d files", len(files))

	// Log files being imported
	for _, file := range files {
		seriesInfo := "Unknown Series"
		if file.Series != nil {
			seriesInfo = file.Series.Title
		}
		f.logger.Debug("      → Importing: %s (%s)", file.Name, seriesInfo)
	}

	// Execute the manual import with "move" mode (safer than copy)
	err := f.client.ExecuteManualImport(ctx, files, "move")
	if err != nil {
		f.logger.Debug("    → Manual import failed: %s", err.Error())
		return false
	}

	f.logger.Debug("    → Manual import command executed successfully")
	return true
}
