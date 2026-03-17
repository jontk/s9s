# Views Overview

s9s provides specialized views for different aspects of your SLURM cluster. Each view is optimized for specific tasks and provides relevant actions and information.

## Available Views

### Core Views

| View | Description | Key Shortcut |
|------|-------------|--------------|
| [Jobs](jobs.md) | Job management, submission, and monitoring | `1` |
| [Nodes](nodes.md) | Node status, resource usage, and operations | `2` |
| [Partitions](partitions.md) | Partition information and queue analysis | `3` |
| [Dashboard](dashboard.md) | Real-time cluster overview with health metrics | `8` |

### Resource Management

| View | Description | Key Shortcut |
|------|-------------|--------|
| [Reservations](reservations.md) | Reservation scheduling and management | `4` |
| [QoS](qos.md) | Quality of Service policies and priorities | `5` |
| [Accounts](accounts.md) | Account hierarchy and associations | `6` |
| [Users](users.md) | User accounts and resource limits | `7` |

### Monitoring

| View | Description | Key Shortcut |
|------|-------------|--------|
| [Health](health.md) | Cluster health checks and alerts | `9` |
| [Performance](performance.md) | Cluster-wide metrics and resource utilization | `0` |

## Switching Between Views

### Using Tab Navigation
Press `Tab` to cycle through views in this order:
- Jobs → Nodes → Partitions → Reservations → QoS → Accounts → Users → Dashboard → Health → Performance

### Using Number Keys
Press a number key to jump directly to a view (works globally):
- `1` - Jobs
- `2` - Nodes
- `3` - Partitions
- `4` - Reservations
- `5` - QoS
- `6` - Accounts
- `7` - Users
- `8` - Dashboard
- `9` - Health
- `0` - Performance

### From Dashboard
The Dashboard provides additional letter shortcuts:
- `J` - Switch to Jobs
- `N` - Switch to Nodes
- `P` - Switch to Partitions
- `H` - Open Health Check modal

Note: The `J`, `N`, and `P` letter shortcuts only work from the Dashboard view, not globally.

## Common Features

All views share these standard capabilities:

### Filtering
- **`/`** - Simple text filter (searches all columns)
- **`Ctrl+F`** - Advanced filter with expression syntax (Jobs and Nodes views)
- **`ESC`** - Exit filter mode

### Search
- **`Ctrl+F`** - Global search across all views
- Search by resource type (jobs, nodes, partitions, etc.)

### Sorting
- **`S`** - Open sort modal
- Click column headers to toggle sort direction

### Refresh
- **`R`** - Manual refresh
- **`F5`** - Manual refresh (global)
- **`m/M`** - Toggle auto-refresh (Jobs view)
- Auto-refresh varies by view: Jobs (30s), Health (10s), Performance (5s). Other views (Nodes, Partitions, Dashboard, Users, Accounts, QoS, Reservations) do not auto-refresh.

### Help
- **`?`** - Show help and keyboard shortcuts
- **`F1`** - Help modal (global)

## View-Specific Features

Each view has specialized features for its domain:

- **Jobs**: Batch operations, job submission wizard, output streaming
- **Nodes**: SSH access, drain/resume operations, resource grouping
- **Partitions**: Queue analytics, efficiency metrics
- **Health**: Alert acknowledgment, health statistics

See individual view documentation for detailed features and shortcuts.
