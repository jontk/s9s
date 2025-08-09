# s9s API Reference

This document describes the public APIs and interfaces in the s9s project that developers can use when extending or integrating with s9s.

## Table of Contents

- [Core Interfaces](#core-interfaces)
- [View Interfaces](#view-interfaces)
- [Data Types](#data-types)
- [UI Components](#ui-components)
- [Event System](#event-system)
- [Plugin API (Future)](#plugin-api-future)

## Core Interfaces

### SlurmClient

The main interface for interacting with SLURM clusters.

```go
type SlurmClient interface {
    // Jobs returns the job manager
    Jobs() JobManager
    
    // Nodes returns the node manager
    Nodes() NodeManager
    
    // Partitions returns the partition manager
    Partitions() PartitionManager
    
    // Reservations returns the reservation manager
    Reservations() ReservationManager
    
    // QoS returns the QoS manager
    QoS() QoSManager
    
    // Accounts returns the account manager
    Accounts() AccountManager
    
    // Users returns the user manager
    Users() UserManager
    
    // Info returns the info manager for cluster information
    Info() InfoManager
    
    // ClusterInfo returns basic cluster information
    ClusterInfo() (*ClusterInfo, error)
    
    // Close closes the client and releases resources
    Close() error
}
```

### JobManager

Manages SLURM job operations.

```go
type JobManager interface {
    // List returns a list of jobs matching the given options
    List(opts *ListJobsOptions) (*JobList, error)
    
    // Get returns details for a specific job
    Get(id string) (*Job, error)
    
    // Submit submits a new job to the cluster
    Submit(job *JobSubmission) (string, error)
    
    // Cancel cancels a running or pending job
    Cancel(id string) error
    
    // Hold places a hold on a pending job
    Hold(id string) error
    
    // Release releases a held job
    Release(id string) error
    
    // Requeue requeues a completed/failed job
    Requeue(id string) (*Job, error)
    
    // GetOutput returns the output of a completed job
    GetOutput(id string) (string, error)
    
    // Notify sends a message to a running job
    Notify(id string, message string) error
}
```

### NodeManager

Manages SLURM compute nodes.

```go
type NodeManager interface {
    // List returns a list of nodes matching the given options
    List(opts *ListNodesOptions) (*NodeList, error)
    
    // Get returns details for a specific node
    Get(name string) (*Node, error)
    
    // Drain drains a node with the given reason
    Drain(name string, reason string) error
    
    // Resume resumes a drained node
    Resume(name string) error
    
    // SetState sets the state of a node
    SetState(name string, state string) error
}
```

### PartitionManager

Manages SLURM partitions.

```go
type PartitionManager interface {
    // List returns a list of all partitions
    List() (*PartitionList, error)
    
    // Get returns details for a specific partition
    Get(name string) (*Partition, error)
}
```

## View Interfaces

### View

Base interface for all views in the application.

```go
type View interface {
    // Name returns the unique identifier for the view
    Name() string
    
    // Title returns the display title for the view
    Title() string
    
    // SetupView creates and returns the root UI primitive
    SetupView() tview.Primitive
    
    // Refresh updates the view with latest data
    Refresh() error
    
    // SetApp sets the application reference
    SetApp(app *tview.Application)
    
    // SetPages sets the pages reference for modal management
    SetPages(pages *tview.Pages)
    
    // SetStatusBar sets the main status bar reference
    SetStatusBar(statusBar *components.StatusBar)
    
    // HandleKey handles custom key events
    HandleKey(event *tcell.EventKey) *tcell.EventKey
    
    // Focus sets focus to the view's primary element
    Focus()
    
    // HasFocus returns whether the view has focus
    HasFocus() bool
}
```

### RefreshableView

Interface for views that support auto-refresh.

```go
type RefreshableView interface {
    View
    
    // StartAutoRefresh begins automatic refresh
    StartAutoRefresh(interval time.Duration)
    
    // StopAutoRefresh stops automatic refresh
    StopAutoRefresh()
    
    // IsAutoRefreshing returns current auto-refresh state
    IsAutoRefreshing() bool
}
```

### SearchableView

Interface for views that support searching.

```go
type SearchableView interface {
    View
    
    // Search performs a search with the given query
    Search(query string) error
    
    // ClearSearch clears the current search
    ClearSearch()
    
    // NextMatch moves to the next search match
    NextMatch()
    
    // PreviousMatch moves to the previous search match
    PreviousMatch()
}
```

## Data Types

### Job

Represents a SLURM job.

```go
type Job struct {
    ID         string            // Job ID
    Name       string            // Job name
    User       string            // Username
    Account    string            // Account name
    Partition  string            // Partition name
    State      string            // Job state (RUNNING, PENDING, etc.)
    Priority   float64           // Job priority
    QOS        string            // Quality of Service
    NodeCount  int               // Number of nodes
    TimeLimit  string            // Time limit
    TimeUsed   string            // Time used
    SubmitTime time.Time         // Submission time
    StartTime  *time.Time        // Start time (nil if not started)
    EndTime    *time.Time        // End time (nil if not ended)
    NodeList   string            // Allocated nodes
    Command    string            // Job command/script
    WorkingDir string            // Working directory
    StdOut     string            // Standard output path
    StdErr     string            // Standard error path
    ExitCode   *int              // Exit code (nil if not completed)
}
```

### Node

Represents a SLURM compute node.

```go
type Node struct {
    Name            string            // Node name
    State           string            // Node state
    Partitions      []string          // Associated partitions
    CPUsTotal       int               // Total CPUs
    CPUsAllocated   int               // Allocated CPUs
    CPUsIdle        int               // Idle CPUs
    MemoryTotal     int64             // Total memory (MB)
    MemoryAllocated int64             // Allocated memory (MB)
    MemoryFree      int64             // Free memory (MB)
    Features        []string          // Node features
    Reason          string            // Reason for current state
    ReasonTime      *time.Time        // When reason was set
    AllocatedJobs   []string          // Currently allocated jobs
}
```

### Partition

Represents a SLURM partition.

```go
type Partition struct {
    Name           string            // Partition name
    State          string            // Partition state
    TotalNodes     int               // Total nodes
    TotalCPUs      int               // Total CPUs
    DefaultTime    string            // Default time limit
    MaxTime        string            // Maximum time limit
    DefaultMemPerCPU int64           // Default memory per CPU
    MaxMemPerCPU   int64             // Maximum memory per CPU
    AllowedQoS     []string          // Allowed QoS
    DenyQoS        []string          // Denied QoS
    AllowedAccounts []string         // Allowed accounts
    DenyAccounts   []string          // Denied accounts
}
```

### ListOptions

Options for listing resources.

```go
type ListJobsOptions struct {
    States    []string          // Filter by job states
    Users     []string          // Filter by users
    Accounts  []string          // Filter by accounts
    Partitions []string         // Filter by partitions
    StartTime *time.Time        // Jobs started after this time
    EndTime   *time.Time        // Jobs ended before this time
    Limit     int               // Maximum results
    Offset    int               // Result offset for pagination
}

type ListNodesOptions struct {
    States     []string         // Filter by node states
    Partitions []string         // Filter by partitions
    Features   []string         // Filter by features
    Reason     string           // Filter by reason pattern
}
```

### JobSubmission

Parameters for submitting a new job.

```go
type JobSubmission struct {
    Name         string           // Job name
    Script       string           // Job script content
    ScriptPath   string           // Path to job script
    Partition    string           // Target partition
    Account      string           // Account to charge
    QOS          string           // Quality of Service
    TimeLimit    string           // Time limit (HH:MM:SS)
    Memory       string           // Memory requirement
    CPUs         int              // Number of CPUs
    Nodes        int              // Number of nodes
    Tasks        int              // Number of tasks
    GPUs         int              // Number of GPUs
    WorkingDir   string           // Working directory
    Output       string           // Output file path
    Error        string           // Error file path
    Environment  map[string]string // Environment variables
    Dependencies []string         // Job dependencies
    Constraints  string           // Node constraints
    Array        string           // Array job specification
}
```

## UI Components

### MultiSelectTable

Table component with multi-selection support.

```go
type MultiSelectTable struct {
    *tview.Table
    // ... private fields
}

// Methods
func (t *MultiSelectTable) SetOnSelect(fn func(row int, data []string))
func (t *MultiSelectTable) SetOnToggle(fn func(row int, selected bool))
func (t *MultiSelectTable) GetSelectedRows() []int
func (t *MultiSelectTable) GetSelectedData() [][]string
func (t *MultiSelectTable) ClearSelection()
func (t *MultiSelectTable) SelectAll()
func (t *MultiSelectTable) SetData(data [][]string)
```

### StatusBar

Application-wide status bar for messages.

```go
type StatusBar struct {
    *tview.TextView
    // ... private fields
}

// Methods
func (s *StatusBar) Info(message string)
func (s *StatusBar) Success(message string)
func (s *StatusBar) Warning(message string)
func (s *StatusBar) Error(message string)
func (s *StatusBar) Clear()
func (s *StatusBar) SetPersistent(message string)
```

### FilterBar

Advanced filtering interface.

```go
type FilterBar struct {
    // ... private fields
}

// Methods
func (f *FilterBar) Show()
func (f *FilterBar) Hide()
func (f *FilterBar) IsVisible() bool
func (f *FilterBar) SetOnFilterChange(fn func(filter *Filter))
func (f *FilterBar) GetFilter() *Filter
func (f *FilterBar) Clear()
```

### Modal

Enhanced modal dialog.

```go
type Modal struct {
    *tview.Modal
    // ... private fields
}

// Methods
func NewConfirmModal(title, text string, onConfirm func()) *Modal
func NewInputModal(title, label string, onSubmit func(text string)) *Modal
func NewFormModal(title string, fields []FormField, onSubmit func(values map[string]string)) *Modal
```

## Event System

### EventBus

Central event distribution system.

```go
type EventBus interface {
    // Subscribe registers a handler for an event type
    Subscribe(eventType string, handler EventHandler) string
    
    // Unsubscribe removes a handler
    Unsubscribe(id string)
    
    // Publish sends an event to all subscribers
    Publish(event Event)
}

type Event interface {
    Type() string
    Data() interface{}
}

type EventHandler func(event Event)
```

### Common Events

```go
// Job state change event
type JobStateChangeEvent struct {
    JobID     string
    OldState  string
    NewState  string
}

// Node state change event
type NodeStateChangeEvent struct {
    NodeName  string
    OldState  string
    NewState  string
}

// Refresh request event
type RefreshRequestEvent struct {
    ViewName  string
    Force     bool
}
```

## Plugin API (Future)

### Plugin Interface

```go
type Plugin interface {
    // Metadata
    Name() string
    Version() string
    Description() string
    Author() string
    
    // Lifecycle
    Init(api PluginAPI) error
    Start() error
    Stop() error
    
    // Registration
    RegisterCommands() []Command
    RegisterViews() []View
    RegisterKeyBindings() []KeyBinding
    RegisterEventHandlers() []EventHandler
}
```

### PluginAPI

API provided to plugins.

```go
type PluginAPI interface {
    // Get SLURM client
    GetSlurmClient() SlurmClient
    
    // UI operations
    ShowModal(modal tview.Primitive)
    ShowNotification(message string, level NotificationLevel)
    
    // Navigation
    SwitchToView(name string)
    
    // Events
    PublishEvent(event Event)
    SubscribeToEvent(eventType string, handler EventHandler) string
    
    // Storage
    LoadConfig(key string, v interface{}) error
    SaveConfig(key string, v interface{}) error
    
    // Logging
    Logger() *log.Logger
}
```

### Command Interface

```go
type Command interface {
    Name() string
    Description() string
    Execute(args []string) error
    Complete(args []string) []string
}
```

### KeyBinding

```go
type KeyBinding struct {
    Key         tcell.Key
    Rune        rune
    Modifiers   tcell.ModMask
    Handler     func() error
    Description string
    ViewName    string    // Empty for global bindings
}
```

## Usage Examples

### Creating a Custom View

```go
type CustomView struct {
    *views.BaseView
    client dao.SlurmClient
    table  *components.MultiSelectTable
}

func NewCustomView(client dao.SlurmClient) *CustomView {
    v := &CustomView{
        BaseView: views.NewBaseView("custom", "Custom View"),
        client:   client,
    }
    return v
}

func (v *CustomView) SetupView() tview.Primitive {
    // Create UI components
    v.table = components.NewMultiSelectTable(config)
    
    // Set up event handlers
    v.table.SetOnSelect(v.onSelect)
    
    return v.table
}

func (v *CustomView) Refresh() error {
    // Fetch and display data
    data, err := v.fetchData()
    if err != nil {
        return err
    }
    v.table.SetData(data)
    return nil
}
```

### Using the SLURM Client

```go
// List running jobs
opts := &dao.ListJobsOptions{
    States: []string{"RUNNING"},
    Limit:  100,
}
jobList, err := client.Jobs().List(opts)
if err != nil {
    return err
}

// Cancel a job
err = client.Jobs().Cancel("12345")
if err != nil {
    return err
}

// Drain a node
err = client.Nodes().Drain("node001", "Maintenance")
if err != nil {
    return err
}
```

### Subscribing to Events

```go
// Subscribe to job state changes
eventBus.Subscribe("job.state.change", func(event Event) {
    if e, ok := event.(*JobStateChangeEvent); ok {
        log.Printf("Job %s changed from %s to %s", 
            e.JobID, e.OldState, e.NewState)
    }
})

// Publish an event
eventBus.Publish(&RefreshRequestEvent{
    ViewName: "jobs",
    Force:    true,
})
```

## Best Practices

1. **Error Handling**: Always return errors with context
2. **Concurrency**: Use the main UI thread for all tview operations
3. **Memory Management**: Clean up resources in view destructors
4. **Testing**: Provide mock implementations for interfaces
5. **Documentation**: Document all public APIs

## Version Compatibility

This API reference is for s9s version 1.x. Breaking changes will be documented in release notes.

---

For more examples and detailed usage, see the [examples/](../examples/) directory.