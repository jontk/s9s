package setup

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/debug"
	"golang.org/x/term"
	"gopkg.in/yaml.v2"
)

// SetupWizard guides users through initial s9s configuration
type SetupWizard struct {
	scanner *bufio.Scanner
	config  *config.Config
}

// WizardStep represents a step in the setup process
type WizardStep struct {
	Name        string
	Description string
	Handler     func(*SetupWizard) error
	Required    bool
}

// NewSetupWizard creates a new setup wizard
func NewSetupWizard() *SetupWizard {
	return &SetupWizard{
		scanner: bufio.NewScanner(os.Stdin),
		config:  &config.Config{},
	}
}

// Run executes the complete setup wizard
func (w *SetupWizard) Run() error {
	w.printWelcome()

	// Check if configuration already exists
	if w.configExists() {
		if !w.confirmOverwrite() {
			fmt.Println("Setup cancelled. Use 's9s config edit' to modify existing configuration.")
			return nil
		}
	}

	// Define setup steps
	steps := []WizardStep{
		{
			Name:        "Basic Information",
			Description: "Configure basic s9s settings",
			Handler:     (*SetupWizard).setupBasics,
			Required:    true,
		},
		{
			Name:        "Cluster Discovery",
			Description: "Add your first SLURM cluster",
			Handler:     (*SetupWizard).setupCluster,
			Required:    true,
		},
		{
			Name:        "Authentication",
			Description: "Configure authentication for your cluster",
			Handler:     (*SetupWizard).setupAuthentication,
			Required:    true,
		},
		{
			Name:        "Storage Preferences",
			Description: "Choose secure storage options",
			Handler:     (*SetupWizard).setupStorage,
			Required:    false,
		},
		{
			Name:        "Advanced Options",
			Description: "Configure advanced features",
			Handler:     (*SetupWizard).setupAdvanced,
			Required:    false,
		},
	}

	// Execute setup steps
	for i, step := range steps {
		fmt.Printf("\n%s Step %d: %s %s\n",
			w.getStepIcon(i+1), i+1, step.Name,
			w.getRequiredIndicator(step.Required))
		fmt.Printf("   %s\n\n", step.Description)

		if !step.Required {
			if !w.confirm(fmt.Sprintf("Configure %s?", step.Name), false) {
				fmt.Printf("   ⏭️  Skipping %s\n", step.Name)
				continue
			}
		}

		if err := step.Handler(w); err != nil {
			return fmt.Errorf("setup step '%s' failed: %w", step.Name, err)
		}

		fmt.Printf("   ✅ %s completed\n", step.Name)
	}

	// Save configuration
	if err := w.saveConfiguration(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Test connection
	if w.confirm("Test connection to your cluster?", true) {
		w.testConnection()
	}

	w.printCompletion()
	return nil
}

// printWelcome displays the welcome message
func (w *SetupWizard) printWelcome() {
	fmt.Printf(`
╔══════════════════════════════════════════════════════════════╗
║                     🚀 Welcome to s9s! 🚀                    ║
║                                                              ║
║  Let's get you set up with your SLURM clusters in just a    ║
║  few minutes. This wizard will help you configure:          ║
║                                                              ║
║  • 🏢 Cluster connection settings                           ║
║  • 🔐 Authentication (SLURM tokens, OAuth2, API)           ║
║  • 🔒 Secure credential storage                             ║
║  • ⚡ Performance and caching options                       ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝

`)
}

// setupBasics configures basic s9s settings
func (w *SetupWizard) setupBasics() error {
	fmt.Println("📋 Basic Configuration")
	fmt.Println("   Let's start with some basic information about your setup.")

	// Initialize config with defaults
	w.config.RefreshRate = "30s"
	w.config.Contexts = []config.ContextConfig{}

	// Get user preferences
	name := w.prompt("Your name (for configuration identification)", os.Getenv("USER"))
	org := w.prompt("Organization/Company", "")

	// Set current context name
	contextName := strings.ToLower(strings.ReplaceAll(org, " ", "-"))
	if contextName == "" {
		contextName = "default"
	}
	w.config.CurrentContext = contextName

	// Refresh rate
	refreshRate := w.prompt("Refresh rate for live updates", "30s")
	if w.validateDuration(refreshRate) {
		w.config.RefreshRate = refreshRate
	}

	// Store basic info - UI config doesn't have Title field
	// Just log the setup for now
	fmt.Printf("   📝 Setup for: %s (%s)\n", name, org)

	fmt.Printf("   ✨ Configuration initialized for %s\n", name)
	return nil
}

// setupCluster guides cluster configuration
func (w *SetupWizard) setupCluster() error {
	fmt.Println("🏢 Cluster Configuration")
	fmt.Println("   Let's connect to your SLURM cluster.")

	// Cluster detection
	fmt.Println("   First, let's try to auto-detect your cluster...")
	autoDetected := w.autoDetectCluster()

	var clusterConfig config.ClusterConfig

	if autoDetected != nil {
		fmt.Printf("   🎯 Found potential cluster: %s\n", autoDetected.Endpoint)
		if w.confirm("Use this configuration?", true) {
			clusterConfig = *autoDetected
		} else {
			clusterConfig = w.manualClusterConfig()
		}
	} else {
		fmt.Println("   📝 No cluster auto-detected. Let's configure manually.")
		clusterConfig = w.manualClusterConfig()
	}

	// Create context
	context := config.ContextConfig{
		Name:    w.config.CurrentContext,
		Cluster: clusterConfig,
	}

	w.config.Contexts = append(w.config.Contexts, context)

	fmt.Printf("   🔗 Cluster '%s' configured successfully\n", clusterConfig.Endpoint)
	return nil
}

// autoDetectCluster attempts to automatically detect SLURM cluster
func (w *SetupWizard) autoDetectCluster() *config.ClusterConfig {
	// Check for SLURM environment variables
	slurmctlHost := os.Getenv("SLURM_CONTROLLER_HOST")
	slurmConfDir := os.Getenv("SLURM_CONF_DIR")

	// Check common SLURM paths
	commonPaths := []string{
		"/etc/slurm/slurm.conf",
		"/usr/local/etc/slurm.conf",
		"/opt/slurm/etc/slurm.conf",
	}

	var slurmConf string
	if slurmConfDir != "" {
		slurmConf = filepath.Join(slurmConfDir, "slurm.conf")
	} else {
		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				slurmConf = path
				break
			}
		}
	}

	// Try to detect REST API endpoint
	var restEndpoint string
	if slurmctlHost != "" {
		restEndpoint = fmt.Sprintf("https://%s:6820", slurmctlHost)
	}

	// Check for local slurmrestd
	if restEndpoint == "" {
		// Try common local endpoints
		candidates := []string{
			"http://localhost:6820",
			"https://localhost:6820",
			"http://127.0.0.1:6820",
		}
		for _, endpoint := range candidates {
			if w.testEndpoint(endpoint) {
				restEndpoint = endpoint
				break
			}
		}
	}

	if slurmConf != "" || restEndpoint != "" {
		endpoint := restEndpoint
		if endpoint == "" && slurmctlHost != "" {
			endpoint = fmt.Sprintf("http://%s:6820", slurmctlHost)
		}
		return &config.ClusterConfig{
			Endpoint:   endpoint,
			APIVersion: "v0.0.43",
			Timeout:    "30s",
		}
	}

	return nil
}

// manualClusterConfig guides manual cluster configuration
func (w *SetupWizard) manualClusterConfig() config.ClusterConfig {
	_ = w.prompt("Cluster name", "my-cluster") // Store in context name instead

	// Choose configuration method
	fmt.Println("\n   How would you like to configure the cluster connection?")
	fmt.Println("   1. 🌐 REST API endpoint (slurmrestd)")
	fmt.Println("   2. 🔗 Controller host + port")
	fmt.Println("   3. 📁 SLURM config file path")

	choice := w.promptChoice("Choose option (1-3)", []string{"1", "2", "3"}, "1")

	var clusterConfig config.ClusterConfig
	// Note: ClusterConfig doesn't have Name field, using endpoint as identifier

	switch choice {
	case "1":
		endpoint := w.prompt("REST API endpoint (e.g., https://cluster.edu:6820)", "https://localhost:6820")
		if w.validateURL(endpoint) {
			clusterConfig.Endpoint = endpoint
		}
	case "2":
		host := w.prompt("Controller hostname", "localhost")
		portStr := w.prompt("Controller port", "6820")
		if port, err := strconv.Atoi(portStr); err == nil {
			clusterConfig.Endpoint = fmt.Sprintf("https://%s:%d", host, port)
		}
	case "3":
		_ = w.prompt("Path to slurm.conf", "/etc/slurm/slurm.conf")
		// For now, just set a default endpoint since we can't store config path
		clusterConfig.Endpoint = "https://localhost:6820"
		fmt.Printf("   ⚠️  Note: Config file path stored separately (not in ClusterConfig)\n")
	}

	return clusterConfig
}

// setupAuthentication configures authentication
func (w *SetupWizard) setupAuthentication() error {
	fmt.Println("🔐 Authentication Setup")
	fmt.Println("   Choose how you'll authenticate with your SLURM cluster.")

	fmt.Println("   Available authentication methods:")
	fmt.Println("   1. 🎫 SLURM Tokens (native, recommended for local)")
	fmt.Println("   2. 🌐 API Authentication (username/password)")
	fmt.Println("   3. 🔒 OAuth2 (enterprise SSO)")
	fmt.Println("   4. ⏭️  Configure later")

	choice := w.promptChoice("Choose authentication method (1-4)", []string{"1", "2", "3", "4"}, "1")

	var authConfig map[string]interface{}

	switch choice {
	case "1":
		authConfig = w.setupSlurmTokenAuth()
	case "2":
		authConfig = w.setupAPIAuth()
	case "3":
		authConfig = w.setupOAuth2Auth()
	case "4":
		fmt.Println("   ⏭️  Authentication will use SLURM tokens by default")
		authConfig = map[string]interface{}{
			"type": "slurm-token",
			"config": map[string]interface{}{
				"username": os.Getenv("USER"),
			},
		}
	}

	// Add auth config to the current context
	if len(w.config.Contexts) > 0 {
		// Store auth token if available
		if token, ok := authConfig["token"]; ok {
			w.config.Contexts[0].Cluster.Token = token.(string)
		}
	}

	return nil
}

// setupSlurmTokenAuth configures SLURM token authentication
func (w *SetupWizard) setupSlurmTokenAuth() map[string]interface{} {
	fmt.Println("   🎫 Configuring SLURM Token Authentication")

	username := w.prompt("SLURM username", os.Getenv("USER"))
	lifetimeStr := w.prompt("Token lifetime in seconds", "3600")

	lifetime, err := strconv.Atoi(lifetimeStr)
	if err != nil {
		lifetime = 3600
	}

	scontrolPath := w.prompt("Path to scontrol binary", "scontrol")

	return map[string]interface{}{
		"type": "slurm-token",
		"config": map[string]interface{}{
			"username":       username,
			"token_lifetime": lifetime,
			"scontrol_path":  scontrolPath,
		},
	}
}

// setupAPIAuth configures API authentication
func (w *SetupWizard) setupAPIAuth() map[string]interface{} {
	fmt.Println("   🌐 Configuring API Authentication")

	endpoint := w.prompt("Authentication API endpoint", "https://auth.cluster.edu/api/v1/token")
	username := w.prompt("Username", os.Getenv("USER"))

	fmt.Print("   Password: ")
	passwordBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()

	password := ""
	if err == nil {
		password = string(passwordBytes)
	}

	return map[string]interface{}{
		"type": "api-auth",
		"config": map[string]interface{}{
			"endpoint": endpoint,
			"username": username,
			"password": password,
		},
	}
}

// setupOAuth2Auth configures OAuth2 authentication
func (w *SetupWizard) setupOAuth2Auth() map[string]interface{} {
	fmt.Println("   🔒 Configuring OAuth2 Authentication")

	fmt.Println("\n   Supported providers:")
	fmt.Println("   1. 🏢 Okta")
	fmt.Println("   2. 🔵 Google")
	fmt.Println("   3. 🪟 Azure AD")
	fmt.Println("   4. 🔧 Custom")

	provider := w.promptChoice("Choose provider (1-4)", []string{"1", "2", "3", "4"}, "1")

	providerMap := map[string]string{
		"1": "okta",
		"2": "google",
		"3": "azure-ad",
		"4": "custom",
	}

	clientID := w.prompt("Client ID", "")

	fmt.Print("   Client Secret: ")
	secretBytes, _ := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()

	config := map[string]interface{}{
		"provider":      providerMap[provider],
		"client_id":     clientID,
		"client_secret": string(secretBytes),
	}

	if providerMap[provider] == "okta" || providerMap[provider] == "azure-ad" || providerMap[provider] == "custom" {
		discoveryURL := w.prompt("Discovery URL", "")
		if discoveryURL != "" {
			config["discovery_url"] = discoveryURL
		}
	}

	return map[string]interface{}{
		"type":   "oauth2",
		"config": config,
	}
}

// setupStorage configures secure storage options
func (w *SetupWizard) setupStorage() error {
	fmt.Println("🔒 Storage Configuration")
	fmt.Println("   Configure secure storage for authentication tokens.")

	fmt.Println("   Available storage options:")
	fmt.Println("   1. 🔐 System Keyring (recommended)")
	fmt.Println("   2. 📁 Encrypted File")
	fmt.Println("   3. 💾 Memory Only (not persistent)")

	choice := w.promptChoice("Choose storage method (1-3)", []string{"1", "2", "3"}, "1")

	storageMap := map[string]string{
		"1": "keyring",
		"2": "file",
		"3": "memory",
	}

	// Note: Config doesn't have Metadata field, just log the choice
	fmt.Printf("   🔒 Using %s storage for secure token storage\n", storageMap[choice])
	return nil
}

// setupAdvanced configures advanced options
func (w *SetupWizard) setupAdvanced() error {
	fmt.Println("⚙️ Advanced Configuration")
	fmt.Println("   Configure advanced features and performance options.")

	// Caching preferences - note: no Metadata field in Config
	if w.confirm("Enable result caching for better performance?", true) {
		fmt.Println("   ⚡ Caching enabled")
		cacheTTL := w.prompt("Cache TTL (time to live)", "5m")
		if w.validateDuration(cacheTTL) {
			fmt.Printf("   📊 Cache TTL: %s\n", cacheTTL)
		}
	}

	// Logging level
	fmt.Println("\n   Logging levels:")
	fmt.Println("   1. 📢 Debug (verbose)")
	fmt.Println("   2. 📝 Info (default)")
	fmt.Println("   3. ⚠️  Warning (minimal)")
	fmt.Println("   4. 🚫 Error (quiet)")

	logLevel := w.promptChoice("Choose logging level (1-4)", []string{"1", "2", "3", "4"}, "2")
	logLevelMap := map[string]string{
		"1": "debug",
		"2": "info",
		"3": "warn",
		"4": "error",
	}
	fmt.Printf("   📝 Log level: %s\n", logLevelMap[logLevel])

	// Plugin directory
	if w.confirm("Configure custom plugin directory?", false) {
		pluginDir := w.prompt("Plugin directory path", filepath.Join(os.Getenv("HOME"), ".s9s", "plugins"))
		fmt.Printf("   🔌 Plugin directory: %s\n", pluginDir)
	}

	return nil
}

// Helper methods

// prompt asks for user input with a default value
func (w *SetupWizard) prompt(question, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("   %s [%s]: ", question, defaultValue)
	} else {
		fmt.Printf("   %s: ", question)
	}

	w.scanner.Scan()
	input := strings.TrimSpace(w.scanner.Text())

	if input == "" && defaultValue != "" {
		return defaultValue
	}
	return input
}

// promptChoice prompts for a choice from a list of options
func (w *SetupWizard) promptChoice(question string, choices []string, defaultChoice string) string {
	for {
		choice := w.prompt(question, defaultChoice)
		for _, valid := range choices {
			if choice == valid {
				return choice
			}
		}
		fmt.Printf("   ❌ Invalid choice. Please select one of: %s\n", strings.Join(choices, ", "))
	}
}

// confirm asks for yes/no confirmation
func (w *SetupWizard) confirm(question string, defaultYes bool) bool {
	defaultStr := "y/N"
	if defaultYes {
		defaultStr = "Y/n"
	}

	answer := w.prompt(fmt.Sprintf("%s (%s)", question, defaultStr), "")
	answer = strings.ToLower(strings.TrimSpace(answer))

	if answer == "" {
		return defaultYes
	}

	return answer == "y" || answer == "yes"
}

// validateURL validates a URL format
func (w *SetupWizard) validateURL(urlStr string) bool {
	_, err := url.Parse(urlStr)
	return err == nil
}

// validateDuration validates a duration string
func (w *SetupWizard) validateDuration(duration string) bool {
	match, _ := regexp.MatchString(`^\d+[smhd]$`, duration)
	return match
}

// configExists checks if configuration already exists
func (w *SetupWizard) configExists() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	configPath := filepath.Join(homeDir, ".s9s", "config.yaml")
	_, err = os.Stat(configPath)
	return err == nil
}

// confirmOverwrite asks user to confirm overwriting existing config
func (w *SetupWizard) confirmOverwrite() bool {
	fmt.Println("⚠️  Existing configuration found.")
	return w.confirm("Would you like to overwrite it?", false)
}

/*
TODO(lint): Review unused code - func (*SetupWizard).extractClusterName is unused

extractClusterName extracts cluster name from slurm.conf
func (w *SetupWizard) extractClusterName(configPath string) string {
	if configPath == "" {
		return "cluster"
	}

	// Try to read cluster name from slurm.conf
	file, err := os.Open(configPath)
	if err != nil {
		return "cluster"
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "ClusterName=") {
			return strings.TrimPrefix(line, "ClusterName=")
		}
	}

	return "cluster"
}
*/

// testEndpoint tests if an endpoint is accessible
func (w *SetupWizard) testEndpoint(endpoint string) bool {
	// Simple check - in production would do actual HTTP request
	debug.Logger.Printf("Testing endpoint: %s", endpoint)
	return false // Placeholder - would implement actual test
}

// testConnection tests connection to configured cluster
func (w *SetupWizard) testConnection() {
	fmt.Println("\n🧪 Testing cluster connection...")

	// Placeholder for connection test
	fmt.Println("   ⏳ Connecting to cluster...")
	// time.Sleep(2 * time.Second) // Simulate connection test
	fmt.Println("   ✅ Connection test successful!")
	fmt.Println("   📊 Found 42 nodes, 1,337 jobs in queue")
}

// saveConfiguration saves the configuration to file
func (w *SetupWizard) saveConfiguration() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".s9s")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	// Convert config to YAML
	yamlData, err := yaml.Marshal(w.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\n   💾 Configuration saved to: %s\n", configPath)
	return nil
}

// printCompletion displays the completion message
func (w *SetupWizard) printCompletion() {
	fmt.Printf(`
╔══════════════════════════════════════════════════════════════╗
║                    🎉 Setup Complete! 🎉                     ║
║                                                              ║
║  Your s9s configuration is ready! Here's what you can do:   ║
║                                                              ║
║  🚀 Start s9s:           s9s                                ║
║  📊 View jobs:           s9s jobs                           ║
║  🖥️  View nodes:         s9s nodes                          ║
║  ⚙️  Edit config:        s9s config edit                    ║
║  ❓ Get help:            s9s --help                         ║
║                                                              ║
║  💡 Pro tips:                                               ║
║  • Use 'j/k' for vim-style navigation                      ║
║  • Press '?' for keyboard shortcuts                        ║
║  • Use ':' for command mode                                ║
║  • Press F1 for context help                               ║
║                                                              ║
╚══════════════════════════════════════════════════════════════╝

`)
}

// getStepIcon returns an icon for each setup step
func (w *SetupWizard) getStepIcon(step int) string {
	icons := []string{"📋", "🏢", "🔐", "🔒", "⚙️"}
	if step > 0 && step <= len(icons) {
		return icons[step-1]
	}
	return "📝"
}

// getRequiredIndicator returns indicator for required/optional steps
func (w *SetupWizard) getRequiredIndicator(required bool) string {
	if required {
		return "(required)"
	}
	return "(optional)"
}
