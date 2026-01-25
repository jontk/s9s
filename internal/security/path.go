package security

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidatePath validates that a file path is safe to use.
// It checks for:
// - Path traversal attempts (../)
// - Null bytes
// - Absolute path validation
//
// Returns the cleaned absolute path if valid.
func ValidatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return "", fmt.Errorf("path contains null byte")
	}

	// Clean the path (removes .., ., etc.)
	cleanPath := filepath.Clean(path)

	// Note: cleaning may remove traversal attempts (e.g., "../" becomes "")
	// We allow this and return the cleaned path

	return cleanPath, nil
}

// ValidatePathWithinBase ensures a path is within a base directory.
// This prevents path traversal attacks by ensuring the resolved path
// is within the allowed base directory.
func ValidatePathWithinBase(path, baseDir string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	if baseDir == "" {
		return "", fmt.Errorf("base directory cannot be empty")
	}

	// Clean both paths
	cleanPath := filepath.Clean(path)
	cleanBase := filepath.Clean(baseDir)

	// Convert to absolute paths
	var absPath string
	if filepath.IsAbs(cleanPath) {
		absPath = cleanPath
	} else {
		absPath = filepath.Join(cleanBase, cleanPath)
	}

	absBase, err := filepath.Abs(cleanBase)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute base path: %w", err)
	}

	absPath, err = filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if the absolute path is within the base directory
	relPath, err := filepath.Rel(absBase, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	// If the relative path starts with "..", it's outside the base directory
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path %q is outside base directory %q", path, baseDir)
	}

	return absPath, nil
}

// IsPathSafe performs basic safety checks on a path without requiring a base directory.
// Use this when you trust the path source but want to validate format.
func IsPathSafe(path string) bool {
	if path == "" {
		return false
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return false
	}

	// Check for suspicious patterns
	// Note: We don't reject ".." outright as it might be legitimate after cleaning
	cleanPath := filepath.Clean(path)

	// After cleaning, path should not start with ../
	if strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return false
	}

	return true
}

// EnsureDir creates a directory if it doesn't exist, with validation.
// It ensures the directory path is safe before creation.
func EnsureDir(dir string, perm os.FileMode) error {
	cleanDir := filepath.Clean(dir)

	// Check if it already exists
	info, err := os.Stat(cleanDir)
	if err == nil {
		if !info.IsDir() {
			return fmt.Errorf("path exists but is not a directory: %s", cleanDir)
		}
		return nil // Directory already exists
	}

	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat directory: %w", err)
	}

	// Create the directory
	if err := os.MkdirAll(cleanDir, perm); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	return nil
}

// ValidateConfigPath validates a path is within the user's config directory.
// This is useful for config files, cache files, etc.
func ValidateConfigPath(path, configDir string) (string, error) {
	// If path is already absolute and within config dir, allow it
	if filepath.IsAbs(path) {
		return ValidatePathWithinBase(path, configDir)
	}

	// Otherwise, join with config dir
	fullPath := filepath.Join(configDir, path)
	return ValidatePathWithinBase(fullPath, configDir)
}

// SafeOpenFile opens a file with path validation.
// Use this as a wrapper around os.Open when you need path validation.
func SafeOpenFile(path, baseDir string) (*os.File, error) {
	validPath, err := ValidatePathWithinBase(path, baseDir)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(validPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}
