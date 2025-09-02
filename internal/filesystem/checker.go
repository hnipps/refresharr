package filesystem

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hnipps/refresharr/internal/arr"
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

// IsSymlink checks if a path is a symbolic link
func (f *FileSystemChecker) IsSymlink(path string) bool {
	if path == "" {
		return false
	}

	info, err := os.Lstat(path) // Use Lstat to check symlink status
	if err != nil {
		return false
	}

	return info.Mode()&os.ModeSymlink != 0
}

// FindBrokenSymlinks recursively finds broken symlinks with specified extensions in a directory
func (f *FileSystemChecker) FindBrokenSymlinks(rootDir string, extensions []string) ([]string, error) {
	var brokenSymlinks []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Log the error but continue walking
			return nil
		}

		// Check if this is a symlink
		if info.Mode()&os.ModeSymlink == 0 {
			return nil
		}

		// Check if it has one of the target extensions
		if !hasTargetExtension(path, extensions) {
			return nil
		}

		// Check if the symlink target exists
		if _, err := os.Stat(path); err != nil {
			// Symlink is broken
			brokenSymlinks = append(brokenSymlinks, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking directory %s: %w", rootDir, err)
	}

	return brokenSymlinks, nil
}

// hasTargetExtension checks if a file has one of the target extensions
func hasTargetExtension(path string, extensions []string) bool {
	if len(extensions) == 0 {
		return true // If no extensions specified, include all files
	}

	pathLower := strings.ToLower(path)
	for _, ext := range extensions {
		extLower := strings.ToLower(ext)
		if !strings.HasPrefix(extLower, ".") {
			extLower = "." + extLower
		}
		if strings.HasSuffix(pathLower, extLower) {
			return true
		}
	}

	return false
}
