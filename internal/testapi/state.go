// Package testapi provides test utilities and application state export for testing.
package testapi

import (
	"encoding/json"

	"github.com/jontk/s9s/internal/app"
)

// AppState represents the exportable state of the application
type AppState struct {
	CurrentView string   `json:"current_view"`
	ModalOpen   bool     `json:"modal_open"`
	ModalName   string   `json:"modal_name"`
	CmdVisible  bool     `json:"cmd_visible"`
	ViewNames   []string `json:"view_names"`
}

// StateExporter provides methods to export application state for testing
type StateExporter struct {
	app *app.S9s
}

// NewStateExporter creates a new state exporter for the given app
func NewStateExporter(s9sApp *app.S9s) *StateExporter {
	return &StateExporter{
		app: s9sApp,
	}
}

// GetState returns the current application state
func (e *StateExporter) GetState() *AppState {
	state := &AppState{
		CurrentView: e.app.GetCurrentViewName(),
		ModalOpen:   e.app.IsModalOpen(),
		ModalName:   e.app.GetModalName(),
		CmdVisible:  e.app.IsCmdVisible(),
	}

	// Get view names from view manager
	if viewMgr := e.app.GetViewManager(); viewMgr != nil {
		state.ViewNames = viewMgr.GetViewNames()
	}

	return state
}

// GetStateJSON returns the current application state as JSON
func (e *StateExporter) GetStateJSON() (string, error) {
	state := e.GetState()
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GetCurrentView returns the name of the current view
func (e *StateExporter) GetCurrentView() string {
	return e.app.GetCurrentViewName()
}

// IsModalOpen returns whether a modal is open
func (e *StateExporter) IsModalOpen() bool {
	return e.app.IsModalOpen()
}

// GetModalName returns the name of the open modal
func (e *StateExporter) GetModalName() string {
	return e.app.GetModalName()
}

// IsCmdVisible returns whether the command line is visible
func (e *StateExporter) IsCmdVisible() bool {
	return e.app.IsCmdVisible()
}

// GetViewNames returns all available view names
func (e *StateExporter) GetViewNames() []string {
	if viewMgr := e.app.GetViewManager(); viewMgr != nil {
		return viewMgr.GetViewNames()
	}
	return nil
}
