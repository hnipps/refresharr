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
		f.logger.Info("[DRY RUN] Would remove %d stuck import(s)", len(stuckItems))
		f.logger.Info("Run without --dry-run to actually remove these items")
		return result, nil
	}

	f.logger.Info("Removing %d stuck imports...", len(stuckItems))

	for _, item := range stuckItems {
		seriesTitle := "Unknown Series"
		if item.Series != nil {
			seriesTitle = item.Series.Title
		}

		f.logger.Info("Removing: %s - %s (ID: %d)", seriesTitle, item.Title, item.ID)

		if err := f.client.RemoveFromQueue(ctx, item.ID, removeFromClient); err != nil {
			errMsg := fmt.Sprintf("Failed to remove queue item %d: %s", item.ID, err.Error())
			f.logger.Error("  ✗ %s", errMsg)
			result.Errors = append(result.Errors, errMsg)
			result.Success = false
		} else {
			f.logger.Info("  ✓ Successfully removed")
			result.FixedItems++
		}
	}

	f.logger.Info("Fixed %d/%d imports", result.FixedItems, result.TotalStuckItems)
	return result, nil
}

// TestConnection tests the connection to the service
func (f *ImportFixer) TestConnection(ctx context.Context) error {
	return f.client.TestConnection(ctx)
}
