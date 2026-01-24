package ssh

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// SSHKey represents an SSH key pair
type SSHKey struct {
	Name        string
	Path        string
	PublicPath  string
	Type        string
	Fingerprint string
	Comment     string
	IsLoaded    bool
	CreatedAt   time.Time
	LastUsed    time.Time
}

// SSHAgent represents an SSH agent connection
type SSHAgent struct {
	conn   net.Conn
	client agent.Agent
	mu     sync.RWMutex
}

// KeyManager manages SSH keys and agent integration
type KeyManager struct {
	keys     map[string]*SSHKey
	agent    *SSHAgent
	keyDir   string // SSH key directory (usually ~/.ssh)
	mu       sync.RWMutex
	autoLoad bool
}

// NewKeyManager creates a new SSH key manager
func NewKeyManager() (*KeyManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	keyDir := filepath.Join(homeDir, ".ssh")

	km := &KeyManager{
		keys:     make(map[string]*SSHKey),
		keyDir:   keyDir,
		autoLoad: true,
	}

	// Try to connect to SSH agent
	if err := km.ConnectToAgent(); err != nil {
		// Agent connection is optional
		fmt.Printf("Warning: Could not connect to SSH agent: %v\n", err)
	}

	// Discover existing keys
	if err := km.DiscoverKeys(); err != nil {
		return nil, fmt.Errorf("failed to discover SSH keys: %w", err)
	}

	return km, nil
}

// ConnectToAgent connects to the SSH agent
func (km *KeyManager) ConnectToAgent() error {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return fmt.Errorf("SSH_AUTH_SOCK not set")
	}

	conn, err := (&net.Dialer{}).DialContext(context.Background(), "unix", authSock)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(conn)

	km.mu.Lock()
	km.agent = &SSHAgent{
		conn:   conn,
		client: agentClient,
	}
	km.mu.Unlock()

	return nil
}

// DisconnectFromAgent disconnects from the SSH agent
func (km *KeyManager) DisconnectFromAgent() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.agent != nil {
		if err := km.agent.conn.Close(); err != nil {
			return fmt.Errorf("failed to close agent connection: %w", err)
		}
		km.agent = nil
	}
	return nil
}

// DiscoverKeys discovers SSH keys in the SSH directory
func (km *KeyManager) DiscoverKeys() error {
	if _, err := os.Stat(km.keyDir); os.IsNotExist(err) {
		// Create SSH directory if it doesn't exist
		if err := os.MkdirAll(km.keyDir, 0700); err != nil {
			return fmt.Errorf("failed to create SSH directory: %w", err)
		}
		return nil
	}

	err := filepath.WalkDir(km.keyDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		// Skip known files that aren't private keys
		basename := filepath.Base(path)
		if strings.HasSuffix(basename, ".pub") ||
			strings.HasSuffix(basename, ".ppk") ||
			basename == "known_hosts" ||
			basename == "authorized_keys" ||
			basename == "config" {
			return nil
		}

		// Try to parse as SSH private key
		if km.isPrivateKey(path) {
			key, err := km.parseKey(path)
			if err != nil {
				// Log but don't fail
				fmt.Printf("Warning: Could not parse key %s: %v\n", path, err)
				return nil
			}

			km.mu.Lock()
			km.keys[key.Name] = key
			km.mu.Unlock()
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to walk SSH directory: %w", err)
	}

	// Auto-load keys to agent if enabled
	if km.autoLoad {
		_ = km.LoadKeysToAgent()
	}

	return nil
}

// isPrivateKey checks if a file is likely an SSH private key
func (km *KeyManager) isPrivateKey(path string) bool {
	// nolint:gosec // G304: path from scanning user's ~/.ssh directory, application-controlled
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		firstLine := strings.TrimSpace(scanner.Text())
		return strings.HasPrefix(firstLine, "-----BEGIN") &&
			(strings.Contains(firstLine, "PRIVATE KEY") ||
				strings.Contains(firstLine, "RSA PRIVATE KEY") ||
				strings.Contains(firstLine, "OPENSSH PRIVATE KEY"))
	}

	return false
}

// parseKey parses an SSH key file
func (km *KeyManager) parseKey(path string) (*SSHKey, error) {
	// nolint:gosec // G304: path from scanning user's ~/.ssh directory, application-controlled
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	// Try to parse the private key
	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Get file info
	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat key file: %w", err)
	}

	// Try to find the corresponding public key
	pubPath := path + ".pub"
	var fingerprint, comment, keyType string

	// nolint:gosec // G304: pubPath derived from path in user's ~/.ssh directory, application-controlled
	if pubData, err := os.ReadFile(pubPath); err == nil {
		pubKey, commentBytes, _, _, err := ssh.ParseAuthorizedKey(pubData)
		if err == nil {
			fingerprint = ssh.FingerprintSHA256(pubKey)
			keyType = pubKey.Type()
			comment = commentBytes
		}
	}

	// If no public key file, try to derive from private key
	if fingerprint == "" {
		if privKey, err := ssh.ParseRawPrivateKey(keyData); err == nil {
			switch k := privKey.(type) {
			case *rsa.PrivateKey:
				pubKey, err := ssh.NewPublicKey(&k.PublicKey)
				if err == nil {
					fingerprint = ssh.FingerprintSHA256(pubKey)
					keyType = pubKey.Type()
				}
			}
		}
	}

	// Check if key is loaded in agent
	isLoaded := km.IsKeyLoadedInAgent(fingerprint)

	key := &SSHKey{
		Name:        filepath.Base(path),
		Path:        path,
		PublicPath:  pubPath,
		Type:        keyType,
		Fingerprint: fingerprint,
		Comment:     comment,
		IsLoaded:    isLoaded,
		CreatedAt:   stat.ModTime(),
		LastUsed:    stat.ModTime(), // We'll update this with agent info if available
	}

	return key, nil
}

// GetKeys returns all discovered SSH keys
func (km *KeyManager) GetKeys() map[string]*SSHKey {
	km.mu.RLock()
	defer km.mu.RUnlock()

	// Return a copy to prevent external modification
	keysCopy := make(map[string]*SSHKey)
	for name, key := range km.keys {
		keyCopy := *key
		keysCopy[name] = &keyCopy
	}

	return keysCopy
}

// GetKey returns a specific SSH key
func (km *KeyManager) GetKey(name string) (*SSHKey, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	key, exists := km.keys[name]
	if !exists {
		return nil, fmt.Errorf("key '%s' not found", name)
	}

	// Return a copy
	keyCopy := *key
	return &keyCopy, nil
}

// GenerateKey generates a new SSH key pair
func (km *KeyManager) GenerateKey(name, comment string, keyType string, bits int) error {
	if keyType == "" {
		keyType = "rsa"
	}
	if bits == 0 {
		bits = 2048
	}

	// Validate that name doesn't contain path separators or traversal attempts
	if strings.Contains(name, string(filepath.Separator)) || strings.Contains(name, "..") {
		return fmt.Errorf("invalid key name: must not contain path separators or traversal")
	}

	keyPath := filepath.Join(km.keyDir, name)
	pubPath := keyPath + ".pub"

	// Check if key already exists
	if _, err := os.Stat(keyPath); err == nil {
		return fmt.Errorf("key '%s' already exists", name)
	}

	switch keyType {
	case "rsa":
		return km.generateRSAKey(keyPath, pubPath, comment, bits)
	case "ed25519":
		return km.generateEd25519Key(keyPath, pubPath, comment)
	default:
		return fmt.Errorf("unsupported key type: %s", keyType)
	}
}

// generateRSAKey generates an RSA key pair
func (km *KeyManager) generateRSAKey(keyPath, pubPath, comment string, bits int) error {
	// Generate RSA private key
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Create private key file
	// nolint:gosec // G304: keyPath validated in GenerateKey() to prevent path traversal
	privFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer func() { _ = privFile.Close() }()

	// Write private key in PKCS#1 format
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	}

	if err := pem.Encode(privFile, privKeyPEM); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Create public key
	pubKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to create public key: %w", err)
	}

	// Write public key file
	// nolint:gosec // G304: pubPath validated in GenerateKey() to prevent path traversal
	pubFile, err := os.OpenFile(pubPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer func() { _ = pubFile.Close() }()

	pubKeyStr := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(pubKey)))
	if comment != "" {
		pubKeyStr += " " + comment
	}
	pubKeyStr += "\n"

	if _, err := pubFile.WriteString(pubKeyStr); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	// Refresh key list
	return km.DiscoverKeys()
}

// generateEd25519Key generates an Ed25519 key pair using ssh-keygen
func (km *KeyManager) generateEd25519Key(keyPath, _pubPath, comment string) error {
	args := []string{
		"-t", "ed25519",
		"-f", keyPath,
		"-N", "", // No passphrase
	}

	if comment != "" {
		args = append(args, "-C", comment)
	}

	// nolint:gosec // G204: ssh-keygen is a well-known system command, args are controlled and validated
	cmd := exec.CommandContext(context.Background(), "ssh-keygen", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate Ed25519 key: %w", err)
	}

	// Refresh key list
	return km.DiscoverKeys()
}

// DeleteKey deletes an SSH key pair
func (km *KeyManager) DeleteKey(name string) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	key, exists := km.keys[name]
	if !exists {
		return fmt.Errorf("key '%s' not found", name)
	}

	// Remove from agent first if loaded
	if key.IsLoaded && km.agent != nil {
		if err := km.removeKeyFromAgent(key); err != nil {
			// Log but don't fail
			fmt.Printf("Warning: Could not remove key from agent: %v\n", err)
		}
	}

	// Delete private key file
	if err := os.Remove(key.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete private key: %w", err)
	}

	// Delete public key file
	if err := os.Remove(key.PublicPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete public key: %w", err)
	}

	// Remove from memory
	delete(km.keys, name)

	return nil
}

// LoadKeyToAgent loads a specific key to the SSH agent
func (km *KeyManager) LoadKeyToAgent(name string) error {
	if km.agent == nil {
		return fmt.Errorf("SSH agent not connected")
	}

	key, exists := km.keys[name]
	if !exists {
		return fmt.Errorf("key '%s' not found", name)
	}

	if key.IsLoaded {
		return nil // Already loaded
	}

	// Read private key
	keyData, err := os.ReadFile(key.Path)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}

	// Parse private key
	privKey, err := ssh.ParseRawPrivateKey(keyData)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Add to agent
	addedKey := agent.AddedKey{
		PrivateKey: privKey,
		Comment:    key.Comment,
	}

	km.agent.mu.Lock()
	err = km.agent.client.Add(addedKey)
	km.agent.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to add key to agent: %w", err)
	}

	// Update key status
	km.mu.Lock()
	key.IsLoaded = true
	km.mu.Unlock()

	return nil
}

// LoadKeysToAgent loads all keys to the SSH agent
func (km *KeyManager) LoadKeysToAgent() error {
	if km.agent == nil {
		return fmt.Errorf("SSH agent not connected")
	}

	var errors []string
	for name := range km.keys {
		if err := km.LoadKeyToAgent(name); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to load some keys: %s", strings.Join(errors, "; "))
	}

	return nil
}

// RemoveKeyFromAgent removes a key from the SSH agent
func (km *KeyManager) RemoveKeyFromAgent(name string) error {
	if km.agent == nil {
		return fmt.Errorf("SSH agent not connected")
	}

	key, exists := km.keys[name]
	if !exists {
		return fmt.Errorf("key '%s' not found", name)
	}

	if !key.IsLoaded {
		return nil // Not loaded
	}

	return km.removeKeyFromAgent(key)
}

// removeKeyFromAgent removes a key from the agent (internal)
func (km *KeyManager) removeKeyFromAgent(key *SSHKey) error {
	// Read public key to get the key for removal
	pubData, err := os.ReadFile(key.PublicPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %w", err)
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(pubData)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	km.agent.mu.Lock()
	err = km.agent.client.Remove(pubKey)
	km.agent.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to remove key from agent: %w", err)
	}

	// Update key status
	km.mu.Lock()
	key.IsLoaded = false
	km.mu.Unlock()

	return nil
}

// RemoveAllKeysFromAgent removes all keys from the SSH agent
func (km *KeyManager) RemoveAllKeysFromAgent() error {
	if km.agent == nil {
		return fmt.Errorf("SSH agent not connected")
	}

	km.agent.mu.Lock()
	err := km.agent.client.RemoveAll()
	km.agent.mu.Unlock()

	if err != nil {
		return fmt.Errorf("failed to remove all keys from agent: %w", err)
	}

	// Update all key statuses
	km.mu.Lock()
	for _, key := range km.keys {
		key.IsLoaded = false
	}
	km.mu.Unlock()

	return nil
}

// GetAgentKeys returns keys currently loaded in the SSH agent
func (km *KeyManager) GetAgentKeys() ([]*agent.Key, error) {
	if km.agent == nil {
		return nil, fmt.Errorf("SSH agent not connected")
	}

	km.agent.mu.RLock()
	keys, err := km.agent.client.List()
	km.agent.mu.RUnlock()

	if err != nil {
		return nil, fmt.Errorf("failed to list agent keys: %w", err)
	}

	return keys, nil
}

// IsKeyLoadedInAgent checks if a key is loaded in the agent
func (km *KeyManager) IsKeyLoadedInAgent(fingerprint string) bool {
	if km.agent == nil || fingerprint == "" {
		return false
	}

	agentKeys, err := km.GetAgentKeys()
	if err != nil {
		return false
	}

	for _, agentKey := range agentKeys {
		if ssh.FingerprintSHA256(agentKey) == fingerprint {
			return true
		}
	}

	return false
}

// RefreshKeyStatus refreshes the loaded status of all keys
func (km *KeyManager) RefreshKeyStatus() error {
	if km.agent == nil {
		return nil
	}

	agentKeys, err := km.GetAgentKeys()
	if err != nil {
		return fmt.Errorf("failed to get agent keys: %w", err)
	}

	agentFingerprints := make(map[string]bool)
	for _, agentKey := range agentKeys {
		agentFingerprints[ssh.FingerprintSHA256(agentKey)] = true
	}

	km.mu.Lock()
	for _, key := range km.keys {
		key.IsLoaded = agentFingerprints[key.Fingerprint]
	}
	km.mu.Unlock()

	return nil
}

// GetSSHConfig generates SSH client configuration using available keys
func (km *KeyManager) GetSSHConfig() (*SSHConfig, error) {
	config := DefaultSSHConfig()

	// Add available private keys
	var keyFiles []string
	for _, key := range km.GetKeys() {
		if key.IsLoaded || key.Path != "" {
			keyFiles = append(keyFiles, key.Path)
		}
	}

	if len(keyFiles) > 0 {
		config.KeyFile = keyFiles[0] // Use first available key as default
	}

	// If agent is connected, prefer agent authentication
	if km.agent != nil {
		config.Options["IdentitiesOnly"] = "no"
		config.Options["UseAgent"] = "yes"
	}

	return config, nil
}

// GetKeyPath returns the path to a specific key
func (km *KeyManager) GetKeyPath(name string) (string, error) {
	key, exists := km.keys[name]
	if !exists {
		return "", fmt.Errorf("key '%s' not found", name)
	}
	return key.Path, nil
}

// SetAutoLoad enables or disables automatic key loading
func (km *KeyManager) SetAutoLoad(enable bool) {
	km.mu.Lock()
	km.autoLoad = enable
	km.mu.Unlock()
}

// IsAgentConnected returns true if connected to SSH agent
func (km *KeyManager) IsAgentConnected() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.agent != nil
}

// GetAgentInfo returns information about the SSH agent
func (km *KeyManager) GetAgentInfo() (map[string]interface{}, error) {
	if km.agent == nil {
		return nil, fmt.Errorf("SSH agent not connected")
	}

	agentKeys, err := km.GetAgentKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to get agent keys: %w", err)
	}

	info := map[string]interface{}{
		"connected":   true,
		"socket_path": os.Getenv("SSH_AUTH_SOCK"),
		"keys_loaded": len(agentKeys),
		"agent_pid":   os.Getenv("SSH_AGENT_PID"),
	}

	// Add key details
	var keyInfos []map[string]string
	for _, agentKey := range agentKeys {
		keyInfo := map[string]string{
			"fingerprint": ssh.FingerprintSHA256(agentKey),
			"type":        agentKey.Type(),
			"comment":     agentKey.Comment,
		}
		keyInfos = append(keyInfos, keyInfo)
	}
	info["keys"] = keyInfos

	return info, nil
}

// StartAgent starts a new SSH agent if none is running
func (km *KeyManager) StartAgent() error {
	// Check if agent is already running
	if km.IsAgentConnected() {
		return nil
	}

	// Try to start ssh-agent
	cmd := exec.CommandContext(context.Background(), "ssh-agent", "-s")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to start SSH agent: %w", err)
	}

	// Parse ssh-agent output to set environment variables
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "SSH_AUTH_SOCK") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				sockPath := strings.Trim(parts[1], "; ")
				_ = os.Setenv("SSH_AUTH_SOCK", sockPath)
			}
		} else if strings.Contains(line, "SSH_AGENT_PID") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				pidStr := strings.Trim(parts[1], "; ")
				_ = os.Setenv("SSH_AGENT_PID", pidStr)
			}
		}
	}

	// Try to connect to the new agent
	return km.ConnectToAgent()
}

// StopAgent stops the SSH agent
func (km *KeyManager) StopAgent() error {
	if !km.IsAgentConnected() {
		return nil
	}

	// Disconnect first
	if err := km.DisconnectFromAgent(); err != nil {
		return fmt.Errorf("failed to disconnect from agent: %w", err)
	}

	// Try to kill the agent
	pidStr := os.Getenv("SSH_AGENT_PID")
	if pidStr != "" {
		pid, err := strconv.Atoi(pidStr)
		if err == nil {
			// nolint:gosec // G204: kill command with validated numeric PID from SSH_AGENT_PID
			cmd := exec.CommandContext(context.Background(), "kill", strconv.Itoa(pid))
			_ = cmd.Run() // Ignore errors
		}
	}

	// Clear environment variables
	_ = os.Unsetenv("SSH_AUTH_SOCK")
	_ = os.Unsetenv("SSH_AGENT_PID")

	return nil
}

// Cleanup cleans up resources
func (km *KeyManager) Cleanup() error {
	return km.DisconnectFromAgent()
}
