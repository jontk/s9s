// Package mock provides mock client validation and mock mode utilities.
package mock

import (
	"fmt"
	"os"
)

// IsMockEnabled checks if mock mode is allowed based on the S9S_ENABLE_MOCK
// environment variable. Any non-empty value enables mock mode.
func IsMockEnabled() bool {
	return os.Getenv("S9S_ENABLE_MOCK") != ""
}

// ValidateMockUsage validates mock usage and returns an error if mock mode
// is requested but not enabled via the environment variable.
func ValidateMockUsage(useMockClient bool) error {
	if !useMockClient {
		return nil
	}

	if !IsMockEnabled() {
		return fmt.Errorf(`mock mode disabled

To enable mock mode, set the S9S_ENABLE_MOCK environment variable:
  export S9S_ENABLE_MOCK=1
  s9s --mock`)
	}

	return nil
}

// SuggestMockSetup provides setup suggestions when mock is requested but not enabled.
func SuggestMockSetup() {
	fmt.Printf("💡 To enable mock mode:\n\n")
	fmt.Printf("  # For current session:\n")
	fmt.Printf("  export S9S_ENABLE_MOCK=1\n\n")
	fmt.Printf("  # For permanent setup, add to your shell profile:\n")
	fmt.Printf("  echo 'export S9S_ENABLE_MOCK=1' >> ~/.bashrc\n")
	fmt.Printf("  echo 'export S9S_ENABLE_MOCK=1' >> ~/.zshrc\n\n")
	fmt.Printf("  # Then run:\n")
	fmt.Printf("  s9s --mock\n\n")
}
