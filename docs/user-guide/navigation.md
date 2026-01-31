# Navigation & Keyboard Shortcuts

S9S follows vim-style navigation patterns, making it intuitive for terminal power users. This guide covers all keyboard shortcuts and navigation methods.

## Navigation Philosophy

S9S uses a modal interface inspired by vim:
- **Normal mode** - Navigate and execute commands
- **Search mode** - Filter and find items
- **Command mode** - Enter advanced commands
- **Selection mode** - Select multiple items

## Global Shortcuts

These shortcuts work in all views:

### Essential Keys

| Key | Action | Description |
|-----|--------|-------------|
| `?` | Help | Show context-sensitive help |
| `q` | Quit | Exit S9S |
| `Q` | Force Quit | Exit immediately without confirmation |
| `:` | Command Mode | Enter command mode |
| `/` | Search | Enter search/filter mode |
| `Esc` | Cancel | Exit current mode/dialog |
| `Tab` | Next View | Cycle to next view |
| `Shift+Tab` | Previous View | Cycle to previous view |

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
| `h` or `←` | Left | Move left/collapse |
| `l` or `→` | Right | Move right/expand |
| `g` | Top | Go to first item |
| `G` | Bottom | Go to last item |
| `Ctrl+f` | Page Down | Scroll down one page |
| `Ctrl+b` | Page Up | Scroll up one page |
| `Ctrl+d` | Half Page Down | Scroll down half page |
| `Ctrl+u` | Half Page Up | Scroll up half page |

### Selection

| Key | Action | Description |
|-----|--------|-------------|
| `Space` | Toggle Selection | Select/deselect current item |
| `v` | Visual Mode | Enter visual selection mode |
| `V` | Select All | Select all visible items |
| `Ctrl+a` | Select All | Alternative select all |
| `Ctrl+Space` | Clear Selection | Deselect all items |

## View-Specific Shortcuts

### Jobs View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Expand/Details | Show job details |
| `c` | Cancel | Cancel selected job(s) |
| `h` | Hold | Hold selected job(s) |
| `r` | Release | Release held job(s) |
| `d` | Delete | Delete job record |
| `o` | Output | View job output/logs |
| `O` | Full Output | View complete output in pager |
| `e` | Edit | Modify job parameters |
| `s` | Submit | Submit new job |
| `S` | Submit Template | Submit from template |
| `t` | Tail | Tail job output |
| `i` | Info | Detailed job information |
| `p` | Priority | Change job priority |
| `m` | Move | Move to different partition |
| `n` | Notify | Set up job notifications |
| `b` | Batch | Enter batch operations mode |
| `f` | Filter Preset | Apply saved filter |
| `F` | Save Filter | Save current filter |
| `x` | Export | Export job data |
| `R` | Requeue | Requeue failed job |
| `D` | Dependencies | View job dependencies |
| `T` | Timeline | View job timeline |

### Nodes View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Show node details |
| `s` | SSH | SSH to selected node |
| `S` | SSH (new window) | SSH in new terminal |
| `d` | Drain | Drain node |
| `D` | Drain (reason) | Drain with reason |
| `r` | Resume | Resume/undrain node |
| `R` | Reboot | Reboot node |
| `p` | Power | Power management |
| `i` | Info | Detailed node info |
| `m` | Maintenance | Set maintenance mode |
| `h` | History | Node history |
| `g` | GPU Info | Show GPU details |
| `l` | Load | Show node load |
| `t` | Temperature | Show temperatures |
| `x` | Export | Export node data |

### Partitions View

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Details | Show partition info |
| `j` | Jobs | Show jobs in partition |
| `n` | Nodes | Show nodes in partition |
| `e` | Edit | Edit partition config |
| `s` | State | Change partition state |
| `l` | Limits | View/edit limits |
| `q` | QoS | Associated QoS |
| `a` | Accounts | Allowed accounts |

### Dashboard View

| Key | Action | Description |
|-----|--------|-------------|
| `r` | Refresh | Force refresh |
| `a` | Auto-refresh | Toggle auto-refresh |
| `+` | Increase Rate | Faster refresh |
| `-` | Decrease Rate | Slower refresh |
| `c` | Customize | Customize dashboard |
| `w` | Add Widget | Add new widget |
| `x` | Remove Widget | Remove widget |
| `m` | Move Widget | Rearrange widgets |

## Search Mode

Enter search mode with `/`:

### Basic Search

```
/keyword           # Simple text search
/RUNNING          # Find all running items
/gpu              # Find GPU-related items
```

### Advanced Search Syntax

```
# Field-specific search
/user:alice       # Items by user alice
/state:PENDING    # Pending items
/name:analysis    # Items with 'analysis' in name

# Operators
/time:>1h         # Running time > 1 hour
/nodes:<=4        # Using <= 4 nodes
/priority:>=1000  # Priority >= 1000

# Compound searches
/user:bob state:RUNNING    # Bob's running items
/partition:gpu nodes:>2    # GPU partition, >2 nodes

# Regex search
/name:~"test.*2023"        # Regex pattern
/output:~"error|fail"      # Find errors

# Time ranges
/time:1h-2h               # Between 1-2 hours
/submit:<1d               # Submitted < 1 day ago
```

### Search Navigation

| Key | Action | Description |
|-----|--------|-------------|
| `Enter` | Apply | Apply search filter |
| `Esc` | Cancel | Cancel search |
| `Ctrl+c` | Clear | Clear search box |
| `Tab` | Autocomplete | Complete search term |
| `↑/↓` | History | Browse search history |
| `Ctrl+r` | Regex Mode | Toggle regex mode |
| `Ctrl+i` | Case Mode | Toggle case sensitivity |

## Command Mode

Enter command mode with `:`:

### Common Commands

```
:help [topic]          # Show help
:quit or :q           # Quit S9S
:write or :w          # Save current state
:view <name>          # Switch to view
:refresh or :r        # Force refresh
:clear                # Clear filters
:export <format>      # Export data
:set <option> <value> # Set option
:source <file>        # Load commands from file
:!<shell-command>     # Execute shell command
```

### View Management

```
:split                # Split view horizontally
:vsplit               # Split view vertically
:close                # Close current pane
:only                 # Close all other panes
:resize +10           # Increase pane size
:resize -10           # Decrease pane size
```

### Settings Commands

```
:set refresh 5s       # Set refresh interval
:set theme dark       # Change theme
:set confirm off      # Disable confirmations
:set columns <list>   # Set visible columns
:show settings        # Display all settings
```

## Advanced Navigation

### Multi-Pane Navigation

| Key | Action | Description |
|-----|--------|-------------|
| `Ctrl+w h` | Left Pane | Move to left pane |
| `Ctrl+w j` | Down Pane | Move to pane below |
| `Ctrl+w k` | Up Pane | Move to pane above |
| `Ctrl+w l` | Right Pane | Move to right pane |
| `Ctrl+w w` | Next Pane | Cycle through panes |
| `Ctrl+w s` | Split Horizontal | Create horizontal split |
| `Ctrl+w v` | Split Vertical | Create vertical split |
| `Ctrl+w c` | Close Pane | Close current pane |
| `Ctrl+w o` | Only Pane | Close other panes |

### Bookmarks

| Key | Action | Description |
|-----|--------|-------------|
| `m<letter>` | Set Mark | Set bookmark |
| `'<letter>` | Go to Mark | Jump to bookmark |
| `:marks` | List Marks | Show all bookmarks |
| `:delmarks <letter>` | Delete Mark | Remove bookmark |

### Macros

| Key | Action | Description |
|-----|--------|-------------|
| `q<letter>` | Record Macro | Start recording |
| `q` | Stop Recording | End recording |
| `@<letter>` | Play Macro | Execute macro |
| `@@` | Repeat Macro | Repeat last macro |
| `:reg` | List Macros | Show all macros |

## Tips & Tricks

### Efficiency Tips

1. **Use number prefixes**: `5j` moves down 5 lines
2. **Combine movements**: `gg` → top, `G` → bottom
3. **Quick filters**: `/` + partial text + `Tab` for autocomplete
4. **Repeat commands**: `.` repeats last action
5. **Quick escape**: `Ctrl+[` works like `Esc`

### Power User Workflows

```bash
# Cancel all failed jobs
/state:FAILED    # Filter failed jobs
V                # Select all visible
c                # Cancel selected
y                # Confirm

# SSH to GPU node
2                # Switch to nodes view
/gpu             # Filter GPU nodes
s                # SSH to selected

# Export running jobs
/state:RUNNING   # Filter running
:export csv      # Export to CSV
```

### Custom Key Bindings

Create custom bindings in `~/.s9s/keybindings.yaml`:

```yaml
keybindings:
  global:
    "ctrl+j": "view jobs"
    "ctrl+n": "view nodes"
    "ctrl+/": "search"

  jobs:
    "K": "cancel --force"
    "H": "hold --reason='Manual hold'"
    "X": "export csv --file=jobs.csv"
```

## Next Steps

- Practice navigation in mock mode: `s9s --mock`
- Create your own keybindings
- Learn advanced filtering in our [Filtering Guide](filtering.md)
- Master batch operations in our [Batch Operations Guide](batch-operations.md)
- Explore individual view guides:
  - [Jobs View](views/jobs.md)
  - [Nodes View](views/nodes.md)
  - [Partitions View](views/partitions.md)
  - [Dashboard View](views/dashboard.md)
  - [Health View](views/health.md)
