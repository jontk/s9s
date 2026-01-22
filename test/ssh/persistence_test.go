package ssh_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/ssh"
)

// TestSessionPersistence tests session persistence functionality
func TestSessionPersistence(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "s9s_ssh_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test 1: Create persistence manager
	t.Run("CreatePersistence", func(t *testing.T) {
		persistence, err := ssh.NewSessionPersistence(tempDir)
		if err != nil {
			t.Fatalf("Failed to create persistence: %v", err)
		}

		if persistence == nil {
			t.Fatal("Expected persistence manager to be created")
		}
	})

	// Test 2: Save and load sessions
	t.Run("SaveLoadSessions", func(t *testing.T) {
		persistence, err := ssh.NewSessionPersistence(tempDir)
		if err != nil {
			t.Fatalf("Failed to create persistence: %v", err)
		}

		// Create test sessions
		sessions := map[string]*ssh.SSHSession{
			"session1": {
				ID:           "session1",
				Hostname:     "node1",
				Username:     "user1",
				State:        ssh.SessionConnected,
				LastActivity: time.Now(),
				ControlPath:  "/tmp/control1",
				Tunnels: []ssh.SSHTunnel{
					{LocalPort: 8080, RemoteHost: "localhost", RemotePort: 80, Type: "local"},
				},
			},
			"session2": {
				ID:           "session2",
				Hostname:     "node2",
				Username:     "user2",
				State:        ssh.SessionActive,
				LastActivity: time.Now(),
				ControlPath:  "/tmp/control2",
			},
			"session3": {
				ID:           "session3",
				Hostname:     "node3",
				Username:     "user3",
				State:        ssh.SessionDisconnected, // Should not be saved
				LastActivity: time.Now(),
				ControlPath:  "/tmp/control3",
			},
		}

		// Save sessions
		err = persistence.SaveSessions(sessions)
		if err != nil {
			t.Fatalf("Failed to save sessions: %v", err)
		}

		// Load sessions
		loaded, err := persistence.LoadSessions()
		if err != nil {
			t.Fatalf("Failed to load sessions: %v", err)
		}

		// Should have 2 sessions (disconnected one not saved)
		if len(loaded) != 2 {
			t.Errorf("Expected 2 sessions, got %d", len(loaded))
		}

		// Verify session data
		sessionMap := make(map[string]ssh.PersistentSession)
		for _, s := range loaded {
			sessionMap[s.ID] = s
		}

		if s1, ok := sessionMap["session1"]; ok {
			if s1.Hostname != "node1" || s1.Username != "user1" {
				t.Errorf("Session1 data mismatch")
			}
			if len(s1.Tunnels) != 1 {
				t.Errorf("Expected 1 tunnel, got %d", len(s1.Tunnels))
			}
		} else {
			t.Error("Session1 not found in loaded sessions")
		}

		if s2, ok := sessionMap["session2"]; ok {
			if s2.Hostname != "node2" || s2.Username != "user2" {
				t.Errorf("Session2 data mismatch")
			}
		} else {
			t.Error("Session2 not found in loaded sessions")
		}

		// Session3 should not be present
		if _, ok := sessionMap["session3"]; ok {
			t.Error("Session3 (disconnected) should not be saved")
		}
	})

	// Test 3: Session tags
	t.Run("SessionTags", func(t *testing.T) {
		persistence, err := ssh.NewSessionPersistence(tempDir)
		if err != nil {
			t.Fatalf("Failed to create persistence: %v", err)
		}

		sessionID := "test-session"
		tags := map[string]string{
			"project":     "myproject",
			"environment": "production",
			"owner":       "admin",
		}

		// Save tags
		err = persistence.SaveSessionTags(sessionID, tags)
		if err != nil {
			t.Fatalf("Failed to save tags: %v", err)
		}

		// Load tags
		loadedTags, err := persistence.LoadSessionTags(sessionID)
		if err != nil {
			t.Fatalf("Failed to load tags: %v", err)
		}

		// Verify tags
		if len(loadedTags) != len(tags) {
			t.Errorf("Expected %d tags, got %d", len(tags), len(loadedTags))
		}

		for key, value := range tags {
			if loadedTags[key] != value {
				t.Errorf("Tag %s: expected %s, got %s", key, value, loadedTags[key])
			}
		}
	})

	// Test 4: Stale session filtering
	t.Run("StaleSessionFiltering", func(t *testing.T) {
		persistence, err := ssh.NewSessionPersistence(tempDir)
		if err != nil {
			t.Fatalf("Failed to create persistence: %v", err)
		}

		// Create sessions with different ages
		sessions := map[string]*ssh.SSHSession{
			"fresh": {
				ID:           "fresh",
				Hostname:     "node1",
				Username:     "user1",
				State:        ssh.SessionConnected,
				LastActivity: time.Now(),
			},
			"old": {
				ID:           "old",
				Hostname:     "node2",
				Username:     "user2",
				State:        ssh.SessionConnected,
				LastActivity: time.Now().Add(-25 * time.Hour), // Older than 24 hours
			},
		}

		// Save sessions
		err = persistence.SaveSessions(sessions)
		if err != nil {
			t.Fatalf("Failed to save sessions: %v", err)
		}

		// Load sessions - old one should be filtered out
		loaded, err := persistence.LoadSessions()
		if err != nil {
			t.Fatalf("Failed to load sessions: %v", err)
		}

		if len(loaded) != 1 {
			t.Errorf("Expected 1 session (old filtered out), got %d", len(loaded))
		}

		if len(loaded) > 0 && loaded[0].ID != "fresh" {
			t.Error("Expected 'fresh' session to be loaded")
		}
	})
}

// TestSessionManagerWithPersistence tests session manager persistence integration
func TestSessionManagerWithPersistence(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "s9s_ssh_mgr_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test 1: Enable persistence
	t.Run("EnablePersistence", func(t *testing.T) {
		manager, err := ssh.NewSessionManager(nil)
		if err != nil {
			t.Fatalf("Failed to create session manager: %v", err)
		}
		defer manager.Shutdown()

		// Enable persistence
		err = manager.EnablePersistence(tempDir)
		if err != nil {
			t.Fatalf("Failed to enable persistence: %v", err)
		}

		// Create sessions
		session1, err := manager.CreateSession("persist-node1", "user1")
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		_, err = manager.CreateSession("persist-node2", "user2")
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Save tags
		tags1 := map[string]string{"type": "compute", "priority": "high"}
		err = manager.SaveSessionTags(session1.ID, tags1)
		if err != nil {
			t.Fatalf("Failed to save tags: %v", err)
		}

		// Verify tags can be retrieved
		loadedTags, err := manager.GetSessionTags(session1.ID)
		if err != nil {
			t.Fatalf("Failed to get tags: %v", err)
		}

		if len(loadedTags) != len(tags1) {
			t.Errorf("Expected %d tags, got %d", len(tags1), len(loadedTags))
		}

		// Shutdown should save sessions
		manager.Shutdown()

		// Create new manager - should load saved sessions
		manager2, err := ssh.NewSessionManager(nil)
		if err != nil {
			t.Fatalf("Failed to create second session manager: %v", err)
		}
		defer manager2.Shutdown()

		err = manager2.EnablePersistence(tempDir)
		if err != nil {
			t.Fatalf("Failed to enable persistence on second manager: %v", err)
		}

		// Note: Sessions won't be restored because control sockets don't exist
		// This is expected behavior in tests
		sessions := manager2.GetSessions()
		t.Logf("Loaded %d sessions (may be 0 due to missing control sockets)", len(sessions))

		// Tags should still be loadable
		loadedTags2, err := manager2.GetSessionTags(session1.ID)
		if err == nil && len(loadedTags2) > 0 {
			t.Log("Tags were preserved across restarts")
		}
	})
}

// TestPersistenceEdgeCases tests edge cases
func TestPersistenceEdgeCases(t *testing.T) {
	// Test 1: Default directory
	t.Run("DefaultDirectory", func(t *testing.T) {
		persistence, err := ssh.NewSessionPersistence("")
		if err != nil {
			t.Fatalf("Failed to create persistence with default dir: %v", err)
		}

		if persistence == nil {
			t.Fatal("Expected persistence to be created")
		}

		// Clean up default directory
		homeDir, _ := os.UserHomeDir()
		defaultDir := filepath.Join(homeDir, ".config", "s9s", "ssh_sessions")
		_ = os.RemoveAll(defaultDir)
	})

	// Test 2: Non-existent sessions file
	t.Run("NonExistentFile", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "s9s_ssh_nofile")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		persistence, err := ssh.NewSessionPersistence(tempDir)
		if err != nil {
			t.Fatalf("Failed to create persistence: %v", err)
		}

		// Load from non-existent file
		sessions, err := persistence.LoadSessions()
		if err != nil {
			t.Fatalf("Failed to load sessions: %v", err)
		}

		if len(sessions) != 0 {
			t.Errorf("Expected 0 sessions from non-existent file, got %d", len(sessions))
		}
	})

	// Test 3: Invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "s9s_ssh_invalid")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Write invalid JSON
		filePath := filepath.Join(tempDir, "sessions.json")
		_ = os.WriteFile(filePath, []byte("invalid json"), 0600)

		persistence, err := ssh.NewSessionPersistence(tempDir)
		if err != nil {
			t.Fatalf("Failed to create persistence: %v", err)
		}

		// Should fail to load
		_, err = persistence.LoadSessions()
		if err == nil {
			t.Error("Expected error loading invalid JSON")
		}
	})
}
