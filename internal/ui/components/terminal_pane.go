package components

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/ssh"
	"github.com/rivo/tview"
)

// TerminalPane represents an SSH terminal pane
type TerminalPane struct {
	mu             sync.RWMutex
	id             string
	hostname       string
	username       string
	sessionID      string
	sessionManager *ssh.SessionManager

	// UI components
	container  *tview.Flex
	terminal   *tview.TextView
	statusLine *tview.TextView

	// State
	connected      bool
	lastActivity   time.Time
	commandHistory []string
	// TODO(lint): Review unused code - field currentInput is unused
	// currentInput   string

	// Callbacks
	onClose       func(string) error
	onTitleChange func(string, string)

	// Input handling
	inputBuffer strings.Builder
	inputMode   bool
}

// NewTerminalPane creates a new terminal pane
func NewTerminalPane(id, hostname, username string, sessionManager *ssh.SessionManager) *TerminalPane {
	tp := &TerminalPane{
		id:             id,
		hostname:       hostname,
		username:       username,
		sessionManager: sessionManager,
		lastActivity:   time.Now(),
		commandHistory: make([]string, 0, 100),
	}

	tp.initializeUI()

	return tp
}

// initializeUI sets up the terminal UI components
func (tp *TerminalPane) initializeUI() {
	// Create terminal display
	tp.terminal = tview.NewTextView()
	tp.terminal.SetDynamicColors(true)
	tp.terminal.SetScrollable(true)
	tp.terminal.SetWrap(true)
	tp.terminal.SetBorder(true)
	tp.terminal.SetTitle(fmt.Sprintf(" %s@%s ", tp.username, tp.hostname))
	tp.terminal.SetTitleAlign(tview.AlignLeft)

	// Create status line
	tp.statusLine = tview.NewTextView()
	tp.statusLine.SetDynamicColors(true)
	tp.statusLine.SetText(tp.getStatusText())

	// Create container
	tp.container = tview.NewFlex()
	tp.container.SetDirection(tview.FlexRow)
	tp.container.AddItem(tp.terminal, 0, 1, true)
	tp.container.AddItem(tp.statusLine, 1, 0, false)

	// Set up input handling
	tp.container.SetInputCapture(tp.handleInput)

	// Initialize with connection message
	tp.addOutput("[yellow]Initializing terminal session...[white]\n")
	tp.addOutput(fmt.Sprintf("[cyan]Connecting to %s@%s[white]\n", tp.username, tp.hostname))
}

// handleInput processes keyboard input for the terminal
func (tp *TerminalPane) handleInput(event *tcell.EventKey) *tcell.EventKey {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	tp.lastActivity = time.Now()

	switch event.Key() {
	case tcell.KeyEnter:
		if tp.handleEnterKey() {
			return nil
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if tp.handleBackspaceKey() {
			return nil
		}
	case tcell.KeyCtrlC:
		if tp.handleCtrlCKey() {
			return nil
		}
	case tcell.KeyCtrlD:
		if tp.handleCtrlDKey() {
			return nil
		}
	case tcell.KeyRune:
		if handled := tp.handleRuneKey(event.Rune()); handled {
			return nil
		}
	}

	return event
}

func (tp *TerminalPane) handleEnterKey() bool {
	if !tp.inputMode {
		return false
	}

	command := tp.inputBuffer.String()
	tp.inputBuffer.Reset()
	tp.executeCommand(command)
	return true
}

func (tp *TerminalPane) handleBackspaceKey() bool {
	if !tp.inputMode || tp.inputBuffer.Len() == 0 {
		return false
	}

	current := tp.inputBuffer.String()
	if len(current) > 0 {
		tp.inputBuffer.Reset()
		tp.inputBuffer.WriteString(current[:len(current)-1])
		tp.updatePrompt()
	}
	return true
}

func (tp *TerminalPane) handleCtrlCKey() bool {
	if !tp.inputMode {
		return false
	}

	tp.inputBuffer.Reset()
	tp.addOutput("[red]^C[white]\n")
	tp.showPrompt()
	return true
}

func (tp *TerminalPane) handleCtrlDKey() bool {
	if !tp.inputMode || tp.inputBuffer.Len() > 0 {
		return false
	}

	tp.addOutput("[yellow]logout[white]\n")
	tp.disconnect()
	return true
}

func (tp *TerminalPane) handleRuneKey(r rune) bool {
	if tp.inputMode {
		tp.inputBuffer.WriteRune(r)
		tp.updatePrompt()
		return true
	}

	// Handle special characters when not in input mode
	handlers := map[rune]func(){
		'i': tp.enterInputMode,
		'I': tp.enterInputMode,
		'c': tp.connect,
		'C': tp.connect,
		'd': tp.disconnect,
		'D': tp.disconnect,
		'r': tp.reconnect,
		'R': tp.reconnect,
		'h': tp.showHelp,
		'H': tp.showHelp,
	}

	if handler, ok := handlers[r]; ok {
		handler()
		return true
	}

	return false
}

// executeCommand executes a command in the terminal
func (tp *TerminalPane) executeCommand(command string) {
	if command == "" {
		tp.showPrompt()
		return
	}

	// Add to history
	tp.commandHistory = append(tp.commandHistory, command)
	if len(tp.commandHistory) > 100 {
		tp.commandHistory = tp.commandHistory[1:]
	}

	// Display command
	tp.addOutput(fmt.Sprintf("[green]%s@%s[white]:[blue]~[white]$ %s\n", tp.username, tp.hostname, command))

	// Handle built-in commands
	switch {
	case command == "exit" || command == "logout":
		tp.addOutput("[yellow]logout[white]\n")
		tp.disconnect()
		return

	case command == "clear":
		tp.clearTerminal()
		tp.showPrompt()
		return

	case command == "help":
		tp.showHelp()
		tp.showPrompt()
		return

	case strings.HasPrefix(command, "echo "):
		message := strings.TrimPrefix(command, "echo ")
		tp.addOutput(fmt.Sprintf("%s\n", message))
		tp.showPrompt()
		return
	}

	// Execute remote command if connected
	if tp.connected && tp.sessionManager != nil && tp.sessionID != "" {
		tp.executeRemoteCommand(command)
	} else {
		tp.addOutput("[red]Not connected to remote host[white]\n")
		tp.addOutput("Use 'c' to connect or 'i' to enter input mode\n")
		tp.showPrompt()
	}
}

// executeRemoteCommand executes a command on the remote host
func (tp *TerminalPane) executeRemoteCommand(command string) {
	// Show executing status
	tp.addOutput("[yellow]Executing...[white]\n")
	tp.updateStatusLine()

	go func() {
		// Simulate command execution (in real implementation, this would use SSH)
		time.Sleep(100 * time.Millisecond) // Simulate network delay

		// Mock response based on command
		var output string
		switch {
		case command == "pwd":
			output = "/home/" + tp.username
		case command == "whoami":
			output = tp.username
		case command == "hostname":
			output = tp.hostname
		case command == "date":
			output = time.Now().Format("Mon Jan 2 15:04:05 MST 2006")
		case command == "uptime":
			output = " 15:42:13 up 2 days,  3:21,  1 user,  load average: 0.15, 0.09, 0.05"
		case strings.HasPrefix(command, "ls"):
			if strings.Contains(command, "-l") {
				output = "total 8\ndrwxr-xr-x 2 " + tp.username + " " + tp.username + " 4096 Jan 15 10:30 Documents\ndrwxr-xr-x 2 " + tp.username + " " + tp.username + " 4096 Jan 15 10:30 Downloads"
			} else {
				output = "Documents  Downloads  Desktop  Pictures"
			}
		default:
			output = fmt.Sprintf("bash: %s: command not found", command)
		}

		// Update UI on main thread
		go tp.addOutput(fmt.Sprintf("%s\n", output))
		go tp.showPrompt()
		go tp.updateStatusLine()
	}()
}

// connect establishes a connection to the remote host
func (tp *TerminalPane) connect() {
	if tp.connected {
		tp.addOutput("[yellow]Already connected[white]\n")
		return
	}

	tp.addOutput(fmt.Sprintf("[yellow]Connecting to %s@%s...[white]\n", tp.username, tp.hostname))
	tp.updateStatusLine()

	go func() {
		// Simulate connection process
		time.Sleep(500 * time.Millisecond)

		if tp.sessionManager != nil {
			// Try to create/connect session
			session, err := tp.sessionManager.CreateSession(tp.hostname, tp.username)
			if err != nil {
				tp.addOutput(fmt.Sprintf("[red]Connection failed: %v[white]\n", err))
				tp.updateStatusLine()
				return
			}

			err = tp.sessionManager.ConnectSession(session.ID)
			if err != nil {
				tp.addOutput(fmt.Sprintf("[red]Connection failed: %v[white]\n", err))
				tp.updateStatusLine()
				return
			}

			tp.sessionID = session.ID
		}

		// Connection successful
		tp.mu.Lock()
		tp.connected = true
		tp.mu.Unlock()

		tp.addOutput(fmt.Sprintf("[green]Connected to %s@%s[white]\n", tp.username, tp.hostname))
		tp.addOutput("Welcome to the SSH terminal!\n")
		tp.addOutput("Type 'help' for available commands or 'i' to enter interactive mode.\n")
		tp.showPrompt()
		tp.updateStatusLine()
		tp.updateTitle()
	}()
}

// disconnect closes the connection
func (tp *TerminalPane) disconnect() {
	if !tp.connected {
		tp.addOutput("[yellow]Not connected[white]\n")
		return
	}

	tp.addOutput("[yellow]Disconnecting...[white]\n")

	if tp.sessionManager != nil && tp.sessionID != "" {
		go func() { _ = tp.sessionManager.CloseSession(tp.sessionID) }()
	}

	tp.mu.Lock()
	tp.connected = false
	tp.sessionID = ""
	tp.inputMode = false
	tp.mu.Unlock()

	tp.addOutput("[red]Connection closed[white]\n")
	tp.updateStatusLine()
	tp.updateTitle()
}

// reconnect attempts to reconnect
func (tp *TerminalPane) reconnect() {
	tp.addOutput("[yellow]Reconnecting...[white]\n")
	tp.disconnect()
	time.Sleep(100 * time.Millisecond)
	tp.connect()
}

// enterInputMode enables interactive input mode
func (tp *TerminalPane) enterInputMode() {
	tp.mu.Lock()
	tp.inputMode = true
	tp.mu.Unlock()

	tp.addOutput("[cyan]Entering interactive mode. Press Ctrl+C to exit.[white]\n")
	tp.showPrompt()
	tp.updateStatusLine()
}

// showPrompt displays the command prompt
func (tp *TerminalPane) showPrompt() {
	if !tp.connected {
		return
	}

	prompt := fmt.Sprintf("[green]%s@%s[white]:[blue]~[white]$ ", tp.username, tp.hostname)
	tp.addOutput(prompt)
}

// updatePrompt updates the current prompt with input
func (tp *TerminalPane) updatePrompt() {
	// This would update the last line with current input
	// For simplicity, we'll just show the input
	current := tp.terminal.GetText(false)
	lines := strings.Split(current, "\n")

	if len(lines) > 0 {
		lastLine := lines[len(lines)-1]
		if strings.Contains(lastLine, "$ ") {
			// Update the last line with current input
			parts := strings.Split(lastLine, "$ ")
			if len(parts) >= 2 {
				newLine := parts[0] + "$ " + tp.inputBuffer.String()
				lines[len(lines)-1] = newLine
				tp.terminal.SetText(strings.Join(lines, "\n"))
			}
		}
	}
}

// addOutput adds text to the terminal display
func (tp *TerminalPane) addOutput(text string) {
	current := tp.terminal.GetText(false)
	tp.terminal.SetText(current + text)

	// Auto-scroll to bottom
	tp.terminal.ScrollToEnd()
}

// clearTerminal clears the terminal display
func (tp *TerminalPane) clearTerminal() {
	tp.terminal.Clear()
	tp.addOutput(fmt.Sprintf("[cyan]Terminal cleared - %s@%s[white]\n", tp.username, tp.hostname))
}

// showHelp displays help information
func (tp *TerminalPane) showHelp() {
	help := `[yellow]Terminal Pane Help[white]

[cyan]Navigation:[white]
  i, I    - Enter interactive input mode
  c, C    - Connect to remote host
  d, D    - Disconnect from remote host
  r, R    - Reconnect
  h, H    - Show this help

[cyan]Interactive Mode:[white]
  Enter   - Execute command
  Ctrl+C  - Cancel current input / Exit interactive mode
  Ctrl+D  - Logout (if no input)

[cyan]Built-in Commands:[white]
  help    - Show this help
  clear   - Clear terminal
  exit    - Disconnect and close
  echo    - Echo text

[cyan]Remote Commands:[white]
  pwd, whoami, hostname, date, uptime, ls

Press any key to continue...
`
	tp.addOutput(help)
}

// updateStatusLine updates the status line
func (tp *TerminalPane) updateStatusLine() {
	tp.statusLine.SetText(tp.getStatusText())
}

// getStatusText returns the current status text
func (tp *TerminalPane) getStatusText() string {
	status := fmt.Sprintf("[white]%s@%s", tp.username, tp.hostname)

	if tp.connected {
		status += " [green]●[white] Connected"
		if tp.inputMode {
			status += " [yellow]INPUT[white]"
		}
	} else {
		status += " [red]●[white] Disconnected"
	}

	status += fmt.Sprintf(" | Last: %s", tp.lastActivity.Format("15:04:05"))

	if len(tp.commandHistory) > 0 {
		status += fmt.Sprintf(" | History: %d", len(tp.commandHistory))
	}

	return status
}

// updateTitle updates the pane title
func (tp *TerminalPane) updateTitle() {
	var title string
	if tp.connected {
		title = fmt.Sprintf(" ● %s@%s ", tp.username, tp.hostname)
	} else {
		title = fmt.Sprintf(" ○ %s@%s ", tp.username, tp.hostname)
	}

	tp.terminal.SetTitle(title)

	if tp.onTitleChange != nil {
		tp.onTitleChange(tp.id, title)
	}
}

// GetContainer returns the main container
func (tp *TerminalPane) GetContainer() tview.Primitive {
	return tp.container
}

// GetID returns the pane ID
func (tp *TerminalPane) GetID() string {
	return tp.id
}

// GetHostname returns the hostname
func (tp *TerminalPane) GetHostname() string {
	return tp.hostname
}

// GetUsername returns the username
func (tp *TerminalPane) GetUsername() string {
	return tp.username
}

// IsConnected returns the connection status
func (tp *TerminalPane) IsConnected() bool {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return tp.connected
}

// GetLastActivity returns the last activity time
func (tp *TerminalPane) GetLastActivity() time.Time {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return tp.lastActivity
}

// SetOnClose sets the close callback
func (tp *TerminalPane) SetOnClose(callback func(string) error) {
	tp.onClose = callback
}

// SetOnTitleChange sets the title change callback
func (tp *TerminalPane) SetOnTitleChange(callback func(string, string)) {
	tp.onTitleChange = callback
}

// Close closes the terminal pane
func (tp *TerminalPane) Close() error {
	if tp.connected {
		tp.disconnect()
	}

	if tp.onClose != nil {
		return tp.onClose(tp.id)
	}

	return nil
}

// SendCommand sends a command to the terminal
func (tp *TerminalPane) SendCommand(command string) error {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	if !tp.connected {
		return fmt.Errorf("terminal not connected")
	}

	tp.executeCommand(command)
	return nil
}

// GetCommandHistory returns the command history
func (tp *TerminalPane) GetCommandHistory() []string {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	history := make([]string, len(tp.commandHistory))
	copy(history, tp.commandHistory)
	return history
}
