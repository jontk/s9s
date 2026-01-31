# Users View

The Users view displays user account information, administrative levels, resource limits, and account associations.

![Users Demo](/assets/demos/users.gif)

*Users view showing user accounts, admin levels, and resource limits*

## Overview

The Users view provides a comprehensive list of all SLURM users with their permissions, resource limits, and account memberships. Use this view to manage user access and understand resource allocation policies.

## Table Columns

| Column | Description |
|--------|-------------|
| **User** | Username |
| **UID** | User ID number |
| **Default Account** | Primary billing account |
| **Admin Level** | User privilege level (color-coded) |
| **Default QoS** | Default Quality of Service |
| **Max Jobs** | Maximum concurrent jobs allowed |
| **Max Nodes** | Maximum nodes per user |
| **Max CPUs** | Maximum CPUs per user |
| **Accounts** | List of associated accounts |

## Admin Levels

Admin levels are color-coded for quick identification:

| Level | Color | Description | Permissions |
|-------|-------|-------------|-------------|
| **Administrator** | Red | Full admin | Cluster configuration, user management, all operations |
| **Operator** | Yellow | Limited admin | Job management, node operations, monitoring |
| **Regular** | White | Standard user | Submit jobs, view own jobs, basic operations |

### Admin Level Capabilities

**Administrator:**
- Modify cluster configuration
- Create/delete users and accounts
- Drain/resume nodes
- Cancel any user's jobs
- Modify QoS policies
- View all cluster data

**Operator:**
- Monitor cluster status
- Drain/resume nodes
- View all jobs
- Limited job control
- No configuration changes

**Regular:**
- Submit jobs
- View own jobs
- Cancel own jobs
- View cluster status

## Resource Limits

### Max Jobs
Maximum number of concurrent jobs a user can have in the system (running + pending).

- **Unlimited**: Shown as "∞" or "unlimited"
- **Limited**: Specific number (e.g., "100")

When limit is reached, new job submissions are rejected until running jobs complete.

### Max Nodes
Maximum number of nodes a single job can request.

- Prevents users from monopolizing large partition resources
- Can be set per-partition
- "Unlimited" means subject to partition limits only

### Max CPUs
Maximum number of CPUs across all concurrent jobs.

- Total CPU allocation limit
- Affects both running and pending jobs
- Helps ensure fair resource sharing

**Example:**
```
User: alice
Max CPUs: 1000
Current usage: 750 CPUs (3 jobs)
Available: 250 CPUs for new jobs
```

## User Actions

### View User Details
**Shortcut**: `Enter`

Shows comprehensive user information:

**Account Information:**
- Username and UID
- Default account
- Associated accounts with priorities
- Administrative level

**QoS Settings:**
- Default QoS
- Allowed QoS levels
- QoS priorities

**Resource Limits:**
- Maximum jobs (running + pending)
- Maximum nodes per job
- Maximum CPUs total
- Maximum wall time
- Maximum memory

**Current Usage:**
- Active jobs count
- Current CPU usage
- Current node usage
- Current memory usage

**Account Associations:**
- List of accounts user can submit to
- Default account designation
- Priority levels per account

### Filter Users

#### Simple Filter
**Shortcut**: `/`

Filter by:
- Username
- Default account
- Admin level
- QoS
- Any displayed field

#### Advanced Filter
**Shortcut**: `F3`

Expression-based filtering:

```
admin:Administrator
account:research
qos:high
maxjobs:>50
```

**Supported fields:**
- `user` - Username
- `uid` - User ID
- `account` - Default account name
- `admin` - Admin level (Administrator/Operator/Regular)
- `qos` - Default QoS
- `maxjobs` - Max jobs limit (supports >, <, >=, <=)
- `maxnodes` - Max nodes limit (supports comparison)
- `maxcpus` - Max CPUs limit (supports comparison)

### Admin Filter Toggle
**Shortcut**: `a/A`

Toggles between:
- **All users**: Shows everyone (default)
- **Admins only**: Shows only Administrators and Operators

Useful for quickly finding users with elevated privileges.

### Global Search
**Shortcut**: `Ctrl+F`

Search across all cluster resources.

## Sorting

Sort users by clicking column headers or using number keys.

**Useful sorting:**
- By admin level (find administrators)
- By default account (group by research team)
- By max CPUs (identify power users)
- By username (alphabetical)

Press `1-9` to sort by column number.

## Keyboard Shortcuts Reference

### User Operations
| Key | Action |
|-----|--------|
| `Enter` | View user details |

### Filtering
| Key | Action |
|-----|--------|
| `/` | Simple filter |
| `F3` | Advanced filter |
| `Ctrl+F` | Global search |
| `a/A` | Toggle admin-only filter |
| `ESC` | Exit filter mode |

### Data Management
| Key | Action |
|-----|--------|
| `R` | Manual refresh |
| `1-9` | Sort by column |

## User Details Example

When viewing user details (`Enter`):

```
User: alice
UID: 1234
Admin Level: Operator

Default Account: ml-team
Default QoS: normal

Resource Limits:
  Max Jobs: 50
  Max Nodes: 16 per job
  Max CPUs: 1000 total
  Max Wall Time: 7-00:00:00 (7 days)
  Max Memory: 2TB

Current Usage:
  Active Jobs: 8
  CPUs Used: 384/1000 (38%)
  Nodes Used: 12/16 (75%)

Associated Accounts:
  ml-team (Default, Priority: 100)
  research (Priority: 50)
  shared (Priority: 10)

Allowed QoS Levels:
  normal (Default)
  high
  low
```

## Understanding Resource Limits

### Per-User vs. Per-Job Limits

**Per-User Limits** (enforced across all jobs):
- Max Jobs - total concurrent jobs
- Max CPUs - total CPU allocation
- Max Nodes - cumulative node usage

**Per-Job Limits** (enforced per individual job):
- Max Nodes per Job - single job cannot exceed
- Max CPUs per Job - single job cannot exceed
- Max Memory per Job - single job cannot exceed

**Example scenario:**
```
User settings:
  Max Jobs: 10
  Max CPUs: 1000
  Max Nodes per Job: 50

Valid: 10 jobs × 100 CPUs each = 1000 CPUs total ✓
Invalid: 5 jobs × 250 CPUs each = 1250 CPUs total ✗
Invalid: 1 job × 60 nodes = exceeds per-job limit ✗
```

### Unlimited Settings

When a limit shows "∞" or "unlimited":
- No user-specific limit set
- Still subject to partition limits
- Still subject to QoS limits
- Still subject to cluster-wide policies

## Account Associations

Users can belong to multiple accounts:

**Default Account:**
- Used when no account is specified in job submission
- Shown in Default Account column
- Can be changed by user or admin

**Secondary Accounts:**
- User can submit jobs to any associated account
- Must specify with `--account=name` flag
- Useful for billing different projects separately

**Priority Levels:**
- Each association has a priority
- Higher priority = better scheduling
- Affects fair-share calculation

## Admin-Only View

Press `a/A` to toggle admin-only filtering:

**Shows:**
- Administrators (red)
- Operators (yellow)

**Hides:**
- Regular users

**Use cases:**
- Find who can help with admin tasks
- Audit admin access
- Contact operators for assistance

## Tips

- **Check resource limits**: Before requesting help with "job won't submit" issues
- **Review account associations**: Users might not realize they have access to other accounts
- **Monitor admin users**: Regularly review who has elevated access
- **Compare QoS**: Different QoS levels provide different priorities and limits
- **Filter by account**: Use `F3` with `account:name` to see all users in a research group
- **Sort by usage limits**: Identify power users with high CPU/node limits
- **Default account matters**: Billing and priority often depend on the default account
- **Unlimited ≠ infinite**: Still subject to partition and cluster limits

## Fair-Share and Priority

User priority in the scheduler depends on:

1. **Account Association Priority**: Set per user-account pair
2. **Fair-Share Score**: Based on historical usage vs. allocation
3. **QoS Priority**: Different QoS levels have different priorities
4. **Job Age**: Older pending jobs get priority boost

View the Jobs view to see how priority affects actual job scheduling.

## Common Issues

### "Job submission failed: User limit exceeded"
- Check Max Jobs limit
- Wait for running jobs to complete
- Contact admin to request limit increase

### "Job submission failed: CPU limit exceeded"
- Check Max CPUs limit
- Current jobs are using up allocation
- Reduce CPUs requested or wait for jobs to finish

### "Cannot submit to account X"
- User not associated with account X
- Check Associated Accounts list
- Contact admin to request account access

### "QoS not allowed"
- User trying to use QoS not in Allowed QoS list
- Specify a different QoS or use default
- Contact admin to request QoS access
