# Command Reference

Complete reference of available S9S commands for efficient cluster management.

## Available Commands

### Navigation Commands
| Command | Description | Shortcut |
|---------|-------------|----------|
| `:view jobs` or `:jobs` | Switch to jobs view | `1` or `j` |
| `:view nodes` or `:nodes` | Switch to nodes view | `2` or `n` |
| `:view users` or `:users` | Switch to users view | `3` |
| `:view partitions` or `:partitions` | Switch to partitions view | `4` or `p` |
| `:view dashboard` or `:dashboard` | Switch to dashboard | `0` |
| `:view reservations` or `:reservations` | Switch to reservations view | - |
| `:view qos` or `:qos` | Switch to QoS view | - |
| `:view accounts` or `:accounts` | Switch to accounts view | - |
| `:view health` or `:health` | Switch to health view | - |
| `:view performance` or `:performance` | Switch to performance view | - |
| `:help` or `:h` | Show help | `?` |
| `:quit` or `:q` | Exit S9S | `q` |

### View Management Commands
| Command | Description | Example |
|---------|-------------|---------|
| `:refresh` or `:r` | Refresh current view | `:refresh` |
| `:prefs` or `:preferences` | Show preferences | `:prefs` |
| `:layout` or `:layouts` | Show layout switcher | `:layout` |
| `:config` or `:configuration` or `:settings` | Show configuration | `:config` |

### Job Management Commands
| Command | Description | Example |
|---------|-------------|---------|
| `:cancel JOBID` | Cancel a specific job | `:cancel 12345` |
| `:hold JOBID` | Hold a specific job | `:hold 12345` |
| `:release JOBID` | Release a held job | `:release 12345` |
| `:requeue JOBID` | Requeue a failed/completed job | `:requeue 12345` |

**Note:** These commands operate on specific job IDs. For batch operations on selected jobs in the UI, use the keyboard shortcuts (`c`, `h`, `r`) described in the Interactive Operations section.

### Node Management Commands
| Command | Description | Example |
|---------|-------------|---------|
| `:drain NODE [REASON]` | Drain a node (make unavailable for new jobs) | `:drain node01 maintenance` |
| `:resume NODE` | Resume a drained node | `:resume node01` |

**Note:** The reason for draining is optional. If not provided, defaults to "Drained via s9s command".

### Filtering and Search
| Command | Description | Example |
|---------|-------------|---------|
| `/` | Quick filter | `/user:alice` |
| `:filter` | Advanced filter | `:filter "state:RUNNING nodes:>4"` |
| `:clear` | Clear all filters | `:clear` |

### Interactive Operations

S9S provides interactive keyboard shortcuts for common operations within each view:

**Jobs View:**
| Key | Action |
|-----|--------|
| `c` | Cancel selected job |
| `h` | Hold selected job |
| `r` | Release selected job |
| `d` | Show job details |
| `o` | Show job output |
| `Space` | Toggle selection |
| `v` | Visual selection mode |

**Nodes View:**
| Key | Action |
|-----|--------|
| `s` | SSH to selected node |
| `d` | Show node details |
| `i` | Node info |
| `Space` | Toggle selection |

**Batch Operations:**
- Select multiple items using `Space` or `v` (visual mode)
- Use keyboard shortcuts on selections for batch operations
- See [Batch Operations Guide](/home/jontk/src/github.com/jontk/s9s/docs/user-guide/batch-operations.md) for details

### Export Commands

Export functionality is available through the interactive UI:

**Available Formats:**
- CSV (`.csv`)
- JSON (`.json`)
- Markdown (`.md`)
- Text (`.txt`)

**Access Export:**
- Press `Ctrl+E` in any view to open export dialog
- Select format and destination
- Configure export options interactively

## Keyboard Shortcuts

### Global Shortcuts
| Key | Action |
|-----|--------|
| `q` | Quit |
| `?` | Help |
| `r` | Refresh |
| `Ctrl+C` | Interrupt/Cancel |
| `Esc` | Clear/Cancel |

### Navigation Shortcuts
| Key | Action |
|-----|--------|
| `0` | Dashboard view |
| `1` | Jobs view |
| `2` | Nodes view |
| `3` | Users view |
| `4` | Partitions view |
| `j` | Jobs view |
| `n` | Nodes view |
| `p` | Partitions view |
| `Tab` | Next view |
| `Shift+Tab` | Previous view |

### View-Specific Shortcuts

**Jobs View:**
| Key | Action |
|-----|--------|
| `c` | Cancel job |
| `h` | Hold job |
| `r` | Release job |
| `d` | Show details |
| `o` | Show output |
| `l` | Show logs |

**Nodes View:**
| Key | Action |
|-----|--------|
| `s` | SSH to node |
| `i` | Node info |
| `d` | Show details |

**Selection Shortcuts:**
| Key | Action |
|-----|--------|
| `Space` | Toggle selection |
| `v` | Visual selection mode |
| `V` | Visual line mode |
| `Ctrl+A` | Select all |

### Filtering Shortcuts
| Key | Action |
|-----|--------|
| `/` | Quick filter |
| `Ctrl+/` | Clear filter |

### Export Shortcuts
| Key | Action |
|-----|--------|
| `Ctrl+E` | Export dialog |

## Advanced Features

### Visual Selection Mode

Select multiple items for batch operations:

```bash
# Enter visual mode
Press v

# Select range with arrow keys
Use ↑/↓ to expand selection

# Apply operation to selection
Press c to cancel selected jobs
Press h to hold selected jobs
Press r to release selected jobs
```

### Filter Syntax

The filter system supports various filter expressions:

```bash
# Filter by user
/user:alice

# Filter by state
/state:RUNNING

# Filter by partition
/partition:gpu

# Combine filters
/user:alice state:RUNNING partition:gpu

# Numeric comparisons
/nodes:>4 time:>2h
```

See the [Filtering Guide](/home/jontk/src/github.com/jontk/s9s/docs/user-guide/filtering.md) for comprehensive filter documentation.

### SSH to Nodes

SSH functionality is available in the Nodes view:

```bash
# Navigate to Nodes view
:nodes

# Select a node with arrow keys
# Press 's' to open SSH session

# SSH terminal opens in the UI
# Exit SSH session to return to S9S
```

## Getting Help

### Command-Specific Help
```bash
:help                    # General help
?                        # Quick help
```

### Documentation Resources

- [User Guide](/home/jontk/src/github.com/jontk/s9s/docs/user-guide) - Comprehensive usage documentation
- [Filtering Guide](/home/jontk/src/github.com/jontk/s9s/docs/user-guide/filtering.md) - Advanced filtering syntax
- [Batch Operations](/home/jontk/src/github.com/jontk/s9s/docs/user-guide/batch-operations.md) - Batch operation workflows

## Next Steps

- Master [keyboard shortcuts](/home/jontk/src/github.com/jontk/s9s/docs/user-guide/navigation.md) for efficient navigation
- Learn [filtering syntax](/home/jontk/src/github.com/jontk/s9s/docs/user-guide/filtering.md) for precise data views
- Explore [batch operations](/home/jontk/src/github.com/jontk/s9s/docs/user-guide/batch-operations.md) for bulk job management
