package util

import (
	"fmt"
	"os"
	"path/filepath"
)

// MkdirWithPerm creates a directory with specified permissions
func MkdirWithPerm(path string, perm os.FileMode) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", path, err)
	}

	// Ensure the directory has the correct permissions
	if err := os.Chmod(path, perm); err != nil {
		return fmt.Errorf("failed to set permissions for directory %s: %v", path, err)
	}

	return nil
}

// EnsureFileDir ensures the parent directory of a file exists
func EnsureFileDir(filePath string) error {
	dir := filepath.Dir(filePath)
	return MkdirWithPerm(dir, 0755)
}

func Exists(filePath string) bool {
	_, err := os.Stat(filePath)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
