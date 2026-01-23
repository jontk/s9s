// Package security provides security utilities for validating command paths,
// file paths, and preventing injection attacks.
package security

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ValidateCommandPath validates that a command path is safe to execute.
// It checks for:
// - Path traversal attempts (../)
// - Shell metacharacters that could enable command injection
// - Absolute paths that exist and are executable
//
// For commands in PATH, it uses exec.LookPath to find the absolute path.
// For absolute paths, it verifies the file exists and is executable.
//
// Security note: This helps prevent command injection but does not guarantee
// complete security. Always validate command arguments separately.
func ValidateCommandPath(cmdPath string) (string, error) {
	if cmdPath == "" {
		return "", fmt.Errorf("command path cannot be empty")
	}

	// Check for path traversal attempts
	if strings.Contains(cmdPath, "..") {
		return "", fmt.Errorf("command path contains path traversal: %s", cmdPath)
	}

	// Check for shell metacharacters that could enable injection
	dangerousChars := []string{";", "|", "&", "$", "`", ">", "<", "(", ")", "{", "}", "[", "]", "\\", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(cmdPath, char) {
			return "", fmt.Errorf("command path contains dangerous character %q: %s", char, cmdPath)
		}
	}

	// If it's an absolute path, verify it exists and is executable
	if filepath.IsAbs(cmdPath) {
		info, err := os.Stat(cmdPath)
		if err != nil {
			return "", fmt.Errorf("command path does not exist: %w", err)
		}

		// Check if it's a regular file
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("command path is not a regular file: %s", cmdPath)
		}

		// Check if it's executable (at least one executable bit set)
		if info.Mode()&0111 == 0 {
			return "", fmt.Errorf("command path is not executable: %s", cmdPath)
		}

		return cmdPath, nil
	}

	// For commands in PATH, use exec.LookPath to find absolute path
	// This also validates that the command exists
	absPath, err := exec.LookPath(cmdPath)
	if err != nil {
		return "", fmt.Errorf("command not found in PATH: %w", err)
	}

	return absPath, nil
}

// AllowedCommands defines a whitelist of commands that are known to be safe
// for specific operations. This provides defense in depth beyond path validation.
var AllowedCommands = map[string][]string{
	"slurm": {"scontrol", "squeue", "scancel", "sinfo", "sacct"},
	"ssh":   {"ssh", "ssh-keygen", "ssh-add", "ssh-agent"},
	"shell": {"bash", "sh", "zsh"},
	"notification": {"notify-send", "osascript"},
	"editor": {"vi", "vim", "nvim", "nano", "emacs", "code", "subl"},
}

// IsAllowedCommand checks if a command is in the whitelist for a given category.
// This provides an additional security layer beyond path validation.
func IsAllowedCommand(cmdPath, category string) bool {
	allowed, ok := AllowedCommands[category]
	if !ok {
		return false
	}

	// Get the base command name
	cmdName := filepath.Base(cmdPath)

	for _, allowedCmd := range allowed {
		if cmdName == allowedCmd {
			return true
		}
	}

	return false
}

// ValidateAndResolveCommand combines path validation with whitelist checking.
// It returns the absolute path to the command if it passes all checks.
func ValidateAndResolveCommand(cmdPath, category string) (string, error) {
	// First validate the path
	absPath, err := ValidateCommandPath(cmdPath)
	if err != nil {
		return "", err
	}

	// Then check against whitelist if category is specified
	if category != "" && !IsAllowedCommand(absPath, category) {
		return "", fmt.Errorf("command %q is not allowed for category %q", filepath.Base(absPath), category)
	}

	return absPath, nil
}
