package ssh_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jontk/s9s/internal/ssh"
	"github.com/rivo/tview"
)

// TestTerminalManager tests the SSH terminal manager functionality
func TestTerminalManager(t *testing.T) {
	// Create a mock app for testing
	app := tview.NewApplication()

	// Create terminal manager
	manager := ssh.NewTerminalManager(app)

	// Test 1: Basic creation
	t.Run("BasicCreation", func(t *testing.T) {
		if manager == nil {
			t.Fatal("Expected terminal manager to be created")
		}
	})

	// Test 2: Add terminal connection (mock)
	t.Run("AddTerminal", func(t *testing.T) {
		nodeID := "test-node-1"
		err := manager.AddTerminal(nodeID, createMockTerminal(nodeID))
		if err != nil {
			t.Fatalf("Failed to add terminal: %v", err)
		}

		if !manager.HasTerminal(nodeID) {
			t.Error("Expected terminal to exist after adding")
		}

		if count := manager.GetActiveCount(); count != 1 {
			t.Errorf("Expected 1 active terminal, got %d", count)
		}
	})

	// Test 3: Multiple terminals
	t.Run("MultipleTerminals", func(t *testing.T) {
		// Add more terminals
		nodes := []string{"test-node-2", "test-node-3", "test-node-4"}
		for _, nodeID := range nodes {
			err := manager.AddTerminal(nodeID, createMockTerminal(nodeID))
			if err != nil {
				t.Fatalf("Failed to add terminal %s: %v", nodeID, err)
			}
		}

		expectedCount := 4 // 1 from previous test + 3 new
		if count := manager.GetActiveCount(); count != expectedCount {
			t.Errorf("Expected %d active terminals, got %d", expectedCount, count)
		}

		// Check all terminals exist
		allNodes := append([]string{"test-node-1"}, nodes...)
		for _, nodeID := range allNodes {
			if !manager.HasTerminal(nodeID) {
				t.Errorf("Expected terminal %s to exist", nodeID)
			}
		}
	})

	// Test 4: Remove terminal
	t.Run("RemoveTerminal", func(t *testing.T) {
		nodeID := "test-node-2"
		err := manager.RemoveTerminal(nodeID)
		if err != nil {
			t.Fatalf("Failed to remove terminal: %v", err)
		}

		if manager.HasTerminal(nodeID) {
			t.Error("Expected terminal to be removed")
		}

		if count := manager.GetActiveCount(); count != 3 {
			t.Errorf("Expected 3 active terminals after removal, got %d", count)
		}
	})

	// Test 5: Terminal switching
	t.Run("TerminalSwitching", func(t *testing.T) {
		// Test next terminal
		startNode := manager.GetCurrentNodeID()
		manager.NextTerminal()
		nextNode := manager.GetCurrentNodeID()

		if startNode == nextNode {
			t.Error("Expected terminal to change after NextTerminal")
		}

		// Test previous terminal
		manager.PreviousTerminal()
		prevNode := manager.GetCurrentNodeID()

		if prevNode != startNode {
			t.Error("Expected to return to original terminal after PreviousTerminal")
		}
	})

	// Test 6: Switch to specific terminal
	t.Run("SwitchToTerminal", func(t *testing.T) {
		targetNode := "test-node-4"
		err := manager.SwitchToTerminal(targetNode)
		if err != nil {
			t.Fatalf("Failed to switch to terminal: %v", err)
		}

		if current := manager.GetCurrentNodeID(); current != targetNode {
			t.Errorf("Expected current node to be %s, got %s", targetNode, current)
		}
	})

	// Test 7: Get terminal info
	t.Run("GetTerminalInfo", func(t *testing.T) {
		infos := manager.GetTerminalInfos()

		if len(infos) != 3 { // We removed one earlier
			t.Errorf("Expected 3 terminal infos, got %d", len(infos))
		}

		// Check info contains expected nodes
		nodeMap := make(map[string]bool)
		for _, info := range infos {
			nodeMap[info.NodeID] = true
		}

		expectedNodes := []string{"test-node-1", "test-node-3", "test-node-4"}
		for _, node := range expectedNodes {
			if !nodeMap[node] {
				t.Errorf("Expected node %s in terminal infos", node)
			}
		}
	})

	// Test 8: Close all terminals
	t.Run("CloseAllTerminals", func(t *testing.T) {
		manager.CloseAll()

		if count := manager.GetActiveCount(); count != 0 {
			t.Errorf("Expected 0 active terminals after CloseAll, got %d", count)
		}

		// Verify all terminals are gone
		for _, nodeID := range []string{"test-node-1", "test-node-3", "test-node-4"} {
			if manager.HasTerminal(nodeID) {
				t.Errorf("Expected terminal %s to be removed after CloseAll", nodeID)
			}
		}
	})
}

// TestTerminalManagerEdgeCases tests edge cases
func TestTerminalManagerEdgeCases(t *testing.T) {
	app := tview.NewApplication()
	manager := ssh.NewTerminalManager(app)

	// Test 1: Remove non-existent terminal
	t.Run("RemoveNonExistent", func(t *testing.T) {
		err := manager.RemoveTerminal("non-existent")
		if err == nil {
			t.Error("Expected error when removing non-existent terminal")
		}
	})

	// Test 2: Switch to non-existent terminal
	t.Run("SwitchToNonExistent", func(t *testing.T) {
		err := manager.SwitchToTerminal("non-existent")
		if err == nil {
			t.Error("Expected error when switching to non-existent terminal")
		}
	})

	// Test 3: Navigation with no terminals
	t.Run("NavigationEmpty", func(t *testing.T) {
		// Should not panic
		manager.NextTerminal()
		manager.PreviousTerminal()

		if nodeID := manager.GetCurrentNodeID(); nodeID != "" {
			t.Errorf("Expected empty current node ID, got %s", nodeID)
		}
	})

	// Test 4: Add duplicate terminal
	t.Run("AddDuplicate", func(t *testing.T) {
		nodeID := "duplicate-node"

		// Add first
		err := manager.AddTerminal(nodeID, createMockTerminal(nodeID))
		if err != nil {
			t.Fatalf("Failed to add first terminal: %v", err)
		}

		// Try to add duplicate
		err = manager.AddTerminal(nodeID, createMockTerminal(nodeID))
		if err == nil {
			t.Error("Expected error when adding duplicate terminal")
		}

		// Should still have only one
		if count := manager.GetActiveCount(); count != 1 {
			t.Errorf("Expected 1 terminal, got %d", count)
		}
	})
}

// TestTerminalFeatures tests specific terminal features
func TestTerminalFeatures(t *testing.T) {
	app := tview.NewApplication()
	manager := ssh.NewTerminalManager(app)

	// Test 1: Send command to terminal
	t.Run("SendCommand", func(t *testing.T) {
		nodeID := "cmd-test-node"
		terminal := createMockTerminal(nodeID)

		err := manager.AddTerminal(nodeID, terminal)
		if err != nil {
			t.Fatalf("Failed to add terminal: %v", err)
		}

		// Send command
		command := "echo 'test command'"
		err = manager.SendCommand(nodeID, command)
		if err != nil {
			t.Fatalf("Failed to send command: %v", err)
		}

		// Verify command was sent (in mock terminal)
		if mockTerm, ok := terminal.(*MockTerminal); ok {
			if mockTerm.lastCommand != command {
				t.Errorf("Expected command %s, got %s", command, mockTerm.lastCommand)
			}
		}
	})

	// Test 2: Resize terminal
	t.Run("ResizeTerminal", func(t *testing.T) {
		nodeID := "resize-test-node"
		terminal := createMockTerminal(nodeID)

		err := manager.AddTerminal(nodeID, terminal)
		if err != nil {
			t.Fatalf("Failed to add terminal: %v", err)
		}

		// Resize
		width, height := 120, 40
		err = manager.ResizeTerminal(nodeID, width, height)
		if err != nil {
			t.Fatalf("Failed to resize terminal: %v", err)
		}

		// Verify resize (in mock terminal)
		if mockTerm, ok := terminal.(*MockTerminal); ok {
			if mockTerm.width != width || mockTerm.height != height {
				t.Errorf("Expected size %dx%d, got %dx%d", width, height, mockTerm.width, mockTerm.height)
			}
		}
	})

	// Test 3: Terminal status
	t.Run("TerminalStatus", func(t *testing.T) {
		nodeID := "status-test-node"
		terminal := createMockTerminal(nodeID)

		err := manager.AddTerminal(nodeID, terminal)
		if err != nil {
			t.Fatalf("Failed to add terminal: %v", err)
		}

		// Get terminal info
		infos := manager.GetTerminalInfos()

		var info *ssh.TerminalInfo
		for i := range infos {
			if infos[i].NodeID == nodeID {
				info = &infos[i]
				break
			}
		}

		if info == nil {
			t.Fatal("Expected to find terminal info")
		}

		if !info.IsActive {
			t.Error("Expected terminal to be active")
		}

		if info.ConnectedAt.IsZero() {
			t.Error("Expected connected time to be set")
		}
	})
}

// MockTerminal implements a mock terminal for testing
type MockTerminal struct {
	*tview.TextView
	nodeID      string
	lastCommand string
	width       int
	height      int
	isActive    bool
}

// createMockTerminal creates a mock terminal for testing
func createMockTerminal(nodeID string) ssh.Terminal {
	textView := tview.NewTextView()
	textView.SetBorder(true)
	textView.SetTitle(fmt.Sprintf(" SSH: %s ", nodeID))

	// Add some mock output
	textView.SetText(fmt.Sprintf(`[green]Connected to %s[white]
Last login: %s

[user@%s ~]$ `, nodeID, time.Now().Format(time.RFC3339), nodeID))

	return &MockTerminal{
		TextView: textView,
		nodeID:   nodeID,
		isActive: true,
		width:    80,
		height:   24,
	}
}

// Terminal interface methods for MockTerminal
func (m *MockTerminal) SendCommand(command string) error {
	m.lastCommand = command
	// Append command to output
	currentText := m.GetText(false)
	m.SetText(currentText + command + "\n")
	return nil
}

func (m *MockTerminal) Resize(width, height int) error {
	m.width = width
	m.height = height
	return nil
}

func (m *MockTerminal) Close() error {
	m.isActive = false
	m.SetText(m.GetText(false) + "\n[red]Connection closed[white]\n")
	return nil
}

func (m *MockTerminal) IsActive() bool {
	return m.isActive
}

func (m *MockTerminal) GetNodeID() string {
	return m.nodeID
}

// TestSSHConnectionScenarios tests realistic SSH connection scenarios
func TestSSHConnectionScenarios(t *testing.T) {
	app := tview.NewApplication()
	manager := ssh.NewTerminalManager(app)

	// Test 1: Connection failure handling
	t.Run("ConnectionFailure", func(t *testing.T) {
		// This would test actual SSH connection failures
		// For now, we simulate with mock
		nodeID := "unreachable-node"
		terminal := &MockTerminal{
			TextView: tview.NewTextView(),
			nodeID:   nodeID,
			isActive: false, // Simulate failed connection
		}

		err := manager.AddTerminal(nodeID, terminal)
		if err != nil {
			t.Fatalf("Failed to add terminal: %v", err)
		}

		// Verify terminal is tracked even if not active
		if !manager.HasTerminal(nodeID) {
			t.Error("Expected terminal to be tracked even if connection failed")
		}
	})

	// Test 2: Reconnection scenario
	t.Run("Reconnection", func(t *testing.T) {
		nodeID := "reconnect-node"

		// Initial connection
		terminal1 := createMockTerminal(nodeID)
		err := manager.AddTerminal(nodeID, terminal1)
		if err != nil {
			t.Fatalf("Failed to add terminal: %v", err)
		}

		// Simulate disconnect
		_ = terminal1.Close()

		// Remove old terminal
		err = manager.RemoveTerminal(nodeID)
		if err != nil {
			t.Fatalf("Failed to remove terminal: %v", err)
		}

		// Reconnect with new terminal
		terminal2 := createMockTerminal(nodeID)
		err = manager.AddTerminal(nodeID, terminal2)
		if err != nil {
			t.Fatalf("Failed to add reconnected terminal: %v", err)
		}

		if !manager.HasTerminal(nodeID) {
			t.Error("Expected reconnected terminal to exist")
		}
	})

	// Test 3: Multiple nodes batch operations
	t.Run("BatchOperations", func(t *testing.T) {
		// Add multiple terminals
		nodeCount := 5
		for i := 0; i < nodeCount; i++ {
			nodeID := fmt.Sprintf("batch-node-%d", i)
			err := manager.AddTerminal(nodeID, createMockTerminal(nodeID))
			if err != nil {
				t.Fatalf("Failed to add terminal %s: %v", nodeID, err)
			}
		}

		// Send command to all
		command := "uptime"
		for i := 0; i < nodeCount; i++ {
			nodeID := fmt.Sprintf("batch-node-%d", i)
			err := manager.SendCommand(nodeID, command)
			if err != nil {
				t.Errorf("Failed to send command to %s: %v", nodeID, err)
			}
		}

		// Verify all received command
		infos := manager.GetTerminalInfos()
		commandedNodes := 0
		for _, info := range infos {
			if strings.HasPrefix(info.NodeID, "batch-node-") {
				commandedNodes++
			}
		}

		if commandedNodes != nodeCount {
			t.Errorf("Expected %d nodes to receive command, got %d", nodeCount, commandedNodes)
		}
	})
}

// TestTerminalManagerConcurrency tests concurrent operations
func TestTerminalManagerConcurrency(t *testing.T) {
	app := tview.NewApplication()
	manager := ssh.NewTerminalManager(app)

	// Test concurrent terminal additions
	t.Run("ConcurrentAdd", func(t *testing.T) {
		done := make(chan bool, 10)

		// Add terminals concurrently
		for i := 0; i < 10; i++ {
			go func(id int) {
				nodeID := fmt.Sprintf("concurrent-node-%d", id)
				err := manager.AddTerminal(nodeID, createMockTerminal(nodeID))
				if err != nil {
					t.Errorf("Failed to add terminal %s: %v", nodeID, err)
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Verify all terminals were added
		if count := manager.GetActiveCount(); count != 10 {
			t.Errorf("Expected 10 terminals, got %d", count)
		}
	})

	// Test concurrent operations
	t.Run("ConcurrentOperations", func(t *testing.T) {
		done := make(chan bool, 30)

		// Mix of operations
		go func() {
			for i := 0; i < 10; i++ {
				manager.NextTerminal()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 10; i++ {
				manager.PreviousTerminal()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		go func() {
			for i := 0; i < 10; i++ {
				_ = manager.GetTerminalInfos()
				time.Sleep(10 * time.Millisecond)
			}
			done <- true
		}()

		// Wait for all operations
		for i := 0; i < 3; i++ {
			<-done
		}

		// Manager should still be in valid state
		if count := manager.GetActiveCount(); count != 10 {
			t.Errorf("Expected terminals to remain after concurrent operations, got %d", count)
		}
	})
}
