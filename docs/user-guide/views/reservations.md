# Reservations View

The Reservations view displays scheduled resource reservations for dedicated access to cluster resources during specific time windows.

![Reservations Demo](/assets/demos/reservations.gif)

*Reservations view showing active and future reservations with time windows*

## Overview

Reservations provide exclusive or prioritized access to cluster resources during defined time periods. They're used for:
- Scheduled maintenance windows
- Dedicated compute time for critical jobs
- Class instruction sessions
- Conference demonstrations
- Guaranteed resource availability

## Table Columns

| Column | Description |
|--------|-------------|
| **Name** | Reservation identifier |
| **State** | Current status (color-coded) |
| **Start Time** | When reservation begins |
| **End Time** | When reservation ends |
| **Duration** | Total reservation length |
| **Nodes** | Number of reserved nodes |
| **Cores** | Number of reserved cores |
| **Users** | Authorized users (truncated list) |
| **Accounts** | Authorized accounts (truncated list) |

## Reservation States

States are color-coded based on time and status:

| State | Color | Description |
|-------|-------|-------------|
| **ACTIVE** | Green | Currently active, resources available |
| **INACTIVE** | Gray | Past reservation, ended |
| Future | Yellow | Scheduled, not yet started |

### State Indicators

**ACTIVE** (Green):
- Reservation window is currently open
- Authorized users can submit jobs
- Resources are reserved
- Shows time remaining

**Future** (Yellow):
- Reservation scheduled but not started
- Shows time until start
- Resources not yet reserved

**INACTIVE** (Gray):
- Reservation has ended
- Historical record
- Resources released

## Time Display

### Start and End Times
- Displayed in local time
- Format: `YYYY-MM-DD HH:MM:SS`
- Example: `2024-01-15 14:00:00`

### Duration
- Total reservation length
- Format: `Dd HH:MM:SS`
- Examples:
  - `04:00:00` - 4 hours
  - `1d 00:00:00` - 1 day
  - `7d 12:00:00` - 7.5 days

### Time-to-End (Active Reservations)
For active reservations, shows countdown:
```
State: ACTIVE (ends in 2h 35m)
```

## Resource Allocation

### Nodes
Number of nodes reserved:
- Specific nodes by name, or
- Any nodes matching criteria
- Shows count in table

### Cores
Total CPU cores reserved:
- Sum across all nodes
- Dedicated or shared access
- Shows in Cores column

## Access Control

### Authorized Users
List of users allowed to use the reservation:
- Comma-separated in table
- Full list in details view
- Can be user=ALL for open access

### Authorized Accounts
List of accounts allowed to use the reservation:
- Jobs must specify reservation name
- Account must be in authorized list
- Billing still applies to account

## Reservation Actions

### View Reservation Details
**Shortcut**: `Enter`

Shows comprehensive reservation information:

**Basic Information:**
- Reservation name
- State and time status
- Creator and creation time

**Time Window:**
- Start time
- End time
- Duration
- Time remaining (if active)
- Time until start (if future)

**Resources:**
- Node list (specific nodes)
- Node count
- Core count
- Features required
- Partition assignment

**Access Control:**
- Full list of authorized users
- Full list of authorized accounts
- Access restrictions

**Flags:**
- IGNORE_JOBS - Override running jobs
- DAILY - Repeating daily reservation
- WEEKLY - Repeating weekly reservation
- REPLACE - Replace existing allocation
- STATIC_ALLOC - Fixed node allocation

**Current Usage** (for active):
- Jobs running in reservation
- Resources in use vs. reserved
- Utilization percentage

### Filter Reservations

#### Simple Filter
**Shortcut**: `/`

Filter by:
- Reservation name
- State
- Authorized users
- Authorized accounts

#### Advanced Filter
**Shortcut**: `F3`

Expression-based filtering:

```
state:ACTIVE
nodes:>16
users:alice
accounts:ml-team
```

**Supported fields:**
- `name` - Reservation name
- `state` - State (ACTIVE/INACTIVE/future)
- `nodes` - Node count (supports >, <, >=, <=)
- `cores` - Core count (supports comparison)
- `users` - Username in authorized list
- `accounts` - Account in authorized list
- `start` - Start time (supports comparison)
- `end` - End time (supports comparison)

### State-Based Filters (Planned)

**Active Only Filter**
**Shortcut**: `a/A` (TODO: not yet implemented)

Show only currently active reservations.

**Future Only Filter**
**Shortcut**: `f/F` (TODO: not yet implemented)

Show only upcoming reservations.

### Global Search
**Shortcut**: `Ctrl+F`

Search across all cluster resources.

## Sorting

Sort reservations by clicking column headers or using number keys.

**Useful sorting:**
- By start time (chronological)
- By end time (find expiring soon)
- By nodes (largest reservations)
- By state (group active/future/past)

Press `1-9` to sort by column number.

## Keyboard Shortcuts Reference

### Reservation Operations
| Key | Action |
|-----|--------|
| `Enter` | View reservation details |

### Filtering
| Key | Action |
|-----|--------|
| `/` | Simple filter |
| `F3` | Advanced filter |
| `Ctrl+F` | Global search |
| `a/A` | Toggle active-only (TODO) |
| `f/F` | Toggle future-only (TODO) |
| `ESC` | Exit filter mode |

### Data Management
| Key | Action |
|-----|--------|
| `R` | Manual refresh |
| `1-9` | Sort by column |

## Reservation Details Example

When viewing reservation details (`Enter`):

```
Reservation: class-demo
State: ACTIVE
Creator: admin
Created: 2024-01-10 10:00:00

Time Window:
  Start: 2024-01-15 14:00:00
  End:   2024-01-15 18:00:00
  Duration: 04:00:00
  Time Remaining: 2h 35m

Resources:
  Nodes: gpu[001-008] (8 nodes)
  Cores: 384 (48 per node)
  Partition: gpu
  Features: nvidia_a100, nvlink

Access Control:
  Users: alice, bob, charlie, david
  Accounts: teaching, research

Flags:
  - DAILY
  - REPLACE

Current Usage:
  Jobs Running: 3
  Cores Used: 192/384 (50%)
  Nodes Used: 4/8 (50%)

Running Jobs:
  12345 (alice) - 96 cores, 2 nodes
  12346 (bob)   - 48 cores, 1 node
  12347 (charlie) - 48 cores, 1 node
```

## Using Reservations

### Submitting Jobs to Reservations

Jobs must explicitly request the reservation:

```bash
sbatch --reservation=class-demo job_script.sh
```

Or in job script:
```bash
#SBATCH --reservation=class-demo
```

### Requirements
1. User must be in authorized users list, or
2. Job account must be in authorized accounts list
3. Reservation must be active (within time window)
4. Resources requested must fit in reservation

### Priority
Jobs in reservations:
- Get priority access to reserved resources
- May bypass normal queue
- Confined to reservation resources
- Cannot use non-reserved resources (usually)

## Common Reservation Patterns

### Maintenance Window
```
Name: maintenance-weekly
State: Future
Start: Every Sunday 02:00:00
Duration: 04:00:00
Nodes: ALL
Users: root, admin
Accounts: operations
Flags: WEEKLY, IGNORE_JOBS
```

### Class Instruction
```
Name: cs101-lab
State: ACTIVE
Start: Mon/Wed/Fri 14:00:00
Duration: 02:00:00
Nodes: cpu[001-016]
Users: student1, student2, ..., instructor
Accounts: teaching
Flags: DAILY, STATIC_ALLOC
```

### Dedicated Research Time
```
Name: grant-deadline
State: ACTIVE
Start: 2024-01-20 00:00:00
Duration: 3d 00:00:00
Nodes: gpu[001-032]
Users: alice, bob, charlie
Accounts: research, ml-team
Flags: REPLACE
```

### Conference Demo
```
Name: sc24-demo
State: Future
Start: 2024-11-18 08:00:00
Duration: 8h 00:00:00
Nodes: gpu[001-004]
Users: presenter, backup
Accounts: marketing
Flags: STATIC_ALLOC
```

## Reservation Flags

**IGNORE_JOBS:**
- Can preempt running jobs to start reservation
- Use for: Critical maintenance windows

**DAILY:**
- Repeats every day at same time
- Use for: Regular class sessions, maintenance

**WEEKLY:**
- Repeats every week on same day/time
- Use for: Weekly maintenance, meetings

**REPLACE:**
- Replace existing resource allocations
- Can take resources from running jobs if needed
- Use for: Emergency reservations

**STATIC_ALLOC:**
- Fixed node allocation (specific nodes)
- Nodes don't change over reservation lifetime
- Use for: Predictable resource access

**FLEX:**
- Flexible node assignment within partition
- Can use any available nodes
- Use for: General-purpose reservations

## Tips

- **Plan ahead**: Create reservations well before needed time
- **Check availability**: Ensure nodes are available during desired window
- **Use specific nodes**: STATIC_ALLOC prevents resource shuffling
- **Communicate**: Notify users when reservations will limit general access
- **Monitor usage**: Check utilization to justify resource dedication
- **Clean up**: Remove or update inactive reservations
- **Access control**: Be specific with user/account lists to prevent unauthorized use
- **Time buffers**: Add buffer time for setup/cleanup
- **Repeating reservations**: Use DAILY/WEEKLY for regular needs
- **Test before critical use**: Test reservation access before important deadlines

## Common Issues

### "Reservation not usable"
- User not in authorized users list
- Job account not in authorized accounts list
- Check reservation access control
- Contact admin to add user/account

### "Reservation is not active"
- Trying to use before start time or after end time
- Check current time vs. reservation window
- Wait for start time or request extension

### "Insufficient resources in reservation"
- Job requests more resources than reserved
- Reduce job resource request
- Request larger reservation

### "Reservation conflict"
- Time window overlaps with existing reservation on same nodes
- Adjust start/end times
- Use different nodes
- Cancel conflicting reservation

## Best Practices

**For Admins:**
- Document reservation policies
- Require justification for large reservations
- Monitor reservation utilization
- Remove unused or expired reservations
- Use DAILY/WEEKLY for recurring needs
- Set reasonable duration limits

**For Users:**
- Only request resources actually needed
- Use reservations efficiently (high utilization)
- Return unused time if possible
- Coordinate with team on shared reservations
- Test jobs before reservation window
- Monitor time remaining during active reservation

**For Planning:**
- Create reservation days in advance
- Add buffer time for setup
- Include authorized backup users
- Document reservation purpose
- Set calendar reminders for start time
- Prepare jobs ahead of window
