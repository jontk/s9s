package plugins

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/rivo/tview"
)

// Plugin represents a s9s plugin
type Plugin interface {
	// GetInfo returns basic plugin information
	GetInfo() PluginInfo

	// Initialize initializes the plugin with the given context and client
	Initialize(ctx context.Context, client dao.SlurmClient) error

	// GetCommands returns the commands this plugin provides
	GetCommands() []Command

	// GetViews returns the views this plugin provides
	GetViews() []View

	// GetKeyBindings returns custom key bindings this plugin provides
	GetKeyBindings() []KeyBinding

	// OnEvent is called when various events occur in the application
	OnEvent(event Event) error

	// Cleanup is called when the plugin is being unloaded
	Cleanup() error
}

// PluginInfo contains basic information about a plugin
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	Author      string
	Website     string
}

// Command represents a plugin command
type Command struct {
	Name        string
	Description string
	Usage       string
	Handler     func(args []string) error
}

// View represents a plugin-provided view
type View interface {
	// GetName returns the view name
	GetName() string

	// GetTitle returns the view title
	GetTitle() string

	// Render returns the tview primitive for this view
	Render() tview.Primitive

	// OnKey handles key events
	OnKey(event *tcell.EventKey) *tcell.EventKey

	// Refresh updates the view data
	Refresh() error

	// Init initializes the view
	Init(ctx context.Context) error
}

// KeyBinding represents a custom key binding
type KeyBinding struct {
	Key         rune
	Modifiers   tcell.ModMask
	Description string
	Handler     func() error
}

// Event represents various events in the application
type Event struct {
	Type EventType
	Data interface{}
}

// EventType represents the type of event
type EventType int

const (
	EventViewChanged EventType = iota
	EventJobSubmitted
	EventJobCompleted
	EventNodeStateChanged
	EventClusterHealthChanged
)

// PluginManager manages all loaded plugins
type PluginManager interface {
	// LoadPlugin loads a plugin from the given path
	LoadPlugin(path string) error

	// LoadPluginsFromDirectory loads all plugins from a directory
	LoadPluginsFromDirectory(dir string) error

	// GetPlugin returns a plugin by name
	GetPlugin(name string) Plugin

	// GetAllPlugins returns all loaded plugins
	GetAllPlugins() []Plugin

	// GetCommands returns all commands from all plugins
	GetCommands() []Command

	// GetViews returns all views from all plugins
	GetViews() []View

	// GetKeyBindings returns all key bindings from all plugins
	GetKeyBindings() []KeyBinding

	// SendEvent sends an event to all plugins
	SendEvent(event Event) error

	// UnloadPlugin unloads a plugin
	UnloadPlugin(name string) error

	// UnloadAllPlugins unloads all plugins
	UnloadAllPlugins() error
}
