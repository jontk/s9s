package ssh

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/security"
	"github.com/rivo/tview"
)

type SSHTerminal struct {
	*tview.TextView
	nodeID   string
	hostname string
	username string
	// TODO(lint): Review unused code - field sessionID is unused
	// sessionID     string
	cmd            *exec.Cmd
	stdin          io.WriteCloser
	stdout         io.ReadCloser
	stderr         io.ReadCloser
	isActive       bool
	mu             sync.RWMutex
	onClose        func()
	buffer         []string
	maxBufferSize  int
	lastActivity   time.Time
	connectedAt    time.Time
	sshCommandPath string // Validated absolute path to ssh command
}

// NewSSHTerminal creates a new SSH terminal
func NewSSHTerminal(nodeID, hostname, username string, config *SSHConfig) (*SSHTerminal, error) {
	// Create text view for terminal display
	textView := tview.NewTextView()
	textView.SetDynamicColors(true)
	textView.SetScrollable(true)
	textView.SetWrap(true)
	textView.SetBorder(true)
	textView.SetTitle(fmt.Sprintf(" SSH: %s@%s ", username, hostname))
	textView.SetTitleAlign(tview.AlignLeft)

	// Set dark background for terminal feel
	textView.SetBackgroundColor(tcell.ColorBlack)
	textView.SetTextColor(tcell.ColorWhite)

	// Validate and resolve ssh command path
	sshPath, err := security.ValidateAndResolveCommand("ssh", "ssh")
	if err != nil {
		return nil, fmt.Errorf("failed to validate ssh command: %w", err)
	}

	terminal := &SSHTerminal{
		TextView:       textView,
		nodeID:         nodeID,
		hostname:       hostname,
		username:       username,
		maxBufferSize:  10000, // Max lines to keep in buffer
		buffer:         make([]string, 0, 10000),
		connectedAt:    time.Now(),
		lastActivity:   time.Now(),
		sshCommandPath: sshPath,
	}

	// Build SSH command
	args := buildSSHArgs(config, hostname, username)
	terminal.cmd = exec.CommandContext(context.Background(), terminal.sshCommandPath, args...)

	// Create pipes
	stdin, err := terminal.cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	terminal.stdin = stdin

	stdout, err := terminal.cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	terminal.stdout = stdout

	stderr, err := terminal.cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}
	terminal.stderr = stderr

	// Start the SSH command
	if err := terminal.cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start SSH: %w", err)
	}

	terminal.isActive = true

	// Start output readers
	go terminal.readOutput(terminal.stdout, false)
	go terminal.readOutput(terminal.stderr, true)

	// Monitor process
	go terminal.monitorProcess()

	// Initial connection message
	terminal.appendLine(fmt.Sprintf("[green]Connecting to %s@%s...[white]", username, hostname))

	return terminal, nil
}

// buildSSHArgs builds SSH command arguments
func buildSSHArgs(config *SSHConfig, hostname, username string) []string {
	var args []string

	// Force pseudo-terminal allocation
	args = append(args, "-tt")

	if config != nil {
		// Add port if specified
		if config.Port != 22 {
			args = append(args, "-p", fmt.Sprintf("%d", config.Port))
		}

		// Add key file if specified
		if config.KeyFile != "" {
			args = append(args, "-i", config.KeyFile)
		}

		// Add config file if specified
		if config.ConfigFile != "" {
			args = append(args, "-F", config.ConfigFile)
		}

		// Add SSH options
		for key, value := range config.Options {
			args = append(args, "-o", fmt.Sprintf("%s=%s", key, value))
		}

		// Add connection timeout
		if config.Timeout > 0 {
			timeoutSecs := int(config.Timeout.Seconds())
			args = append(args, "-o", fmt.Sprintf("ConnectTimeout=%d", timeoutSecs))
		}
	}

	// Add username@hostname
	if username != "" {
		args = append(args, fmt.Sprintf("%s@%s", username, hostname))
	} else {
		args = append(args, hostname)
	}

	return args
}

// readOutput reads output from stdout/stderr
func (t *SSHTerminal) readOutput(reader io.ReadCloser, isError bool) {
	defer func() { _ = reader.Close() }()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()

		// Color error output in red
		if isError {
			line = fmt.Sprintf("[red]%s[white]", line)
		}

		t.appendLine(line)
		t.lastActivity = time.Now()
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		t.appendLine(fmt.Sprintf("[red]Read error: %v[white]", err))
	}
}

// monitorProcess monitors the SSH process
func (t *SSHTerminal) monitorProcess() {
	err := t.cmd.Wait()

	t.mu.Lock()
	t.isActive = false
	t.mu.Unlock()

	if err != nil {
		t.appendLine(fmt.Sprintf("[red]SSH connection closed: %v[white]", err))
	} else {
		t.appendLine("[yellow]SSH connection closed[white]")
	}

	// Call close callback if set
	if t.onClose != nil {
		t.onClose()
	}
}

// appendLine appends a line to the terminal buffer
func (t *SSHTerminal) appendLine(line string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Add to buffer
	t.buffer = append(t.buffer, line)

	// Trim buffer if too large
	if len(t.buffer) > t.maxBufferSize {
		t.buffer = t.buffer[len(t.buffer)-t.maxBufferSize:]
	}

	// Update text view
	text := ""
	for _, l := range t.buffer {
		text += l + "\n"
	}

	t.SetText(text)

	// Auto-scroll to bottom
	t.ScrollToEnd()
}

// SendCommand sends a command to the terminal
func (t *SSHTerminal) SendCommand(command string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.isActive {
		return fmt.Errorf("terminal is not active")
	}

	_, err := fmt.Fprintf(t.stdin, "%s\n", command)
	if err != nil {
		return fmt.Errorf("failed to send command: %w", err)
	}

	t.lastActivity = time.Now()
	return nil
}

// Resize resizes the terminal
func (t *SSHTerminal) Resize(_, _ int) error {
	// SSH doesn't easily support resize through the command interface
	// This would require using a PTY library for full terminal emulation
	// For now, we'll just resize the text view
	return nil
}

// Close closes the terminal
func (t *SSHTerminal) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.isActive {
		return nil
	}

	t.isActive = false

	// Close stdin to signal end of input
	if t.stdin != nil {
		_ = t.stdin.Close()
	}

	// Try to terminate the process gracefully
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Signal(os.Interrupt)

		// Give it a moment to exit gracefully
		time.Sleep(100 * time.Millisecond)

		// Force kill if still running
		_ = t.cmd.Process.Kill()
	}

	return nil
}

// IsActive returns whether the terminal is active
func (t *SSHTerminal) IsActive() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isActive
}

// GetNodeID returns the node ID
func (t *SSHTerminal) GetNodeID() string {
	return t.nodeID
}

// SetOnClose sets the callback for when the terminal closes
func (t *SSHTerminal) SetOnClose(callback func()) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.onClose = callback
}

// GetInfo returns terminal information
func (t *SSHTerminal) GetInfo() TerminalInfo {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return TerminalInfo{
		NodeID:       t.nodeID,
		Hostname:     t.hostname,
		Username:     t.username,
		IsActive:     t.isActive,
		ConnectedAt:  t.connectedAt,
		LastActivity: t.lastActivity,
	}
}

// Clear clears the terminal buffer
func (t *SSHTerminal) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.buffer = make([]string, 0, t.maxBufferSize)
	t.TextView.Clear()
	t.appendLine(fmt.Sprintf("[green]Terminal cleared - %s@%s[white]", t.username, t.hostname))
}

// GetBuffer returns the current terminal buffer
func (t *SSHTerminal) GetBuffer() []string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make([]string, len(t.buffer))
	copy(result, t.buffer)
	return result
}

// SetInputCapture sets up keyboard input handling
func (t *SSHTerminal) SetInputCapture(_ *tview.Application) {
	t.TextView.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Handle special keys
		switch event.Key() {
		case tcell.KeyCtrlC:
			// Send Ctrl+C to terminal
			_ = t.SendCommand("\x03")
			return nil
		case tcell.KeyCtrlD:
			// Send Ctrl+D to terminal
			_ = t.SendCommand("\x04")
			return nil
		case tcell.KeyCtrlZ:
			// Send Ctrl+Z to terminal
			_ = t.SendCommand("\x1a")
			return nil
		case tcell.KeyEscape:
			// Let escape pass through for vim, etc
			_ = t.SendCommand("\x1b")
			return nil
		case tcell.KeyEnter:
			// Send Enter
			_ = t.SendCommand("")
			return nil
		case tcell.KeyBackspace, tcell.KeyBackspace2:
			// Send backspace
			_ = t.SendCommand("\x08")
			return nil
		case tcell.KeyTab:
			// Send tab for completion
			_ = t.SendCommand("\t")
			return nil
		default:
			// Send regular characters
			if event.Rune() != 0 {
				_ = t.SendCommand(string(event.Rune()))
			}
		}

		return event
	})
}
