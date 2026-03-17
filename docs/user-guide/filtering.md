# Advanced Filtering Guide

s9s provides powerful filtering capabilities to help you find exactly what you need across thousands of jobs, nodes, and resources. This guide covers all filtering features and syntax.

## Quick Filter Demo

See filtering in action:

![Search Demo](/assets/demos/search.gif)

## Basic Filtering

### Quick Filter

Press `/` in any view to activate quick filter:

```bash
# Simple text search (matches across all visible columns)
/analysis      # Find items containing "analysis"
/GPU          # Case-insensitive search for "GPU"
/node001      # Find specific node
```

The quick filter is a plain text search only. The only special prefix is `p:` for partition filtering (works in both Jobs and Nodes views).

### Clear Filters

- `Esc` - Clear current filter and exit search mode

## Advanced Filter

### Field-Specific Filters

Press `Ctrl+F` to open the advanced filter (available in all data views). Use `field=value` syntax:

```bash
# Job filters (advanced filter, Ctrl+F)
user=alice              # Jobs by user alice
name~simulation        # Jobs containing "simulation" in name
state=RUNNING          # Jobs in RUNNING state
partition=gpu          # Jobs in GPU partition

# Node filters (advanced filter, Ctrl+F)
state=idle             # Idle nodes
name~compute           # Nodes containing "compute"
```

### Operators

s9s supports the following comparison operators in the advanced filter:

| Operator | Description | Example |
|----------|-------------|---------|
| `=` | Equals | `state=RUNNING` |
| `!=` | Not equals | `state!=FAILED` |
| `>` | Greater than | `cpus>4` |
| `<` | Less than | `priority<1000` |
| `>=` | Greater or equal | `priority>=1000` |
| `<=` | Less or equal | `cpus<=64` |
| `~` | Contains | `name~analysis` |
| `!~` | Not contains | `name!~test` |
| `=~` | Regex match | `name=~"test.*2023"` |
| `in` | In list | `state in (RUNNING,PENDING)` |
| `not in` | Not in list | `state not in (COMPLETED,FAILED)` |

## Compound Filters

### AND Logic

Combine multiple expressions with spaces (AND logic) in the advanced filter:

```bash
# Jobs by alice in GPU partition
user=alice partition=gpu

# Running jobs with more than 4 CPUs
state=RUNNING cpus>4
```

### OR Logic (in operator)

Use the `in` operator for matching against multiple values:

```bash
# Jobs in RUNNING or PENDING state
state in (RUNNING,PENDING)

# Jobs in gpu or cpu partition
partition in (gpu,cpu)
```

## Numeric and Time Filters

The advanced filter supports numeric comparisons and automatic parsing of memory sizes (e.g., `4G`, `1024M`) and durations (e.g., `2:30:00`, `30m`):

```bash
# Resource filters (advanced filter, Ctrl+F)
cpus>48              # More than 48 CPUs
cpus>=8              # 8 or more CPUs
memory>4G            # More than 4GB memory
priority>=1000       # High priority jobs

# QoS
qos=normal           # Normal QoS
```

## Regular Expressions

Use regex for complex pattern matching with the `=~` operator in the advanced filter:

```bash
# Enable regex with =~ operator (advanced filter, Ctrl+F)
name=~"analysis_\d{4}"      # analysis_0001, analysis_0002, etc.
user=~"(alice|bob)_.*"       # alice_* or bob_* users
name=~"(?i)GPU"              # Case-insensitive match

# Note: ~ (without =) is the contains operator, not regex
name~analysis                 # Matches if name contains "analysis"
```

## State Filters

### Job States (Advanced Filter)

```bash
# Single state
state=RUNNING
state=PENDING
state=COMPLETED
state=FAILED

# Multiple states using in operator
state in (RUNNING,PENDING)
state!=COMPLETED              # Not completed
```

### Node States (Advanced Filter)

```bash
# Basic states
state=idle
state=allocated
state=down
state=drain
```

### State Toggle Shortcuts

In the Jobs view, use keyboard shortcuts to toggle state filters quickly:
- `p`/`P` -- toggle pending state filter
- `a`/`A` -- show all states (clear filter)

In the Nodes view:
- `i`/`I` -- toggle idle state filter
- `m`/`M` -- toggle mixed state filter
- `a`/`A` -- show all states (clear filter)

## Saved Filters and Presets (Planned)

> **Note**: Saved filters, filter presets (`~` prefix shortcuts like `/~active`, `/~mine`), and `:filter save/load/list/delete` commands are planned features. See [#119](https://github.com/jontk/s9s/issues/119) for details.
>
> In the meantime, you can re-enter filters manually using `/` in any view.

## Filter Behavior

### Auto-Refresh

When auto-refresh is enabled (toggle with `m`/`M` in Jobs view), filters remain active as data refreshes. The filtered view updates automatically with each refresh cycle.

### Context-Aware Fields

The available filter fields depend on the current view. The advanced filter supports field aliases for convenience:

| Alias | Canonical Field |
|-------|----------------|
| `name` | Name |
| `user` | User |
| `state` | State |
| `partition` | Partition |
| `node`/`nodes` | NodeList |
| `cpu`/`cpus` | CPUs |
| `mem`/`memory` | Memory |
| `account` | Account |
| `qos` | QoS |
| `priority` | Priority |
| `time` | TimeUsed |
| `timelimit` | TimeLimit |

## Filter Examples

### Common Use Cases (Advanced Filter, Ctrl+F)

#### Find Pending Jobs
```bash
state=PENDING
```

#### GPU Partition Jobs
```bash
partition=gpu state=RUNNING
```

#### Jobs by User
```bash
user=alice state!=COMPLETED
```

#### High CPU Jobs
```bash
cpus>64 state in (RUNNING,PENDING)
```

#### Multi-User Team Filter
```bash
user=~"^(alice|bob|charlie)"
```

## Performance Tips

### Efficient Filtering

1. **Use state toggles**: Keyboard shortcuts like `p` (pending) are fastest
2. **Quick filter first**: Use `/` for simple text searches
3. **Advanced filter for precision**: Use `Ctrl+F` when you need field-specific matching (Jobs and Nodes views)

## Filter Shortcuts

### Keyboard Shortcuts

| Key | Action | Description |
|-----|--------|-------------|
| `/` | Quick filter | Enter plain text filter mode |
| `Ctrl+F` | Advanced filter | Open advanced field-specific filter (all data views) |
| `Esc` | Clear | Clear current filter |

### Using Filters

Use `/` in any view for plain text search across all columns. Use `Ctrl+F` to open the advanced filter for field-specific queries with operators (available in Jobs and Nodes views). Press `Esc` to clear the filter.

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

- Practice filters in [Mock Mode](../guides/mock-mode.md)
- Learn [Batch Operations](../user-guide/job-management.md) with filters
- Explore filtering in specific views:
  - [Jobs View](../user-guide/views/jobs.md)
  - [Nodes View](../user-guide/views/nodes.md)
  - [Partitions View](../user-guide/views/partitions.md)
