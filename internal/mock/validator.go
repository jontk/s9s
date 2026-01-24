package mock

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// IsMockEnabled checks if mock mode is allowed based on environment variables
func IsMockEnabled() bool {
	mockEnv := os.Getenv("S9S_ENABLE_MOCK")

	// Allow specific development/testing values
	allowedValues := []string{
		"true",
		"development",
		"dev",
		"testing",
		"test",
		"debug",
		"local",
	}

	for _, allowed := range allowedValues {
		if strings.ToLower(mockEnv) == allowed {
			return true
		}
	}

	return false
}

// IsProductionEnvironment detects if we're running in production
func IsProductionEnvironment() bool {
	env := strings.ToLower(os.Getenv("ENVIRONMENT"))
	if env == "production" || env == "prod" {
		return true
	}

	// Check common production indicators
	prodIndicators := []string{
		"NODE_ENV=production",
		"GO_ENV=production",
		"RAILS_ENV=production",
	}

	for _, indicator := range prodIndicators {
		parts := strings.SplitN(indicator, "=", 2)
		if len(parts) == 2 && strings.ToLower(os.Getenv(parts[0])) == parts[1] {
			return true
		}
	}

	return false
}

// ValidateMockUsage validates mock usage with appropriate warnings
func ValidateMockUsage(useMockClient bool) error {
	if !useMockClient {
		return nil // No mock usage, nothing to validate
	}

	// Check if mock is enabled
	if !IsMockEnabled() {
		return fmt.Errorf(`mock mode disabled

To enable mock mode, set one of these environment variables:
  S9S_ENABLE_MOCK=development  # For development
  S9S_ENABLE_MOCK=testing      # For testing
  S9S_ENABLE_MOCK=debug        # For debugging
  S9S_ENABLE_MOCK=true         # Generic enable

Example:
  export S9S_ENABLE_MOCK=development
  s9s --mock`)
	}

	// Show warning in production environments
	if IsProductionEnvironment() {
		fmt.Printf("🚨 WARNING: Mock SLURM client enabled in production environment!\n")
		fmt.Printf("   This should only be used for debugging purposes.\n")
		fmt.Printf("   Mock mode provides simulated data, not real cluster information.\n\n")

		// Require explicit confirmation in production
		if !confirmMockInProduction() {
			return fmt.Errorf("mock mode cancelled by user")
		}

		fmt.Printf("✅ Proceeding with mock mode in production (user confirmed)\n\n")
	}

	return nil
}

// confirmMockInProduction asks user for confirmation when using mock in production
func confirmMockInProduction() bool {
	fmt.Print("Are you sure you want to continue with mock mode in production? (yes/no): ")

	// Try to read from terminal first
	if term.IsTerminal(int(syscall.Stdin)) {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			response := strings.ToLower(strings.TrimSpace(scanner.Text()))
			return response == "yes" || response == "y"
		}
	}

	// Fallback for non-interactive environments - default to no
	fmt.Println("(non-interactive terminal detected, defaulting to 'no')")
	return false
}

// GetMockStatusMessage returns a user-friendly message about mock status
func GetMockStatusMessage() string {
	if IsMockEnabled() {
		env := os.Getenv("S9S_ENABLE_MOCK")
		return fmt.Sprintf("Mock mode available (S9S_ENABLE_MOCK=%s)", env)
	}

	return "Mock mode disabled (set S9S_ENABLE_MOCK to enable)"
}

// SuggestMockSetup provides setup suggestions when mock is requested but not enabled
func SuggestMockSetup() {
	fmt.Printf("💡 To enable mock mode for development:\n\n")
	fmt.Printf("  # For current session:\n")
	fmt.Printf("  export S9S_ENABLE_MOCK=development\n\n")
	fmt.Printf("  # For permanent setup, add to your shell profile:\n")
	fmt.Printf("  echo 'export S9S_ENABLE_MOCK=development' >> ~/.bashrc\n")
	fmt.Printf("  echo 'export S9S_ENABLE_MOCK=development' >> ~/.zshrc\n\n")
	fmt.Printf("  # Then run:\n")
	fmt.Printf("  s9s --mock\n\n")
}
