# Command Reference

Complete reference of all S9S commands, shortcuts, and operations for efficient cluster management.

## Command Categories

### Navigation Commands
| Command | Description | Shortcut |
|---------|-------------|----------|
| `:view jobs` | Switch to jobs view | `1` |
| `:view nodes` | Switch to nodes view | `2` |
| `:view users` | Switch to users view | `3` |
| `:view partitions` | Switch to partitions view | `4` |
| `:view dashboard` | Switch to dashboard | `0` |
| `:help` | Show help | `?` |
| `:quit` | Exit S9S | `q` |

### Job Management Commands
| Command | Description | Example |
|---------|-------------|---------|
| `:submit` | Submit a new job | `:submit script.sh --cores=4` |
| `:cancel` | Cancel job(s) | `:cancel 12345` |
| `:hold` | Hold job(s) | `:hold 12345 --reason="Debug"` |
| `:release` | Release held job(s) | `:release 12345` |
| `:requeue` | Requeue job(s) | `:requeue 12345` |
| `:priority` | Change job priority | `:priority 12345 --priority=1000` |

### Node Management Commands
| Command | Description | Example |
|---------|-------------|---------|
| `:drain` | Drain node(s) | `:drain node001 --reason="Maintenance"` |
| `:resume` | Resume drained node(s) | `:resume node001` |
| `:ssh` | SSH to node | `:ssh node001` |
| `:reboot` | Reboot node | `:reboot node001` |

### Filtering Commands
| Command | Description | Example |
|---------|-------------|---------|
| `/` | Quick filter | `/user:alice` |
| `:filter` | Advanced filter | `:filter "state:RUNNING nodes:>4"` |
| `:clear` | Clear all filters | `:clear` |
| `:save-filter` | Save current filter | `:save-filter my-jobs` |

### Export Commands
| Command | Description | Example |
|---------|-------------|---------|
| `:export` | Export data | `:export csv --output=jobs.csv` |
| `:report` | Generate report | `:report utilization --period=week` |

## Detailed Command Reference

### Job Commands

**:submit** - Submit Jobs
```bash
:submit <script> [options]

Options:
  --cores=N           Number of CPU cores
  --mem=SIZE          Memory requirement (e.g., 32GB)
  --time=DURATION     Time limit (e.g., 2:00:00)
  --partition=NAME    Partition name
  --gpus=N           Number of GPUs
  --array=RANGE      Job array (e.g., 1-100)
  --dependency=TYPE:ID Job dependency
  --name=NAME        Job name
  --account=ACCOUNT  Account for billing

Examples:
  :submit analysis.sh --cores=16 --mem=64GB --time=4:00:00
  :submit ml_train.py --gpus=2 --partition=gpu
  :submit job_array.sh --array=1-100 --cores=4
```

**:cancel** - Cancel Jobs
```bash
:cancel <job_ids> [options]

Options:
  --force            Force cancellation
  --reason=TEXT      Cancellation reason
  --signal=SIGNAL    Signal to send (default: SIGTERM)

Examples:
  :cancel 12345
  :cancel 12345,12346,12347
  :cancel job[001-100]
  :cancel --force 12345 --reason="Emergency maintenance"
```

**:hold** - Hold Jobs
```bash
:hold <job_ids> [options]

Options:
  --reason=TEXT      Hold reason
  --user-hold        User hold (can release)
  --system-hold      System hold (admin required)

Examples:
  :hold 12345 --reason="Debugging required"
  :hold /state:PENDING --user-hold
```

**:release** - Release Jobs
```bash
:release <job_ids>

Examples:
  :release 12345
  :release /state:HELD user:alice
```

**:requeue** - Requeue Jobs
```bash
:requeue <job_ids> [options]

Options:
  --hold             Hold after requeue
  --reason=TEXT      Requeue reason

Examples:
  :requeue 12345
  :requeue /state:FAILED --hold
```

**:priority** - Change Priority
```bash
:priority <job_ids> --priority=VALUE

Examples:
  :priority 12345 --priority=1000
  :priority /user:alice --priority=+100
  :priority /state:PENDING --priority=500
```

### Node Commands

**:drain** - Drain Nodes
```bash
:drain <nodes> [options]

Options:
  --reason=TEXT      Drain reason
  --timeout=DURATION Max time to wait for jobs
  --force            Force drain (kill jobs)

Examples:
  :drain node001 --reason="Hardware upgrade"
  :drain node[001-010] --timeout=2h
  :drain /features:gpu --force --reason="Emergency"
```

**:resume** - Resume Nodes
```bash
:resume <nodes>

Examples:
  :resume node001
  :resume node[001-010]
  :resume /state:DRAIN
```

**:ssh** - SSH to Nodes
```bash
:ssh <node> [command] [options]

Options:
  --user=USERNAME    SSH username
  --key=KEYFILE      SSH key file
  --port=PORT        SSH port

Examples:
  :ssh node001
  :ssh node001 "uptime"
  :ssh node001 --user=admin "sudo reboot"
```

**:reboot** - Reboot Nodes
```bash
:reboot <nodes> [options]

Options:
  --force            Force reboot
  --delay=SECONDS    Delay before reboot
  --reason=TEXT      Reboot reason

Examples:
  :reboot node001
  :reboot node[001-010] --delay=300
  :reboot /state:MAINT --force
```

### Filtering Commands

**/** - Quick Filter
```bash
/<filter_expression>

Examples:
  /user:alice                    # Jobs by user alice
  /state:RUNNING                 # Running jobs
  /partition:gpu state:PENDING   # Pending GPU jobs
  /nodes:>4 time:>2h            # Large, long-running jobs
```

**:filter** - Advanced Filter
```bash
:filter "<complex_filter>"

Examples:
  :filter "user:alice,bob state:RUNNING partition:gpu"
  :filter "submitted:>1h runtime:<30m"
  :filter "priority:>1000 OR qos:high"
```

**:save-filter** - Save Filters
```bash
:save-filter <name> [filter]

Examples:
  :save-filter my-running-jobs "/user:alice state:RUNNING"
  :save-filter gpu-jobs "/partition:gpu"
  :save-filter # Save current filter
```

**:load-filter** - Load Filters
```bash
:load-filter <name>

Examples:
  :load-filter my-running-jobs
  :load-filter gpu-jobs
```

### Export Commands

**:export** - Export Data
```bash
:export <format> [options]

Formats:
  csv, json, excel, html, xml

Options:
  --output=FILE      Output filename
  --fields=LIST      Fields to export
  --filter=EXPR      Filter expression
  --selected         Export selected items only

Examples:
  :export csv --output=jobs.csv
  :export json --fields=JobID,User,State
  :export excel --selected
  :export csv --filter="user:alice" --output=alice_jobs.csv
```

**:report** - Generate Reports
```bash
:report <type> [options]

Types:
  utilization, efficiency, summary, costs, compliance

Options:
  --period=DURATION  Time period (day, week, month, year)
  --format=FORMAT    Output format (pdf, html, csv)
  --email=ADDRESS    Email report
  --users=LIST       Specific users
  --partitions=LIST  Specific partitions

Examples:
  :report utilization --period=month --format=pdf
  :report efficiency --users=alice,bob
  :report costs --partitions=gpu --email=admin@example.com
```

### View Commands

**:view** - Switch Views
```bash
:view <view_name>

Views:
  jobs, nodes, users, partitions, dashboard, performance

Examples:
  :view jobs
  :view nodes
  :view dashboard
```

**:columns** - Configure Columns
```bash
:columns <action> [column_name]

Actions:
  show, hide, add, remove, reset

Examples:
  :columns show Priority
  :columns hide Account
  :columns add GPUs
  :columns reset
```

**:sort** - Sort Data
```bash
:sort <column> [direction]

Direction:
  asc, desc

Examples:
  :sort JobID desc
  :sort SubmitTime asc
  :sort Priority desc
```

### Configuration Commands

**:config** - Configuration Management
```bash
:config <action> [key] [value]

Actions:
  get, set, list, reset, validate

Examples:
  :config get refresh_interval
  :config set theme dark
  :config list
  :config validate
```

**:theme** - Theme Management
```bash
:theme <theme_name>

Themes:
  dark, light, terminal, high-contrast

Examples:
  :theme dark
  :theme terminal
```

### System Commands

**:status** - System Status
```bash
:status [component]

Components:
  cluster, scheduler, storage, network

Examples:
  :status
  :status cluster
  :status scheduler
```

**:log** - View Logs
```bash
:log [options]

Options:
  --lines=N          Number of lines
  --follow           Follow log output
  --level=LEVEL      Log level filter
  --component=COMP   Component filter

Examples:
  :log --lines=100
  :log --follow --level=error
  :log --component=scheduler
```

**:refresh** - Refresh Data
```bash
:refresh [view]

Examples:
  :refresh           # Refresh current view
  :refresh jobs      # Refresh jobs data
  :refresh nodes     # Refresh nodes data
```

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
| `Tab` | Next view |
| `Shift+Tab` | Previous view |

### View-Specific Shortcuts

**Jobs View:**
| Key | Action |
|-----|--------|
| `c` | Cancel job |
| `h` | Hold job |
| `r` | Release job |
| `k` | Kill job |
| `d` | Show details |
| `s` | SSH to job nodes |
| `o` | Show output |
| `l` | Show logs |

**Nodes View:**
| Key | Action |
|-----|--------|
| `d` | Drain node |
| `r` | Resume node |
| `s` | SSH to node |
| `m` | Maintenance mode |
| `i` | Node info |
| `j` | Jobs on node |

**Selection Shortcuts:**
| Key | Action |
|-----|--------|
| `Space` | Toggle selection |
| `v` | Visual selection mode |
| `V` | Visual line mode |
| `Ctrl+A` | Select all |
| `Ctrl+D` | Deselect all |

### Filtering Shortcuts
| Key | Action |
|-----|--------|
| `/` | Quick filter |
| `n` | Next match |
| `N` | Previous match |
| `Ctrl+/` | Clear filter |

### Export Shortcuts
| Key | Action |
|-----|--------|
| `Ctrl+E` | Export CSV |
| `Ctrl+Shift+E` | Export JSON |
| `Alt+E` | Export Excel |
| `Ctrl+P` | Print/PDF |

## Advanced Usage

### Command Chaining
```bash
# Chain multiple commands
:filter "state:FAILED" ; :cancel --selected ; :clear

# Conditional execution
:submit job.sh --cores=16 && :priority $JOB_ID --priority=1000

# Pipe operations
:export json --fields=JobID,User | :import --target=database
```

### Variables and Substitution
```bash
# Use environment variables
:submit $HOME/scripts/analysis.sh --account=$SLURM_ACCOUNT

# Job ID substitution
:cancel $LAST_JOB_ID
:ssh node001 "kill $JOB_PID"

# Date/time substitution
:export csv --output=jobs_$(date +%Y%m%d).csv
```

### Scripting Commands
```bash
# Script execution
:script /path/to/commands.s9s

# Batch mode
:batch < batch_commands.txt

# Interactive scripting
:begin-script
:filter "user:alice state:RUNNING"
:priority --selected --priority=500
:end-script
```

## Emergency Commands

### Emergency Stop
```bash
:emergency-stop [options]

Options:
  --reason=TEXT      Emergency reason
  --notify           Send notifications
  --graceful         Graceful shutdown

Examples:
  :emergency-stop --reason="Power outage" --notify
```

### Emergency Evacuation
```bash
:evacuate <nodes> [options]

Options:
  --timeout=DURATION Evacuation timeout
  --force            Force evacuation
  --destination=NODE Target nodes

Examples:
  :evacuate rack01 --timeout=30m
  :evacuate /features:gpu --force
```

## Command Tips

### Efficiency Tips
1. **Use tab completion** - Most commands support tab completion
2. **Learn shortcuts** - Master keyboard shortcuts for speed
3. **Save filters** - Save commonly used filters
4. **Use aliases** - Create command aliases for frequent operations
5. **Batch operations** - Use visual selection for bulk actions

### Best Practices
1. **Be specific** - Use precise filters and job IDs
2. **Confirm destructive actions** - Always verify before canceling jobs
3. **Use dry-run mode** - Test commands with `--dry-run` when available
4. **Document reasons** - Always provide reasons for administrative actions
5. **Check permissions** - Ensure you have necessary permissions

## Command Help

### Getting Help
```bash
:help                    # General help
:help <command>         # Command-specific help
:help shortcuts         # Keyboard shortcuts
:help examples          # Command examples
:man <command>          # Detailed manual page
```

### Command Discovery
```bash
:commands               # List all commands
:commands --category=job # Commands by category
:search <keyword>       # Search commands
:recent                 # Recently used commands
```

## Next Steps

- Explore [configuration options](./configuration.md)
- Learn [API integration](./api.md)
- Review filtering syntax in [filtering guide](../guides/filtering.md)
- Master [batch operations](../guides/batch-operations.md) for efficiency
