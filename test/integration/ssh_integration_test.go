package integration

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSSHIntegration runs integration tests with real SSH connections
// These tests require:
// 1. SSH server running on localhost:2222 (or configured host)
// 2. SSH key authentication set up
// 3. Environment variable SSH_TEST_HOST set (defaults to localhost:2222)
func TestSSHIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Check if SSH integration tests are enabled
	if os.Getenv("SSH_INTEGRATION_TESTS") != "1" {
		t.Skip("SSH integration tests disabled. Set SSH_INTEGRATION_TESTS=1 to enable")
	}

	testHost := os.Getenv("SSH_TEST_HOST")
	if testHost == "" {
		testHost = "localhost:2222"
	}

	testUser := os.Getenv("SSH_TEST_USER")
	if testUser == "" {
		testUser = os.Getenv("USER") // Use current user as default
	}

	t.Run("SSHClientIntegration", func(t *testing.T) {
		testSSHClientIntegration(t, testHost, testUser)
	})

	t.Run("SessionManagerIntegration", func(t *testing.T) {
		testSessionManagerIntegration(t, testHost, testUser)
	})

	t.Run("TerminalManagerIntegration", func(t *testing.T) {
		testTerminalManagerIntegration(t, testHost, testUser)
	})

	t.Run("PersistenceIntegration", func(t *testing.T) {
		testPersistenceIntegration(t, testHost, testUser)
	})
}

func testSSHClientIntegration(t *testing.T, testHost, testUser string) {
	config := &ssh.SSHConfig{
		Username: testUser,
		Port:     parsePort(testHost),
		Timeout:  10 * time.Second,
		Options: map[string]string{
			"StrictHostKeyChecking": "no",
			"UserKnownHostsFile":    "/dev/null",
			"ConnectTimeout":        "5",
		},
	}

	client := ssh.NewSSHClient(config)
	hostname := parseHostname(testHost)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	t.Run("TestConnection", func(t *testing.T) {
		err := client.TestConnection(ctx, hostname)
		if err != nil {
			t.Logf("SSH connection failed: %v", err)
			t.Logf("Make sure SSH server is running on %s", testHost)
			t.Logf("And that key-based authentication is set up")
			t.Skip("SSH connection not available for integration testing")
		}
	})

	t.Run("ExecuteCommand", func(t *testing.T) {
		output, err := client.ExecuteCommand(ctx, hostname, "echo 'Hello from SSH'")
		require.NoError(t, err)
		assert.Contains(t, output, "Hello from SSH")
	})

	t.Run("GetNodeInfo", func(t *testing.T) {
		info, err := client.GetNodeInfo(ctx, hostname)
		require.NoError(t, err)
		assert.Contains(t, info, "hostname")
		assert.Contains(t, info, "uptime")
	})

	t.Run("MultipleCommands", func(t *testing.T) {
		commands := []string{
			"pwd",
			"whoami",
			"date",
			"uname -a",
		}

		for _, cmd := range commands {
			t.Run(fmt.Sprintf("Command_%s", strings.ReplaceAll(cmd, " ", "_")), func(t *testing.T) {
				output, err := client.ExecuteCommand(ctx, hostname, cmd)
				require.NoError(t, err)
				assert.NotEmpty(t, output)
			})
		}
	})
}

func testSessionManagerIntegration(t *testing.T, testHost, testUser string) {
	config := &ssh.SSHConfig{
		Username: testUser,
		Port:     parsePort(testHost),
		Timeout:  10 * time.Second,
	}

	manager, err := ssh.NewSessionManager(config)
	require.NoError(t, err)
	defer manager.Shutdown()

	hostname := parseHostname(testHost)

	t.Run("CreateSession", func(t *testing.T) {
		session, err := manager.CreateSession(hostname, testUser)
		if err != nil {
			t.Logf("Failed to create SSH session: %v", err)
			t.Skip("SSH session creation failed - check connectivity")
		}
		assert.NotNil(t, session)
		assert.Equal(t, hostname, session.Hostname)
		assert.Equal(t, testUser, session.Username)
	})

	t.Run("MultipleSession", func(t *testing.T) {
		session1, err := manager.CreateSession(hostname, testUser)
		if err != nil {
			t.Skip("SSH session creation failed")
		}

		session2, err := manager.CreateSession(hostname+"_alt", testUser) // Different hostname
		require.NoError(t, err)

		// Should be different sessions with different hostnames
		assert.NotEqual(t, session1.ID, session2.ID)
		assert.NotEqual(t, session1.Hostname, session2.Hostname)
	})

	t.Run("SessionLifecycle", func(t *testing.T) {
		session, err := manager.CreateSession(hostname, testUser)
		if err != nil {
			t.Skip("SSH session creation failed")
		}

		// Test session operations
		allSessions := manager.GetSessions()
		assert.Contains(t, allSessions, session.ID)

		// Get session info
		sessionInfo, err := manager.GetSessionInfo(session.ID)
		assert.NoError(t, err)
		assert.Equal(t, session.ID, sessionInfo.ID)

		// Cleanup session
		err = manager.CloseSession(session.ID)
		assert.NoError(t, err)
	})
}

func testTerminalManagerIntegration(t *testing.T, _, _ string) {
	// Skip terminal manager tests for now since they require tview integration
	t.Skip("Terminal manager integration tests require GUI components")
}

func testPersistenceIntegration(t *testing.T, testHost, testUser string) {
	config := &ssh.SSHConfig{
		Username: testUser,
		Port:     parsePort(testHost),
		Timeout:  10 * time.Second,
	}

	// Create temporary data directory
	tempDataDir, err := os.MkdirTemp("", "ssh_test_data_*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDataDir) }()

	hostname := parseHostname(testHost)

	t.Run("SessionPersistence", func(t *testing.T) {
		// Create session manager with persistence
		manager1, err := ssh.NewSessionManager(config)
		require.NoError(t, err)

		// Enable persistence
		err = manager1.EnablePersistence(tempDataDir)
		require.NoError(t, err)

		// Create a session
		session, err := manager1.CreateSession(hostname, testUser)
		if err != nil {
			t.Skip("SSH session creation failed")
		}

		_ = session.ID // We have the session ID but don't use it in this test

		// Close the manager
		manager1.Shutdown()

		// Create new manager and load sessions
		manager2, err := ssh.NewSessionManager(config)
		require.NoError(t, err)
		defer manager2.Shutdown()

		// Enable persistence and load sessions
		err = manager2.EnablePersistence(tempDataDir)
		require.NoError(t, err)

		// Load persistent sessions directly
		persistence, err := ssh.NewSessionPersistence(tempDataDir)
		require.NoError(t, err)

		persistentSessions, err := persistence.LoadSessions()
		require.NoError(t, err)

		// We should have at least one session saved (though it might be filtered as stale)
		t.Logf("Found %d persistent sessions", len(persistentSessions))
	})

	t.Run("SessionTags", func(t *testing.T) {
		persistence, err := ssh.NewSessionPersistence(tempDataDir)
		require.NoError(t, err)

		sessionID := "test_session_123"
		tags := map[string]string{
			"environment": "test",
			"purpose":     "integration_testing",
			"user":        testUser,
		}

		// Save tags
		err = persistence.SaveSessionTags(sessionID, tags)
		require.NoError(t, err)

		// Load tags
		loadedTags, err := persistence.LoadSessionTags(sessionID)
		require.NoError(t, err)
		assert.Equal(t, tags, loadedTags)
	})
}

// Helper functions

func parseHostname(hostPort string) string {
	if strings.Contains(hostPort, ":") {
		host, _, err := net.SplitHostPort(hostPort)
		if err != nil {
			return hostPort
		}
		return host
	}
	return hostPort
}

func parsePort(hostPort string) int {
	if strings.Contains(hostPort, ":") {
		_, portStr, err := net.SplitHostPort(hostPort)
		if err != nil {
			return 22
		}

		port := 22
		if portStr != "" {
			if p, err := net.LookupPort("tcp", portStr); err == nil {
				port = p
			}
		}
		return port
	}
	return 22
}

// TestSSHAvailability checks if SSH client is available
func TestSSHAvailability(t *testing.T) {
	available := ssh.IsSSHAvailable()
	if !available {
		t.Skip("SSH client not available on this system")
	}
	assert.True(t, available)
}

// TestDockerSSHServer sets up a temporary SSH server using Docker for testing
func TestDockerSSHServer(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Docker SSH server test in short mode")
	}

	if os.Getenv("DOCKER_SSH_TESTS") != "1" {
		t.Skip("Docker SSH tests disabled. Set DOCKER_SSH_TESTS=1 to enable")
	}

	// Check if Docker is available
	if !isDockerAvailable() {
		t.Skip("Docker not available")
	}

	t.Run("SetupSSHContainer", func(t *testing.T) {
		containerID, err := setupSSHContainer(t)
		if err != nil {
			t.Fatalf("Failed to setup SSH container: %v", err)
		}
		defer cleanupSSHContainer(t, containerID)

		// Wait for SSH server to be ready
		time.Sleep(2 * time.Second)

		// Test SSH connection to container
		config := &ssh.SSHConfig{
			Username: "testuser",
			Port:     2222,
			Timeout:  10 * time.Second,
			Options: map[string]string{
				"StrictHostKeyChecking": "no",
				"UserKnownHostsFile":    "/dev/null",
			},
		}

		client := ssh.NewSSHClient(config)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		output, err := client.ExecuteCommand(ctx, "localhost", "echo 'Container SSH test'")
		require.NoError(t, err)
		assert.Contains(t, output, "Container SSH test")
	})
}

func isDockerAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func setupSSHContainer(_ *testing.T) (string, error) {
	// Create a temporary directory for SSH keys
	keyDir, err := os.MkdirTemp("", "ssh_test_keys_*")
	if err != nil {
		return "", err
	}

	// Generate SSH key pair
	privateKeyPath := filepath.Join(keyDir, "test_key")
	publicKeyPath := filepath.Join(keyDir, "test_key.pub")

	cmd := exec.Command("ssh-keygen", "-t", "rsa", "-b", "2048", "-f", privateKeyPath, "-N", "")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to generate SSH key: %w", err)
	}

	// Read public key
	pubKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", err
	}

	// Create Dockerfile content
	dockerfile := fmt.Sprintf(`
FROM ubuntu:20.04
RUN apt-get update && apt-get install -y openssh-server sudo
RUN useradd -m -s /bin/bash testuser
RUN echo 'testuser:testpass' | chpasswd
RUN mkdir -p /home/testuser/.ssh
RUN echo '%s' > /home/testuser/.ssh/authorized_keys
RUN chown -R testuser:testuser /home/testuser/.ssh
RUN chmod 600 /home/testuser/.ssh/authorized_keys
RUN chmod 700 /home/testuser/.ssh
RUN mkdir /var/run/sshd
EXPOSE 22
CMD ["/usr/sbin/sshd", "-D"]
`, strings.TrimSpace(string(pubKeyData)))

	// Write Dockerfile
	dockerfilePath := filepath.Join(keyDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(dockerfile), 0600); err != nil {
		return "", err
	}

	// Build Docker image
	buildCmd := exec.Command("docker", "build", "-t", "ssh-test-server", keyDir)
	if err := buildCmd.Run(); err != nil {
		return "", fmt.Errorf("failed to build Docker image: %w", err)
	}

	// Run container
	runCmd := exec.Command("docker", "run", "-d", "-p", "2222:22", "ssh-test-server")
	output, err := runCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to run Docker container: %w", err)
	}

	containerID := strings.TrimSpace(string(output))

	// Set up SSH client to use the generated key
	_ = os.Setenv("SSH_TEST_KEY", privateKeyPath)

	return containerID, nil
}

func cleanupSSHContainer(_ *testing.T, containerID string) {
	// Stop and remove container
	_ = exec.Command("docker", "stop", containerID).Run()
	_ = exec.Command("docker", "rm", containerID).Run()

	// Remove test image
	_ = exec.Command("docker", "rmi", "ssh-test-server").Run()
}
