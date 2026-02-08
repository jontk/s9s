# Views Overview

s9s provides specialized views for different aspects of your SLURM cluster. Each view is optimized for specific tasks and provides relevant actions and information.

## Available Views

### Core Views

| View | Description | Key Shortcut |
|------|-------------|--------------|
| [Dashboard](dashboard.md) | Real-time cluster overview with health metrics | Default view |
| [Jobs](jobs.md) | Job management, submission, and monitoring | `Tab` or `J` |
| [Nodes](nodes.md) | Node status, resource usage, and operations | `N` |
| [Partitions](partitions.md) | Partition information and queue analysis | `P` |

### Resource Management

| View | Description | Access |
|------|-------------|--------|
| [Users](users.md) | User accounts and resource limits | Via navigation |
| [Accounts](accounts.md) | Account hierarchy and associations | Via navigation |
| [QoS](qos.md) | Quality of Service policies and priorities | Via navigation |
| [Reservations](reservations.md) | Reservation scheduling and management | Via navigation |

### Monitoring

| View | Description | Access |
|------|-------------|--------|
| [Performance](performance.md) | Cluster-wide metrics and resource utilization | `9` or via navigation |
| [Health](health.md) | Cluster health checks and alerts | `H` from Dashboard |

## Switching Between Views

### Using Tab Navigation
Press `Tab` to cycle through the main views:
- Dashboard → Jobs → Nodes → Partitions → Users → Accounts → QoS → Reservations

### Direct Navigation
Press the corresponding key to jump directly to a view:
- `J` - Jobs view
- `N` - Nodes view
- `P` - Partitions view

### From Dashboard
The Dashboard provides quick navigation shortcuts:
- `J` - Switch to Jobs
- `N` - Switch to Nodes
- `P` - Switch to Partitions
- `H` - Open Health Check modal

## Common Features

All views share these standard capabilities:

### Filtering
- **`/`** - Simple text filter (searches all columns)
- **`F3`** - Advanced filter with expression syntax
- **`ESC`** - Exit filter mode

### Search
- **`Ctrl+F`** - Global search across all views
- Search by resource type (jobs, nodes, partitions, etc.)

### Sorting
- **`1-9`** - Sort by column number
- Click column headers to toggle sort direction

### Refresh
- **`R`** - Manual refresh
- **`m/M`** - Toggle auto-refresh (Jobs view)
- Views auto-refresh every 10-30 seconds

### Help
- **`?`** - Show help and keyboard shortcuts
- **`F1`** - Action menu (context-specific)

## View-Specific Features

Each view has specialized features for its domain:

- **Jobs**: Batch operations, job submission wizard, output streaming
- **Nodes**: SSH access, drain/resume operations, resource grouping
- **Partitions**: Queue analytics, efficiency metrics
- **Health**: Alert acknowledgment, health statistics

See individual view documentation for detailed features and shortcuts.
