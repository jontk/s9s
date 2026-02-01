# Navigation & Keyboard Shortcuts

S9S follows vim-style navigation patterns, making it intuitive for terminal power users. This guide covers all keyboard shortcuts and navigation methods currently implemented.

## Navigation Philosophy

S9S uses a simple interface inspired by vim with basic navigation and view switching capabilities.

## Global Shortcuts

These shortcuts work across all views:

### Essential Keys

| Key | Action | Description |
|-----|--------|-------------|
| `Ctrl+C` | Quit | Exit S9S |
| `Esc` | Cancel | Exit current mode/dialog |
| `Tab` | Next View | Cycle to next view |
| `F1` | Help | Show context-sensitive help |
| `F2` | Alerts | Show system alerts |
| `F5` | Refresh | Refresh current view |

### View Navigation

| Key | Action | View |
|-----|--------|------|
| `1` | Switch View | Jobs |
| `2` | Switch View | Nodes |
| `3` | Switch View | Partitions |
| `4` | Switch View | Users |
| `5` | Switch View | Accounts |
| `6` | Switch View | QoS |
| `7` | Switch View | Reservations |
| `8` | Switch View | Dashboard |
| `9` | Switch View | Health |

### Movement

| Key | Action | Description |
|-----|--------|-------------|
| `j` or `↓` | Down | Move cursor down |
| `k` or `↑` | Up | Move cursor up |
| `g` | Top | Go to first item |
| `G` | Bottom | Go to last item |

## View-Specific Shortcuts

### Jobs View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Show job details |
| `s` | Submit | Submit new job |
| `c` | Cancel | Cancel selected job |
| `H` | Hold/Release | Hold or release job |
| `r` | Release | Release held job |
| `o` | Output | View job output |
| `d` | Dependencies | Show job dependencies |
| `q` | Requeue | Requeue failed job |
| `b` | Batch Ops | Enter batch operations mode |
| `v` | Multi-Select | Toggle multi-select mode |
| `m` | Auto Refresh | Toggle auto-refresh |
| `/` | Filter | Filter jobs |
| `F3` | Adv Filter | Advanced filter mode |
| `Ctrl+F` | Search | Global search |
| `F1` | Actions Menu | Show actions menu |
| `F2` | Templates | Show job templates |
| `1-9` | Sort | Sort by column |
| `R` | Refresh | Force refresh view |

### Nodes View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Show node details |
| `s` | SSH | SSH to selected node |
| `d` | Drain | Drain node |
| `r` | Resume | Resume drained node |
| `/` | Filter | Filter nodes |
| `F3` | Adv Filter | Advanced filter mode |
| `Ctrl+F` | Search | Global search |
| `1-9` | Sort | Sort by column |
| `R` | Refresh | Force refresh view |
| `p` | Partition | Filter by partition |
| `a` | All States | Show all node states |
| `g` | Group By | Group nodes by attribute |
| `Space` | Toggle Group | Toggle group expansion |

### Partitions View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Show partition details |
| `J` | Jobs | Show jobs in partition |
| `N` | Nodes | Show nodes in partition |
| `A` | Analytics | Show partition analytics |
| `W` | Wait Times | Show wait time analytics |
| `/` | Filter | Filter partitions |
| `F3` | Adv Filter | Advanced filter mode |
| `Ctrl+F` | Search | Global search |
| `1-9` | Sort | Sort by column |
| `R` | Refresh | Force refresh view |

### QoS View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Show QoS details |
| `/` | Filter | Filter QoS policies |
| `F3` | Adv Filter | Advanced filter mode |
| `Ctrl+F` | Search | Global search |
| `1-9` | Sort | Sort by column |
| `R` | Refresh | Force refresh view |

### Accounts View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Show account details |
| `H` | Hierarchy | Show account hierarchy |
| `/` | Filter | Filter accounts |
| `F3` | Adv Filter | Advanced filter mode |
| `Ctrl+F` | Search | Global search |
| `1-9` | Sort | Sort by column |
| `R` | Refresh | Force refresh view |

### Users View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Show user details |
| `a` | Toggle Filter | Show admin users / all users |
| `/` | Filter | Filter users |
| `F3` | Adv Filter | Advanced filter mode |
| `Ctrl+F` | Search | Global search |
| `1-9` | Sort | Sort by column |
| `R` | Refresh | Force refresh view |

### Reservations View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Show reservation details |
| `a` | Active Only | Filter active reservations |
| `f` | Future Only | Filter future reservations |
| `/` | Filter | Filter reservations |
| `F3` | Adv Filter | Advanced filter mode |
| `Ctrl+F` | Search | Global search |
| `1-9` | Sort | Sort by column |
| `R` | Refresh | Force refresh view |

### Dashboard View

| Key | Action | Description |
|-----|--------|-------------|
| `J` | Jobs View | Jump to jobs view |
| `N` | Nodes View | Jump to nodes view |
| `P` | Partitions View | Jump to partitions view |
| `A` | Analytics | Show advanced analytics |
| `H` | Health Check | Show health check |
| `R` | Refresh | Refresh all panels |

### Health View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Alert Details | Show alert details |
| `a` | Acknowledge | Acknowledge alert |
| `r` | Resolve | Resolve alert |
| `c` | Clear Resolved | Clear resolved alerts |
| `R` | Refresh | Refresh view |
| `H` | Health Details | Show health details |
| `s` | Health Stats | Show health statistics |

## Search and Filter Mode

Enter filter mode with `/` or advanced filter with `F3`:

### Basic Filter

```
/keyword           # Simple text search in current view
/RUNNING          # Find all running items
/gpu              # Find GPU-related items
```

### Advanced Filter

Press `F3` to access the advanced filter builder with:
- Field-specific filtering
- Operator support (equals, contains, greater than, less than)
- Multiple filter conditions
- Save and load filter presets

### Global Search

Press `Ctrl+F` to search across all views simultaneously and jump to results.

## Tips & Tricks

### Efficiency Tips

1. **Use vim keys**: `j`/`k` for up/down, `g`/`G` for top/bottom
2. **Quick view switching**: Number keys `1-9` switch directly to views
3. **Quick filters**: `/` for instant filtering in any view
4. **Multi-select**: Use `v` in jobs view for batch operations
5. **Sort quickly**: Use `1-9` to sort by different columns

### Power User Workflows

```bash
# Cancel all failed jobs
1                # Switch to jobs view
/FAILED         # Filter failed jobs
v                # Enter multi-select mode
c                # Cancel selected jobs

# SSH to GPU node
2                # Switch to nodes view
/gpu             # Filter GPU nodes
s                # SSH to selected node

# Check partition health
3                # Switch to partitions view
W                # View wait time analytics
```

## Command Mode

Press `:` to enter vim-style command mode with intelligent tab completion:

### Quick Examples
```
:cancel 12345      # Cancel job 12345
:drain node01      # Drain node01
:requeue 67890     # Requeue job 67890
```

### Tab Completion
- Type `:req` and press `Tab` → completes to `:requeue`
- Type `:cancel ` and press `Tab` → shows available job IDs
- Type `:drain ` and press `Tab` → shows available node names

Commands use cached data from views for instant autocomplete without API calls.

See [Commands Reference](../reference/commands.md) for complete command documentation.

## Features

- **Real-time updates** - Data refreshes automatically
- **Command mode with autocomplete** - Vim-style `:` commands with Tab completion
- **ASCII visualizations** - Resource usage shown with colored progress bars
- **Advanced filtering** - Use `/` to filter data in any view or `F3` for advanced filters
- **Sortable columns** - Use number keys `1-9` to sort
- **Detailed analytics** - Press `A` or `W` in partitions for insights
- **Node grouping** - Group nodes by partition, state, or features
- **Wait time analysis** - Predictive queue analytics in partitions
- **Health monitoring** - Cluster health scoring and alerts in health view
- **Hierarchical views** - Account hierarchy and dependency trees
- **Global search** - Cross-view search with `Ctrl+F`

## Color Coding

- **Green** - Running, Available, Healthy states
- **Yellow** - Pending, Mixed, Warning states
- **Red** - Failed, Down, Error states
- **Blue** - Allocated, Active states
- **Cyan** - Information, Special states
- **Orange** - Drain, Maintenance states
- **Gray** - Unknown, Inactive states

## Resource Usage Bars

Visual representation of resource utilization:
- `█` - Used/Actual usage
- `▒` - Allocated by SLURM
- `▱` - Free/Available

Bar colors:
- **Green bars** - Low usage (< 50%)
- **Yellow bars** - Medium usage (50-80%)
- **Red bars** - High usage (> 80%)

## Next Steps

- Practice navigation in mock mode: `s9s --mock`
- Learn advanced filtering techniques
- Explore individual view guides:
  - [Jobs View](views/jobs.md)
  - [Nodes View](views/nodes.md)
  - [Partitions View](views/partitions.md)
  - [Dashboard View](views/dashboard.md)
  - [Health View](views/health.md)
