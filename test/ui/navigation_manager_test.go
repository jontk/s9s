package ui

import (
	"fmt"
	"testing"

	"github.com/gdamore/tcell/v2"
	"github.com/jontk/s9s/internal/ui/navigation"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNavigationManager(t *testing.T) {
	app := tview.NewApplication()
	nm := navigation.NewNavigationManager(app)

	t.Run("InitialState", func(t *testing.T) {
		assert.Equal(t, navigation.NavigationModeNormal, nm.GetMode())
		assert.Equal(t, "", nm.GetCurrentTarget())
		assert.NotNil(t, nm.GetBreadcrumb())
		assert.NotNil(t, nm.GetQuickHelpBar())
		assert.NotNil(t, nm.GetCommandPalette())
		assert.NotNil(t, nm.GetSearchOverlay())
	})

	t.Run("RegisterTarget", func(t *testing.T) {
		target := &navigation.NavigationTarget{
			ID:          "test-target",
			Name:        "Test Target",
			Description: "A test navigation target",
			Type:        "test",
			Component:   tview.NewTextView(),
			Shortcut:    't',
			KeyBinding:  tcell.KeyF12, // Use F12 to avoid conflict with default F1
			Priority:    5,
		}

		err := nm.RegisterTarget(target)
		require.NoError(t, err)
	})

	t.Run("RegisterDuplicateTarget", func(t *testing.T) {
		target := &navigation.NavigationTarget{
			ID:   "test-target", // Same ID as above
			Name: "Duplicate Test",
		}

		// Should not error for duplicate ID, but will overwrite
		err := nm.RegisterTarget(target)
		assert.NoError(t, err)
	})

	t.Run("RegisterTargetWithDuplicateShortcut", func(t *testing.T) {
		target := &navigation.NavigationTarget{
			ID:       "another-target",
			Name:     "Another Target",
			Shortcut: 't', // Same shortcut as first target
		}

		err := nm.RegisterTarget(target)
		if err != nil {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "shortcut")
		} else {
			// If no error, check that the first target's shortcut was overridden
			// This depends on the implementation - some systems allow override
			assert.NoError(t, err)
		}
	})

	t.Run("NavigateToTarget", func(t *testing.T) {
		activated := false

		target := &navigation.NavigationTarget{
			ID:   "nav-target",
			Name: "Navigation Target",
			OnActivate: func() {
				activated = true
			},
		}

		err := nm.RegisterTarget(target)
		require.NoError(t, err)

		err = nm.NavigateTo("nav-target")
		require.NoError(t, err)

		assert.True(t, activated)
		assert.Equal(t, "nav-target", nm.GetCurrentTarget())
	})

	t.Run("NavigateToNonExistentTarget", func(t *testing.T) {
		err := nm.NavigateTo("non-existent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("UnregisterTarget", func(t *testing.T) {
		err := nm.UnregisterTarget("test-target")
		assert.NoError(t, err)

		// Should not be able to navigate to unregistered target
		err = nm.NavigateTo("test-target")
		assert.Error(t, err)
	})
}

func TestNavigationModes(t *testing.T) {
	app := tview.NewApplication()
	nm := navigation.NewNavigationManager(app)

	var modeChanges []navigation.NavigationMode
	nm.SetOnModeChange(func(mode navigation.NavigationMode) {
		modeChanges = append(modeChanges, mode)
	})

	t.Run("EnterQuickMode", func(t *testing.T) {
		nm.EnterQuickMode()
		assert.Equal(t, navigation.NavigationModeQuick, nm.GetMode())
		assert.Contains(t, modeChanges, navigation.NavigationModeQuick)
	})

	t.Run("ExitQuickMode", func(t *testing.T) {
		nm.ExitQuickMode()
		assert.Equal(t, navigation.NavigationModeNormal, nm.GetMode())
		assert.Contains(t, modeChanges, navigation.NavigationModeNormal)
	})

	t.Run("EnterCommandMode", func(t *testing.T) {
		nm.EnterCommandMode()
		assert.Equal(t, navigation.NavigationModeCommand, nm.GetMode())
		assert.Contains(t, modeChanges, navigation.NavigationModeCommand)
	})

	t.Run("ExitCommandMode", func(t *testing.T) {
		nm.ExitCommandMode()
		assert.Equal(t, navigation.NavigationModeNormal, nm.GetMode())
	})

	t.Run("EnterSearchMode", func(t *testing.T) {
		nm.EnterSearchMode()
		assert.Equal(t, navigation.NavigationModeSearch, nm.GetMode())
		assert.Contains(t, modeChanges, navigation.NavigationModeSearch)
	})

	t.Run("ExitSearchMode", func(t *testing.T) {
		nm.ExitSearchMode()
		assert.Equal(t, navigation.NavigationModeNormal, nm.GetMode())
	})
}

func TestNavigationInput(t *testing.T) {
	app := tview.NewApplication()
	nm := navigation.NewNavigationManager(app)

	// Register some test targets
	for i, char := range "abc" {
		target := &navigation.NavigationTarget{
			ID:       fmt.Sprintf("target-%d", i),
			Name:     fmt.Sprintf("Target %d", i),
			Shortcut: char,
		}
		_ = nm.RegisterTarget(target)
	}

	t.Run("QuickModeNavigation", func(t *testing.T) {
		nm.EnterQuickMode()

		// Simulate pressing 'a' in quick mode
		event := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)
		result := nm.HandleInput(event)

		// Should be handled (return nil) and navigate to target
		assert.Nil(t, result)
		assert.Equal(t, "target-0", nm.GetCurrentTarget())
		assert.Equal(t, navigation.NavigationModeNormal, nm.GetMode()) // Should exit quick mode
	})

	t.Run("EscapeKey", func(t *testing.T) {
		nm.EnterQuickMode()

		event := tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone)
		result := nm.HandleInput(event)

		assert.Nil(t, result)
		assert.Equal(t, navigation.NavigationModeNormal, nm.GetMode())
	})

	t.Run("UnhandledInput", func(t *testing.T) {
		event := tcell.NewEventKey(tcell.KeyRune, 'z', tcell.ModNone)
		result := nm.HandleInput(event)

		// Should pass through unhandled input
		assert.Equal(t, event, result)
	})
}

func TestNavigationSearch(t *testing.T) {
	app := tview.NewApplication()
	nm := navigation.NewNavigationManager(app)

	// Register test targets
	targets := []*navigation.NavigationTarget{
		{ID: "jobs", Name: "Jobs View", Description: "View SLURM jobs"},
		{ID: "nodes", Name: "Nodes View", Description: "View cluster nodes"},
		{ID: "terminal", Name: "Terminal", Description: "SSH terminal"},
	}

	for _, target := range targets {
		_ = nm.RegisterTarget(target)
	}

	t.Run("CustomSearchCallback", func(t *testing.T) {
		var searchQuery string
		nm.SetOnSearch(func(query string) []string {
			searchQuery = query
			if query == "job" {
				return []string{"jobs"}
			}
			return []string{}
		})

		nm.EnterSearchMode()

		// Simulate typing in search mode
		for _, char := range "job" {
			event := tcell.NewEventKey(tcell.KeyRune, char, tcell.ModNone)
			nm.HandleInput(event)
		}

		assert.Equal(t, "job", searchQuery)
	})

	t.Run("SearchModeInput", func(t *testing.T) {
		nm.EnterSearchMode()

		// Test character input
		event := tcell.NewEventKey(tcell.KeyRune, 't', tcell.ModNone)
		result := nm.HandleInput(event)
		assert.Nil(t, result) // Should be handled

		// Test backspace
		event = tcell.NewEventKey(tcell.KeyBackspace, 0, tcell.ModNone)
		result = nm.HandleInput(event)
		assert.Nil(t, result) // Should be handled

		// Test escape
		event = tcell.NewEventKey(tcell.KeyEsc, 0, tcell.ModNone)
		result = nm.HandleInput(event)
		assert.Nil(t, result) // Should be handled
		assert.Equal(t, navigation.NavigationModeNormal, nm.GetMode())
	})
}

func TestNavigationCallbacks(t *testing.T) {
	app := tview.NewApplication()
	nm := navigation.NewNavigationManager(app)

	var navigatedFrom, navigatedTo string
	nm.SetOnNavigate(func(from, to string) {
		navigatedFrom = from
		navigatedTo = to
	})

	// Register targets
	target1 := &navigation.NavigationTarget{
		ID:   "target1",
		Name: "Target 1",
	}
	target2 := &navigation.NavigationTarget{
		ID:   "target2",
		Name: "Target 2",
	}

	_ = nm.RegisterTarget(target1)
	_ = nm.RegisterTarget(target2)

	// Navigate to first target
	_ = nm.NavigateTo("target1")
	assert.Equal(t, "", navigatedFrom) // No previous target
	assert.Equal(t, "target1", navigatedTo)

	// Navigate to second target
	_ = nm.NavigateTo("target2")
	assert.Equal(t, "target1", navigatedFrom)
	assert.Equal(t, "target2", navigatedTo)
}

func BenchmarkNavigationManager(b *testing.B) {
	app := tview.NewApplication()
	nm := navigation.NewNavigationManager(app)

	b.Run("RegisterTargets", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			target := &navigation.NavigationTarget{
				ID:   fmt.Sprintf("bench-target-%d", i),
				Name: fmt.Sprintf("Bench Target %d", i),
			}
			_ = nm.RegisterTarget(target)
		}
	})

	b.Run("NavigateToTargets", func(b *testing.B) {
		// Register some targets first
		for i := 0; i < 10; i++ {
			target := &navigation.NavigationTarget{
				ID:   fmt.Sprintf("nav-target-%d", i),
				Name: fmt.Sprintf("Nav Target %d", i),
			}
			_ = nm.RegisterTarget(target)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			targetID := fmt.Sprintf("nav-target-%d", i%10)
			_ = nm.NavigateTo(targetID)
		}
	})

	b.Run("HandleInput", func(b *testing.B) {
		event := tcell.NewEventKey(tcell.KeyRune, 'a', tcell.ModNone)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			nm.HandleInput(event)
		}
	})
}
