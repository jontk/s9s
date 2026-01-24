package views

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/ssh"
	"github.com/rivo/tview"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// SSHTerminalView provides an SSH terminal interface
type SSHTerminalView struct {
	app            *tview.Application
	pages          *tview.Pages
	sessionManager *ssh.SessionManager

	// UI components
	modal        *tview.Flex
	sessionList  *tview.List
	nodeInfoView *tview.TextView
	statusBar    *tview.TextView

	// State
	selectedNode  string
	activeSession *ssh.SSHSession
	nodes         []string // Available nodes
}

// NewSSHTerminalView creates a new SSH terminal view
func NewSSHTerminalView(app *tview.Application) *SSHTerminalView {
	sshConfig := ssh.DefaultSSHConfig()
	sessionManager, err := ssh.NewSessionManager(sshConfig)
	if err != nil {
		// Fallback to basic configuration
		sessionManager = nil
	}

	return &SSHTerminalView{
		app:            app,
		sessionManager: sessionManager,
		nodes:          []string{}, // Will be populated from cluster data
	}
}

// SetPages sets the pages manager for modal display
func (v *SSHTerminalView) SetPages(pages *tview.Pages) {
	v.pages = pages
}

// SetNodes sets the available nodes for SSH connections
func (v *SSHTerminalView) SetNodes(nodes []string) {
	v.nodes = nodes
	v.updateNodeList()
}

// ShowSSHInterface displays the SSH terminal interface
func (v *SSHTerminalView) ShowSSHInterface(defaultNode string) {
	v.selectedNode = defaultNode
	v.buildUI()
	v.show()
}

// buildUI creates the SSH terminal interface
func (v *SSHTerminalView) buildUI() {
	// Create session list
	v.sessionList = tview.NewList()
	v.sessionList.SetBorder(true)
	v.sessionList.SetTitle(" SSH Sessions ")
	v.sessionList.SetTitleAlign(tview.AlignCenter)
	v.sessionList.ShowSecondaryText(true)

	// Add available nodes
	v.updateNodeList()

	// Create node info view
	v.nodeInfoView = tview.NewTextView()
	v.nodeInfoView.SetBorder(true)
	v.nodeInfoView.SetTitle(" Node Information ")
	v.nodeInfoView.SetTitleAlign(tview.AlignCenter)
	v.nodeInfoView.SetDynamicColors(true)
	v.nodeInfoView.SetScrollable(true)
	v.nodeInfoView.SetWrap(true)

	// Create status bar
	v.statusBar = tview.NewTextView()
	v.statusBar.SetDynamicColors(true)
	v.statusBar.SetTextAlign(tview.AlignCenter)
	v.statusBar.SetText("[green]SSH Terminal Ready[white] | [yellow]Keys:[white] Enter=Connect c=New Connection i=Info t=Terminal s=System Info Esc=Close")

	// Create help text
	helpText := tview.NewTextView()
	helpText.SetDynamicColors(true)
	helpText.SetText("[yellow]SSH Terminal Controls:[white]\n" +
		"[green]Enter[white] - Connect to selected node\n" +
		"[green]c[white] - Create new SSH connection\n" +
		"[green]i[white] - Show node information\n" +
		"[green]t[white] - Open terminal session\n" +
		"[green]s[white] - Show system information\n" +
		"[green]m[white] - Monitor session status\n" +
		"[green]x[white] - Close selected session\n" +
		"[green]Esc[white] - Close SSH interface")
	helpText.SetTextAlign(tview.AlignLeft)
	helpText.SetBorder(true)
	helpText.SetTitle(" Help ")

	// Create main layout
	leftPanel := tview.NewFlex()
	leftPanel.SetDirection(tview.FlexRow)
	leftPanel.AddItem(v.sessionList, 0, 3, true)
	leftPanel.AddItem(helpText, 0, 1, false)

	rightPanel := tview.NewFlex()
	rightPanel.SetDirection(tview.FlexRow)
	rightPanel.AddItem(v.nodeInfoView, 0, 1, false)

	contentPanel := tview.NewFlex()
	contentPanel.AddItem(leftPanel, 0, 1, true)
	contentPanel.AddItem(rightPanel, 0, 2, false)

	// Create modal container
	v.modal = tview.NewFlex()
	v.modal.SetDirection(tview.FlexRow)
	v.modal.AddItem(nil, 0, 1, false)
	v.modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(contentPanel, 0, 1, true).
			AddItem(v.statusBar, 1, 0, false), 0, 4, true).
		AddItem(nil, 0, 1, false), 0, 3, true)
	v.modal.AddItem(nil, 0, 1, false)

	v.modal.SetBorder(true)
	v.modal.SetTitle(" SSH Terminal Manager ")
	v.modal.SetTitleAlign(tview.AlignCenter)

	// Setup event handlers
	v.setupEventHandlers()

	// Load initial node info if a node is selected
	if v.selectedNode != "" {
		v.loadNodeInfo(v.selectedNode)
	}
}

// updateNodeList updates the session list with available nodes and sessions
func (v *SSHTerminalView) updateNodeList() {
	if v.sessionList == nil {
		return
	}

	v.sessionList.Clear()

	// Add existing sessions
	if v.sessionManager != nil {
		sessions := v.sessionManager.GetSessions()
		for _, session := range sessions {
			status := session.State.String()
			secondary := fmt.Sprintf("User: %s | Status: %s | Started: %s",
				session.Username, status, session.StartTime.Format("15:04:05"))

			sessionCopy := session // Capture for closure
			v.sessionList.AddItem(
				fmt.Sprintf("🔗 %s", session.Hostname),
				secondary,
				0,
				func() {
					v.activeSession = sessionCopy
					v.selectedNode = sessionCopy.Hostname
					v.loadNodeInfo(sessionCopy.Hostname)
				},
			)
		}
	}

	// Add separator if we have both sessions and available nodes
	if v.sessionManager != nil && len(v.sessionManager.GetSessions()) > 0 && len(v.nodes) > 0 {
		v.sessionList.AddItem("──────────────", "Available Nodes", 0, nil)
	}

	// Add available nodes
	for _, node := range v.nodes {
		nodeCopy := node // Capture for closure
		v.sessionList.AddItem(
			fmt.Sprintf("🖥  %s", node),
			"Available for connection",
			0,
			func() {
				v.selectedNode = nodeCopy
				v.activeSession = nil
				v.loadNodeInfo(nodeCopy)
			},
		)
	}

	// Add quick connect option
	v.sessionList.AddItem("➕ Quick Connect", "Connect to a custom hostname", 0, func() {
		v.showQuickConnectDialog()
	})
}

// setupEventHandlers configures keyboard shortcuts
func (v *SSHTerminalView) setupEventHandlers() {
	v.modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			v.close()
			return nil
		case tcell.KeyEnter:
			v.connectToSelectedNode()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'c', 'C':
				v.createNewConnection()
				return nil
			case 'i', 'I':
				v.showNodeInfo()
				return nil
			case 't', 'T':
				v.openTerminalSession()
				return nil
			case 's', 'S':
				v.showSystemInfo()
				return nil
			case 'm', 'M':
				v.monitorSession()
				return nil
			case 'x', 'X':
				v.closeSelectedSession()
				return nil
			case 'r', 'R':
				v.refreshSessions()
				return nil
			}
		}
		return event
	})

	// Handle session list selection
	v.sessionList.SetSelectedFunc(func(index int, primaryText, secondaryText string, shortcut rune) {
		v.connectToSelectedNode()
	})
}

// connectToSelectedNode connects to the currently selected node
func (v *SSHTerminalView) connectToSelectedNode() {
	if v.selectedNode == "" {
		v.updateStatus("[red]No node selected[white]")
		return
	}

	if v.sessionManager == nil {
		v.fallbackSSHConnection(v.selectedNode)
		return
	}

	v.updateStatus(fmt.Sprintf("[yellow]Connecting to %s...[white]", v.selectedNode))

	// Create or get existing session
	session, err := v.sessionManager.CreateSession(v.selectedNode, "")
	if err != nil {
		v.updateStatus(fmt.Sprintf("[red]Failed to create session: %v[white]", err))
		return
	}

	// Connect session
	if err := v.sessionManager.ConnectSession(session.ID); err != nil {
		v.updateStatus(fmt.Sprintf("[red]Connection failed: %v[white]", err))
		return
	}

	v.activeSession = session
	v.updateStatus(fmt.Sprintf("[green]Connected to %s[white]", v.selectedNode))
	v.updateNodeList()
}

// createNewConnection creates a new SSH connection
func (v *SSHTerminalView) createNewConnection() {
	v.showQuickConnectDialog()
}

// showNodeInfo displays information about the selected node
func (v *SSHTerminalView) showNodeInfo() {
	if v.selectedNode == "" {
		return
	}
	v.loadNodeInfo(v.selectedNode)
}

// openTerminalSession opens an interactive terminal session
func (v *SSHTerminalView) openTerminalSession() {
	if v.activeSession == nil {
		if v.selectedNode != "" {
			v.connectToSelectedNode()
			if v.activeSession == nil {
				return
			}
		} else {
			v.updateStatus("[red]No active session[white]")
			return
		}
	}

	v.updateStatus(fmt.Sprintf("[yellow]Opening terminal for %s...[white]", v.activeSession.Hostname))

	// Close the SSH interface
	v.close()

	// Start interactive session (this will take over the terminal)
	go func() {
		if err := v.sessionManager.StartInteractiveSession(v.activeSession.ID); err != nil {
			// Show error in a modal when we return to the UI
			v.app.QueueUpdateDraw(func() {
				v.showErrorModal(fmt.Sprintf("Failed to start terminal: %v", err))
			})
		}
	}()
}

// showSystemInfo shows detailed system information for the selected node
func (v *SSHTerminalView) showSystemInfo() {
	if v.activeSession == nil {
		v.updateStatus("[red]No active session. Connect to a node first.[white]")
		return
	}

	v.updateStatus("[yellow]Gathering system information...[white]")

	go func() {
		info, err := v.sessionManager.GetNodeSystemInfo(v.activeSession.ID)
		if err != nil {
			v.app.QueueUpdateDraw(func() {
				v.updateStatus(fmt.Sprintf("[red]Failed to get system info: %v[white]", err))
			})
			return
		}

		// Format system information
		var infoText strings.Builder
		infoText.WriteString(fmt.Sprintf("[yellow]System Information for %s[white]\n\n", v.activeSession.Hostname))

		sections := map[string][]string{
			"System":      {"hostname", "kernel", "distribution", "uptime"},
			"Hardware":    {"cpu_info", "memory"},
			"Storage":     {"disk_usage"},
			"Network":     {"network"},
			"Performance": {"load_avg", "processes"},
		}

		for section, keys := range sections {
			infoText.WriteString(fmt.Sprintf("[green]%s:[white]\n", section))
			for _, key := range keys {
				if value, exists := info[key]; exists {
					// Truncate long values
					if len(value) > 200 {
						value = value[:200] + "..."
					}
					infoText.WriteString(fmt.Sprintf("  [cyan]%s:[white] %s\n", cases.Title(language.English).String(strings.ReplaceAll(key, "_", " ")), value))
				}
			}
			infoText.WriteString("\n")
		}

		v.app.QueueUpdateDraw(func() {
			v.nodeInfoView.SetText(infoText.String())
			v.updateStatus("[green]System information updated[white]")
		})
	}()
}

// monitorSession monitors the status of the active session
func (v *SSHTerminalView) monitorSession() {
	if v.activeSession == nil {
		v.updateStatus("[red]No active session to monitor[white]")
		return
	}

	statusChan, err := v.sessionManager.MonitorSession(v.activeSession.ID, 5*time.Second)
	if err != nil {
		v.updateStatus(fmt.Sprintf("[red]Failed to start monitoring: %v[white]", err))
		return
	}

	v.updateStatus(fmt.Sprintf("[yellow]Monitoring session %s...[white]", v.activeSession.Hostname))

	go func() {
		for status := range statusChan {
			v.app.QueueUpdateDraw(func() {
				v.updateStatus(fmt.Sprintf("[blue]Session %s status: %s[white]", v.activeSession.Hostname, status.String()))
			})
		}
	}()
}

// closeSelectedSession closes the currently selected session
func (v *SSHTerminalView) closeSelectedSession() {
	if v.activeSession == nil {
		v.updateStatus("[red]No active session to close[white]")
		return
	}

	if err := v.sessionManager.CloseSession(v.activeSession.ID); err != nil {
		v.updateStatus(fmt.Sprintf("[red]Failed to close session: %v[white]", err))
		return
	}

	v.updateStatus(fmt.Sprintf("[green]Session %s closed[white]", v.activeSession.Hostname))
	v.activeSession = nil
	v.updateNodeList()
}

// refreshSessions refreshes the session list
func (v *SSHTerminalView) refreshSessions() {
	v.updateNodeList()
	v.updateStatus("[green]Sessions refreshed[white]")
}

// loadNodeInfo loads and displays information about a node
func (v *SSHTerminalView) loadNodeInfo(hostname string) {
	if v.nodeInfoView == nil {
		return
	}

	info := "[yellow]Node Information[white]\n\n"
	info += fmt.Sprintf("[cyan]Hostname:[white] %s\n", hostname)
	info += fmt.Sprintf("[cyan]Status:[white] %s\n", v.getNodeStatus(hostname))

	if v.activeSession != nil && v.activeSession.Hostname == hostname {
		info += fmt.Sprintf("[cyan]Session ID:[white] %s\n", v.activeSession.ID)
		info += fmt.Sprintf("[cyan]Connected Since:[white] %s\n", v.activeSession.StartTime.Format("2006-01-02 15:04:05"))
		info += fmt.Sprintf("[cyan]Last Activity:[white] %s\n", v.activeSession.LastActivity.Format("15:04:05"))

		if len(v.activeSession.Tunnels) > 0 {
			info += "\n[cyan]Active Tunnels:[white]\n"
			for _, tunnel := range v.activeSession.Tunnels {
				info += fmt.Sprintf("  %s:%d -> %s:%d (%s)\n",
					"localhost", tunnel.LocalPort, tunnel.RemoteHost, tunnel.RemotePort, tunnel.Type)
			}
		}
	}

	info += "\n[gray]Use 's' to get detailed system information[white]"

	v.nodeInfoView.SetText(info)
}

// getNodeStatus returns the current status of a node
func (v *SSHTerminalView) getNodeStatus(hostname string) string {
	if v.sessionManager == nil {
		return "Unknown"
	}

	sessions := v.sessionManager.GetSessions()
	for _, session := range sessions {
		if session.Hostname == hostname {
			return session.State.String()
		}
	}

	return "Available"
}

// showQuickConnectDialog shows a dialog to connect to a custom hostname
func (v *SSHTerminalView) showQuickConnectDialog() {
	form := tview.NewForm()
	form.SetBorder(true)
	form.SetTitle(" Quick Connect ")
	form.SetTitleAlign(tview.AlignCenter)

	hostname := ""
	username := os.Getenv("USER")

	form.AddInputField("Hostname", "", 30, nil, func(text string) {
		hostname = text
	})
	form.AddInputField("Username", username, 20, nil, func(text string) {
		username = text
	})

	form.AddButton("Connect", func() {
		v.pages.RemovePage("quick-connect")
		if hostname != "" {
			v.selectedNode = hostname
			v.connectToNode(hostname, username)
		}
	})
	form.AddButton("Cancel", func() {
		v.pages.RemovePage("quick-connect")
	})

	// Create modal container
	modal := tview.NewFlex()
	modal.SetDirection(tview.FlexRow)
	modal.AddItem(nil, 0, 1, false)
	modal.AddItem(tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(form, 0, 1, true).
		AddItem(nil, 0, 1, false), 0, 1, true)
	modal.AddItem(nil, 0, 1, false)

	v.pages.AddPage("quick-connect", modal, true, true)
}

// connectToNode connects to a specific node with username
func (v *SSHTerminalView) connectToNode(hostname, username string) {
	if v.sessionManager == nil {
		v.fallbackSSHConnection(hostname)
		return
	}

	v.updateStatus(fmt.Sprintf("[yellow]Connecting to %s@%s...[white]", username, hostname))

	session, err := v.sessionManager.CreateSession(hostname, username)
	if err != nil {
		v.updateStatus(fmt.Sprintf("[red]Failed to create session: %v[white]", err))
		return
	}

	if err := v.sessionManager.ConnectSession(session.ID); err != nil {
		v.updateStatus(fmt.Sprintf("[red]Connection failed: %v[white]", err))
		return
	}

	v.activeSession = session
	v.selectedNode = hostname
	v.updateStatus(fmt.Sprintf("[green]Connected to %s@%s[white]", username, hostname))
	v.updateNodeList()
	v.loadNodeInfo(hostname)
}

// fallbackSSHConnection uses basic SSH when session manager is not available
func (v *SSHTerminalView) fallbackSSHConnection(hostname string) {
	v.updateStatus(fmt.Sprintf("[yellow]Connecting to %s (basic SSH)...[white]", hostname))

	v.close()

	// Use basic SSH connection
	go func() {
		cmd := exec.CommandContext(context.Background(), "ssh", hostname)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			v.app.QueueUpdateDraw(func() {
				v.showErrorModal(fmt.Sprintf("SSH connection failed: %v", err))
			})
		}
	}()
}

// updateStatus updates the status bar
func (v *SSHTerminalView) updateStatus(message string) {
	if v.statusBar != nil {
		v.statusBar.SetText(message)
	}
}

// showErrorModal displays an error message
func (v *SSHTerminalView) showErrorModal(message string) {
	modal := tview.NewModal()
	modal.SetText(message)
	modal.AddButtons([]string{"OK"})
	modal.SetDoneFunc(func(buttonIndex int, buttonLabel string) {
		v.pages.RemovePage("error-modal")
	})
	v.pages.AddPage("error-modal", modal, true, true)
}

// show displays the SSH terminal interface
func (v *SSHTerminalView) show() {
	if v.pages != nil {
		v.pages.AddPage("ssh-terminal", v.modal, true, true)
		v.app.SetFocus(v.sessionList)
	}
}

// close closes the SSH terminal interface
func (v *SSHTerminalView) close() {
	if v.pages != nil {
		v.pages.RemovePage("ssh-terminal")
	}
}

// Shutdown gracefully shuts down the SSH terminal view
func (v *SSHTerminalView) Shutdown() {
	if v.sessionManager != nil {
		v.sessionManager.Shutdown()
	}
}
