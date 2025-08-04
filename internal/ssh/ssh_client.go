package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"
)

// SSHConfig holds SSH connection configuration
type SSHConfig struct {
	Username    string
	Port        int
	KeyFile     string
	ConfigFile  string
	Options     map[string]string
	Timeout     time.Duration
}

// SSHClient handles SSH connections to cluster nodes
type SSHClient struct {
	config *SSHConfig
}

// NewSSHClient creates a new SSH client
func NewSSHClient(config *SSHConfig) *SSHClient {
	if config == nil {
		config = &SSHConfig{
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

	return &SSHClient{
		config: config,
	}
}

// ConnectToNode opens an SSH connection to a cluster node
func (c *SSHClient) ConnectToNode(ctx context.Context, hostname string) error {
	// Build SSH command
	args := c.buildSSHArgs(hostname)

	// Create SSH command
	cmd := exec.CommandContext(ctx, "ssh", args...)

	// Connect stdin/stdout/stderr to current terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run SSH command
	return cmd.Run()
}

// ConnectToNodeInTerminal opens SSH in the current terminal
func (c *SSHClient) ConnectToNodeInTerminal(hostname string) error {
	// Build SSH command
	args := c.buildSSHArgs(hostname)

	// Create SSH command
	cmd := exec.Command("ssh", args...)

	// Connect to current terminal
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Run SSH command
	return cmd.Run()
}

// ExecuteCommand executes a command on a remote node via SSH
func (c *SSHClient) ExecuteCommand(ctx context.Context, hostname, command string) (string, error) {
	// Build SSH command with remote command
	args := c.buildSSHArgs(hostname)
	args = append(args, command)

	// Create command
	cmd := exec.CommandContext(ctx, "ssh", args...)

	// Execute and capture output
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// buildSSHArgs builds SSH command arguments
func (c *SSHClient) buildSSHArgs(hostname string) []string {
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
func (c *SSHClient) TestConnection(ctx context.Context, hostname string) error {
	// Try to execute a simple command
	_, err := c.ExecuteCommand(ctx, hostname, "echo 'SSH connection test'")
	return err
}

// GetNodeInfo retrieves basic information about a node
func (c *SSHClient) GetNodeInfo(ctx context.Context, hostname string) (map[string]string, error) {
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
	return &SSHConfig{
		Port:    22,
		Timeout: 30 * time.Second,
		Options: map[string]string{
			"StrictHostKeyChecking": "no",
			"UserKnownHostsFile":   "/dev/null",
			"ConnectTimeout":       "10",
			"ServerAliveInterval":  "60",
			"ServerAliveCountMax":  "3",
		},
	}
}