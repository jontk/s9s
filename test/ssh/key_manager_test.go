package ssh

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jontk/s9s/internal/ssh"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyManager(t *testing.T) {
	// Create temporary SSH directory
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	err := os.MkdirAll(sshDir, 0700)
	require.NoError(t, err)

	// Set up mock environment
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	t.Run("NewKeyManager", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		require.NotNil(t, km)
		defer func() { _ = km.Cleanup() }()

		keys := km.GetKeys()
		assert.Empty(t, keys) // No keys initially
	})

	t.Run("GenerateRSAKey", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		err = km.GenerateKey("test_rsa", "test@example.com", "rsa", 2048)
		require.NoError(t, err)

		// Check key files exist
		keyPath := filepath.Join(sshDir, "test_rsa")
		pubPath := keyPath + ".pub"

		assert.FileExists(t, keyPath)
		assert.FileExists(t, pubPath)

		// Check key was discovered
		keys := km.GetKeys()
		assert.Len(t, keys, 1)
		assert.Contains(t, keys, "test_rsa")

		key := keys["test_rsa"]
		assert.Equal(t, "test_rsa", key.Name)
		assert.Equal(t, keyPath, key.Path)
		assert.Equal(t, pubPath, key.PublicPath)
		assert.Equal(t, "ssh-rsa", key.Type)
		assert.NotEmpty(t, key.Fingerprint)

		// Clean up
		err = km.DeleteKey("test_rsa")
		require.NoError(t, err)
	})

	t.Run("GenerateEd25519Key", func(t *testing.T) {
		// Skip if ssh-keygen not available
		if !ssh.IsSSHAvailable() {
			t.Skip("ssh-keygen not available")
		}

		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		err = km.GenerateKey("test_ed25519", "test@example.com", "ed25519", 0)
		require.NoError(t, err)

		// Check key files exist
		keyPath := filepath.Join(sshDir, "test_ed25519")
		pubPath := keyPath + ".pub"

		assert.FileExists(t, keyPath)
		assert.FileExists(t, pubPath)

		// Check key was discovered
		keys := km.GetKeys()
		assert.Contains(t, keys, "test_ed25519")

		key := keys["test_ed25519"]
		assert.Equal(t, "test_ed25519", key.Name)
		assert.Equal(t, "ssh-ed25519", key.Type)

		// Clean up
		err = km.DeleteKey("test_ed25519")
		require.NoError(t, err)
	})

	t.Run("GetKey", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		// Generate a key first
		err = km.GenerateKey("test_get", "test@example.com", "rsa", 2048)
		require.NoError(t, err)

		// Test getting existing key
		key, err := km.GetKey("test_get")
		require.NoError(t, err)
		assert.Equal(t, "test_get", key.Name)

		// Test getting non-existent key
		_, err = km.GetKey("nonexistent")
		assert.Error(t, err)

		// Clean up
		err = km.DeleteKey("test_get")
		require.NoError(t, err)
	})

	t.Run("DeleteKey", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		// Generate a key first
		err = km.GenerateKey("test_delete", "test@example.com", "rsa", 2048)
		require.NoError(t, err)

		keyPath := filepath.Join(sshDir, "test_delete")
		pubPath := keyPath + ".pub"

		// Verify files exist
		assert.FileExists(t, keyPath)
		assert.FileExists(t, pubPath)

		// Delete the key
		err = km.DeleteKey("test_delete")
		require.NoError(t, err)

		// Verify files are gone
		assert.NoFileExists(t, keyPath)
		assert.NoFileExists(t, pubPath)

		// Verify key is removed from memory
		keys := km.GetKeys()
		assert.NotContains(t, keys, "test_delete")
	})

	t.Run("DiscoverExistingKeys", func(t *testing.T) {
		// Skip if ssh-keygen not available
		if !ssh.IsSSHAvailable() {
			t.Skip("ssh-keygen not available")
		}

		// Create a key manager and generate a real key
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		// Generate a real RSA key
		keyName := "manual_key"
		err = km.GenerateKey(keyName, "test@example.com", "rsa", 2048)
		require.NoError(t, err)

		// Get all keys to verify discovery
		keys := km.GetKeys()
		assert.Contains(t, keys, keyName)

		// Verify key properties
		discoveredKey := keys[keyName]
		assert.Equal(t, keyName, discoveredKey.Name)
		keyPath := filepath.Join(sshDir, keyName)
		pubPath := keyPath + ".pub"
		assert.Equal(t, keyPath, discoveredKey.Path)
		assert.Equal(t, pubPath, discoveredKey.PublicPath)

		// Clean up the key to avoid interfering with other tests
		err = km.DeleteKey(keyName)
		require.NoError(t, err)
	})

	t.Run("GetKeyPath", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		// Generate a key first
		err = km.GenerateKey("test_path", "test@example.com", "rsa", 2048)
		require.NoError(t, err)

		// Test getting key path
		path, err := km.GetKeyPath("test_path")
		require.NoError(t, err)
		expectedPath := filepath.Join(sshDir, "test_path")
		assert.Equal(t, expectedPath, path)

		// Test getting path for non-existent key
		_, err = km.GetKeyPath("nonexistent")
		assert.Error(t, err)

		// Clean up
		err = km.DeleteKey("test_path")
		require.NoError(t, err)
	})

	t.Run("SetAutoLoad", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		// Test setting auto load
		km.SetAutoLoad(false)
		km.SetAutoLoad(true)

		// No assertions needed, just test that it doesn't panic
	})

	t.Run("IsAgentConnected", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		// Initially should not be connected (no SSH_AUTH_SOCK)
		assert.False(t, km.IsAgentConnected())
	})

	t.Run("GetSSHConfig", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		// Generate a key first
		err = km.GenerateKey("test_config", "test@example.com", "rsa", 2048)
		require.NoError(t, err)

		// Get SSH config
		config, err := km.GetSSHConfig()
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Should have the key file set
		expectedKeyPath := filepath.Join(sshDir, "test_config")
		assert.Equal(t, expectedKeyPath, config.KeyFile)

		// Clean up
		err = km.DeleteKey("test_config")
		require.NoError(t, err)
	})
}

func TestKeyManagerWithAgent(t *testing.T) {
	// These tests require an SSH agent - skip if not available
	if !ssh.IsAgentAvailable() {
		t.Skip("SSH agent not available")
	}

	// Create temporary SSH directory
	tempDir := t.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	err := os.MkdirAll(sshDir, 0700)
	require.NoError(t, err)

	// Set up mock environment
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	t.Run("ConnectToAgent", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		// Should be connected to agent
		assert.True(t, km.IsAgentConnected())

		// Get agent info
		info, err := km.GetAgentInfo()
		require.NoError(t, err)
		assert.True(t, info["connected"].(bool))
		assert.NotEmpty(t, info["socket_path"])
	})

	t.Run("LoadKeyToAgent", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		if !km.IsAgentConnected() {
			t.Skip("SSH agent not connected")
		}

		// Generate a key
		err = km.GenerateKey("test_load", "test@example.com", "rsa", 2048)
		require.NoError(t, err)

		// Load key to agent
		err = km.LoadKeyToAgent("test_load")
		require.NoError(t, err)

		// Check if key is loaded
		key, err := km.GetKey("test_load")
		require.NoError(t, err)
		assert.True(t, key.IsLoaded)

		// Cleanup - remove key from agent
		err = km.RemoveKeyFromAgent("test_load")
		require.NoError(t, err)
	})

	t.Run("RefreshKeyStatus", func(t *testing.T) {
		km, err := ssh.NewKeyManager()
		require.NoError(t, err)
		defer func() { _ = km.Cleanup() }()

		if !km.IsAgentConnected() {
			t.Skip("SSH agent not connected")
		}

		// Refresh key status
		err = km.RefreshKeyStatus()
		require.NoError(t, err)

		// No specific assertions, just ensure it doesn't error
	})
}

func TestAgentOperations(t *testing.T) {
	if !ssh.IsAgentAvailable() {
		t.Skip("SSH agent not available")
	}

	t.Run("TestAgentConnection", func(t *testing.T) {
		err := ssh.TestAgentConnection()
		assert.NoError(t, err)
	})

	t.Run("GetAgentKeyCount", func(t *testing.T) {
		count, err := ssh.GetAgentKeyCount()
		require.NoError(t, err)
		assert.GreaterOrEqual(t, count, 0)
	})

	t.Run("NewAgentAuth", func(t *testing.T) {
		auth, err := ssh.NewAgentAuth()
		require.NoError(t, err)
		require.NotNil(t, auth)
		defer func() { _ = auth.Close() }()

		// Test getting signers
		signers := auth.GetSigners()
		assert.NotNil(t, signers)

		// Test getting auth method
		authMethod := auth.GetAuthMethod()
		assert.NotNil(t, authMethod)

		// Test getting fingerprints
		fingerprints := auth.GetKeyFingerprints()
		assert.NotNil(t, fingerprints)

		// Test getting key info
		keyInfos := auth.GetKeyInfo()
		assert.NotNil(t, keyInfos)
	})
}

func BenchmarkKeyDiscovery(b *testing.B) {
	// Create temporary SSH directory with some keys
	tempDir := b.TempDir()
	sshDir := filepath.Join(tempDir, ".ssh")
	err := os.MkdirAll(sshDir, 0700)
	require.NoError(b, err)

	// Set up mock environment
	originalHome := os.Getenv("HOME")
	_ = os.Setenv("HOME", tempDir)
	defer func() { _ = os.Setenv("HOME", originalHome) }()

	// Create some dummy key files
	for i := 0; i < 5; i++ {
		// nolint:gosec // G101: test RSA key only, non-production
		keyContent := `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1234567...
-----END RSA PRIVATE KEY-----`

		keyPath := filepath.Join(sshDir, fmt.Sprintf("test_key_%d", i))
		err := os.WriteFile(keyPath, []byte(keyContent), 0600)
		require.NoError(b, err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		km, err := ssh.NewKeyManager()
		require.NoError(b, err)
		_ = km.Cleanup()
	}
}
