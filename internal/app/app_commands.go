package app

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
)

// showCommandLine shows the command input field
func (s *S9s) showCommandLine() {
	// Ensure clean state: hide first if already visible
	if s.cmdVisible {
		s.mainLayout.ResizeItem(s.cmdLine, 0, 0)
	}

	s.cmdVisible = true
	s.cmdLine.SetText("")
	s.mainLayout.ResizeItem(s.cmdLine, 1, 0)
	s.app.SetFocus(s.cmdLine)
}

// hideCommandLine hides the command input field
func (s *S9s) hideCommandLine() {
	s.cmdVisible = false
	s.mainLayout.ResizeItem(s.cmdLine, 0, 0)

	// Return focus to current view
	if currentView, err := s.viewMgr.GetCurrentView(); err == nil {
		s.app.SetFocus(currentView.Render())
	}
}

// onCommandDone handles command line completion
func (s *S9s) onCommandDone(key tcell.Key) {
	defer s.hideCommandLine()

	if key != tcell.KeyEnter {
		return
	}

	command := s.cmdLine.GetText()
	if command == "" {
		return
	}

	s.executeCommand(command)
}

// executeCommand parses and executes commands with arguments
func (s *S9s) executeCommand(input string) {
	name, args := ParseCommand(input)
	if name == "" {
		return
	}

	cmd := s.findCommand(name)
	if cmd == nil {
		s.statusBar.Error(fmt.Sprintf("Unknown command: %s", name))
		return
	}

	// Validate argument count
	if len(args) < cmd.MinArgs {
		s.statusBar.Error(fmt.Sprintf("Usage: %s", cmd.Usage))
		return
	}
	if cmd.MaxArgs >= 0 && len(args) > cmd.MaxArgs {
		s.statusBar.Error(fmt.Sprintf("Too many arguments. Usage: %s", cmd.Usage))
		return
	}

	result := cmd.Handler(args)
	if result.Success {
		s.statusBar.Success(result.Message)
	} else {
		s.statusBar.Error(result.Message)
	}
}

// commandRegistry returns all available commands
func (s *S9s) commandRegistry() map[string]*CommandDef {
	return map[string]*CommandDef{
		// Navigation (no args)
		"quit": {
			Name:    "quit",
			Aliases: []string{"q"},
			Usage:   ":quit",
			MaxArgs: 0,
			Handler: s.cmdQuit,
		},
		"jobs": {
			Name:    "jobs",
			Aliases: []string{"j"},
			Usage:   ":jobs",
			MaxArgs: 0,
			Handler: s.cmdJobs,
		},
		"nodes": {
			Name:    "nodes",
			Aliases: []string{"n"},
			Usage:   ":nodes",
			MaxArgs: 0,
			Handler: s.cmdNodes,
		},
		"partitions": {
			Name:    "partitions",
			Aliases: []string{"p"},
			Usage:   ":partitions",
			MaxArgs: 0,
			Handler: s.cmdPartitions,
		},
		"reservations": {
			Name:    "reservations",
			Usage:   ":reservations",
			MaxArgs: 0,
			Handler: s.cmdReservations,
		},
		"qos": {
			Name:    "qos",
			Usage:   ":qos",
			MaxArgs: 0,
			Handler: s.cmdQos,
		},
		"accounts": {
			Name:    "accounts",
			Usage:   ":accounts",
			MaxArgs: 0,
			Handler: s.cmdAccounts,
		},
		"users": {
			Name:    "users",
			Usage:   ":users",
			MaxArgs: 0,
			Handler: s.cmdUsers,
		},
		"dashboard": {
			Name:    "dashboard",
			Usage:   ":dashboard",
			MaxArgs: 0,
			Handler: s.cmdDashboard,
		},
		"health": {
			Name:    "health",
			Usage:   ":health",
			MaxArgs: 0,
			Handler: s.cmdHealth,
		},
		"performance": {
			Name:    "performance",
			Usage:   ":performance",
			MaxArgs: 0,
			Handler: s.cmdPerformance,
		},
		"refresh": {
			Name:    "refresh",
			Aliases: []string{"r"},
			Usage:   ":refresh",
			MaxArgs: 0,
			Handler: s.cmdRefresh,
		},
		"help": {
			Name:    "help",
			Aliases: []string{"h"},
			Usage:   ":help",
			MaxArgs: 0,
			Handler: s.cmdHelp,
		},
		"prefs": {
			Name:    "prefs",
			Aliases: []string{"preferences"},
			Usage:   ":prefs",
			MaxArgs: 0,
			Handler: s.cmdPrefs,
		},
		"layout": {
			Name:    "layout",
			Aliases: []string{"layouts"},
			Usage:   ":layout",
			MaxArgs: 0,
			Handler: s.cmdLayout,
		},
		"config": {
			Name:    "config",
			Aliases: []string{"configuration", "settings"},
			Usage:   ":config",
			MaxArgs: 0,
			Handler: s.cmdConfig,
		},

		// Job operations (with args)
		"cancel": {
			Name:    "cancel",
			Usage:   ":cancel JOBID",
			MinArgs: 1,
			MaxArgs: 1,
			Handler: s.cmdCancelJob,
		},
		"hold": {
			Name:    "hold",
			Usage:   ":hold JOBID",
			MinArgs: 1,
			MaxArgs: 1,
			Handler: s.cmdHoldJob,
		},
		"release": {
			Name:    "release",
			Usage:   ":release JOBID",
			MinArgs: 1,
			MaxArgs: 1,
			Handler: s.cmdReleaseJob,
		},
		"requeue": {
			Name:    "requeue",
			Usage:   ":requeue JOBID",
			MinArgs: 1,
			MaxArgs: 1,
			Handler: s.cmdRequeueJob,
		},

		// Node operations
		"drain": {
			Name:    "drain",
			Usage:   ":drain NODE [REASON]",
			MinArgs: 1,
			MaxArgs: -1, // Unlimited for reason
			Handler: s.cmdDrainNode,
		},
		"resume": {
			Name:    "resume",
			Usage:   ":resume NODE",
			MinArgs: 1,
			MaxArgs: 1,
			Handler: s.cmdResumeNode,
		},
	}
}

// findCommand finds a command by name or alias
func (s *S9s) findCommand(name string) *CommandDef {
	registry := s.commandRegistry()

	// Check exact name match first
	if cmd, ok := registry[name]; ok {
		return cmd
	}

	// Check aliases
	for _, cmd := range registry {
		for _, alias := range cmd.Aliases {
			if alias == name {
				return cmd
			}
		}
	}

	return nil
}

// Command handlers that don't need arguments
func (s *S9s) cmdQuit(args []string) CommandResult {
	_ = s.Stop()
	return CommandResult{Success: true, Message: "Quitting..."}
}

func (s *S9s) cmdJobs(args []string) CommandResult {
	s.switchToView("jobs")
	return CommandResult{Success: true, Message: "Switched to jobs view"}
}

func (s *S9s) cmdNodes(args []string) CommandResult {
	s.switchToView("nodes")
	return CommandResult{Success: true, Message: "Switched to nodes view"}
}

func (s *S9s) cmdPartitions(args []string) CommandResult {
	s.switchToView("partitions")
	return CommandResult{Success: true, Message: "Switched to partitions view"}
}

func (s *S9s) cmdReservations(args []string) CommandResult {
	s.switchToView("reservations")
	return CommandResult{Success: true, Message: "Switched to reservations view"}
}

func (s *S9s) cmdQos(args []string) CommandResult {
	s.switchToView("qos")
	return CommandResult{Success: true, Message: "Switched to QoS view"}
}

func (s *S9s) cmdAccounts(args []string) CommandResult {
	s.switchToView("accounts")
	return CommandResult{Success: true, Message: "Switched to accounts view"}
}

func (s *S9s) cmdUsers(args []string) CommandResult {
	s.switchToView("users")
	return CommandResult{Success: true, Message: "Switched to users view"}
}

func (s *S9s) cmdDashboard(args []string) CommandResult {
	s.switchToView("dashboard")
	return CommandResult{Success: true, Message: "Switched to dashboard view"}
}

func (s *S9s) cmdHealth(args []string) CommandResult {
	s.switchToView("health")
	return CommandResult{Success: true, Message: "Switched to health view"}
}

func (s *S9s) cmdPerformance(args []string) CommandResult {
	s.switchToView("performance")
	return CommandResult{Success: true, Message: "Switched to performance view"}
}

func (s *S9s) cmdHelp(args []string) CommandResult {
	s.showHelp()
	return CommandResult{Success: true, Message: "Showing help"}
}

func (s *S9s) cmdPrefs(args []string) CommandResult {
	s.showPreferences()
	return CommandResult{Success: true, Message: "Showing preferences"}
}

func (s *S9s) cmdLayout(args []string) CommandResult {
	s.showLayoutSwitcher()
	return CommandResult{Success: true, Message: "Showing layout switcher"}
}

func (s *S9s) cmdConfig(args []string) CommandResult {
	s.showConfiguration()
	return CommandResult{Success: true, Message: "Showing configuration"}
}

func (s *S9s) cmdRefresh(args []string) CommandResult {
	s.handleRefreshCommand()
	return CommandResult{Success: true, Message: "Refreshing..."}
}

// handleRefreshCommand handles the refresh command
func (s *S9s) handleRefreshCommand() {
	currentView, err := s.viewMgr.GetCurrentView()
	if err != nil {
		return
	}

	go func() {
		if err := currentView.Refresh(); err != nil {
			s.statusBar.Error(fmt.Sprintf("Refresh failed: %v", err))
		} else {
			s.statusBar.Success("Refreshed")
			// Restore hints after success message expires
			time.Sleep(3500 * time.Millisecond)
			if cv, err := s.viewMgr.GetCurrentView(); err == nil {
				s.statusBar.SetHints(cv.Hints())
			}
		}
	}()
}
