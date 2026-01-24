package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/jontk/s9s/internal/logging"
	"github.com/jontk/s9s/internal/security"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// getHostKeyCallback returns an appropriate host key callback based on configuration.
//
// SECURITY WARNING: By default, this disables host key verification for cluster
// environments where compute nodes are frequently rebuilt. This is INSECURE and
// makes SSH connections vulnerable to man-in-the-middle attacks.
//
// For production use, set StrictHostKeyChecking=yes in SSHConfig.Options and provide
// a valid UserKnownHostsFile path to enable proper host key verification.
func getHostKeyCallback(config *SSHConfig) ssh.HostKeyCallback {
	// Check if strict host key checking is enabled
	if config != nil && config.Options != nil {
		if strictCheck, ok := config.Options["StrictHostKeyChecking"]; ok && strictCheck == "yes" {
			// Use known_hosts file if specified
			if knownHostsFile, ok := config.Options["UserKnownHostsFile"]; ok {
				callback, err := knownhosts.New(knownHostsFile)
				if err != nil {
					logging.Warnf("Failed to load known_hosts from %s: %v. Falling back to insecure mode.", knownHostsFile, err)
					return ssh.InsecureIgnoreHostKey() // #nosec G106 -- fallback when known_hosts cannot be loaded
				}
				return callback
			}
		}
	}

	// Default: Insecure mode for cluster environments
	//nolint:gosec // G106: Intentionally insecure for cluster environments where nodes are frequently rebuilt
	return ssh.InsecureIgnoreHostKey()
}

// Config holds SSH connection configuration
type Config struct {
	Username     string
	Port         int
	KeyFile      string
	ConfigFile   string
	Options      map[string]string
	Timeout      time.Duration
	UseAgent     bool              // Use SSH agent for authentication
	KeyManager   *KeyManager       // Key manager for advanced key handling
	ClientConfig *ssh.ClientConfig // Native SSH client config
}

// SSHConfig is an alias for backward compatibility
type SSHConfig = Config

// Client handles SSH connections to cluster nodes
type Client struct {
	config         *Config
	sshCommandPath string // Validated absolute path to ssh command
}

// SSHClient is an alias for backward compatibility
type SSHClient = Client

// NewSSHClient creates a new SSH client
func NewSSHClient(config *Config) *Client {
	if config == nil {
		config = &Config{
			Port:    22,
			Timeout: 30 * time.Second,
			Options: make(map[string]string),
		}
	}

	// Set default options for cluster environments
	if config.Options == nil {
		config.Options = make(map[string]string)
	}

	// Common SSH options for cluster environments
	if _, exists := config.Options["StrictHostKeyChecking"]; !exists {
		config.Options["StrictHostKeyChecking"] = "no"
	}
	if _, exists := config.Options["UserKnownHostsFile"]; !exists {
		config.Options["UserKnownHostsFile"] = "/dev/null"
	}
	if _, exists := config.Options["ConnectTimeout"]; !exists {
		config.Options["ConnectTimeout"] = "10"
	}

	// Validate and resolve ssh command path
	// Fall back to "ssh" if validation fails (will fail later with clear error)
	sshPath := "ssh"
	if validated, err := security.ValidateAndResolveCommand("ssh", "ssh"); err == nil {
		sshPath = validated
	}

	return &Client{
		config:         config,
		sshCommandPath: sshPath,
	}
}

// ConnectToNode opens an SSH connection to a cluster node
func (c *Client) ConnectToNode(ctx context.Context, hostname string) error {
	// Build SSH command
	args := c.buildSSHArgs(hostname)

	// Create SSH command
	//nolint:gosec // G204: Command path validated at initialization, arguments from application config
	cmd := exec.CommandContext(ctx, c.sshCommandPath, args...)

	// Connect stdin/stdout/stderr to current terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run SSH command
	return cmd.Run()
}

// ConnectToNodeInTerminal opens SSH in the current terminal
func (c *Client) ConnectToNodeInTerminal(hostname string) error {
	// Build SSH command
	args := c.buildSSHArgs(hostname)

	// Create SSH command
	//nolint:gosec // G204: Command path validated at initialization, arguments from application config
	cmd := exec.CommandContext(context.Background(), c.sshCommandPath, args...)

	// Connect to current terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run SSH command
	return cmd.Run()
}

// ExecuteCommand executes a command on a remote node via SSH
func (c *Client) ExecuteCommand(ctx context.Context, hostname, command string) (string, error) {
	// Build SSH command with remote command
	args := c.buildSSHArgs(hostname)
	args = append(args, command)

	// Create command
	//nolint:gosec // G204: Command path validated at initialization, arguments from application config
	cmd := exec.CommandContext(ctx, c.sshCommandPath, args...)

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// buildSSHArgs builds SSH command arguments
func (c *Client) buildSSHArgs(hostname string) []string {
	var args []string

	// Add port if specified
	if c.config.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", c.config.Port))
	}

	// Add key file if specified
	if c.config.KeyFile != "" {
		args = append(args, "-i", c.config.KeyFile)
	}

	// Add config file if specified
	if c.config.ConfigFile != "" {
		args = append(args, "-F", c.config.ConfigFile)
	}

	// Add SSH options
	for key, value := range c.config.Options {
		args = append(args, "-o", fmt.Sprintf("%s=%s", key, value))
	}

	// Add connection timeout
	if c.config.Timeout > 0 {
		timeoutSecs := int(c.config.Timeout.Seconds())
		args = append(args, "-o", fmt.Sprintf("ConnectTimeout=%d", timeoutSecs))
	}

	// Add username@hostname
	if c.config.Username != "" {
		args = append(args, fmt.Sprintf("%s@%s", c.config.Username, hostname))
	} else {
		args = append(args, hostname)
	}

	return args
}

// TestConnection tests SSH connectivity to a node
func (c *Client) TestConnection(ctx context.Context, hostname string) error {
	// Try to execute a simple command
	_, err := c.ExecuteCommand(ctx, hostname, "echo 'SSH connection test'")
	return err
}

// GetNodeInfo retrieves basic information about a node
func (c *Client) GetNodeInfo(ctx context.Context, hostname string) (map[string]string, error) {
	info := make(map[string]string)

	// Get hostname
	if output, err := c.ExecuteCommand(ctx, hostname, "hostname"); err == nil {
		info["hostname"] = output
	}

	// Get uptime
	if output, err := c.ExecuteCommand(ctx, hostname, "uptime"); err == nil {
		info["uptime"] = output
	}

	// Get memory info
	if output, err := c.ExecuteCommand(ctx, hostname, "free -h"); err == nil {
		info["memory"] = output
	}

	// Get CPU info
	if output, err := c.ExecuteCommand(ctx, hostname, "nproc"); err == nil {
		info["cpu_count"] = output
	}

	// Get disk usage
	if output, err := c.ExecuteCommand(ctx, hostname, "df -h /"); err == nil {
		info["disk_usage"] = output
	}

	return info, nil
}

// IsSSHAvailable checks if SSH command is available
func IsSSHAvailable() bool {
	_, err := exec.LookPath("ssh")
	return err == nil
}

// DefaultSSHConfig returns a default SSH configuration for cluster environments
func DefaultSSHConfig() *SSHConfig {
	config := &SSHConfig{
		Port:     22,
		Timeout:  30 * time.Second,
		UseAgent: false, // Don't check agent availability during startup to avoid hanging
		Options: map[string]string{
			"StrictHostKeyChecking": "no",
			"UserKnownHostsFile":    "/dev/null",
			"ConnectTimeout":        "10",
			"ServerAliveInterval":   "60",
			"ServerAliveCountMax":   "3",
		},
	}

	// Skip key manager initialization during startup to avoid potential hanging
	// Key manager will be initialized later when actually needed for SSH connections
	// if km, err := NewKeyManager(); err == nil {
	//	config.KeyManager = km
	//
	//	// Use SSH config from key manager if available
	//	if sshConfig, err := km.GetSSHConfig(); err == nil {
	//		config.KeyFile = sshConfig.KeyFile
	//		config.UseAgent = km.IsAgentConnected()
	//	}
	// }

	return config
}

// WithKeyManager sets a key manager for the SSH config
func (c *SSHConfig) WithKeyManager(km *KeyManager) *SSHConfig {
	c.KeyManager = km
	c.UseAgent = km.IsAgentConnected()

	// Update key file from key manager
	if sshConfig, err := km.GetSSHConfig(); err == nil {
		c.KeyFile = sshConfig.KeyFile
	}

	return c
}

// WithAgent enables or disables SSH agent usage
func (c *SSHConfig) WithAgent(useAgent bool) *SSHConfig {
	c.UseAgent = useAgent
	if useAgent {
		c.Options["UseAgent"] = "yes"
		c.Options["IdentitiesOnly"] = "no"
	} else {
		c.Options["UseAgent"] = "no"
		c.Options["IdentitiesOnly"] = "yes"
	}
	return c
}

// GetNativeClientConfig returns a native SSH client config
func (c *SSHConfig) GetNativeClientConfig(_ string) (*ssh.ClientConfig, error) {
	if c.ClientConfig != nil {
		return c.ClientConfig, nil
	}

	var authMethods []ssh.AuthMethod

	// Try SSH agent first if enabled
	if c.UseAgent && IsAgentAvailable() {
		if agentAuth, err := NewAgentAuth(); err == nil {
			authMethods = append(authMethods, agentAuth.GetAuthMethod())
		}
	}

	// Add key file authentication if available
	if c.KeyFile != "" {
		if keyAuth, err := c.getKeyFileAuth(); err == nil {
			authMethods = append(authMethods, keyAuth)
		}
	}

	// If no auth methods, try to use key manager
	if len(authMethods) == 0 && c.KeyManager != nil {
		if kmAuth, err := c.getKeyManagerAuth(); err == nil {
			authMethods = append(authMethods, kmAuth...)
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication methods available")
	}

	clientConfig := &ssh.ClientConfig{
		User:            c.Username,
		Auth:            authMethods,
		HostKeyCallback: getHostKeyCallback(c),
		Timeout:         c.Timeout,
	}

	return clientConfig, nil
}

// getKeyFileAuth creates authentication from key file
func (c *SSHConfig) getKeyFileAuth() (ssh.AuthMethod, error) {
	keyData, err := os.ReadFile(c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}

// getKeyManagerAuth creates authentication methods from key manager
func (c *SSHConfig) getKeyManagerAuth() ([]ssh.AuthMethod, error) {
	if c.KeyManager == nil {
		return nil, fmt.Errorf("no key manager available")
	}

	var authMethods []ssh.AuthMethod

	// If agent is connected, use agent auth
	if c.KeyManager.IsAgentConnected() {
		if agentAuth, err := NewAgentAuth(); err == nil {
			authMethods = append(authMethods, agentAuth.GetAuthMethod())
		}
	}

	// Add individual key file auth methods
	keys := c.KeyManager.GetKeys()
	for _, key := range keys {
		if keyAuth, err := c.getKeyFileAuthFromPath(key.Path); err == nil {
			authMethods = append(authMethods, keyAuth)
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no usable keys found in key manager")
	}

	return authMethods, nil
}

// getKeyFileAuthFromPath creates authentication from a specific key path
func (c *SSHConfig) getKeyFileAuthFromPath(keyPath string) (ssh.AuthMethod, error) {
	//nolint:gosec // G304: keyPath from KeyManager.GetKeys(), paths from user's ~/.ssh directory
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	return ssh.PublicKeys(signer), nil
}
