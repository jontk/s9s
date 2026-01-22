package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	Version   = "0.1.0-dev"
	CommitSHA = "unknown"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "s9s",
	Short: "Terminal UI for SLURM cluster management",
	Long: `s9s provides a terminal interface for managing SLURM clusters, inspired by k9s.

It allows HPC administrators and users to monitor and manage jobs, nodes, 
and cluster resources efficiently from the terminal.

Features:
• Real-time job and node monitoring
• Interactive job management (submit, cancel, hold, release)
• SSH integration for direct node access  
• Export capabilities (CSV, JSON, Markdown)
• Plugin system for extensibility
• Vim-like navigation`,

	Example: `  s9s                         # Launch interactive TUI
  s9s --mock                   # Use mock SLURM for testing
  s9s setup                    # Run configuration wizard
  s9s setup --auto-discover   # Auto-discover clusters
  s9s config edit             # Edit configuration file
  s9s version                  # Show version information`,

	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🚀 s9s - Terminal UI for SLURM")
		fmt.Println("")
		fmt.Println("This is a demo of the new CLI system.")
		fmt.Println("")
		fmt.Println("Available commands:")
		fmt.Println("  setup    - Interactive configuration wizard")
		fmt.Println("  config   - Configuration management")
		fmt.Println("  version  - Version information")
		fmt.Println("")
		fmt.Println("Use 's9s --help' to see all options.")
		return nil
	},
}

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
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🧙‍♂️ Starting s9s setup wizard...")
		fmt.Println("")
		fmt.Println("The setup wizard would guide you through:")
		fmt.Println("  1. 🏢 Cluster connection settings")
		fmt.Println("  2. 🔐 Authentication configuration")
		fmt.Println("  3. 🔒 Secure credential storage")
		fmt.Println("  4. ⚡ Performance optimization")
		fmt.Println("")
		fmt.Println("(This is a demo - full wizard is implemented in the main app)")
		return nil
	},
}

// configCmd represents the config command group
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
	Long: `Manage s9s configuration files and settings.

Configuration files are searched in the following order:
1. ~/.s9s/config.yaml
2. ~/.config/s9s/config.yaml
3. Environment variables
4. Command-line flags`,
}

// configEditCmd represents the config edit command
var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit the configuration file",
	Long: `Open the s9s configuration file in your default editor.

If no configuration file exists, a new one will be created with default settings.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("📝 Opening configuration file for editing...")
		fmt.Println("   Path: ~/.s9s/config.yaml")
		fmt.Println("   (This is a demo - full implementation available in main app)")
		return nil
	},
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version information for s9s including build details.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("s9s version %s\n", Version)
		fmt.Printf("Git commit: %s\n", CommitSHA)
		fmt.Printf("Built: %s\n", BuildTime)
		fmt.Printf("Go version: %s\n", GoVersion)
		fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	// Setup flags
	setupCmd.Flags().Bool("auto-discover", false, "Auto-discover SLURM clusters before setup")
	setupCmd.Flags().Bool("validate-only", false, "Only validate existing configuration")
	setupCmd.Flags().Bool("force", false, "Force setup even if configuration exists")

	// Add subcommands
	configCmd.AddCommand(configEditCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)

	// Global flags
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	rootCmd.Flags().Bool("mock", false, "use mock SLURM client for testing")
	rootCmd.Flags().BoolP("version", "v", false, "show version information")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
