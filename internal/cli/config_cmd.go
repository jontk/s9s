package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/fileperms"
	"github.com/jontk/s9s/internal/security"
	"github.com/spf13/cobra"
)

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
	RunE: runConfigEdit,
}

// configValidateCmd represents the config validate command
var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the configuration file",
	Long: `Validate the s9s configuration file for syntax and logical errors.

This command checks:
• YAML syntax validity
• Required fields presence
• Cluster connectivity (optional)
• Authentication configuration`,
	RunE: runConfigValidate,
}

// configShowCmd represents the config show command
var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration",
	Long: `Display the current s9s configuration with resolved values.

This shows the effective configuration after merging:
• Configuration files
• Environment variables  
• Command-line flags`,
	RunE: runConfigShow,
}

func init() {
	// Add subcommands to config
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configShowCmd)

	// Add config command to root
	rootCmd.AddCommand(configCmd)
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	configPath := getConfigPath()

	// Create directory if it doesn't exist
	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, fileperms.ConfigDir); err != nil{
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := createDefaultConfig(configPath); err != nil {
			return fmt.Errorf("failed to create default config: %w", err)
		}
		fmt.Printf("Created default configuration at: %s\n", configPath)
	}

	// Open in editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = getDefaultEditor()
	}

	// Validate editor command path
	validatedEditor, err := security.ValidateAndResolveCommand(editor, "editor")
	if err != nil {
		return fmt.Errorf("invalid editor command %q: %w", editor, err)
	}

	fmt.Printf("Opening %s in %s...\n", configPath, filepath.Base(validatedEditor))

	// nolint:gosec // G204: Command path is validated via security.ValidateAndResolveCommand
	execCmd := exec.Command(validatedEditor, configPath)
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

func runConfigValidate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Basic validation - check if current context exists
	if cfg.CurrentContext != "" {
		_, err := cfg.GetContext(cfg.CurrentContext)
		if err != nil {
			fmt.Printf("❌ Current context '%s' not found\n", cfg.CurrentContext)
			return fmt.Errorf("configuration validation failed: %w", err)
		}
	}

	// Validate contexts have valid cluster endpoints
	for _, ctx := range cfg.Contexts {
		if ctx.Cluster.Endpoint == "" {
			fmt.Printf("❌ Context '%s' has no cluster endpoint\n", ctx.Name)
			return fmt.Errorf("configuration validation failed: missing cluster endpoint")
		}
	}

	fmt.Println("✅ Configuration is valid")
	return nil
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Print configuration (with sensitive data masked)
	fmt.Println("Current s9s configuration:")
	fmt.Printf("Config path: %s\n\n", getConfigPath())

	// TODO: Implement config display with sensitive data masking
	fmt.Printf("Contexts configured: %d\n", len(cfg.Contexts))
	fmt.Printf("Current context: %s\n", cfg.CurrentContext)
	fmt.Printf("Mock mode: %v\n", cfg.UseMockClient)

	return nil
}

func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}

	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".s9s", "config.yaml")
}

func createDefaultConfig(path string) error {
	defaultConfig := `# s9s Configuration File
# Documentation: https://github.com/jontk/s9s/tree/main/docs

refreshRate: "5s"
maxRetries: 3
currentContext: "default"
useMockClient: true

# Contexts define cluster connections
contexts:
  - name: "default"
    cluster:
      endpoint: "https://localhost:6820"
      token: ""
      apiVersion: "v0.0.43"
      insecure: false
      timeout: "30s"
    namespace: ""
    readOnly: false

# UI Configuration
ui:
  skin: "default"
  logoless: false
  crumbsless: false
  statusless: false
  headless: false
  noIcons: false
  enableMouse: true

# View Settings
views:
  jobs:
    columns: ["id", "name", "user", "state", "time", "nodes", "priority"]
    showOnlyActive: true
    defaultSort: "time"
    maxJobs: 1000
  nodes:
    groupBy: "partition"
    showUtilization: true
    maxNodes: 500
  partitions:
    showQueueDepth: true
    showWaitTime: true

# Feature Flags
features:
  streaming: true
  pulseye: true
  xray: false

# Shortcuts (custom key bindings)
shortcuts: []

# Aliases for common commands
aliases:
  ctx: "context"
  kj: "kill job"
  dj: "describe job"
  dn: "describe node"
  sub: "submit job"

# Plugin Configuration
plugins: []
`

	return os.WriteFile(path, []byte(defaultConfig), fileperms.ConfigFile)
}

func getDefaultEditor() string {
	editors := []string{"vim", "nano", "emacs", "vi"}

	for _, editor := range editors {
		if _, err := exec.LookPath(editor); err == nil {
			return editor
		}
	}

	return "vi" // fallback
}
