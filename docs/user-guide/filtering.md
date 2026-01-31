# Advanced Filtering Guide

s9s provides powerful filtering capabilities to help you find exactly what you need across thousands of jobs, nodes, and resources. This guide covers all filtering features and syntax.

## Quick Filter Demo

See filtering in action:

![Search Demo](/assets/demos/search.gif)

## Basic Filtering

### Quick Filter

Press `/` in any view to activate quick filter:

```bash
# Simple text search
/analysis      # Find items containing "analysis"
/GPU          # Case-insensitive search for "GPU"
/node001      # Find specific node
```

### Clear Filters

- `Esc` - Clear current filter and exit search mode
- `:clear` - Clear all active filters
- `Ctrl+/` - Toggle last filter on/off

## Filter Syntax

### Field-Specific Filters

Target specific fields with the `field:value` syntax:

```bash
# Job filters
/user:alice              # Jobs by user alice
/name:simulation        # Jobs with "simulation" in name
/jobid:12345           # Specific job ID
/state:RUNNING         # Jobs in RUNNING state
/partition:gpu         # Jobs in GPU partition

# Node filters
/node:node001          # Specific node
/state:idle           # Idle nodes
/features:gpu         # Nodes with GPU feature
/memory:>128GB       # Nodes with >128GB RAM
```

### Operators

s9s supports various comparison operators:

| Operator | Description | Example |
|----------|-------------|---------|
| `:` | Equals | `/state:RUNNING` |
| `!` | Not equals | `/state:!FAILED` |
| `>` | Greater than | `/nodes:>4` |
| `<` | Less than | `/runtime:<1h` |
| `>=` | Greater or equal | `/priority:>=1000` |
| `<=` | Less or equal | `/memory:<=64GB` |
| `~` | Regex match | `/name:~"test.*2023"` |
| `*` | Wildcard | `/name:analysis*` |
| `..` | Range | `/nodes:2..8` |

## Advanced Filtering

### Compound Filters

Combine multiple filters with spaces (AND logic):

```bash
# Jobs by alice in GPU partition
/user:alice partition:gpu

# Running jobs with more than 4 nodes
/state:RUNNING nodes:>4

# Failed jobs submitted today
/state:FAILED submitted:today
```

### OR Logic

Use comma-separated values for OR logic:

```bash
# Jobs in RUNNING or PENDING state
/state:RUNNING,PENDING

# Jobs in gpu or cpu partition
/partition:gpu,cpu

# Jobs by alice or bob
/user:alice,bob
```

### Complex Queries

Combine AND and OR logic:

```bash
# Alice or Bob's GPU jobs
/user:alice,bob partition:gpu state:RUNNING

# High priority pending jobs in specific partitions
/state:PENDING priority:>1000 partition:gpu,highmem
```

## Time-Based Filters

### Relative Time

Use human-readable time expressions:

```bash
# Submitted time
/submitted:<1h         # Less than 1 hour ago
/submitted:>1d        # More than 1 day ago
/submitted:today      # Submitted today
/submitted:yesterday  # Submitted yesterday
/submitted:thisweek   # This week
/submitted:lastweek   # Last week

# Runtime
/runtime:>30m         # Running more than 30 minutes
/runtime:<2h         # Running less than 2 hours
/runtime:1h-3h      # Between 1 and 3 hours
```

### Absolute Time

Use specific dates and times:

```bash
# ISO format
/submitted:>2023-12-01
/started:<2023-12-25T18:00:00

# Date shortcuts
/submitted:2023-12-01..2023-12-31
/ended:yesterday..today
```

## Numeric Filters

### Resource Filters

Filter by resource usage:

```bash
# Node count
/nodes:4              # Exactly 4 nodes
/nodes:>8            # More than 8 nodes
/nodes:2..16        # Between 2 and 16 nodes

# Memory (supports units)
/memory:>32GB        # More than 32GB
/memory:64GB..128GB  # Between 64GB and 128GB
/memory:<=1TB       # Up to 1TB

# CPUs
/cpus:>48           # More than 48 CPUs
/cpus:24,48,96     # Specific CPU counts

# GPUs
/gpus:>0           # Has GPUs
/gpus:8            # Exactly 8 GPUs
```

### Priority and QoS

```bash
# Priority
/priority:>1000      # High priority jobs
/priority:0..500    # Low to medium priority

# QoS
/qos:normal         # Normal QoS
/qos:high,critical  # High or critical QoS
```

## Regular Expressions

Use regex for complex pattern matching:

```bash
# Enable regex with ~ operator
/name:~"analysis_\d{4}"      # analysis_0001, analysis_0002, etc.
/user:~"(alice|bob)_.*"      # alice_* or bob_* users
/output:~"error|warning"      # Find errors or warnings
/script:~"python.*\.py$"     # Python scripts

# Case-insensitive regex
/name:~"(?i)GPU"             # Matches gpu, GPU, Gpu, etc.

# Negative lookahead
/name:~"^(?!test).*"         # Not starting with "test"
```

## State Filters

### Job States

```bash
# Single state
/state:RUNNING
/state:PENDING
/state:COMPLETED
/state:FAILED

# Multiple states
/state:RUNNING,PENDING
/state:!COMPLETED        # Not completed

# State groups (s9s shortcuts)
/state:active           # RUNNING,PENDING
/state:ended           # COMPLETED,FAILED,CANCELLED
/state:problem         # FAILED,TIMEOUT,NODE_FAIL
```

### Node States

```bash
# Basic states
/state:idle
/state:allocated
/state:down
/state:drain

# Compound states
/state:idle+drain      # Idle and draining
/state:allocated+drain # Allocated but draining

# State shortcuts
/state:available       # idle,mixed
/state:unusable       # down,drain,maint
```

## Saved Filters

### Creating Saved Filters

Save frequently used filters:

```bash
# Save current filter
:filter save my-jobs "/user:${USER} state:active"

# Save with description
:filter save gpu-pending "/partition:gpu state:PENDING" --desc "Pending GPU jobs"

# Save complex filter
:filter save inefficient "/state:RUNNING efficiency:<0.5 runtime:>1h"
```

### Using Saved Filters

```bash
# List saved filters
:filter list

# Load saved filter
:filter load my-jobs
# Or use shortcut
/~my-jobs

# Edit saved filter
:filter edit gpu-pending

# Delete saved filter
:filter delete old-filter
```

### Filter Presets

s9s includes built-in filter presets:

| Preset | Description | Filter |
|--------|-------------|--------|
| `~active` | Active jobs | `state:RUNNING,PENDING` |
| `~mine` | My jobs | `user:${USER}` |
| `~recent` | Recent jobs | `submitted:<1d` |
| `~failed` | Failed jobs | `state:FAILED,TIMEOUT` |
| `~gpu` | GPU jobs | `gpus:>0` |
| `~long` | Long jobs | `runtime:>1d` |
| `~today` | Today's jobs | `submitted:today` |

## Dynamic Filters

### Auto-Refresh Filters

Filters that update automatically:

```bash
# Jobs submitted in last hour (updates)
/submitted:<1h

# Currently running jobs
/state:RUNNING elapsed:>0

# Recently completed
/state:COMPLETED ended:<10m
```

### Context-Aware Filters

Filters that adapt to current view:

```bash
# In Jobs view
/efficiency:<0.7      # Low efficiency jobs

# In Nodes view
/load:>0.8           # High load nodes

# In Users view
/active:true         # Users with running jobs
```

## Filter Examples

### Common Use Cases

#### Find Stuck Jobs
```bash
/state:PENDING submitted:>1h reason:!Resources
```

#### Inefficient GPU Jobs
```bash
/partition:gpu state:RUNNING gpu_util:<50%
```

#### Failed Jobs Today
```bash
/state:FAILED ended:today user:${USER}
```

#### High Memory Jobs
```bash
/memory:>500GB state:RUNNING,PENDING
```

#### Jobs Near Time Limit
```bash
/state:RUNNING time_left:<30m
```

### Power User Filters

#### Complex Resource Query
```bash
/nodes:>16 cpus:>512 memory:>1TB partition:large state:PENDING
```

#### Multi-User Team Filter
```bash
/user:~"^(alice|bob|charlie)" project:ml_research state:!COMPLETED
```

#### Performance Analysis
```bash
/state:RUNNING runtime:>2h efficiency:<0.5 partition:!debug
```

## Performance Tips

### Efficient Filtering

1. **Use indexes**: Filter by indexed fields first (JobID, User, State)
2. **Narrow scope**: Start with restrictive filters, then broaden
3. **Save complex filters**: Reuse instead of retyping
4. **Use shortcuts**: Learn preset filters for common queries

### Filter Optimization

```bash
# Good: Indexed field first
/state:RUNNING user:alice partition:gpu

# Less efficient: Unindexed field first
/name:analysis state:RUNNING user:alice
```

## Filter Shortcuts

### Keyboard Shortcuts

| Key | Action | Description |
|-----|--------|-------------|
| `/` | Start filter | Enter filter mode |
| `Tab` | Autocomplete | Complete filter fields |
| `↑/↓` | History | Browse filter history |
| `Ctrl+/` | Toggle | Toggle last filter |
| `Alt+/` | Presets | Show filter presets |
| `Esc` | Clear | Clear current filter |

### Command Mode Filters

```bash
# Apply filter via command
:filter "state:RUNNING nodes:>4"

# Chain filters
:filter add "user:alice"
:filter add "partition:gpu"

# Remove specific filter
:filter remove "user:alice"
```

## Troubleshooting Filters

### Common Issues

**No Results**
- Check filter syntax
- Verify field names
- Try broader criteria
- Check for typos

**Too Many Results**
- Add more specific criteria
- Use compound filters
- Filter by state first
- Add time constraints

**Slow Filters**
- Avoid regex on large datasets
- Use indexed fields first
- Limit time range
- Consider saved filters

## Next Steps

- Practice filters in [Mock Mode](../MOCK_MODE.md)
- Set up [Saved Filters](../getting-started/configuration.md#filters)
- Learn [Batch Operations](../user-guide/job-management.md) with filters
- Explore filtering in specific views:
  - [Jobs View](../user-guide/views/jobs.md)
  - [Nodes View](../user-guide/views/nodes.md)
  - [Partitions View](../user-guide/views/partitions.md)
