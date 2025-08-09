# s9s Architecture

## Overview

s9s follows a modular, layered architecture designed for maintainability, testability, and extensibility. The application is built using Go and the tview terminal UI framework.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Terminal UI (tview)                      │
├─────────────────────────────────────────────────────────────┤
│                         Views Layer                          │
│  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌─────────────┐     │
│  │  Jobs   │ │  Nodes  │ │Dashboard │ │   Others    │     │
│  └─────────┘ └─────────┘ └──────────┘ └─────────────┘     │
├─────────────────────────────────────────────────────────────┤
│                      UI Components                           │
│  ┌──────────┐ ┌───────────┐ ┌────────┐ ┌─────────────┐    │
│  │  Table   │ │StatusBar  │ │ Modal  │ │  FilterBar  │    │
│  └──────────┘ └───────────┘ └────────┘ └─────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                    Data Access Layer                         │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              DAO (Data Access Objects)               │   │
│  │  ┌─────────┐ ┌─────────┐ ┌──────────┐ ┌─────────┐ │   │
│  │  │JobMgr   │ │NodeMgr  │ │PartMgr   │ │Others   │ │   │
│  │  └─────────┘ └─────────┘ └──────────┘ └─────────┘ │   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│                     SLURM Adapter                            │
│  ┌─────────────────┐           ┌──────────────────────┐    │
│  │  SLURM Client   │           │    Mock Client       │    │
│  │  (Production)   │           │   (Development)      │    │
│  └─────────────────┘           └──────────────────────┘    │
├─────────────────────────────────────────────────────────────┤
│                   External Services                          │
│  ┌─────────────┐  ┌──────────────┐  ┌─────────────────┐   │
│  │ SLURM REST  │  │ SSH Service  │  │ Export Service  │   │
│  │     API     │  │              │  │                 │   │
│  └─────────────┘  └──────────────┘  └─────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Application Layer (`cmd/s9s/`)

The entry point of the application that:
- Parses command-line arguments
- Initializes configuration
- Sets up the application context
- Handles graceful shutdown

```go
// cmd/s9s/main.go
func main() {
    app := app.New()
    if err := app.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### 2. Views Layer (`internal/views/`)

Views implement the UI for different resources:

- **BaseView**: Common functionality for all views
- **JobsView**: Job listing and management
- **NodesView**: Node monitoring and control
- **DashboardView**: Cluster overview
- **PartitionsView**: Partition information
- **Additional Views**: Users, QoS, Reservations, etc.

Key interfaces:
```go
type View interface {
    Name() string
    Title() string
    SetupView() tview.Primitive
    Refresh() error
}
```

### 3. UI Components (`internal/ui/`)

Reusable UI components:

- **MultiSelectTable**: Table with multi-selection support
- **StatusBar**: Application-wide status messages
- **FilterBar**: Advanced filtering interface
- **LoadingManager**: Loading state management
- **Modal dialogs**: Confirmations, forms, etc.

### 4. Data Access Objects (`internal/dao/`)

Abstracts SLURM operations with clean interfaces:

```go
type SlurmClient interface {
    Jobs() JobManager
    Nodes() NodeManager
    Partitions() PartitionManager
    // ... other managers
}

type JobManager interface {
    List(opts *ListJobsOptions) (*JobList, error)
    Get(id string) (*Job, error)
    Cancel(id string) error
    Hold(id string) error
    Release(id string) error
    Submit(job *JobSubmission) (string, error)
}
```

### 5. SLURM Adapter (`internal/dao/slurm_adapter.go`)

Bridges the DAO interfaces with the actual SLURM client:
- Handles API version compatibility
- Manages authentication
- Implements retry logic
- Provides error translation

### 6. Mock Implementation (`pkg/slurm/mock.go`)

Full SLURM simulator for development:
- Simulates job lifecycle
- Provides realistic test data
- Supports all SLURM operations
- Configurable delays and behaviors

## Key Design Patterns

### 1. Interface Segregation

Each manager interface is focused on a single resource type:
```go
type NodeManager interface {
    List(opts *ListNodesOptions) (*NodeList, error)
    Get(name string) (*Node, error)
    Drain(name string, reason string) error
    Resume(name string) error
}
```

### 2. Dependency Injection

Views receive dependencies through constructors:
```go
func NewJobsView(client dao.SlurmClient) *JobsView {
    return &JobsView{
        client: client,
        // ... other initialization
    }
}
```

### 3. Observer Pattern

Views can subscribe to refresh events:
```go
type RefreshObserver interface {
    OnRefresh() error
}
```

### 4. Command Pattern

User actions are encapsulated as commands:
```go
type Command interface {
    Execute() error
    Undo() error
}
```

## Data Flow

### 1. User Input Flow
```
User Input → View Handler → Validation → DAO Call → SLURM API → Response → UI Update
```

### 2. Refresh Flow
```
Timer/Manual Trigger → View.Refresh() → DAO.List() → Parse Response → Update Table → Render
```

### 3. Error Handling Flow
```
Error Occurs → Log Error → User-Friendly Message → Status Bar Display → Optional Retry
```

## Configuration Management

### Configuration Hierarchy

1. **Default Configuration** (built-in)
2. **System Configuration** (`/etc/s9s/config.yaml`)
3. **User Configuration** (`~/.s9s/config.yaml`)
4. **Environment Variables** (`S9S_*`)
5. **Command-line Flags**

### Configuration Structure

```go
type Config struct {
    Clusters      map[string]*ClusterConfig
    Preferences   *UserPreferences
    Debug         bool
    LogFile       string
}

type ClusterConfig struct {
    URL           string
    Auth          AuthConfig
    Timeout       time.Duration
    RetryAttempts int
}
```

## State Management

### View State

Each view maintains its own state:
```go
type JobsView struct {
    jobs         []*dao.Job        // Current job list
    filter       string            // Active filter
    selectedJobs map[string]bool   // Multi-selection state
    autoRefresh  bool              // Auto-refresh enabled
    mu           sync.RWMutex      // Thread safety
}
```

### Global State

Application-wide state is managed by:
- Configuration manager
- Theme manager
- Key binding registry
- Status bar coordinator

## Concurrency Model

### Goroutine Usage

1. **Main UI Thread**: Handles all tview rendering
2. **Refresh Workers**: Background data fetching
3. **SSH Sessions**: Separate goroutine per session
4. **Export Operations**: Async file operations

### Synchronization

```go
// Mutex for data protection
v.mu.Lock()
v.jobs = newJobs
v.mu.Unlock()

// Channels for communication
done := make(chan bool)
go func() {
    // Long operation
    done <- true
}()
```

## Error Handling Strategy

### Error Types

1. **Network Errors**: Retry with backoff
2. **Authentication Errors**: Prompt for credentials
3. **Permission Errors**: Clear user message
4. **Data Errors**: Log and skip invalid entries

### Error Propagation

```go
// DAO layer adds context
if err != nil {
    return fmt.Errorf("failed to list jobs: %w", err)
}

// View layer shows user-friendly message
if err != nil {
    v.mainStatusBar.Error("Unable to refresh job list")
    debug.Logger.Printf("Refresh error: %v", err)
}
```

## Security Considerations

### Authentication

- Token-based authentication preferred
- Credentials stored securely (keyring integration planned)
- Session management with timeout
- No credentials in logs or debug output

### Input Validation

- All user input sanitized
- Command injection prevention
- Path traversal protection
- Size limits on data operations

## Performance Optimization

### Caching Strategy

1. **View-level caching**: Recent data retained
2. **API response caching**: Configurable TTL
3. **Lazy loading**: Load data on demand
4. **Incremental updates**: Refresh only changed data

### Resource Management

```go
// Limit concurrent operations
sem := make(chan struct{}, 5)
for _, job := range jobs {
    sem <- struct{}{}
    go func(j *Job) {
        defer func() { <-sem }()
        // Process job
    }(job)
}
```

## Testing Architecture

### Test Layers

1. **Unit Tests**: Individual component testing
2. **Integration Tests**: Component interaction
3. **End-to-End Tests**: Full workflow validation
4. **Performance Tests**: Benchmarking

### Mock Strategy

```go
// Interface for easy mocking
type TimeProvider interface {
    Now() time.Time
}

// Mock implementation
type MockTimeProvider struct {
    CurrentTime time.Time
}
```

## Extension Points

### Plugin System (Planned)

```go
type Plugin interface {
    Name() string
    Version() string
    Init(api PluginAPI) error
    RegisterCommands() []Command
    RegisterViews() []View
}
```

### Custom Views

Developers can add custom views by:
1. Implementing the View interface
2. Registering with the view manager
3. Adding key bindings

### Export Formats

New export formats can be added by implementing:
```go
type Exporter interface {
    Export(data interface{}, writer io.Writer) error
    FileExtension() string
    MimeType() string
}
```

## Future Architectural Considerations

### Planned Improvements

1. **Plugin System**: Dynamic loading of extensions
2. **Event Bus**: Decoupled component communication
3. **State Store**: Centralized state management
4. **API Gateway**: Multiple backend support
5. **Metrics Collection**: Performance monitoring

### Scalability

- Pagination for large datasets
- Virtual scrolling for tables
- Progressive data loading
- Background prefetching

## Development Guidelines

### Adding a New View

1. Create view file in `internal/views/`
2. Implement View interface
3. Add to view registry
4. Define key bindings
5. Add tests

### Adding a New Command

1. Define command in view
2. Add key binding
3. Implement business logic
4. Update help text
5. Add tests

### Code Organization

```
internal/views/newfeature.go      # View implementation
internal/views/newfeature_test.go # Unit tests
internal/dao/types.go             # Add data types
docs/                            # Update documentation
```

## Debugging and Diagnostics

### Debug Mode

Enable with `--debug` flag:
- Detailed logging to file
- Performance metrics
- API request/response logging
- State change tracking

### Health Checks

Built-in diagnostics:
- Connection status
- API availability
- Performance metrics
- Error rate monitoring

---

This architecture provides a solid foundation for a maintainable, extensible terminal UI application while keeping the codebase organized and testable.