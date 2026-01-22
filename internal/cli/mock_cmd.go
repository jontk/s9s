package cli

import (
	"fmt"

	"github.com/jontk/s9s/internal/mock"
	"github.com/spf13/cobra"
)

// mockCmd represents the mock command group
var mockCmd = &cobra.Command{
	Use:   "mock",
	Short: "Mock mode utilities and status",
	Long: `Utilities for working with s9s mock mode.

Mock mode provides simulated SLURM cluster data for development and testing.
It requires S9S_ENABLE_MOCK environment variable to be set for security.`,
}

// mockStatusCmd shows current mock status
var mockStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show mock mode status and configuration",
	Long: `Display current mock mode status including:
• Whether mock mode is enabled via environment variables
• Production environment detection
• Available environment variable options`,
	RunE: runMockStatus,
}

func init() {
	// Add subcommands
	mockCmd.AddCommand(mockStatusCmd)

	// Add mock command to root
	rootCmd.AddCommand(mockCmd)
}

func runMockStatus(cmd *cobra.Command, args []string) error {
	fmt.Println("🔍 Mock Mode Status")
	fmt.Println("==================")
	fmt.Println()

	// Check mock availability
	if mock.IsMockEnabled() {
		fmt.Println("✅ Mock mode: ENABLED")
		fmt.Printf("   %s\n", mock.GetMockStatusMessage())
	} else {
		fmt.Println("❌ Mock mode: DISABLED")
		fmt.Printf("   %s\n", mock.GetMockStatusMessage())
	}
	fmt.Println()

	// Check production environment
	if mock.IsProductionEnvironment() {
		fmt.Println("🚨 Production environment: DETECTED")
		fmt.Println("   Mock usage will require explicit confirmation")
	} else {
		fmt.Println("🔧 Development environment: DETECTED")
		fmt.Println("   Mock mode can be used without additional warnings")
	}
	fmt.Println()

	// Show configuration options
	fmt.Println("⚙️  Configuration Options:")
	fmt.Println("   To enable mock mode, set one of:")
	fmt.Println("   • S9S_ENABLE_MOCK=development  (recommended for dev)")
	fmt.Println("   • S9S_ENABLE_MOCK=testing      (for testing)")
	fmt.Println("   • S9S_ENABLE_MOCK=debug        (for debugging)")
	fmt.Println("   • S9S_ENABLE_MOCK=local        (for local use)")
	fmt.Println("   • S9S_ENABLE_MOCK=true         (generic enable)")
	fmt.Println()

	fmt.Println("📝 Usage:")
	fmt.Println("   export S9S_ENABLE_MOCK=development")
	fmt.Println("   s9s --mock")
	fmt.Println()

	return nil
}
