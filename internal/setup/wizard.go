// Package setup provides interactive setup wizard for s9s configuration.
package setup

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jontk/s9s/internal/config"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/debug"
	"github.com/jontk/s9s/internal/fileperms"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

// Wizard guides users through initial s9s configuration
type Wizard struct {
	scanner *bufio.Scanner
	config  *config.Config
}

type SetupWizard = Wizard

// NewSetupWizard creates a new setup wizard
func NewSetupWizard() *Wizard {
	return &Wizard{
		scanner: bufio.NewScanner(os.Stdin),
		config:  &config.Config{},
	}
}

// Run executes the complete setup wizard
func (w *Wizard) Run() error {
	w.printWelcome()

	// Check if configuration already exists
	if w.configExists() {
		if !w.confirmOverwrite() {
			fmt.Println("Setup canceled.")
			return nil
		}
	}

	// Initialize config defaults
	w.config.RefreshRate = "30s"
	w.config.Clusters = []config.ClusterContext{}

	// Step 1: Cluster endpoint
	fmt.Println("🏢 Cluster Connection")
	fmt.Println("   Checking for slurmrestd...")
	autoDetected := w.autoDetectCluster()

	var clusterConfig config.ClusterConfig

	if autoDetected != nil {
		fmt.Printf("   🎯 Found slurmrestd at %s\n", autoDetected.Endpoint)
		if w.confirm("Use this endpoint?", true) {
			clusterConfig = *autoDetected
		} else {
			clusterConfig = w.manualClusterConfig()
		}
	} else {
		fmt.Println("   No slurmrestd detected. Let's configure manually.")
		clusterConfig = w.manualClusterConfig()
	}

	// Step 2: Cluster name
	clusterName := w.prompt("\n   Cluster name", "default")
	clusterName = strings.ToLower(strings.ReplaceAll(clusterName, " ", "-"))

	// Step 3: Authentication token (optional)
	fmt.Println("\n🔐 Authentication")
	fmt.Println("   s9s authenticates via JWT token (from `scontrol token`).")
	fmt.Println("   If no token is configured, s9s will try SLURM_JWT env var")
	fmt.Println("   and then `scontrol token` automatically.")
	if w.confirm("Enter a token now?", false) {
		fmt.Print("   Token: ")
		tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err == nil && len(tokenBytes) > 0 {
			clusterConfig.Token = string(tokenBytes)
		}
	}

	// Build cluster entry
	w.config.DefaultCluster = clusterName
	w.config.Clusters = append(w.config.Clusters, config.ClusterContext{
		Name:    clusterName,
		Cluster: clusterConfig,
	})

	// Save configuration
	if err := w.saveConfiguration(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Test connection
	if w.confirm("\nTest connection to your cluster?", true) {
		w.testConnection()
	}

	w.printCompletion()
	return nil
}

// printWelcome displays the welcome message
func (w *Wizard) printWelcome() {
	const boxWidth = 62
	pad := func(s string) string {
		w := runewidth.StringWidth(s)
		if w >= boxWidth {
			return s
		}
		return s + strings.Repeat(" ", boxWidth-w)
	}

	fmt.Println()
	fmt.Println("╔" + strings.Repeat("═", boxWidth) + "╗")
	fmt.Println("║" + pad("                     🚀 Welcome to s9s! 🚀") + "║")
	fmt.Println("║" + pad("") + "║")
	fmt.Println("║" + pad("  Let's get you set up with your SLURM clusters in just a") + "║")
	fmt.Println("║" + pad("  few minutes. This wizard will help you configure:") + "║")
	fmt.Println("║" + pad("") + "║")
	fmt.Println("║" + pad("  • 🏢 Cluster connection settings") + "║")
	fmt.Println("║" + pad("  • 🔐 Authentication (SLURM JWT tokens)") + "║")
	fmt.Println("║" + pad("") + "║")
	fmt.Println("╚" + strings.Repeat("═", boxWidth) + "╝")
	fmt.Println()
}

// autoDetectCluster attempts to automatically detect SLURM cluster
func (w *Wizard) autoDetectCluster() *config.ClusterConfig {
	slurmctlHost := os.Getenv("SLURM_CONTROLLER_HOST")
	slurmConfDir := os.Getenv("SLURM_CONF_DIR")

	slurmConf := w.findSlurmConfig(slurmConfDir)
	restEndpoint := w.detectRESTEndpoint(slurmctlHost)

	if slurmConf == "" && restEndpoint == "" {
		return nil
	}

	// We found a SLURM installation (config file or env vars),
	// but slurmrestd may not be running
	if restEndpoint == "" {
		fmt.Println()
		fmt.Println("   ⚠️  Found SLURM installation but slurmrestd is not running.")
		if slurmConf != "" {
			fmt.Printf("   📁 SLURM config:  %s\n", slurmConf)
		}
		fmt.Println()
		fmt.Println("   s9s requires the SLURM REST API daemon (slurmrestd),")
		fmt.Println("   typically on port 6820. Having slurmctld and slurmdbd")
		fmt.Println("   running is not sufficient.")
		fmt.Println()
		fmt.Println("   Start slurmrestd with:")
		fmt.Println("     slurmrestd 0.0.0.0:6820")
		fmt.Println()
		fmt.Println("   You can finish setup now and start slurmrestd later.")
	}

	return w.buildClusterConfig(restEndpoint, slurmctlHost)
}

// findSlurmConfig locates SLURM configuration file
func (w *Wizard) findSlurmConfig(slurmConfDir string) string {
	if slurmConfDir != "" {
		return filepath.Join(slurmConfDir, "slurm.conf")
	}

	commonPaths := []string{
		"/etc/slurm/slurm.conf",
		"/usr/local/etc/slurm.conf",
		"/opt/slurm/etc/slurm.conf",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// detectRESTEndpoint attempts to find a working REST API endpoint
func (w *Wizard) detectRESTEndpoint(slurmctlHost string) string {
	var candidates []string

	if slurmctlHost != "" {
		candidates = append(candidates,
			fmt.Sprintf("http://%s:6820", slurmctlHost),
			fmt.Sprintf("https://%s:6820", slurmctlHost),
		)
	}

	candidates = append(candidates,
		"http://localhost:6820",
		"https://localhost:6820",
		"http://127.0.0.1:6820",
	)

	for _, endpoint := range candidates {
		if w.testEndpoint(endpoint) {
			return endpoint
		}
	}

	return ""
}

// buildClusterConfig creates cluster configuration from detected values
func (w *Wizard) buildClusterConfig(restEndpoint, slurmctlHost string) *config.ClusterConfig {
	endpoint := restEndpoint
	if endpoint == "" && slurmctlHost != "" {
		endpoint = fmt.Sprintf("http://%s:6820", slurmctlHost)
	}

	return &config.ClusterConfig{
		Endpoint: endpoint,
		Timeout:  "30s",
	}
}

// manualClusterConfig guides manual cluster configuration
func (w *Wizard) manualClusterConfig() config.ClusterConfig {
	fmt.Println("\n   How would you like to configure the connection?")
	fmt.Println("   1. Enter REST API endpoint URL")
	fmt.Println("   2. Enter hostname + port")

	choice := w.promptChoice("   Choose (1-2)", []string{"1", "2"}, "1")

	var clusterConfig config.ClusterConfig

	switch choice {
	case "1":
		endpoint := w.prompt("   REST API endpoint", "http://localhost:6820")
		if w.validateURL(endpoint) {
			clusterConfig.Endpoint = endpoint
		}
	case "2":
		host := w.prompt("   Hostname", "localhost")
		portStr := w.prompt("   Port", "6820")
		scheme := "http"
		if w.confirm("   Use HTTPS?", false) {
			scheme = "https"
		}
		if port, err := strconv.Atoi(portStr); err == nil {
			clusterConfig.Endpoint = fmt.Sprintf("%s://%s:%d", scheme, host, port)
		}
	}

	clusterConfig.Timeout = "30s"
	return clusterConfig
}

// Helper methods

// prompt asks for user input with a default value
func (w *Wizard) prompt(question, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", question, defaultValue)
	} else {
		fmt.Printf("%s: ", question)
	}

	w.scanner.Scan()
	input := strings.TrimSpace(w.scanner.Text())

	if input == "" && defaultValue != "" {
		return defaultValue
	}
	return input
}

// promptChoice prompts for a choice from a list of options
func (w *Wizard) promptChoice(question string, choices []string, defaultChoice string) string {
	for {
		choice := w.prompt(question, defaultChoice)
		for _, valid := range choices {
			if choice == valid {
				return choice
			}
		}
		fmt.Printf("   Invalid choice. Please select one of: %s\n", strings.Join(choices, ", "))
	}
}

// confirm asks for yes/no confirmation
func (w *Wizard) confirm(question string, defaultYes bool) bool {
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
func (w *Wizard) validateURL(urlStr string) bool {
	_, err := url.Parse(urlStr)
	return err == nil
}

// configExists checks if configuration already exists
func (w *Wizard) configExists() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	configPath := filepath.Join(homeDir, ".s9s", "config.yaml")
	_, err = os.Stat(configPath)
	return err == nil
}

// confirmOverwrite asks user to confirm overwriting existing config
func (w *Wizard) confirmOverwrite() bool {
	fmt.Println("Existing configuration found at ~/.s9s/config.yaml")
	return w.confirm("Overwrite?", false)
}

// testEndpoint tests if an endpoint is accessible
func (w *Wizard) testEndpoint(endpoint string) bool {
	debug.Logger.Printf("Testing endpoint: %s", endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		debug.Logger.Printf("Endpoint test failed for %s: %v", endpoint, err)
		return false
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		debug.Logger.Printf("Endpoint test failed for %s: %v", endpoint, err)
		return false
	}
	_ = resp.Body.Close()
	return true
}

// testConnection tests connection to configured cluster using the SLURM REST API
func (w *Wizard) testConnection() {
	fmt.Println("\n🧪 Testing cluster connection...")

	if len(w.config.Clusters) == 0 {
		fmt.Println("   ❌ No cluster configured")
		return
	}

	clusterCfg := w.config.Clusters[0].Cluster

	if clusterCfg.Endpoint == "" {
		fmt.Println("   ❌ No endpoint configured")
		return
	}

	fmt.Printf("   ⏳ Connecting to %s...\n", clusterCfg.Endpoint)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	adapter, err := dao.NewSlurmAdapter(ctx, &clusterCfg)
	if err != nil {
		fmt.Printf("   ❌ Connection failed: %v\n", err)
		fmt.Println()
		fmt.Println("   💡 Make sure slurmrestd is running on your cluster.")
		fmt.Println("      s9s requires the SLURM REST API (slurmrestd), typically on port 6820.")
		fmt.Println("      Start it with: slurmrestd 0.0.0.0:6820")
		return
	}
	defer func() { _ = adapter.Close() }()

	// Try to get cluster info as a connectivity check
	info, err := adapter.ClusterInfo()
	if err != nil {
		fmt.Printf("   ❌ Connected but failed to query cluster: %v\n", err)
		fmt.Println()
		fmt.Println("   💡 The endpoint is reachable but the API request failed.")
		fmt.Println("      Check that slurmrestd is running and authentication is configured.")
		return
	}

	// Also try to list jobs to verify full API access
	jobs, err := adapter.Jobs().List(&dao.ListJobsOptions{Limit: 1})
	if err != nil {
		fmt.Printf("   ⚠️  Cluster info OK but failed to list jobs: %v\n", err)
		return
	}

	// Try to get node count
	nodes, err := adapter.Nodes().List(nil)
	nodeCount := 0
	if err == nil && nodes != nil {
		nodeCount = len(nodes.Nodes)
	}

	fmt.Println("   ✅ Connection test successful!")
	if info.Name != "" {
		fmt.Printf("   🏢 Cluster: %s (SLURM %s)\n", info.Name, info.Version)
	}
	fmt.Printf("   📊 Found %d nodes, %d jobs in queue\n", nodeCount, jobs.Total)
}

// saveConfiguration saves the configuration to file
func (w *Wizard) saveConfiguration() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".s9s")
	if err := os.MkdirAll(configDir, fileperms.ConfigDir); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	yamlData := w.renderConfig()

	if err := os.WriteFile(configPath, []byte(yamlData), fileperms.ConfigFile); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\n   💾 Configuration saved to: %s\n", configPath)
	return nil
}

// renderConfig generates a clean YAML config string with proper camelCase keys
func (w *Wizard) renderConfig() string {
	var b strings.Builder

	b.WriteString("# s9s configuration — generated by s9s setup\n\n")

	if w.config.DefaultCluster != "" {
		fmt.Fprintf(&b, "defaultCluster: %s\n\n", w.config.DefaultCluster)
	}

	if len(w.config.Clusters) > 0 {
		b.WriteString("clusters:\n")
		for _, ctx := range w.config.Clusters {
			fmt.Fprintf(&b, "  - name: %s\n", ctx.Name)
			b.WriteString("    cluster:\n")
			fmt.Fprintf(&b, "      endpoint: %s\n", ctx.Cluster.Endpoint)
			if ctx.Cluster.Token != "" {
				fmt.Fprintf(&b, "      token: \"%s\"\n", ctx.Cluster.Token)
			}
			if ctx.Cluster.APIVersion != "" {
				fmt.Fprintf(&b, "      apiVersion: %s\n", ctx.Cluster.APIVersion)
			}
			if ctx.Cluster.Insecure {
				b.WriteString("      insecure: true\n")
			}
			if ctx.Cluster.Timeout != "" {
				fmt.Fprintf(&b, "      timeout: %s\n", ctx.Cluster.Timeout)
			}
		}
		b.WriteString("\n")
	}

	if w.config.RefreshRate != "" && w.config.RefreshRate != "30s" {
		fmt.Fprintf(&b, "refreshRate: %s\n", w.config.RefreshRate)
	}

	b.WriteString("discovery:\n  enableEndpoint: false\n")

	return b.String()
}

// printCompletion displays the completion message
func (w *Wizard) printCompletion() {
	const boxWidth = 62
	pad := func(s string) string {
		w := runewidth.StringWidth(s)
		if w >= boxWidth {
			return s
		}
		return s + strings.Repeat(" ", boxWidth-w)
	}

	fmt.Println()
	fmt.Println("╔" + strings.Repeat("═", boxWidth) + "╗")
	fmt.Println("║" + pad("                    🎉 Setup Complete! 🎉") + "║")
	fmt.Println("║" + pad("") + "║")
	fmt.Println("║" + pad("  🚀 Start s9s:        s9s") + "║")
	fmt.Println("║" + pad("  📝 Edit config:      s9s config edit") + "║")
	fmt.Println("║" + pad("  ❓ Get help:          s9s --help") + "║")
	fmt.Println("║" + pad("") + "║")
	fmt.Println("║" + pad("  💡 Pro tips:") + "║")
	fmt.Println("║" + pad("  • Use Tab to switch between views") + "║")
	fmt.Println("║" + pad("  • Use 'j/k' for vim-style navigation") + "║")
	fmt.Println("║" + pad("  • Press '?' for keyboard shortcuts") + "║")
	fmt.Println("║" + pad("  • Use ':' for command mode") + "║")
	fmt.Println("║" + pad("") + "║")
	fmt.Println("╚" + strings.Repeat("═", boxWidth) + "╝")
	fmt.Println()
}
