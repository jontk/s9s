package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/discovery"
	"github.com/jontk/s9s/internal/setup"
	"github.com/spf13/cobra"
)

// setupCmd represents the setup command
var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard for s9s configuration",
	Long: `Launch the interactive setup wizard to configure s9s for first-time use.

The setup wizard will guide you through:
• 🏢 Cluster connection settings  
• 🔐 Authentication configuration
• 🔒 Secure credential storage
• ⚡ Performance optimization

Run this command when you first install s9s or want to reconfigure your setup.`,
	Example: `  s9s setup                    # Run interactive setup
  s9s setup --auto-discover     # Auto-discover clusters first
  s9s setup --validate-only     # Just validate current config`,
	RunE: runSetup,
}

// setupFlags holds the flags for the setup command
type setupFlags struct {
	autoDiscover bool
	validateOnly bool
	force        bool
	configPath   string
}

var setupFlagValues setupFlags

func init() {
	// Add setup command flags
	setupCmd.Flags().BoolVar(&setupFlagValues.autoDiscover, "auto-discover", false, "Auto-discover SLURM clusters before setup")
	setupCmd.Flags().BoolVar(&setupFlagValues.validateOnly, "validate-only", false, "Only validate existing configuration")
	setupCmd.Flags().BoolVar(&setupFlagValues.force, "force", false, "Force setup even if configuration exists")
	setupCmd.Flags().StringVar(&setupFlagValues.configPath, "config", "", "Path to configuration file")

	// Add to root command
	rootCmd.AddCommand(setupCmd)
}

// runSetup executes the setup command
func runSetup(cmd *cobra.Command, args []string) error {
	if setupFlagValues.validateOnly {
		return runConfigValidation()
	}

	if setupFlagValues.autoDiscover {
		if err := runAutoDiscovery(); err != nil {
			fmt.Printf("⚠️  Auto-discovery failed: %v\n", err)
			fmt.Println("Continuing with manual setup...")
		}
	}

	// Run the setup wizard
	wizard := setup.NewSetupWizard()
	return wizard.Run()
}

// runConfigValidation validates the current configuration
func runConfigValidation() error {
	fmt.Println("🔍 Validating s9s configuration...")

	// Load current configuration
	cfg, err := config.LoadConfig("")
	if err != nil {
		fmt.Printf("❌ Failed to load configuration: %v\n", err)
		fmt.Println("\n💡 Run 's9s setup' to create initial configuration")
		return nil
	}

	// Validate configuration
	result := config.ValidateAndFix(cfg, false)
	config.PrintValidationResult(result, true)

	if !result.Valid {
		fmt.Println("\n🔧 To fix issues automatically, run: s9s setup --validate-only --auto-fix")
		return fmt.Errorf("configuration validation failed")
	}

	return nil
}

// runAutoDiscovery attempts to auto-discover SLURM clusters
func runAutoDiscovery() error {
	fmt.Println("🔍 Auto-discovering SLURM clusters...")

	discoveryService := discovery.NewClusterDiscovery()
	clusters, err := discoveryService.DiscoverClusters(context.Background())
	if err != nil {
		return fmt.Errorf("cluster discovery failed: %w", err)
	}

	if len(clusters) == 0 {
		fmt.Println("   📭 No clusters found")
		return nil
	}

	fmt.Printf("   🎯 Found %d potential cluster(s):\n\n", len(clusters))

	for i, cluster := range clusters {
		fmt.Printf("   %d. %s\n", i+1, cluster.Name)
		fmt.Printf("      🌐 Host: %s:%d\n", cluster.Host, cluster.Port)
		if len(cluster.RestEndpoints) > 0 {
			fmt.Printf("      🔗 Endpoint: %s\n", cluster.RestEndpoints[0])
		}
		fmt.Printf("      🎯 Confidence: %.1f%%\n", cluster.Confidence*100)
		fmt.Printf("      📝 Methods: %s\n", strings.Join(cluster.DetectionMethods, ", "))
		fmt.Println()
	}

	fmt.Println("These clusters will be available during setup configuration.")
	return nil
}
