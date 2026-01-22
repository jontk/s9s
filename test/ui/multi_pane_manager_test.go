package ui

import (
	"fmt"
	"testing"

	"github.com/jontk/s9s/internal/ui/components"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMultiPaneManager(t *testing.T) {
	app := tview.NewApplication()
	mpm := components.NewMultiPaneManager(app)

	t.Run("InitialState", func(t *testing.T) {
		assert.Equal(t, 0, mpm.GetPaneCount())
		assert.Nil(t, mpm.GetActivePane())
		assert.NotNil(t, mpm.GetContainer())
	})

	t.Run("AddPane", func(t *testing.T) {
		content := tview.NewTextView()
		content.SetText("Test content")

		pane := &components.Pane{
			ID:        "test1",
			Title:     "Test Pane 1",
			Type:      components.PaneTypeView,
			Content:   content,
			Closable:  true,
			Resizable: true,
		}

		err := mpm.AddPane(pane)
		require.NoError(t, err)

		assert.Equal(t, 1, mpm.GetPaneCount())
		activePane := mpm.GetActivePane()
		require.NotNil(t, activePane)
		assert.Equal(t, "test1", activePane.ID)
		assert.True(t, activePane.Active)
	})

	t.Run("AddMultiplePanes", func(t *testing.T) {
		// Add second pane
		content2 := tview.NewTextView()
		content2.SetText("Test content 2")

		pane2 := &components.Pane{
			ID:      "test2",
			Title:   "Test Pane 2",
			Type:    components.PaneTypeTerminal,
			Content: content2,
		}

		err := mpm.AddPane(pane2)
		require.NoError(t, err)

		assert.Equal(t, 2, mpm.GetPaneCount())

		// First pane should still be active
		activePane := mpm.GetActivePane()
		require.NotNil(t, activePane)
		assert.Equal(t, "test1", activePane.ID)
	})

	t.Run("NavigatePanes", func(t *testing.T) {
		// Switch to next pane
		mpm.NextPane()

		activePane := mpm.GetActivePane()
		require.NotNil(t, activePane)
		assert.Equal(t, "test2", activePane.ID)

		// Switch back to previous pane
		mpm.PreviousPane()

		activePane = mpm.GetActivePane()
		require.NotNil(t, activePane)
		assert.Equal(t, "test1", activePane.ID)
	})

	t.Run("SwitchToPaneByIndex", func(t *testing.T) {
		mpm.SwitchToPane(1)

		activePane := mpm.GetActivePane()
		require.NotNil(t, activePane)
		assert.Equal(t, "test2", activePane.ID)
	})

	t.Run("RemovePane", func(t *testing.T) {
		err := mpm.RemovePane("test1")
		require.NoError(t, err)

		assert.Equal(t, 1, mpm.GetPaneCount())

		// Should switch to remaining pane
		activePane := mpm.GetActivePane()
		require.NotNil(t, activePane)
		assert.Equal(t, "test2", activePane.ID)
	})

	t.Run("NewPaneCreation", func(t *testing.T) {
		err := mpm.NewPane("", "Terminal", components.PaneTypeTerminal, nil)
		require.NoError(t, err)

		assert.Equal(t, 2, mpm.GetPaneCount())
	})

	t.Run("LayoutSwitching", func(t *testing.T) {
		mpm.SetLayout(components.LayoutHorizontal)
		mpm.SetLayout(components.LayoutVertical)
		mpm.SetLayout(components.LayoutGrid)
		mpm.SetLayout(components.LayoutTabs)

		// Should not crash and should be able to switch layouts
		assert.Equal(t, 2, mpm.GetPaneCount())
	})

	t.Run("MaxPanesLimit", func(t *testing.T) {
		// Set a low limit for testing
		originalMax := 8

		// Add panes up to the limit
		for j := 0; j < originalMax-2; j++ {
			err := mpm.NewPane("", fmt.Sprintf("Pane %d", j+3), components.PaneTypeView, nil)
			if err != nil {
				// We hit the limit
				break
			}
		}

		// Try to add one more pane beyond the limit
		err := mpm.NewPane("", "Over Limit", components.PaneTypeView, nil)
		if mpm.GetPaneCount() >= originalMax {
			assert.Error(t, err)
		}
	})

	t.Run("CleanupAllPanes", func(t *testing.T) {
		mpm.Close()
		assert.Equal(t, 0, mpm.GetPaneCount())
		assert.Nil(t, mpm.GetActivePane())
	})
}

func TestPaneTypes(t *testing.T) {
	paneTypes := []components.PaneType{
		components.PaneTypeView,
		components.PaneTypeTerminal,
		components.PaneTypeLog,
		components.PaneTypeMonitor,
		components.PaneTypeEditor,
	}

	for i, paneType := range paneTypes {
		t.Run(string(paneType), func(t *testing.T) {
			// Create fresh manager for each test
			app := tview.NewApplication()
			mpm := components.NewMultiPaneManager(app)

			err := mpm.NewPane(fmt.Sprintf("test-%d", i), fmt.Sprintf("Test %s", paneType), paneType, nil)
			require.NoError(t, err)

			pane := mpm.GetActivePane()
			require.NotNil(t, pane)
			assert.Equal(t, paneType, pane.Type, "Expected pane type %s but got %s", paneType, pane.Type)
		})
	}
}

func TestPaneCallbacks(t *testing.T) {
	app := tview.NewApplication()
	mpm := components.NewMultiPaneManager(app)

	var closeCalled bool
	var activateCalled bool
	var switchFromPane, switchToPane string

	// Set up callbacks
	mpm.SetOnPaneSwitch(func(from, to string) {
		switchFromPane = from
		switchToPane = to
	})

	// Create pane with callbacks
	content := tview.NewTextView()
	pane := &components.Pane{
		ID:      "callback-test",
		Title:   "Callback Test",
		Type:    components.PaneTypeView,
		Content: content,
		OnClose: func(id string) error {
			closeCalled = true
			return nil
		},
		OnActivate: func(id string) {
			activateCalled = true
		},
	}

	err := mpm.AddPane(pane)
	require.NoError(t, err)

	// Activate callback should be called
	assert.True(t, activateCalled)

	// Add another pane and switch to test pane switch callback
	err = mpm.NewPane("second", "Second", components.PaneTypeView, nil)
	require.NoError(t, err)

	// Switch to second pane first (index 1)
	mpm.SwitchToPane(1)

	// Now switch back to first pane (index 0)
	mpm.SwitchToPane(0)

	assert.Equal(t, "second", switchFromPane)
	assert.Equal(t, "callback-test", switchToPane)

	// Remove pane to test close callback
	err = mpm.RemovePane("callback-test")
	require.NoError(t, err)

	assert.True(t, closeCalled)
}

func TestKeyboardShortcuts(t *testing.T) {
	app := tview.NewApplication()
	mpm := components.NewMultiPaneManager(app)

	// Add some panes with unique IDs
	for i := 0; i < 3; i++ {
		err := mpm.NewPane(fmt.Sprintf("shortcut-pane-%d", i), fmt.Sprintf("Pane %d", i+1), components.PaneTypeView, nil)
		require.NoError(t, err)
	}

	t.Run("ManualNavigation", func(t *testing.T) {
		// Test programmatic navigation instead of simulating key events
		initialPane := mpm.GetActivePane()
		require.NotNil(t, initialPane)

		// Navigate to next pane
		mpm.NextPane()
		nextPane := mpm.GetActivePane()

		// Should have switched to a different pane if multiple panes exist
		if mpm.GetPaneCount() > 1 {
			assert.NotEqual(t, initialPane.ID, nextPane.ID)
		}

		// Navigate to previous pane
		mpm.PreviousPane()
		prevPane := mpm.GetActivePane()

		// Should be back to original pane
		assert.Equal(t, initialPane.ID, prevPane.ID)
	})

	t.Run("PaneCreationAndClosure", func(t *testing.T) {
		initialCount := mpm.GetPaneCount()

		// Create a new pane programmatically
		err := mpm.NewPane("test-new", "Test New", components.PaneTypeView, nil)
		assert.NoError(t, err)

		// Should have added a new pane
		assert.Equal(t, initialCount+1, mpm.GetPaneCount())

		// Close the active pane
		mpm.CloseActivePane()

		// Should have removed a pane
		assert.Equal(t, initialCount, mpm.GetPaneCount())
	})
}

func BenchmarkPaneManagement(b *testing.B) {
	app := tview.NewApplication()
	mpm := components.NewMultiPaneManager(app)

	b.Run("AddPanes", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if mpm.GetPaneCount() >= 8 {
				mpm.Close()
			}
			_ = mpm.NewPane("", fmt.Sprintf("Bench Pane %d", i), components.PaneTypeView, nil)
		}
	})

	b.Run("SwitchPanes", func(b *testing.B) {
		// Add some panes first
		for i := 0; i < 5; i++ {
			_ = mpm.NewPane("", fmt.Sprintf("Switch Pane %d", i), components.PaneTypeView, nil)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			mpm.NextPane()
		}
	})

	b.Run("LayoutSwitching", func(b *testing.B) {
		layouts := []components.PaneLayout{
			components.LayoutTabs,
			components.LayoutHorizontal,
			components.LayoutVertical,
			components.LayoutGrid,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			layout := layouts[i%len(layouts)]
			mpm.SetLayout(layout)
		}
	})
}
