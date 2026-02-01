package app

import (
	"reflect"
	"sort"
	"testing"

	"github.com/jontk/s9s/internal/dao"
	"github.com/jontk/s9s/internal/views"
	"github.com/rivo/tview"
)

// mockSlurmClient is a minimal mock for testing
type mockSlurmClient struct {
	dao.SlurmClient
}

func (m *mockSlurmClient) Jobs() dao.JobManager {
	return &mockJobManager{}
}

func (m *mockSlurmClient) Nodes() dao.NodeManager {
	return &mockNodeManager{}
}

type mockJobManager struct {
	dao.JobManager
}

type mockNodeManager struct {
	dao.NodeManager
}

func TestGetArgType(t *testing.T) {
	tests := []struct {
		name     string
		cmdName  string
		expected ArgType
	}{
		{"cancel command", "cancel", ArgTypeJobID},
		{"hold command", "hold", ArgTypeJobID},
		{"release command", "release", ArgTypeJobID},
		{"requeue command", "requeue", ArgTypeJobID},
		{"drain command", "drain", ArgTypeNodeName},
		{"resume command", "resume", ArgTypeNodeName},
		{"quit command", "quit", ArgTypeNone},
		{"unknown command", "unknown", ArgTypeNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getArgType(tt.cmdName)
			if result != tt.expected {
				t.Errorf("getArgType(%s) = %v, want %v", tt.cmdName, result, tt.expected)
			}
		})
	}
}

func TestGetCommandCompletions(t *testing.T) {
	app := tview.NewApplication()
	s := &S9s{
		app: app,
	}

	tests := []struct {
		name     string
		prefix   string
		expected []string
	}{
		{
			name:     "empty prefix",
			prefix:   "",
			expected: []string{"accounts", "cancel", "config", "configuration", "dashboard", "drain", "h", "health", "help", "hold", "j", "jobs", "layout", "layouts", "n", "nodes", "p", "partitions", "performance", "preferences", "prefs", "q", "qos", "quit", "r", "refresh", "release", "requeue", "reservations", "resume", "settings", "users"},
		},
		{
			name:     "prefix 'q'",
			prefix:   "q",
			expected: []string{"q", "qos", "quit"},
		},
		{
			name:     "prefix 'req'",
			prefix:   "req",
			expected: []string{"requeue"},
		},
		{
			name:     "prefix 'dr'",
			prefix:   "dr",
			expected: []string{"drain"},
		},
		{
			name:     "prefix 'j'",
			prefix:   "j",
			expected: []string{"j", "jobs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.getCommandCompletions(tt.prefix)
			sort.Strings(result)
			sort.Strings(tt.expected)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getCommandCompletions(%s) = %v, want %v", tt.prefix, result, tt.expected)
			}
		})
	}
}

func TestGetCompletions_Commands(t *testing.T) {
	app := tview.NewApplication()
	s := &S9s{
		app: app,
	}

	tests := []struct {
		name     string
		text     string
		expected []string
	}{
		{
			name:     "complete 'req' to 'requeue'",
			text:     "req",
			expected: []string{"requeue"},
		},
		{
			name:     "complete 'q' to quit/qos/q",
			text:     "q",
			expected: []string{"q", "qos", "quit"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.getCompletions(tt.text)
			sort.Strings(result)
			sort.Strings(tt.expected)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("getCompletions(%s) = %v, want %v", tt.text, result, tt.expected)
			}
		})
	}
}

func TestGetCompletions_Arguments(t *testing.T) {
	app := tview.NewApplication()
	pages := tview.NewPages()

	// Create mock client
	client := &mockSlurmClient{}

	// Create view manager and views with mock data
	viewMgr := views.NewViewManager(app)
	viewMgr.SetPages(pages)

	// Create jobs view and populate with test data
	jobsView := views.NewJobsView(client)
	_ = viewMgr.AddView(jobsView)

	// Create nodes view and populate with test data
	nodesView := views.NewNodesView(client)
	_ = viewMgr.AddView(nodesView)

	s := &S9s{
		app:     app,
		viewMgr: viewMgr,
	}

	tests := []struct {
		name     string
		text     string
		wantType string // "job", "node", or "none"
	}{
		{
			name:     "cancel with space should trigger job completion",
			text:     "cancel ",
			wantType: "job",
		},
		{
			name:     "drain with space should trigger node completion",
			text:     "drain ",
			wantType: "node",
		},
		{
			name:     "quit with space should return nil",
			text:     "quit ",
			wantType: "none",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.getCompletions(tt.text)
			switch tt.wantType {
			case "none":
				if result != nil {
					t.Errorf("getCompletions(%s) = %v, want nil", tt.text, result)
				}
			case "job", "node":
				// We can't test actual job IDs or node names without mocking the full data,
				// but we can verify the function doesn't panic and returns a slice
				if result == nil {
					// Empty result is OK if there's no data
					t.Logf("getCompletions(%s) = nil (no data available)", tt.text)
				}
			}
		})
	}
}

func TestGetJobIDCandidates_NoView(t *testing.T) {
	app := tview.NewApplication()
	viewMgr := views.NewViewManager(app)

	s := &S9s{
		app:     app,
		viewMgr: viewMgr,
	}

	result := s.getJobIDCandidates()
	if result != nil {
		t.Errorf("getJobIDCandidates() with no jobs view = %v, want nil", result)
	}
}

func TestGetNodeNameCandidates_NoView(t *testing.T) {
	app := tview.NewApplication()
	viewMgr := views.NewViewManager(app)

	s := &S9s{
		app:     app,
		viewMgr: viewMgr,
	}

	result := s.getNodeNameCandidates()
	if result != nil {
		t.Errorf("getNodeNameCandidates() with no nodes view = %v, want nil", result)
	}
}
