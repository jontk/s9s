package ssh_test

import (
	"context"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/ssh"
)

// TestSSHClient tests the SSH client functionality
func TestSSHClient(t *testing.T) {
	// Test 1: Create client with default config
	t.Run("DefaultClient", func(t *testing.T) {
		client := ssh.NewSSHClient(nil)
		if client == nil {
			t.Fatal("Expected client to be created with default config")
		}
	})

	// Test 2: Create client with custom config
	t.Run("CustomClient", func(t *testing.T) {
		config := &ssh.SSHConfig{
			Username: "testuser",
			Port:     2222,
			KeyFile:  "~/.ssh/custom_key",
			Timeout:  5 * time.Second,
			Options: map[string]string{
				"PreferredAuthentications": "publickey",
				"LogLevel":                 "ERROR",
			},
		}

		client := ssh.NewSSHClient(config)
		if client == nil {
			t.Fatal("Expected client to be created with custom config")
		}
	})

	// Test 3: Test SSH availability
	t.Run("SSHAvailability", func(t *testing.T) {
		available := ssh.IsSSHAvailable()
		t.Logf("SSH available: %v", available)
		// Don't fail if SSH is not available, just log it
	})
}

// TestSSHClientArguments tests SSH argument building
func TestSSHClientArguments(t *testing.T) {
	// This would require exposing buildSSHArgs method or testing through ExecuteCommand
	// For now, we'll test through the public interface

	t.Run("BasicConnection", func(t *testing.T) {
		config := &ssh.SSHConfig{
			Username: "testuser",
			Port:     22,
		}
		client := ssh.NewSSHClient(config)

		// Test connection would fail without a real server
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := client.TestConnection(ctx, "test-host")
		// We expect this to fail since there's no real SSH server
		if err == nil {
			t.Error("Expected connection to fail for non-existent host")
		}
	})

	t.Run("CustomPort", func(t *testing.T) {
		config := &ssh.SSHConfig{
			Username: "testuser",
			Port:     2222,
		}
		client := ssh.NewSSHClient(config)

		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		err := client.TestConnection(ctx, "test-host")
		// We expect this to fail since there's no real SSH server
		if err == nil {
			t.Error("Expected connection to fail for non-existent host")
		}
	})
}

// TestDefaultSSHConfig tests the default configuration
func TestDefaultSSHConfig(t *testing.T) {
	config := ssh.DefaultSSHConfig()

	if config.Port != 22 {
		t.Errorf("Expected default port 22, got %d", config.Port)
	}

	if config.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout 30s, got %v", config.Timeout)
	}

	expectedOptions := map[string]string{
		"StrictHostKeyChecking": "no",
		"UserKnownHostsFile":    "/dev/null",
		"ConnectTimeout":        "10",
		"ServerAliveInterval":   "60",
		"ServerAliveCountMax":   "3",
	}

	for key, expectedValue := range expectedOptions {
		if value, exists := config.Options[key]; !exists {
			t.Errorf("Expected option %s to exist", key)
		} else if value != expectedValue {
			t.Errorf("Expected option %s=%s, got %s", key, expectedValue, value)
		}
	}
}

// TestSSHClientEdgeCases tests edge cases
func TestSSHClientEdgeCases(t *testing.T) {
	// Test with various invalid configurations
	t.Run("EmptyHostname", func(t *testing.T) {
		client := ssh.NewSSHClient(nil)
		ctx := context.Background()

		err := client.TestConnection(ctx, "")
		if err == nil {
			t.Error("Expected error for empty hostname")
		}
	})

	t.Run("NilContext", func(t *testing.T) {
		client := ssh.NewSSHClient(nil)

		// This should handle nil context gracefully or panic
		defer func() {
			if r := recover(); r != nil {
				t.Logf("Recovered from panic: %v", r)
			}
		}()

		// Using nil context is generally bad practice but test behavior
		err := client.TestConnection(nil, "test-host") //nolint:staticcheck
		if err == nil {
			t.Error("Expected error with nil context")
		}
	})

	t.Run("CancelledContext", func(t *testing.T) {
		client := ssh.NewSSHClient(nil)

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := client.TestConnection(ctx, "test-host")
		if err == nil {
			t.Error("Expected error with cancelled context")
		}
	})
}

// TestGetNodeInfo tests node information retrieval
func TestGetNodeInfo(t *testing.T) {
	client := ssh.NewSSHClient(nil)
	ctx := context.Background()

	// GetNodeInfo always returns nil error, but the individual commands will fail
	// So the info map will be empty or contain error messages
	info, err := client.GetNodeInfo(ctx, "test-host")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Info map should be returned
	if info == nil {
		t.Error("Expected info map to be initialized")
	}

	// The map might be empty since SSH commands will fail
	t.Logf("Info map has %d entries", len(info))
}
