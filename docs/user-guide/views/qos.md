# QoS View

The Quality of Service (QoS) view displays QoS policies that control job priorities, resource limits, and preemption behavior.

![QoS Demo](/assets/demos/qos.gif)

*QoS view showing policies, priorities, and resource limits*

## Overview

Quality of Service (QoS) policies define service tiers with different priorities, limits, and behaviors. QoS affects job scheduling priority and resource access, allowing differentiation between urgent production jobs and low-priority background tasks.

## Table Columns

| Column | Description |
|--------|-------------|
| **Name** | QoS policy name |
| **Priority** | Scheduling priority (color-coded) |
| **Preempt Mode** | Preemption behavior |
| **Max Jobs (User)** | Max jobs per user |
| **Max Submit (User)** | Max submitted jobs per user |
| **Max CPUs (User)** | Max CPUs per user |
| **Max Nodes (User)** | Max nodes per user |
| **Max Wall Time** | Maximum job duration |
| **Grace Time** | Time before preemption |
| **Flags** | Special QoS flags |

## QoS Priority

Priority determines scheduling order when resources are limited.

### Priority Levels

| Range | Color | Description | Typical Use |
|-------|-------|-------------|-------------|
| **>1000** | Green | High priority | Production, urgent work |
| **100-1000** | Yellow | Normal priority | Regular research |
| **<100** | White | Low priority | Background tasks |

**How priority works:**
- Higher priority jobs schedule before lower priority
- Priority combined with fair-share and age
- Does not guarantee immediate scheduling
- Affects queue position, not running jobs

**Example priority ladder:**
```
urgent:    10000  (Green)  - Critical production
high:       5000  (Green)  - Important deadlines
normal:      100  (Yellow) - Standard work
low:          10  (White)  - Best-effort
preemptible:   1  (White)  - Scavenger jobs
```

## Preemption Modes

QoS policies can preempt (interrupt) lower-priority jobs:

| Mode | Description | Use Case |
|------|-------------|----------|
| **None** | No preemption | Standard QoS |
| **Suspend** | Suspend lower priority jobs | Pausable workloads |
| **Requeue** | Requeue lower priority jobs | Checkpointed jobs |
| **Cancel** | Cancel lower priority jobs | Urgent production work |

### Preemption Example

```
Job A: QoS=normal, Priority=100, Running
Job B: QoS=urgent, Priority=10000, Pending

If urgent can preempt normal:
  → Job A suspended/requeued/canceled
  → Job B starts immediately
  → After grace time expires
```

### Grace Time

Time before preemption occurs:
- Allows job to checkpoint
- Gives notice for cleanup
- Job receives SIGTERM, then SIGKILL
- Common values: 60s, 300s, 600s

## Resource Limits

### Per-User Limits

**Max Jobs (User):**
- Maximum concurrent jobs per user with this QoS
- Running + pending

**Max Submit (User):**
- Maximum submitted jobs per user (including pending)
- Often higher than Max Jobs

**Max CPUs (User):**
- Total CPU allocation per user across all jobs

**Max Nodes (User):**
- Total nodes per user across all jobs

### Per-Job Limits

**Max Wall Time:**
- Maximum duration for a single job
- Format: `days-hours:minutes:seconds`
- Jobs requesting longer duration are rejected

**Examples:**
```
00:30:00 - 30 minutes
01:00:00 - 1 hour
12:00:00 - 12 hours
1-00:00:00 - 1 day
7-00:00:00 - 7 days
30-00:00:00 - 30 days
```

## QoS Actions

### View QoS Details
**Shortcut**: `Enter`

Shows comprehensive QoS policy information:

**Priority & Preemption:**
- Priority level and rating
- Preemption mode
- Grace time before preemption
- QoS policies this can preempt

**Per-User Limits:**
- Max jobs (concurrent)
- Max submit jobs (total)
- Max CPUs (total allocation)
- Max nodes (total allocation)
- Max memory (total)

**Per-Job Limits:**
- Max CPUs per job
- Max nodes per job
- Max memory per job
- Max wall time

**Usage Factors:**
- CPU usage multiplier
- Job size factor
- Time limit factor

**Flags:**
- DenyOnLimit - Deny jobs when limit reached (vs. queue)
- EnforceUsageThreshold - Enforce usage limits strictly
- NoReserve - Don't reserve resources
- PartitionMaxNodes - Respect partition node limits
- RequiresReservation - Requires active reservation
- UsageFactorSafe - Safe usage factor calculation

### Filtering

#### Simple Filter
**Shortcut**: `/`

Filter by:
- QoS name
- Priority level
- Preempt mode
- Flags

#### Advanced Filter
**Shortcut**: `F3`

Expression-based filtering:

```
priority:>1000
preempt:Suspend
maxjobs:>100
walltime:>7-00:00:00
```

**Supported fields:**
- `name` - QoS name
- `priority` - Priority value (supports >, <, >=, <=)
- `preempt` - Preemption mode
- `maxjobs` - Max jobs per user (supports comparison)
- `maxcpus` - Max CPUs per user (supports comparison)
- `walltime` - Max wall time (supports comparison)
- `flags` - Flag names

### Global Search
**Shortcut**: `Ctrl+F`

Search across all cluster resources.

## Sorting

Sort QoS by clicking column headers or using number keys.

**Useful sorting:**
- By priority (highest to lowest)
- By name (alphabetical)
- By max wall time (identify long-running QoS)

Press `1-9` to sort by column number.

## Keyboard Shortcuts Reference

### QoS Operations
| Key | Action |
|-----|--------|
| `Enter` | View QoS details |

### Filtering
| Key | Action |
|-----|--------|
| `/` | Simple filter |
| `F3` | Advanced filter |
| `Ctrl+F` | Global search |
| `ESC` | Exit filter mode |

### Data Management
| Key | Action |
|-----|--------|
| `R` | Manual refresh |
| `1-9` | Sort by column |

## QoS Details Example

When viewing QoS details (`Enter`):

```
QoS: high
Priority: 5000 (High)

Preemption:
  Mode: Suspend
  Grace Time: 300 seconds (5 minutes)
  Can Preempt: normal, low
  Cannot Preempt: urgent

Per-User Limits:
  Max Jobs: 50
  Max Submit Jobs: 100
  Max CPUs: 2048
  Max Nodes: 64
  Max Memory: 4TB

Per-Job Limits:
  Max CPUs: 512 per job
  Max Nodes: 32 per job
  Max Memory: 1TB per job
  Max Wall Time: 7-00:00:00 (7 days)

Usage Factors:
  CPU Usage Factor: 1.0
  Job Size Factor: 1.0

Flags:
  - DenyOnLimit
  - EnforceUsageThreshold

Users: 25
Accounts: 8
Current Jobs: 127
```

## Understanding QoS Limits

### QoS vs. Account vs. User Limits

Limits are enforced at multiple levels:

```
Most Restrictive Limit Applies

User:    Max 100 jobs
Account: Max 200 jobs total
QoS:     Max 50 jobs per user

Result: User can submit 50 jobs (QoS is most restrictive)
```

**Limit hierarchy:**
1. Partition limits (physical capacity)
2. Account limits (billing/allocation)
3. QoS limits (service level)
4. User limits (individual quotas)

### Limit Interactions Example

```
User: alice
Account: ml-team (Max CPUs: 3000)
QoS: high (Max CPUs/User: 2000)
Partition: gpu (Max CPUs: 5000)

Alice's effective limit: 2000 CPUs
  (QoS limit is most restrictive)

Even if account has 3000 available and
partition has 5000 capacity, alice can only
use 2000 CPUs with high QoS.
```

## Common QoS Patterns

### Tiered Service Levels

```
QoS          Priority  Max Wall  Preempt
-----------------------------------------------
urgent       10000     7d        Cancel normal,low
high          5000     7d        Suspend low
normal         100     30d       None
low             10     30d       None (preemptible)
```

### Use-Case Specific QoS

```
QoS          Priority  Max Wall  Max Nodes  Use Case
-----------------------------------------------------------
production   10000     1d        32         Critical jobs
interactive   8000     4h         1         Debugging
longrun        100     30d       64         Research
scavenger       10     7d       128         Best-effort
```

### Department QoS

```
QoS              Priority  Accounts
----------------------------------------
chemistry-high    1000     chemistry
physics-high      1000     physics
biology-high      1000     biology
shared-normal      100     All
```

## QoS Flags Explained

**DenyOnLimit:**
- Reject job submission when limit reached
- Alternative: queue jobs until resources available
- Use for: Hard limits, prevent queue buildup

**EnforceUsageThreshold:**
- Strictly enforce usage thresholds
- Deny jobs that would exceed fair-share
- Use for: Preventing usage abuse

**NoReserve:**
- Don't create advance reservations
- Jobs run when resources available
- Use for: Best-effort QoS

**RequiresReservation:**
- Jobs must run in active reservation
- Cannot run outside reservation window
- Use for: Dedicated resource allocations

**PartitionMaxNodes:**
- Respect partition node limits
- Even if QoS allows more
- Use for: Multi-partition QoS

## Tips

- **Choose appropriate QoS**: Don't use high priority for routine work
- **Understand preemption**: High-priority QoS may preempt your low-priority jobs
- **Check limits**: QoS limits may be more restrictive than account limits
- **Wall time matters**: Longer wall time != higher priority
- **Monitor usage**: Exceeding limits prevents job submission
- **Flag implications**: DenyOnLimit provides immediate feedback vs. silent queuing
- **Preemption grace time**: Use for checkpointing before job killed
- **Priority isn't immediate**: High priority doesn't bypass resource availability
- **Combine with account**: QoS + appropriate account for best scheduling

## Common Issues

### "QoS limit exceeded"
- Reached max jobs/CPUs for this QoS
- Wait for running jobs to complete
- Use different QoS if available
- Request limit increase

### "Job preempted"
- Higher priority QoS needed resources
- Job was suspended/requeued/canceled
- Review job logs for checkpoint
- Consider using higher priority QoS
- Implement checkpointing for resilience

### "QoS not allowed for account"
- Account not permitted to use this QoS
- Check account's allowed QoS list
- Use default or alternate QoS
- Contact admin to request access

### "Wall time exceeds QoS maximum"
- Requested time longer than QoS allows
- Reduce wall time request
- Use different QoS with longer limit
- Split job into smaller chunks
