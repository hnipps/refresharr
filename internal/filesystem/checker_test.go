package filesystem

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSystemChecker_FileExists(t *testing.T) {
	checker := NewFileSystemChecker()

	// Create a temporary directory and file for testing
	tempDir, err := os.MkdirTemp("", "refresharr-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "test-file.txt")
	if err := os.WriteFile(tempFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "existing file",
			path:     tempFile,
			expected: true,
		},
		{
			name:     "non-existent file",
			path:     filepath.Join(tempDir, "non-existent.txt"),
			expected: false,
		},
		{
			name:     "directory path",
			path:     tempDir,
			expected: false, // Should return false for directories
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
		{
			name:     "invalid path",
			path:     "/invalid/path/that/does/not/exist.txt",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.FileExists(tt.path)
			if result != tt.expected {
				t.Errorf("FileExists(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestFileSystemChecker_IsReadable(t *testing.T) {
	checker := NewFileSystemChecker()

	// Create a temporary directory and files for testing
	tempDir, err := os.MkdirTemp("", "refresharr-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a readable file
	readableFile := filepath.Join(tempDir, "readable.txt")
	if err := os.WriteFile(readableFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create readable file: %v", err)
	}

	// Create a file with no read permissions (skip on Windows)
	unreadableFile := filepath.Join(tempDir, "unreadable.txt")
	if err := os.WriteFile(unreadableFile, []byte("test content"), 0000); err != nil {
		t.Fatalf("Failed to create unreadable file: %v", err)
	}

	tests := []struct {
		name          string
		path          string
		expected      bool
		skipOnWindows bool
	}{
		{
			name:     "readable file",
			path:     readableFile,
			expected: true,
		},
		{
			name:          "unreadable file",
			path:          unreadableFile,
			expected:      false,
			skipOnWindows: true, // Windows doesn't respect file permissions in the same way
		},
		{
			name:     "non-existent file",
			path:     filepath.Join(tempDir, "non-existent.txt"),
			expected: false,
		},
		{
			name:     "directory path",
			path:     tempDir,
			expected: false, // Should return false for directories
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip permission tests on Windows
			if tt.skipOnWindows && os.Getenv("GOOS") == "windows" {
				t.Skip("Skipping permission test on Windows")
			}

			result := checker.IsReadable(tt.path)
			if result != tt.expected {
				t.Errorf("IsReadable(%s) = %v, expected %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestNewFileSystemChecker(t *testing.T) {
	checker := NewFileSystemChecker()
	if checker == nil {
		t.Error("NewFileSystemChecker() returned nil")
	}

	// Verify it implements the FileChecker interface by testing method calls
	exists := checker.FileExists("/dev/null") // This path exists on Unix-like systems
	_ = exists                                // We don't care about the result, just that the method exists

	readable := checker.IsReadable("/dev/null")
	_ = readable // We don't care about the result, just that the method exists
}

func TestFileSystemChecker_DeleteSymlink(t *testing.T) {
	checker := NewFileSystemChecker()

	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "refresharr-symlink-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a target file
	targetFile := filepath.Join(tempDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("target content"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	// Create a valid symlink
	validSymlink := filepath.Join(tempDir, "valid-symlink.txt")
	if err := os.Symlink(targetFile, validSymlink); err != nil {
		t.Skipf("Symlink creation not supported on this system: %v", err)
	}

	// Create a broken symlink
	brokenSymlink := filepath.Join(tempDir, "broken-symlink.txt")
	nonExistentTarget := filepath.Join(tempDir, "does-not-exist.txt")
	if err := os.Symlink(nonExistentTarget, brokenSymlink); err != nil {
		t.Skipf("Symlink creation not supported on this system: %v", err)
	}

	// Create a regular file (not a symlink)
	regularFile := filepath.Join(tempDir, "regular.txt")
	if err := os.WriteFile(regularFile, []byte("regular content"), 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		shouldError bool
		errorText   string
	}{
		{
			name:        "delete valid symlink",
			path:        validSymlink,
			shouldError: false,
		},
		{
			name:        "delete broken symlink",
			path:        brokenSymlink,
			shouldError: false,
		},
		{
			name:        "try to delete regular file",
			path:        regularFile,
			shouldError: true,
			errorText:   "is not a symlink",
		},
		{
			name:        "try to delete non-existent file",
			path:        filepath.Join(tempDir, "does-not-exist.txt"),
			shouldError: true,
			errorText:   "failed to stat symlink",
		},
		{
			name:        "empty path",
			path:        "",
			shouldError: true,
			errorText:   "failed to stat symlink",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checker.DeleteSymlink(tt.path)

			if tt.shouldError {
				if err == nil {
					t.Errorf("DeleteSymlink(%s) expected error but got nil", tt.path)
				} else if tt.errorText != "" && !containsString(err.Error(), tt.errorText) {
					t.Errorf("DeleteSymlink(%s) error = %v, expected to contain %s", tt.path, err, tt.errorText)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteSymlink(%s) unexpected error = %v", tt.path, err)
				}

				// Verify the symlink was actually deleted
				if _, err := os.Lstat(tt.path); err == nil {
					t.Errorf("DeleteSymlink(%s) did not delete the symlink", tt.path)
				}
			}
		})
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr))))
}

// Simple substring search helper
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
