package arr

import (
	"context"
	"testing"

	"github.com/hnipps/refresharr/pkg/models"
)

func TestImportFixer_isAlreadyImportedIssue(t *testing.T) {
	logger := &mockLogger{}
	fixer := NewImportFixer(nil, logger, true)

	tests := []struct {
		name     string
		item     models.QueueItem
		expected bool
	}{
		{
			name: "completed status with import issue",
			item: models.QueueItem{
				Status: "completed",
				StatusMessages: []models.StatusMessage{
					{Title: "One or more episodes expected in this release were not imported or missing from the release"},
				},
			},
			expected: true,
		},
		{
			name: "completed status with already imported issue",
			item: models.QueueItem{
				Status: "completed",
				StatusMessages: []models.StatusMessage{
					{Title: "Episode file already imported"},
				},
			},
			expected: true,
		},
		{
			name: "completed status with error message containing already imported",
			item: models.QueueItem{
				Status:       "completed",
				ErrorMessage: "This file has already imported",
			},
			expected: true,
		},
		{
			name: "downloading status with import issue should not match",
			item: models.QueueItem{
				Status: "downloading",
				StatusMessages: []models.StatusMessage{
					{Title: "One or more episodes expected in this release were not imported or missing from the release"},
				},
			},
			expected: false,
		},
		{
			name: "completed status with no import issues",
			item: models.QueueItem{
				Status: "completed",
				StatusMessages: []models.StatusMessage{
					{Title: "Some other message"},
				},
			},
			expected: false,
		},
		{
			name: "empty status messages",
			item: models.QueueItem{
				Status:         "completed",
				StatusMessages: []models.StatusMessage{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixer.isAlreadyImportedIssue(tt.item)
			if result != tt.expected {
				t.Errorf("isAlreadyImportedIssue() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestImportFixer_containsImportIssueKeywords(t *testing.T) {
	logger := &mockLogger{}
	fixer := NewImportFixer(nil, logger, true)

	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "contains 'already imported'",
			message:  "This file has already imported",
			expected: true,
		},
		{
			name:     "contains 'episode file already imported'",
			message:  "Episode file already imported from previous download",
			expected: true,
		},
		{
			name:     "contains 'one or more episodes expected'",
			message:  "one or more episodes expected in this release were not imported",
			expected: true,
		},
		{
			name:     "contains 'missing from the release'",
			message:  "episodes are missing from the release",
			expected: true,
		},
		{
			name:     "case insensitive matching (already lowercase)",
			message:  "already imported file",
			expected: true,
		},
		{
			name:     "no matching keywords",
			message:  "Some other error message",
			expected: false,
		},
		{
			name:     "empty message",
			message:  "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fixer.containsImportIssueKeywords(tt.message)
			if result != tt.expected {
				t.Errorf("containsImportIssueKeywords() = %v, want %v for message: %s", result, tt.expected, tt.message)
			}
		})
	}
}

func TestNewImportFixer(t *testing.T) {
	logger := &mockLogger{}
	client := &mockClient{}

	tests := []struct {
		name   string
		dryRun bool
	}{
		{
			name:   "create fixer with dry run enabled",
			dryRun: true,
		},
		{
			name:   "create fixer with dry run disabled",
			dryRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixer := NewImportFixer(client, logger, tt.dryRun)

			if fixer == nil {
				t.Fatal("NewImportFixer() returned nil")
			}

			if fixer.client != client {
				t.Error("NewImportFixer() did not set client correctly")
			}

			if fixer.logger != logger {
				t.Error("NewImportFixer() did not set logger correctly")
			}

			if fixer.dryRun != tt.dryRun {
				t.Errorf("NewImportFixer() dryRun = %v, want %v", fixer.dryRun, tt.dryRun)
			}
		})
	}
}

func TestImportFixer_TestConnection(t *testing.T) {
	logger := &mockLogger{}
	client := &mockClient{}
	fixer := NewImportFixer(client, logger, true)

	ctx := context.Background()
	err := fixer.TestConnection(ctx)

	if err != nil {
		t.Errorf("TestConnection() returned error: %v", err)
	}
}
