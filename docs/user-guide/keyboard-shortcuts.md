# Keyboard Shortcuts

Comprehensive guide to all keyboard shortcuts in s9s.

## Global Shortcuts

These shortcuts work from any view:

| Key | Action | Description |
|-----|--------|-------------|
| `?` | Show help | Display help modal with shortcuts |
| `q` | Quit | Exit s9s application |
| `:` | Command mode | Enter command mode (vim-style) |
| `Tab` | Next view | Cycle through views |
| `Shift+Tab` | Previous view | Cycle backwards through views |
| `Ctrl+r` | Force refresh | Refresh current view data |
| `R` | Manual refresh | Refresh view (most views) |
| `Ctrl+F` | Global search | Search across all cluster resources |
| `Ctrl+C` | Cancel | Cancel current operation or close modal |
| `ESC` | Exit/Close | Exit filter mode, close modal, cancel operation |

## Navigation Shortcuts

| Key | Action | Description |
|-----|--------|-------------|
| `J` | Jobs view | Jump to Jobs view |
| `N` | Nodes view | Jump to Nodes view |
| `P` | Partitions view | Jump to Partitions view |

## Dashboard View

| Key | Action | Description |
|-----|--------|-------------|
| `J` | Jobs view | Switch to Jobs view |
| `N` | Nodes view | Switch to Nodes view |
| `P` | Partitions view | Switch to Partitions view |
| `A` | Advanced analytics | Open analytics modal |
| `H` | Health check | Open health check modal |
| `R` | Refresh dashboard | Manual refresh all panels |

### Analytics Modal (from Dashboard)
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Refresh analytics | Update analytics data |
| `ESC` | Close modal | Return to dashboard |

### Health Check Modal (from Dashboard)
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Refresh health | Update health check |
| `ESC` | Close modal | Return to dashboard |

## Jobs View

### Job Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show detailed job information |
| `s/S` | Submit job | Open job submission wizard |
| `F2` | Job templates | Open job templates/submission form |
| `c/C` | Cancel job | Cancel selected job |
| `H` | Hold job | Place job on hold |
| `r` | Release job | Release held job |
| `q/Q` | Requeue job | Requeue completed/failed job |
| `o/O` | View output | View job output/logs |
| `d/D` | View dependencies | Show job dependency graph |

### Batch Operations
| Key | Action | Description |
|-----|--------|-------------|
| `b/B` | Batch operations | Open batch operations menu |
| `v/V` | Multi-select mode | Toggle multi-select mode |
| `Space` | Toggle selection | Select/deselect current row (in multi-select) |
| `Ctrl+A` | Select all | Select all jobs (in multi-select) |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `F3` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search all cluster resources |
| `a/A` | Filter all states | Show all job states |
| `p/P` | Filter pending | Show pending jobs only |
| `u/U` | Filter by user | Filter jobs by username |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh jobs data |
| `m/M` | Toggle auto-refresh | Enable/disable auto-refresh (30s) |
| `F1` | Action menu | Show context-sensitive actions |
| `1-9` | Sort by column | Sort by column number |

## Nodes View

### Node Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show node details and metrics |
| `d/D` | Drain node | Drain selected node |
| `r` | Resume node | Resume drained node |
| `s/S` | SSH to node | Open SSH options menu |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `F3` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search all cluster resources |
| `p/P` | Partition filter | Filter by partition |
| `a/A` | Toggle all states | Toggle "all states" filter |
| `i/I` | Toggle idle | Toggle idle state filter |
| `m/M` | Toggle mixed | Toggle mixed state filter |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Grouping
| Key | Action | Description |
|-----|--------|-------------|
| `g/G` | Group by | Open grouping dialog (none/partition/state/features) |
| `Space` | Toggle expansion | Expand/collapse group (when grouped) |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh nodes data |
| `1-9` | Sort by column | Sort by column number |

## Partitions View

### Partition Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show partition details |
| `J` | View jobs | Switch to Jobs view filtered by partition |
| `N` | View nodes | Switch to Nodes view filtered by partition |
| `A` | Analytics | Open partition analytics dashboard |
| `W` | Wait time analytics | Show wait time analytics |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `F3` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search all cluster resources |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh partitions data |
| `1-9` | Sort by column | Sort by column number |

## Users View

### User Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show user details and limits |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `F3` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search all cluster resources |
| `a/A` | Toggle admin filter | Show admins/operators only or all users |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh users data |
| `1-9` | Sort by column | Sort by column number |

## Accounts View

### Account Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show account details and hierarchy |
| `H` | Hierarchy tree | Display full account hierarchy tree |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `F3` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search all cluster resources |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh accounts data |
| `1-9` | Sort by column | Sort by column number |

## QoS View

### QoS Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show QoS policy details and limits |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `F3` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search all cluster resources |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh QoS data |
| `1-9` | Sort by column | Sort by column number |

## Reservations View

### Reservation Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show reservation details and usage |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `F3` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search all cluster resources |
| `a/A` | Toggle active | Show active reservations only (TODO) |
| `f/F` | Toggle future | Show future reservations only (TODO) |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh reservations data |
| `1-9` | Sort by column | Sort by column number |

## Health View

### Alert Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View alert | Show alert details and recommendations |
| `a/A` | Acknowledge | Acknowledge first unacknowledged alert |
| `r` | Resolve | Resolve first active alert |
| `c/C` | Clear resolved | Remove resolved alerts >24h old |

### Health Information
| Key | Action | Description |
|-----|--------|-------------|
| `H` | Health checks | Show detailed health check report |
| `s/S` | Statistics | Show health statistics dashboard |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh health data |
| `ESC` | Close modal | Close detail modals |

## Advanced Filter Syntax

When using `F3` for advanced filtering, use these expression patterns:

### Field Matching
```
field:value          - Contains match
field=value          - Exact match
field:>value         - Greater than (numeric)
field:<value         - Less than (numeric)
field:>=value        - Greater or equal (numeric)
field:<=value        - Less or equal (numeric)
```

### Examples

**Jobs:**
```
state:RUNNING partition:gpu
user:alice priority:>500
nodes:>=8 state:PENDING
```

**Nodes:**
```
state:IDLE partition:gpu
features:nvlink cpus:>64
memory:>256GB
```

**Partitions:**
```
state:UP nodes:>100
efficiency:>80 qos:high
```

**Users:**
```
admin:Administrator
account:research qos:high
maxjobs:>50
```

**Accounts:**
```
organization:research
parent:root qos:high
maxcpus:>1000
```

**QoS:**
```
priority:>1000
preempt:Suspend
maxjobs:>100
```

**Reservations:**
```
state:ACTIVE nodes:>16
users:alice accounts:ml-team
```

## Special Filter Syntax

### Partition Filtering (Simple Filter)
In simple filter mode (`/`), use special syntax:

```
p:gpu               - Filter by partition "gpu"
partition:cpu       - Filter by partition "cpu"
```

Works in Jobs and Nodes views.

## Tips & Tricks

### Quick Navigation
- Press `Tab` repeatedly to cycle through all views
- Use `J`, `N`, `P` for direct jumps to common views
- `Ctrl+F` for global search finds resources across all views

### Efficient Filtering
- Use `/` for quick keyword search
- Use `F3` when you need complex multi-field filtering
- Press `ESC` to quickly clear filters
- Special syntax like `p:gpu` saves time

### Batch Operations (Jobs)
1. Press `v/V` to enter multi-select mode
2. Use `Space` to select specific jobs
3. Use `Ctrl+A` to select all
4. Press `b/B` for batch operations menu

### Grouping (Nodes)
1. Press `g/G` to group by partition/state/features
2. Use `Space` to expand/collapse groups
3. Makes large node lists easier to navigate

### Refresh Strategies
- Most views auto-refresh every 10-30 seconds
- Use `R` for immediate manual refresh
- Jobs view: `m/M` toggles auto-refresh on/off

### Help When Stuck
- Press `?` for context-sensitive help
- Press `F1` for action menu (shows available operations)
- Press `ESC` to cancel most operations

## Customizing Shortcuts

See [Configuration Guide](../getting-started/configuration.md) for information on customizing keyboard shortcuts.

## Accessibility

s9s keyboard shortcuts are designed for:
- Keyboard-only navigation
- Screen reader compatibility
- No mouse required for any operation
- Consistent patterns across views

## Vim-Style Commands

Advanced users can use `:` for vim-style command mode:

```
:quit              - Exit application
:refresh           - Refresh current view
:help              - Show help
```

See [Commands Reference](../../reference/commands.md) for full command list.
