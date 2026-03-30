# Command Reference

Complete reference of available S9S commands for efficient cluster management.

## Command Mode

Press `:` to enter command mode (vim-style). Commands support tab completion for both command names and arguments.

### Using Tab Completion

- **Browse all commands**: Press `Tab` on an empty prompt to see all available commands
- **Command completion**: Type `:req` and press `Tab` to complete to `:requeue`
- **Argument completion**: Type `:cancel ` and press `Tab` to see available job IDs
- **Smart suggestions**: Completions are context-aware based on cached view data

**Examples:**
```
:<Tab>              → Shows all available commands
:req<Tab>           → :requeue
:requeue <Tab>      → Shows: 12345, 12346, 12347...
:drain nod<Tab>     → Shows: node01, node02, node03...
:q<Tab>             → Shows: q, qos, quit
```

## Available Commands

### Navigation Commands
| Command | Description | Shortcut |
|---------|-------------|----------|
| `:jobs` | Switch to jobs view | `1` |
| `:nodes` | Switch to nodes view | `2` |
| `:partitions` | Switch to partitions view | `3` |
| `:reservations` | Switch to reservations view | `4` |
| `:qos` | Switch to QoS view | `5` |
| `:accounts` | Switch to accounts view | `6` |
| `:users` | Switch to users view | `7` |
| `:dashboard` | Switch to dashboard | `8` |
| `:health` | Switch to health view | `9` |
| `:performance` | Switch to performance view | `0` |
| `:help` or `:h` | Show help | `?` |
| `:quit` or `:q` | Exit S9S | `q` |

### View Management Commands
| Command | Description | Example |
|---------|-------------|---------|
| `:refresh` or `:r` | Refresh current view | `:refresh` |
| `:layout` or `:layouts` | Show layout switcher | `:layout` |
| `:config` or `:configuration` or `:settings` | Show configuration | `:config` |

### Job Management Commands
| Command | Description | Example | Autocomplete |
|---------|-------------|---------|--------------|
| `:cancel JOBID` | Cancel a specific job | `:cancel 12345` | Job IDs from active jobs |
| `:hold JOBID` | Hold a specific job | `:hold 12345` | Job IDs from active jobs |
| `:release JOBID` | Release a held job | `:release 12345` | Job IDs from active jobs |
| `:requeue JOBID` | Requeue a failed/completed job | `:requeue 12345` | Job IDs from active jobs |

**Tab Completion:** After typing the command and pressing space, press `Tab` to see available job IDs from the currently loaded jobs view.

**Note:** These commands operate on specific job IDs. For batch operations on selected jobs in the UI, use the keyboard shortcuts (`c`, `h`, `r`) described in the Interactive Operations section.

### Node Management Commands
| Command | Description | Example | Autocomplete |
|---------|-------------|---------|--------------|
| `:drain NODE [REASON]` | Drain a node (make unavailable for new jobs) | `:drain node01 maintenance` | Node names from active nodes |
| `:resume NODE` | Resume a drained node | `:resume node01` | Node names from active nodes |

**Tab Completion:** After typing the command and pressing space, press `Tab` to see available node names from the currently loaded nodes view.

**Note:** The reason for draining is optional. If not provided, defaults to "Drained via s9s command".

### Update Commands

Check for and install new versions of s9s directly from the terminal.

| Command | Description | Example |
|---------|-------------|---------|
| `s9s update` | Check for and install the latest version | `s9s update` |
| `s9s update --check` | Only check for updates, don't install | `s9s update --check` |
| `s9s update --force` | Skip confirmation prompt | `s9s update --force` |
| `s9s update --pre-release` | Include pre-release versions | `s9s update --pre-release` |
| `s9s update --target VERSION` | Update to a specific version | `s9s update --target v0.7.0` |

**Example output of `s9s update --check`:**
```
Current version: 0.8.0
Latest version:  0.9.0
Release:         https://github.com/jontk/s9s/releases/tag/v0.9.0

A new version is available: 0.8.0 -> 0.9.0
```

**Notes:**
- Development builds (`Version=dev`) cannot be updated; install a release version first
- If the binary is in a system directory (e.g., `/usr/local/bin`), you may need `sudo s9s update`
- Using `--target` with an older version will show a downgrade warning before proceeding
- Auto-update checks can be configured in `~/.s9s/config.yaml` — see [Auto-Update Configuration](configuration.md#auto-update-configuration)

### Template Management Commands

Manage job submission templates from the command line. Templates can originate from three sources: **builtin** (shipped with s9s), **config** (defined in your configuration file), and **saved** (user-exported templates stored on disk).

| Command | Description | Example |
|---------|-------------|---------|
| `s9s templates list` | List all templates with source | `s9s templates list` |
| `s9s templates export` | Export templates to ~/.s9s/templates/ | `s9s templates export` |
| `s9s templates export NAME` | Export a specific template | `s9s templates export "GPU Job"` |
| `s9s templates export --force` | Overwrite existing files | `s9s templates export --force` |
| `s9s templates export --dir PATH` | Export to custom directory | `s9s templates export --dir /tmp/templates` |

**Example output of `s9s templates list`:**
```
NAME                   SOURCE    DESCRIPTION
Basic Batch Job        builtin   Simple batch job for serial computations
GPU Training Job       config    PyTorch training on GPU partition
My Custom Job          saved     Custom template from user
```

See [Job Submission Configuration](../getting-started/configuration.md#job-submission-configuration) for details on defining templates in your configuration file.

### Filtering
| Key | Action |
|-----|--------|
| `/` | Quick filter (type to filter current view) |
| `Esc` | Clear filter |

### Interactive Operations

S9S provides interactive keyboard shortcuts for common operations within each view:

**Jobs View:**
| Key | Action |
|-----|--------|
| `c` | Cancel selected job |
| `H` | Hold selected job |
| `r` | Release selected job |
| `d` | Show job dependencies |
| `o` | Show job output |
| `f` | Advanced filter |
| `x` | Actions menu |
| `Enter` | Show job details |
| `Space` | Toggle selection |
| `v` | Visual selection mode |

**Nodes View:**
| Key | Action |
|-----|--------|
| `s` | SSH to selected node |
| `d` | Drain selected node |
| `r` | Resume drained node |
| `i` | Toggle idle state filter |
| `Enter` | Show node details |
| `Space` | Toggle group expansion |

**Batch Operations:**
- Select multiple items using `Space` or `v` (visual mode)
- Use keyboard shortcuts on selections for batch operations
- See [Batch Operations Guide](../user-guide/batch-operations.md) for details

### Export Commands

Export functionality is available through the interactive UI:

**Available Formats:**
- CSV (`.csv`)
- JSON (`.json`)
- Markdown (`.md`)
- Text (`.txt`)
- HTML (`.html`)

**Access Export:**
- Press `e` in any view to open export dialog
- Select format and destination
- Configure export options interactively

## Keyboard Shortcuts

### Global Shortcuts
| Key | Action |
|-----|--------|
| `q` / `Q` | Quit |
| `?` | Help |
| `Ctrl+C` | Quit |
| `Ctrl+K` | Cluster switcher |
| `Esc` | Clear/Cancel |
| `F1` | Help |
| `F5` | Refresh |

### Navigation Shortcuts
| Key | Action |
|-----|--------|
| `1` | Jobs view |
| `2` | Nodes view |
| `3` | Partitions view |
| `4` | Reservations view |
| `5` | QoS view |
| `6` | Accounts view |
| `7` | Users view |
| `8` | Dashboard view |
| `9` | Health view |
| `0` | Performance view |
| `h` | Previous view |
| `l` | Next view |
| `Tab` | Next view |
| `Shift+Tab` | Previous view |

### View-Specific Shortcuts

**Jobs View:**
| Key | Action |
|-----|--------|
| `c` | Cancel job |
| `H` | Hold job |
| `r` | Release job |
| `d` | Show dependencies |
| `o` | Show output |
| `f` | Advanced filter |
| `x` | Actions menu |
| `Enter` | Show details |

**Nodes View:**
| Key | Action |
|-----|--------|
| `s` | SSH to node |
| `d` | Drain node |
| `i` | Toggle idle filter |
| `Enter` | Node details |

**Selection Shortcuts:**
| Key | Action |
|-----|--------|
| `Space` | Toggle selection |
| `v` / `V` | Toggle multi-select mode |

### Filtering Shortcuts
| Key | Action |
|-----|--------|
| `/` | Quick filter |
| `Esc` | Clear filter |

### Export Shortcuts
| Key | Action |
|-----|--------|
| `e` | Export dialog (all data views) |

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

Press `/` for plain text search across all columns. Press `Ctrl+F` in any data view to open global search across all entity types. The advanced filter bar supports field-specific expressions:

```bash
# Advanced filter examples (Ctrl+F)
# Filter by user
user=alice

# Filter by state
state=RUNNING

# Filter by partition
partition=gpu

# Combine filters
user=alice state=RUNNING partition=gpu

# Numeric comparisons
nodes>4 priority>=1000
```

See the [Filtering Guide](../user-guide/filtering.md) for comprehensive filter documentation.

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

- [User Guide](../user-guide/navigation.md) - Comprehensive usage documentation
- [Filtering Guide](../user-guide/filtering.md) - Advanced filtering syntax
- [Batch Operations](../user-guide/batch-operations.md) - Batch operation workflows

## Next Steps

- Master [keyboard shortcuts](../user-guide/navigation.md) for efficient navigation
- Learn [filtering syntax](../user-guide/filtering.md) for precise data views
- Explore [batch operations](../user-guide/batch-operations.md) for bulk job management
