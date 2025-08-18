package filesystem

import (
	"github.com/hnipps/refresharr/internal/arr"
	"os"
)

// FileSystemChecker implements the FileChecker interface
type FileSystemChecker struct{}

// NewFileSystemChecker creates a new FileSystemChecker
func NewFileSystemChecker() arr.FileChecker {
	return &FileSystemChecker{}
}

// FileExists checks if a file exists at the given path
func (f *FileSystemChecker) FileExists(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Make sure it's actually a file, not a directory
	return !info.IsDir()
}

// IsReadable checks if a file exists and is readable
func (f *FileSystemChecker) IsReadable(path string) bool {
	if !f.FileExists(path) {
		return false
	}

	// Try to open the file for reading
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	return true
}
