package ssh

import (
	"fmt"
	"net"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

// AgentAuth provides SSH agent-based authentication
type AgentAuth struct {
	agentConn net.Conn
	signers   []ssh.Signer
}

// NewAgentAuth creates a new agent-based authenticator
func NewAgentAuth() (*AgentAuth, error) {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return nil, fmt.Errorf("SSH_AUTH_SOCK not set - no SSH agent available")
	}

	conn, err := net.Dial("unix", authSock)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(conn)
	signers, err := agentClient.Signers()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to get signers from SSH agent: %w", err)
	}

	return &AgentAuth{
		agentConn: conn,
		signers:   signers,
	}, nil
}

// GetSigners returns SSH signers from the agent
func (aa *AgentAuth) GetSigners() []ssh.Signer {
	return aa.signers
}

// GetAuthMethod returns the SSH auth method for agent authentication
func (aa *AgentAuth) GetAuthMethod() ssh.AuthMethod {
	return ssh.PublicKeys(aa.signers...)
}

// RefreshSigners refreshes the list of signers from the agent
func (aa *AgentAuth) RefreshSigners() error {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return fmt.Errorf("SSH_AUTH_SOCK not set")
	}

	// Close existing connection
	if aa.agentConn != nil {
		_ = aa.agentConn.Close()
	}

	// Reconnect
	conn, err := net.Dial("unix", authSock)
	if err != nil {
		return fmt.Errorf("failed to reconnect to SSH agent: %w", err)
	}

	agentClient := agent.NewClient(conn)
	signers, err := agentClient.Signers()
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("failed to get signers from SSH agent: %w", err)
	}

	aa.agentConn = conn
	aa.signers = signers

	return nil
}

// Close closes the agent connection
func (aa *AgentAuth) Close() error {
	if aa.agentConn != nil {
		return aa.agentConn.Close()
	}
	return nil
}

// GetKeyFingerprints returns fingerprints of all keys in the agent
func (aa *AgentAuth) GetKeyFingerprints() []string {
	var fingerprints []string
	for _, signer := range aa.signers {
		fingerprint := ssh.FingerprintSHA256(signer.PublicKey())
		fingerprints = append(fingerprints, fingerprint)
	}
	return fingerprints
}

// HasKey checks if a key with the given fingerprint is available
func (aa *AgentAuth) HasKey(fingerprint string) bool {
	for _, signer := range aa.signers {
		if ssh.FingerprintSHA256(signer.PublicKey()) == fingerprint {
			return true
		}
	}
	return false
}

// GetKeyInfo returns information about keys in the agent
func (aa *AgentAuth) GetKeyInfo() []map[string]string {
	var keyInfos []map[string]string

	for _, signer := range aa.signers {
		pubKey := signer.PublicKey()
		keyInfo := map[string]string{
			"type":        pubKey.Type(),
			"fingerprint": ssh.FingerprintSHA256(pubKey),
		}
		keyInfos = append(keyInfos, keyInfo)
	}

	return keyInfos
}

// CreateSSHClientConfig creates an SSH client config with agent authentication
func (aa *AgentAuth) CreateSSHClientConfig(username string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			aa.GetAuthMethod(),
		},
		HostKeyCallback: getHostKeyCallback(nil), // Use default (insecure) for backward compatibility
	}
}

// IsAgentAvailable checks if SSH agent is available and has keys
func IsAgentAvailable() bool {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return false
	}

	conn, err := net.Dial("unix", authSock)
	if err != nil {
		return false
	}
	defer func() { _ = conn.Close() }()

	agentClient := agent.NewClient(conn)
	keys, err := agentClient.List()
	if err != nil {
		return false
	}

	return len(keys) > 0
}

// GetAgentKeyCount returns the number of keys in the SSH agent
func GetAgentKeyCount() (int, error) {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return 0, fmt.Errorf("SSH_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", authSock)
	if err != nil {
		return 0, fmt.Errorf("failed to connect to SSH agent: %w", err)
	}
	defer func() { _ = conn.Close() }()

	agentClient := agent.NewClient(conn)
	keys, err := agentClient.List()
	if err != nil {
		return 0, fmt.Errorf("failed to list agent keys: %w", err)
	}

	return len(keys), nil
}

// TestAgentConnection tests the connection to the SSH agent
func TestAgentConnection() error {
	authSock := os.Getenv("SSH_AUTH_SOCK")
	if authSock == "" {
		return fmt.Errorf("SSH_AUTH_SOCK not set")
	}

	conn, err := net.Dial("unix", authSock)
	if err != nil {
		return fmt.Errorf("failed to connect to SSH agent: %w", err)
	}
	defer func() { _ = conn.Close() }()

	agentClient := agent.NewClient(conn)
	_, err = agentClient.List()
	if err != nil {
		return fmt.Errorf("failed to communicate with SSH agent: %w", err)
	}

	return nil
}
