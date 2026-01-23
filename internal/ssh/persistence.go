package ssh

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PersistentSession represents a session that can be saved/restored
type PersistentSession struct {
	ID           string            `json:"id"`
	Hostname     string            `json:"hostname"`
	Username     string            `json:"username"`
	LastActivity time.Time         `json:"last_activity"`
	ControlPath  string            `json:"control_path"`
	Tunnels      []SSHTunnel       `json:"tunnels"`
	Tags         map[string]string `json:"tags"` // User-defined tags
}

// SessionPersistence handles saving and loading SSH sessions
type SessionPersistence struct {
	dataDir string
}

// NewSessionPersistence creates a new session persistence manager
func NewSessionPersistence(dataDir string) (*SessionPersistence, error) {
	if dataDir == "" {
		// Use default location in user's config directory
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dataDir = filepath.Join(homeDir, ".config", "s9s", "ssh_sessions")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &SessionPersistence{
		dataDir: dataDir,
	}, nil
}

// SaveSessions saves all active sessions to disk
func (sp *SessionPersistence) SaveSessions(sessions map[string]*SSHSession) error {
	// Convert to persistent format
	persistentSessions := make([]PersistentSession, 0, len(sessions))

	for _, session := range sessions {
		session.mu.RLock()
		ps := PersistentSession{
			ID:           session.ID,
			Hostname:     session.Hostname,
			Username:     session.Username,
			LastActivity: session.LastActivity,
			ControlPath:  session.ControlPath,
			Tunnels:      make([]SSHTunnel, len(session.Tunnels)),
		}
		copy(ps.Tunnels, session.Tunnels)
		session.mu.RUnlock()

		// Only save sessions that are connected or active
		if session.State == SessionConnected || session.State == SessionActive {
			persistentSessions = append(persistentSessions, ps)
		}
	}

	// Save to file
	filePath := filepath.Join(sp.dataDir, "sessions.json")
	data, err := json.MarshalIndent(persistentSessions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sessions: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write sessions file: %w", err)
	}

	return nil
}

// LoadSessions loads previously saved sessions from disk
func (sp *SessionPersistence) LoadSessions() ([]PersistentSession, error) {
	filePath := filepath.Join(sp.dataDir, "sessions.json")

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// No saved sessions
		return []PersistentSession{}, nil
	}

	// nolint:gosec // G304: filePath is application-controlled (dataDir + "sessions.json"), not user input
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read sessions file: %w", err)
	}

	var sessions []PersistentSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sessions: %w", err)
	}

	// Filter out stale sessions (older than 24 hours)
	validSessions := make([]PersistentSession, 0, len(sessions))
	cutoff := time.Now().Add(-24 * time.Hour)

	for _, session := range sessions {
		if session.LastActivity.After(cutoff) {
			validSessions = append(validSessions, session)
		}
	}

	return validSessions, nil
}

// SaveSessionTags saves user-defined tags for a session
func (sp *SessionPersistence) SaveSessionTags(sessionID string, tags map[string]string) error {
	filePath := filepath.Join(sp.dataDir, fmt.Sprintf("tags_%s.json", sessionID))

	data, err := json.MarshalIndent(tags, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write tags file: %w", err)
	}

	return nil
}

// LoadSessionTags loads user-defined tags for a session
func (sp *SessionPersistence) LoadSessionTags(sessionID string) (map[string]string, error) {
	filePath := filepath.Join(sp.dataDir, fmt.Sprintf("tags_%s.json", sessionID))

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// No saved tags
		return make(map[string]string), nil
	}

	// nolint:gosec // G304: filePath is application-controlled (dataDir + "tags.json"), not user input
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read tags file: %w", err)
	}

	var tags map[string]string
	if err := json.Unmarshal(data, &tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %w", err)
	}

	return tags, nil
}

// CleanupOldData removes old session data files
func (sp *SessionPersistence) CleanupOldData() error {
	entries, err := os.ReadDir(sp.dataDir)
	if err != nil {
		return fmt.Errorf("failed to read data directory: %w", err)
	}

	cutoff := time.Now().Add(-7 * 24 * time.Hour) // 7 days old

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(sp.dataDir, entry.Name())
			_ = os.Remove(filePath) // Ignore errors
		}
	}

	return nil
}
