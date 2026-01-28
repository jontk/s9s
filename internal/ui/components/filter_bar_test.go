package components

import (
	"testing"

	"github.com/jontk/s9s/internal/ui/filters"
	"github.com/rivo/tview"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFilterBar(t *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)

	require.NotNil(t, fb)
	assert.NotNil(t, fb.Flex)
	assert.NotNil(t, fb.input)
	assert.NotNil(t, fb.helpText)
	assert.NotNil(t, fb.parser)
	assert.NotNil(t, fb.presetManager)
	assert.Equal(t, "jobs", fb.viewType)
	assert.Equal(t, app, fb.app)
	assert.Nil(t, fb.currentFilter)
}

func TestFilterBarSetPages(t *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)
	pages := tview.NewPages()

	fb.SetPages(pages)

	assert.Equal(t, pages, fb.pages)
}

func TestFilterBarSetOnFilterChange(t *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)

	callbackCalled := false
	var receivedFilter *filters.Filter

	fb.SetOnFilterChange(func(filter *filters.Filter) {
		callbackCalled = true
		receivedFilter = filter
	})

	assert.NotNil(t, fb.onFilterChange)

	// Trigger the callback
	testFilter := &filters.Filter{}
	fb.onFilterChange(testFilter)

	assert.True(t, callbackCalled)
	assert.Equal(t, testFilter, receivedFilter)
}

func TestFilterBarSetOnClose(t *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)

	callbackCalled := false

	fb.SetOnClose(func() {
		callbackCalled = true
	})

	assert.NotNil(t, fb.onClose)

	// Trigger the callback
	fb.onClose()

	assert.True(t, callbackCalled)
}

func TestFilterBarGetCurrentFilter(t *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)

	// Initially nil
	assert.Nil(t, fb.GetCurrentFilter())

	// After setting a filter
	testFilter := &filters.Filter{}
	fb.currentFilter = testFilter

	assert.Equal(t, testFilter, fb.GetCurrentFilter())
}

func TestFilterBarHide(t *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)

	callbackCalled := false
	fb.SetOnClose(func() {
		callbackCalled = true
	})

	fb.Hide()

	assert.True(t, callbackCalled)
}

func TestFilterBarHideWithoutCallback(_ *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)

	// Should not panic when onClose is nil
	fb.Hide()
}

func TestFilterBarGetExamplesForView(t *testing.T) {
	tests := []struct {
		name              string
		viewType          string
		expectedSubstring string
	}{
		{
			name:              "jobs view has job-specific examples",
			viewType:          "jobs",
			expectedSubstring: "state=running",
		},
		{
			name:              "nodes view has node-specific examples",
			viewType:          "nodes",
			expectedSubstring: "state=idle",
		},
		{
			name:              "partitions view has partition-specific examples",
			viewType:          "partitions",
			expectedSubstring: "state=up",
		},
		{
			name:              "unknown view has generic examples",
			viewType:          "unknown",
			expectedSubstring: "field=value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := tview.NewApplication()
			fb := NewFilterBar(tt.viewType, app)

			examples := fb.getExamplesForView()

			assert.Contains(t, examples, tt.expectedSubstring)
		})
	}
}

func TestFilterBarGetHelpText(t *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)

	helpText := fb.getHelpText()

	assert.Contains(t, helpText, "Examples:")
	assert.Contains(t, helpText, "Tab")
	assert.Contains(t, helpText, "F1")
}

func TestCreateCenteredModal(t *testing.T) {
	content := tview.NewTextView()

	modal := createCenteredModal(content, 60, 20)

	require.NotNil(t, modal)
	// Modal should be a flex container
	_, ok := modal.(*tview.Flex)
	assert.True(t, ok, "createCenteredModal should return a Flex container")
}

func TestFilterBarSetFilter(t *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)

	filterCallbackCalled := false
	var receivedFilter *filters.Filter

	fb.SetOnFilterChange(func(filter *filters.Filter) {
		filterCallbackCalled = true
		receivedFilter = filter
	})

	// Set a valid filter
	fb.SetFilter("state=running")

	// Input should be set
	assert.Equal(t, "state=running", fb.input.GetText())

	// Callback should have been called with parsed filter
	assert.True(t, filterCallbackCalled)
	assert.NotNil(t, receivedFilter)
	assert.NotNil(t, fb.currentFilter)
}

func TestFilterBarSetFilterInvalid(t *testing.T) {
	app := tview.NewApplication()
	fb := NewFilterBar("jobs", app)

	// Set an invalid filter (empty is typically invalid for parsed filters)
	// The behavior depends on the parser implementation
	fb.SetFilter("")

	// Input should be set even if filter is invalid
	assert.Equal(t, "", fb.input.GetText())
}
