package cli

import (
	"fmt"

	"github.com/jontk/s9s/internal/mock"
	"github.com/spf13/cobra"
)

// mockCmd represents the mock command group
// Hidden from help as it's for development/testing only
var mockCmd = &cobra.Command{
	Use:    "mock",
	Short:  "Mock mode utilities and status",
	Hidden: true,
	Long: `Utilities for working with s9s mock mode.

Mock mode provides simulated SLURM cluster data for development and testing.
It requires S9S_ENABLE_MOCK environment variable to be set.`,
}

// mockStatusCmd shows current mock status
var mockStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mock mode status and configuration",
	RunE:  runMockStatus,
}

func init() {
	mockCmd.AddCommand(mockStatusCmd)
	rootCmd.AddCommand(mockCmd)
}

func runMockStatus(_ *cobra.Command, _ []string) error {
	fmt.Println("🔍 Mock Mode Status")
	fmt.Println("==================")
	fmt.Println()

	if mock.IsMockEnabled() {
		fmt.Println("✅ Mock mode: ENABLED")
	} else {
		fmt.Println("❌ Mock mode: DISABLED")
		fmt.Println()
		mock.SuggestMockSetup()
	}

	return nil
}
