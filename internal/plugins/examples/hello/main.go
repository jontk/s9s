// +build plugin

package main

import (
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/plugins"
	"github.com/rivo/tview"
)

// HelloPlugin is an example plugin that demonstrates the plugin interface
type HelloPlugin struct {
	client dao.SlurmClient
}

// NewPlugin is the plugin entry point - this function must be exported
func NewPlugin() plugins.Plugin {
	return &HelloPlugin{}
}

// GetInfo returns basic plugin information
func (p *HelloPlugin) GetInfo() plugins.PluginInfo {
	return plugins.PluginInfo{
		Name:        "hello",
		Version:     "1.0.0",
		Description: "Example Hello World plugin for s9s",
		Author:      "s9s Team",
		Website:     "https://s9s.dev",
	}
}

// Initialize initializes the plugin
func (p *HelloPlugin) Initialize(ctx context.Context, client dao.SlurmClient) error {
	p.client = client
	fmt.Println("Hello Plugin initialized!")
	return nil
}

// GetCommands returns the commands this plugin provides
func (p *HelloPlugin) GetCommands() []plugins.Command {
	return []plugins.Command{
		{
			Name:        "hello",
			Description: "Say hello to the world",
			Usage:       "hello [name]",
			Handler: func(args []string) error {
				name := "World"
				if len(args) > 0 {
					name = args[0]
				}
				fmt.Printf("Hello, %s!\n", name)
				return nil
			},
		},
		{
			Name:        "cluster-info",
			Description: "Show cluster information",
			Usage:       "cluster-info",
			Handler: func(args []string) error {
				if p.client != nil {
					info, err := p.client.ClusterInfo()
					if err != nil {
						return err
					}
					fmt.Printf("Cluster: %s\n", info.Name)
					fmt.Printf("Version: %s\n", info.Version)
				}
				return nil
			},
		},
	}
}

// GetViews returns the views this plugin provides
func (p *HelloPlugin) GetViews() []plugins.View {
	return []plugins.View{
		&HelloView{},
	}
}

// GetKeyBindings returns custom key bindings
func (p *HelloPlugin) GetKeyBindings() []plugins.KeyBinding {
	return []plugins.KeyBinding{
		{
			Key:         'H',
			Description: "Show Hello message",
			Handler: func() error {
				fmt.Println("Hello from key binding!")
				return nil
			},
		},
	}
}

// OnEvent handles events from the application
func (p *HelloPlugin) OnEvent(event plugins.Event) error {
	switch event.Type {
	case plugins.EventViewChanged:
		fmt.Printf("View changed to: %v\n", event.Data)
	case plugins.EventJobSubmitted:
		fmt.Printf("Job submitted: %v\n", event.Data)
	}
	return nil
}

// Cleanup is called when the plugin is being unloaded
func (p *HelloPlugin) Cleanup() error {
	fmt.Println("Hello Plugin cleaned up!")
	return nil
}

// HelloView is an example view provided by the plugin
type HelloView struct {
	content *tview.TextView
}

// GetName returns the view name
func (v *HelloView) GetName() string {
	return "hello"
}

// GetTitle returns the view title
func (v *HelloView) GetTitle() string {
	return "Hello Plugin View"
}

// Render returns the tview primitive for this view
func (v *HelloView) Render() tview.Primitive {
	if v.content == nil {
		v.content = tview.NewTextView()
		v.content.SetBorder(true).SetTitle("Hello Plugin")
		v.content.SetText("Hello from the plugin system!\n\nThis is an example view provided by a plugin.\n\nPress 'q' to go back.")
	}
	return v.content
}

// OnKey handles key events
func (v *HelloView) OnKey(event *tcell.EventKey) *tcell.EventKey {
	switch event.Rune() {
	case 'q', 'Q':
		// Signal to close this view
		return nil
	}
	return event
}

// Refresh updates the view data
func (v *HelloView) Refresh() error {
	if v.content != nil {
		v.content.SetText(fmt.Sprintf("Hello from the plugin system!\n\nRefreshed at: %v\n\nPress 'q' to go back.", "now"))
	}
	return nil
}

// Init initializes the view
func (v *HelloView) Init(ctx context.Context) error {
	return nil
}

// Main function is required for plugin compilation but not used
func main() {
	// This is a plugin, main() will never be called
}