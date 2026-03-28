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
| `h` | Previous view | Move to previous view |
| `l` | Next view | Move to next view |
| `F1` | Help | Show help modal |
| `F2` | Alerts | Show system alerts |
| `F3` | Preferences | Show preferences dialog |
| `F5` | Force refresh | Refresh current view data |
| `F10` | Configuration | Show configuration |
| `Ctrl+K` | Switch cluster | Switch between configured clusters |
| `Ctrl+C` | Exit application | Exit s9s entirely |
| `ESC` | Exit/Close | Exit filter mode, close modal, cancel operation |

## View Switching

| Key | View |
|-----|------|
| `1` | Jobs |
| `2` | Nodes |
| `3` | Partitions |
| `4` | Reservations |
| `5` | QoS |
| `6` | Accounts |
| `7` | Users |
| `8` | Dashboard |
| `9` | Health |
| `0` | Performance |

## Dashboard View

| Key | Action | Description |
|-----|--------|-------------|
| `L` | Switch layout | Toggle between default dashboard and monitoring layout |
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
| `s` | Submit job | Open job submission wizard |
| `c/C` | Cancel job | Cancel selected job |
| `H` | Hold job | Place job on hold |
| `r` | Release job | Release held job |
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
| `f` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search across all entity types (all data views) |
| `a/A` | Filter all states | Show all job states |
| `p/P` | Filter pending | Show pending jobs only |
| `u/U` | Filter by user | Filter jobs by username |
| `x` | Actions menu | Open actions menu for selected job |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh jobs data |
| `m/M` | Toggle auto-refresh | Enable/disable auto-refresh (30s) |
| `S` | Sort modal | Open interactive sorting dialog |
| `e/E` | Export | Export job list to CSV/JSON/Text/Markdown/HTML |

## Nodes View

### Node Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show node details and metrics |
| `d/D` | Drain node | Drain selected node |
| `r` | Resume node | Resume drained node |
| `s` | SSH to node | Open SSH connection to node |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `f` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search across all entity types (all data views) |
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
| `S` | Sort modal | Open interactive sorting dialog |
| `e/E` | Export | Export node list to CSV/JSON/Text/Markdown/HTML |

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
| `f` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search across all entity types (all data views) |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh partitions data |
| `S` | Sort modal | Open interactive sorting dialog |
| `e/E` | Export | Export partition list to CSV/JSON/Text/Markdown/HTML |

## Users View

### User Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show user details and limits |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `f` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search across all entity types (all data views) |
| `a/A` | Toggle admin filter | Show admins/operators only or all users |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh users data |
| `S` | Sort modal | Open interactive sorting dialog |
| `e/E` | Export | Export user list to CSV/JSON/Text/Markdown/HTML |

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
| `f` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search across all entity types (all data views) |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh accounts data |
| `S` | Sort modal | Open interactive sorting dialog |
| `e/E` | Export | Export account list to CSV/JSON/Text/Markdown/HTML |

## QoS View

### QoS Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show QoS policy details and limits |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `f` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search across all entity types (all data views) |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh QoS data |
| `S` | Sort modal | Open interactive sorting dialog |
| `e/E` | Export | Export QoS list to CSV/JSON/Text/Markdown/HTML |

## Reservations View

### Reservation Operations
| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | View details | Show reservation details and usage |

### Filtering & Search
| Key | Action | Description |
|-----|--------|-------------|
| `/` | Simple filter | Activate filter input |
| `f` | Advanced filter | Open advanced filter bar |
| `Ctrl+F` | Global search | Search across all entity types (all data views) |
| `a/A` | Toggle active | Show active reservations only |
| `t/T` | Toggle future | Show future reservations only |
| `ESC` | Exit filter | Clear filter and exit filter mode |

### Data Management
| Key | Action | Description |
|-----|--------|-------------|
| `R` | Manual refresh | Refresh reservations data |
| `S` | Sort modal | Open interactive sorting dialog |
| `e/E` | Export | Export reservation list to CSV/JSON/Text/Markdown/HTML |

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

When using the advanced filter, use these expression patterns:

### Operators
```
field=value          - Exact match
field!=value         - Not equals
field~value          - Contains match
field!~value         - Not contains
field>value          - Greater than (numeric)
field<value          - Less than (numeric)
field>=value         - Greater or equal (numeric)
field<=value         - Less or equal (numeric)
field=~pattern       - Regex match
field in (a,b,c)     - In list
field not in (a,b)   - Not in list
```

### Examples

**Jobs:**
```
state=RUNNING partition=gpu
user=alice priority>500
cpus>=8 state=PENDING
name~test memory>4G
```

**Nodes:**
```
state=idle partition=gpu
features~nvlink cpus>64
memory>256G
```

**Partitions:**
```
state=up nodes>100
qos~normal
```

**Users:**
```
account=research qos~high
```

**Accounts:**
```
name~research
qos~high
```

**QoS:**
```
priority>1000
```

**Reservations:**
```
state=ACTIVE nodes>16
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
- Use number keys `1`-`9` and `0` to jump directly to a view by position

### Efficient Filtering
- Use `/` for quick keyword search
- Use `Ctrl+F` in any data view for cross-resource search
- Press `ESC` to quickly clear filters
- Special syntax like `p:gpu` saves time in simple filter mode

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
- Three views auto-refresh: Jobs (30s), Health (10s), Performance (5s)
- Use `R` for immediate manual refresh
- Jobs view: `m/M` toggles auto-refresh on/off

### Help When Stuck
- Press `?` for context-sensitive help
- Press `F1` for help modal
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

Press `:` to enter vim-style command mode with tab completion support:

### Command Completion
```
:<Tab>             - Shows all available commands
:q<Tab>            - Shows: q, qos, quit
:req<Tab>          - Completes to: requeue
:dr<Tab>           - Completes to: drain
```

### Argument Completion
```
:cancel <Tab>      - Shows available job IDs
:drain <Tab>       - Shows available node names
:requeue <Tab>     - Shows available job IDs
```

### Common Commands
```
:quit              - Exit application
:refresh           - Refresh current view
:help              - Show help
:cancel JOBID      - Cancel specific job (with autocomplete)
:drain NODE        - Drain specific node (with autocomplete)
```

**How it works:** Completions are context-aware and use data from the currently loaded views, so you'll see real job IDs and node names from your cluster.

See [Commands Reference](../reference/commands.md) for full command list.
