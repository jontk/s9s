package ssh

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jontk/s9s/internal/security"
)

// SessionState represents the state of an SSH session
type SessionState int

const (
	SessionIdle SessionState = iota
	SessionConnecting
	SessionConnected
	SessionActive
	SessionDisconnected
	SessionError
)

func (s SessionState) String() string {
	switch s {
	case SessionIdle:
		return "idle"
	case SessionConnecting:
		return "connecting"
	case SessionConnected:
		return "connected"
	case SessionActive:
		return "active"
	case SessionDisconnected:
		return "disconnected"
	case SessionError:
		return "error"
	default:
		return "unknown"
	}
}

// SSHSession represents an active SSH session
type SSHSession struct {
	ID           string
	Hostname     string
	Username     string
	State        SessionState
	StartTime    time.Time
	LastActivity time.Time
	Command      *exec.Cmd
	Process      *os.Process
	ErrorMessage string
	ControlPath  string // For connection multiplexing
	Tunnels      []SSHTunnel
	mu           sync.RWMutex
}

// SSHTunnel represents an SSH port forwarding tunnel
type SSHTunnel struct {
	LocalPort  int
	RemoteHost string
	RemotePort int
	Type       string // "local" or "remote"
	Active     bool
}

// SessionManager manages SSH sessions with advanced features
type SessionManager struct {
	sessions       map[string]*SSHSession
	config         *SSHConfig
	mu             sync.RWMutex
	controlDir     string
	cleanupDone    chan struct{}
	persistence    *SessionPersistence
	shutdownOnce   sync.Once
	sshCommandPath string // Validated absolute path to ssh command
}

// NewSessionManager creates a new SSH session manager
func NewSessionManager(config *SSHConfig) (*SessionManager, error) {
	if config == nil {
		config = DefaultSSHConfig()
	}

	// Create control directory for SSH multiplexing
	currentUser, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	controlDir := filepath.Join(currentUser.HomeDir, ".ssh", "s9s_control")
	if err := os.MkdirAll(controlDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create control directory: %w", err)
	}

	// Validate and resolve ssh command path
	sshPath, err := security.ValidateAndResolveCommand("ssh", "ssh")
	if err != nil {
		return nil, fmt.Errorf("failed to validate ssh command: %w", err)
	}

	// Create persistence manager
	persistence, err := NewSessionPersistence("")
	if err != nil {
		// Non-fatal, continue without persistence
		persistence = nil
	}

	sm := &SessionManager{
		sessions:       make(map[string]*SSHSession),
		config:         config,
		controlDir:     controlDir,
		cleanupDone:    make(chan struct{}),
		persistence:    persistence,
		sshCommandPath: sshPath,
	}

	// Start cleanup goroutine
	go sm.cleanupLoop()

	// Load persistent sessions
	sm.loadPersistentSessions()

	return sm, nil
}

// CreateSession creates a new SSH session
func (sm *SessionManager) CreateSession(hostname, username string) (*SSHSession, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sessionID := fmt.Sprintf("%s@%s_%d", username, hostname, time.Now().Unix())

	// Check if session already exists
	if existing, exists := sm.sessions[sessionID]; exists {
		return existing, nil
	}

	// Create control path for multiplexing
	controlPath := filepath.Join(sm.controlDir, fmt.Sprintf("%s_%s", username, hostname))

	session := &SSHSession{
		ID:           sessionID,
		Hostname:     hostname,
		Username:     username,
		State:        SessionIdle,
		StartTime:    time.Now(),
		LastActivity: time.Now(),
		ControlPath:  controlPath,
		Tunnels:      make([]SSHTunnel, 0),
	}

	sm.sessions[sessionID] = session
	return session, nil
}

// ConnectSession establishes an SSH connection
func (sm *SessionManager) ConnectSession(sessionID string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if session.State == SessionConnected || session.State == SessionActive {
		return nil // Already connected
	}

	session.State = SessionConnecting

	// Build SSH args with multiplexing
	args := sm.buildSessionArgs(session, true)

	// Test connection first
	testCmd := exec.CommandContext(context.Background(), sm.sshCommandPath, args...)
	testCmd.Args = append(testCmd.Args, "echo", "connection_test")

	if err := testCmd.Run(); err != nil {
		session.State = SessionError
		session.ErrorMessage = fmt.Sprintf("Connection failed: %v", err)
		return err
	}

	session.State = SessionConnected
	session.LastActivity = time.Now()
	return nil
}

// StartInteractiveSession starts an interactive SSH terminal session
func (sm *SessionManager) StartInteractiveSession(sessionID string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Ensure connection is established
	if session.State != SessionConnected && session.State != SessionActive {
		session.mu.Unlock()
		if err := sm.ConnectSession(sessionID); err != nil {
			session.mu.Lock()
			return err
		}
		session.mu.Lock()
	}

	// Build SSH args for interactive session
	args := sm.buildSessionArgs(session, false)

	// Create SSH command with proper terminal allocation
	cmd := exec.CommandContext(context.Background(), sm.sshCommandPath, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	session.Command = cmd
	session.State = SessionActive
	session.LastActivity = time.Now()

	// Start the command
	if err := cmd.Start(); err != nil {
		session.State = SessionError
		session.ErrorMessage = fmt.Sprintf("Failed to start interactive session: %v", err)
		return err
	}

	session.Process = cmd.Process

	// Wait for command completion in background
	go func() {
		_ = cmd.Wait()
		session.mu.Lock()
		session.State = SessionDisconnected
		session.Command = nil
		session.Process = nil
		session.mu.Unlock()
	}()

	return nil
}

// ExecuteCommand executes a command in an SSH session
func (sm *SessionManager) ExecuteCommand(sessionID, command string) (string, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Build SSH args
	args := sm.buildSessionArgs(session, false)
	args = append(args, command)

	// Execute command
	cmd := exec.CommandContext(context.Background(), sm.sshCommandPath, args...)
	output, err := cmd.CombinedOutput()

	session.LastActivity = time.Now()

	if err != nil {
		session.ErrorMessage = fmt.Sprintf("Command failed: %v", err)
		return string(output), err
	}

	return string(output), nil
}

// CreateTunnel creates an SSH tunnel for port forwarding
func (sm *SessionManager) CreateTunnel(sessionID string, localPort int, remoteHost string, remotePort int, tunnelType string) error {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Check if tunnel already exists
	for _, tunnel := range session.Tunnels {
		if tunnel.LocalPort == localPort && tunnel.RemoteHost == remoteHost && tunnel.RemotePort == remotePort {
			return fmt.Errorf("tunnel already exists")
		}
	}

	// Build tunnel args
	args := sm.buildSessionArgs(session, false)

	var tunnelArg string
	switch tunnelType {
	case "local":
		tunnelArg = fmt.Sprintf("-L%d:%s:%d", localPort, remoteHost, remotePort)
	case "remote":
		tunnelArg = fmt.Sprintf("-R%s:%d:%s:%d", session.Hostname, localPort, remoteHost, remotePort)
	default:
		return fmt.Errorf("invalid tunnel type: %s", tunnelType)
	}

	args = append(args, tunnelArg, "-N") // -N means don't execute remote command

	// Start tunnel
	cmd := exec.CommandContext(context.Background(), sm.sshCommandPath, args...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	// Add tunnel to session
	tunnel := SSHTunnel{
		LocalPort:  localPort,
		RemoteHost: remoteHost,
		RemotePort: remotePort,
		Type:       tunnelType,
		Active:     true,
	}
	session.Tunnels = append(session.Tunnels, tunnel)

	return nil
}

// GetSessions returns all active sessions
func (sm *SessionManager) GetSessions() map[string]*SSHSession {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Create a copy to avoid race conditions
	sessions := make(map[string]*SSHSession)
	for k, v := range sm.sessions {
		// Lock individual session to safely read its fields
		v.mu.RLock()
		sessions[k] = &SSHSession{
			ID:           v.ID,
			Hostname:     v.Hostname,
			Username:     v.Username,
			State:        v.State,
			StartTime:    v.StartTime,
			LastActivity: v.LastActivity,
			ErrorMessage: v.ErrorMessage,
			ControlPath:  v.ControlPath,
			Tunnels:      make([]SSHTunnel, len(v.Tunnels)),
		}
		copy(sessions[k].Tunnels, v.Tunnels)
		v.mu.RUnlock()
	}
	return sessions
}

// CloseSession closes an SSH session
func (sm *SessionManager) CloseSession(sessionID string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	// Terminate active process if any
	if session.Process != nil {
		_ = session.Process.Kill()
		session.Process = nil
	}

	// Close SSH control connection
	if session.ControlPath != "" {
		args := []string{"-O", "exit", "-S", session.ControlPath, session.Hostname}
		_ = exec.CommandContext(context.Background(), sm.sshCommandPath, args...).Run() // Ignore errors
	}

	session.State = SessionDisconnected
	delete(sm.sessions, sessionID)

	return nil
}

// CloseAllSessions closes all active sessions
func (sm *SessionManager) CloseAllSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for sessionID := range sm.sessions {
		session := sm.sessions[sessionID]
		session.mu.Lock()

		if session.Process != nil {
			_ = session.Process.Kill()
		}

		if session.ControlPath != "" {
			args := []string{"-O", "exit", "-S", session.ControlPath, session.Hostname}
			_ = exec.CommandContext(context.Background(), sm.sshCommandPath, args...).Run()
		}

		session.mu.Unlock()
	}

	sm.sessions = make(map[string]*SSHSession)
}

// GetSessionInfo returns detailed information about a session
func (sm *SessionManager) GetSessionInfo(sessionID string) (*SSHSession, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}

// buildSessionArgs builds SSH command arguments for a session
func (sm *SessionManager) buildSessionArgs(session *SSHSession, enableMultiplexing bool) []string {
	var args []string

	// Add port if specified
	if sm.config.Port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", sm.config.Port))
	}

	// Add key file if specified
	if sm.config.KeyFile != "" {
		args = append(args, "-i", sm.config.KeyFile)
	}

	// Add config file if specified
	if sm.config.ConfigFile != "" {
		args = append(args, "-F", sm.config.ConfigFile)
	}

	// Add multiplexing options
	if enableMultiplexing {
		args = append(args, "-M")                       // Master mode
		args = append(args, "-S", session.ControlPath)  // Control socket
		args = append(args, "-o", "ControlPersist=600") // Keep connection for 10 minutes
	} else if session.ControlPath != "" {
		args = append(args, "-S", session.ControlPath) // Use existing control socket
	}

	// Add SSH options
	for key, value := range sm.config.Options {
		args = append(args, "-o", fmt.Sprintf("%s=%s", key, value))
	}

	// Add connection timeout
	if sm.config.Timeout > 0 {
		timeoutSecs := int(sm.config.Timeout.Seconds())
		args = append(args, "-o", fmt.Sprintf("ConnectTimeout=%d", timeoutSecs))
	}

	// Add username@hostname
	if session.Username != "" {
		args = append(args, fmt.Sprintf("%s@%s", session.Username, session.Hostname))
	} else {
		args = append(args, session.Hostname)
	}

	return args
}

// cleanupLoop periodically cleans up stale sessions and saves state
func (sm *SessionManager) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.cleanupStaleSessions()
			// Save sessions periodically
			sm.savePersistentSessions()
		case <-sm.cleanupDone:
			return
		}
	}
}

// cleanupStaleSessions removes stale sessions
func (sm *SessionManager) cleanupStaleSessions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	staleThreshold := time.Now().Add(-30 * time.Minute)

	for sessionID, session := range sm.sessions {
		session.mu.RLock()
		if session.State == SessionDisconnected ||
			session.State == SessionError ||
			(session.State == SessionIdle && session.LastActivity.Before(staleThreshold)) {
			session.mu.RUnlock()

			// Clean up control socket
			if session.ControlPath != "" {
				_ = os.Remove(session.ControlPath)
			}

			delete(sm.sessions, sessionID)
		} else {
			session.mu.RUnlock()
		}
	}
}

// Shutdown gracefully shuts down the session manager
func (sm *SessionManager) Shutdown() {
	sm.shutdownOnce.Do(func() {
		close(sm.cleanupDone)

		// Save sessions before shutdown
		sm.savePersistentSessions()

		sm.CloseAllSessions()

		// Clean up control directory
		if sm.controlDir != "" {
			_ = os.RemoveAll(sm.controlDir)
		}
	})
}

// TestNodeConnectivity tests SSH connectivity to multiple nodes
func (sm *SessionManager) TestNodeConnectivity(hostnames []string, timeout time.Duration) map[string]error {
	results := make(map[string]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, hostname := range hostnames {
		wg.Add(1)
		go func(host string) {
			defer wg.Done()

			// Create temporary session for testing
			session, err := sm.CreateSession(host, sm.config.Username)
			if err != nil {
				mu.Lock()
				results[host] = err
				mu.Unlock()
				return
			}

			// Test connection
			_, err = sm.ExecuteCommand(session.ID, "echo 'test'")

			mu.Lock()
			results[host] = err
			mu.Unlock()

			// Clean up test session
			_ = sm.CloseSession(session.ID)
		}(hostname)
	}

	wg.Wait()
	return results
}

// GetNodeSystemInfo retrieves comprehensive system information from a node
func (sm *SessionManager) GetNodeSystemInfo(sessionID string) (map[string]string, error) {
	info := make(map[string]string)

	commands := map[string]string{
		"hostname":     "hostname -f",
		"uptime":       "uptime",
		"load_avg":     "cat /proc/loadavg",
		"memory":       "free -h",
		"cpu_info":     "lscpu | head -20",
		"disk_usage":   "df -h",
		"network":      "ip -o link show",
		"processes":    "ps aux --sort=-%cpu | head -10",
		"kernel":       "uname -a",
		"distribution": "lsb_release -a 2>/dev/null || cat /etc/os-release",
	}

	for key, command := range commands {
		if output, err := sm.ExecuteCommand(sessionID, command); err == nil {
			info[key] = strings.TrimSpace(output)
		} else {
			info[key] = fmt.Sprintf("Error: %v", err)
		}
	}

	return info, nil
}

// MonitorSession continuously monitors a session and reports status
func (sm *SessionManager) MonitorSession(sessionID string, interval time.Duration) (<-chan SessionState, error) {
	sm.mu.RLock()
	session, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	statusChan := make(chan SessionState, 10)

	go func() {
		defer close(statusChan)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			session.mu.RLock()
			currentState := session.State
			session.mu.RUnlock()

			select {
			case statusChan <- currentState:
			default:
				// Channel full, skip this update
			}

			// Stop monitoring if session is disconnected or error
			if currentState == SessionDisconnected || currentState == SessionError {
				return
			}
		}
	}()

	return statusChan, nil
}

// loadPersistentSessions loads previously saved sessions
func (sm *SessionManager) loadPersistentSessions() {
	if sm.persistence == nil {
		return
	}

	sessions, err := sm.persistence.LoadSessions()
	if err != nil {
		// Log error but continue
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, ps := range sessions {
		// Check if control socket still exists
		if _, err := os.Stat(ps.ControlPath); err == nil {
			// Try to verify the connection is still alive
			args := []string{"-O", "check", "-S", ps.ControlPath, ps.Hostname}
			if err := exec.CommandContext(context.Background(), sm.sshCommandPath, args...).Run(); err == nil {
				// Connection is still alive, restore session
				session := &SSHSession{
					ID:           ps.ID,
					Hostname:     ps.Hostname,
					Username:     ps.Username,
					State:        SessionConnected,
					StartTime:    ps.LastActivity, // Use last activity as start time
					LastActivity: time.Now(),
					ControlPath:  ps.ControlPath,
					Tunnels:      ps.Tunnels,
				}
				sm.sessions[ps.ID] = session
			}
		}
	}
}

// savePersistentSessions saves current sessions to disk
func (sm *SessionManager) savePersistentSessions() {
	if sm.persistence == nil {
		return
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Save sessions
	if err := sm.persistence.SaveSessions(sm.sessions); err != nil {
		// Log error but continue
		return
	}

	// Cleanup old data
	_ = sm.persistence.CleanupOldData()
}

// EnablePersistence enables session persistence with a custom data directory
func (sm *SessionManager) EnablePersistence(dataDir string) error {
	persistence, err := NewSessionPersistence(dataDir)
	if err != nil {
		return err
	}

	sm.mu.Lock()
	sm.persistence = persistence
	sm.mu.Unlock()

	// Load any existing sessions
	sm.loadPersistentSessions()

	return nil
}

// SaveSessionTags saves user-defined tags for a session
func (sm *SessionManager) SaveSessionTags(sessionID string, tags map[string]string) error {
	if sm.persistence == nil {
		return fmt.Errorf("persistence not enabled")
	}

	sm.mu.RLock()
	_, exists := sm.sessions[sessionID]
	sm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	return sm.persistence.SaveSessionTags(sessionID, tags)
}

// GetSessionTags gets user-defined tags for a session
func (sm *SessionManager) GetSessionTags(sessionID string) (map[string]string, error) {
	if sm.persistence == nil {
		return nil, fmt.Errorf("persistence not enabled")
	}

	return sm.persistence.LoadSessionTags(sessionID)
}
