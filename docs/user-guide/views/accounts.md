# Accounts View

The Accounts view displays the account hierarchy, resource limits, and organizational structure for billing and access control.

![Accounts Demo](/assets/demos/accounts.gif)

*Accounts view showing account hierarchy, limits, and associations*

## Overview

The Accounts view shows all SLURM accounts used for billing, resource allocation, and access control. Accounts are organized in a hierarchical tree structure with parent-child relationships.

## Table Columns

| Column | Description |
|--------|-------------|
| **Account** | Account name |
| **Description** | Account purpose or description |
| **Organization** | Organizational unit |
| **Parent** | Parent account in hierarchy |
| **Default QoS** | Default Quality of Service |
| **Max Jobs** | Maximum concurrent jobs |
| **Max Nodes** | Maximum nodes per account |
| **Max CPUs** | Maximum CPUs per account |
| **Max Wall Time** | Maximum job duration |
| **Coordinators** | Account administrators |

## Account Hierarchy

Accounts are organized in parent-child relationships:

```
root
├── research
│   ├── ml-team
│   │   ├── deep-learning
│   │   └── nlp
│   └── bio-team
│       ├── genomics
│       └── proteomics
└── operations
    ├── production
    └── testing
```

### Root Accounts
- Top-level accounts with no parent
- Color-coded differently
- Often represent departments or major divisions

### Child Accounts
- Inherit limits from parent (unless overridden)
- Can have more restrictive limits than parent
- Cannot have more permissive limits than parent
- Useful for sub-projects or teams

## Account Actions

### View Account Details
**Shortcut**: `Enter`

Shows comprehensive account information:

**Basic Information:**
- Account name and description
- Organization affiliation
- Parent account
- Coordinators (account administrators)

**QoS Settings:**
- Default QoS for jobs
- Allowed QoS levels
- QoS priorities

**Resource Limits:**
- Maximum jobs (concurrent)
- Maximum nodes (total across all jobs)
- Maximum CPUs (total across all jobs)
- Maximum wall time (per job)
- Maximum memory (total)

**Usage Information:**
- Current active jobs
- Current CPU usage
- Current node usage
- Usage vs. limits

**Child Accounts:**
- List of sub-accounts
- Child account resource consumption

**Associated Users:**
- Users with access to this account
- User priorities for this account

### View Account Hierarchy
**Shortcut**: `H`

Opens hierarchical tree view showing:
- Full account structure
- Parent-child relationships
- Indentation showing depth
- Unicode tree connectors

**Example hierarchy view:**
```
root
├─ research
│  ├─ ml-team
│  │  ├─ deep-learning
│  │  └─ nlp
│  └─ bio-team
│     ├─ genomics
│     └─ proteomics
└─ operations
   ├─ production
   └─ testing
```

**Tree connectors:**
- `├─` Branch with siblings
- `└─` Last branch
- `│` Vertical connection
- `─` Horizontal connection

### Filtering

#### Simple Filter
**Shortcut**: `/`

Filter accounts by:
- Account name
- Description
- Organization
- Parent account
- Coordinator name

#### Advanced Filter
**Shortcut**: `F3`

Expression-based filtering:

```
organization:research
parent:root
qos:high
maxcpus:>1000
```

**Supported fields:**
- `account` - Account name
- `description` - Description text
- `organization` - Organization name
- `parent` - Parent account name
- `qos` - Default QoS
- `maxjobs` - Max jobs (supports >, <, >=, <=)
- `maxnodes` - Max nodes (supports comparison)
- `maxcpus` - Max CPUs (supports comparison)
- `coordinator` - Coordinator name

### Global Search
**Shortcut**: `Ctrl+F`

Search across all cluster resources.

## Sorting

Sort accounts by clicking column headers or using number keys.

**Useful sorting:**
- By organization (group by department)
- By max CPUs (identify large allocations)
- By account name (alphabetical)
- By parent (group by hierarchy level)

Press `1-9` to sort by column number.

## Keyboard Shortcuts Reference

### Account Operations
| Key | Action |
|-----|--------|
| `Enter` | View account details |
| `H` | Show hierarchy tree |

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

## Account Details Example

When viewing account details (`Enter`):

```
Account: ml-team
Description: Machine Learning Research Team
Organization: Computer Science Dept
Parent: research

Coordinators:
  - alice (PI)
  - bob (Lab Manager)

Default QoS: normal
Allowed QoS: normal, high, low

Resource Limits:
  Max Jobs: 100
  Max Nodes: 64
  Max CPUs: 3072
  Max Wall Time: 7-00:00:00 (7 days)
  Max Memory: 8TB

Current Usage:
  Active Jobs: 23/100 (23%)
  Nodes Used: 18/64 (28%)
  CPUs Used: 864/3072 (28%)

Child Accounts:
  - deep-learning (CPU: 512/864)
  - nlp (CPU: 352/864)

Associated Users: 15
  alice (Priority: 100, Coordinator)
  bob (Priority: 100, Coordinator)
  charlie (Priority: 50)
  ...
```

## Understanding Account Limits

### Cumulative Limits
Account limits apply to **all users combined**:

```
Account: ml-team
Max CPUs: 3072

Current usage:
  alice: 1000 CPUs (5 jobs)
  bob: 800 CPUs (3 jobs)
  charlie: 600 CPUs (2 jobs)
  Total: 2400/3072 CPUs

Available: 672 CPUs for any user in ml-team
```

### Hierarchical Limits

Child accounts consume parent account resources:

```
Parent: research (Max CPUs: 5000)
├─ ml-team (Max CPUs: 3072)
│  Currently using: 2400 CPUs
└─ bio-team (Max CPUs: 2000)
   Currently using: 1800 CPUs

Total research usage: 4200/5000 CPUs
Available: 800 CPUs (shared across all children)
```

**Rules:**
1. Child cannot have limit > parent limit
2. Child usage counts against parent
3. Parent limit can be reached even if child limits have room
4. More restrictive limit (parent or child) applies

### Wall Time Limits

Maximum wall time per job:
- 1-00:00:00 = 1 day
- 7-00:00:00 = 7 days (1 week)
- 30-00:00:00 = 30 days (1 month)

Jobs requesting longer duration will be rejected.

**Format**: `days-hours:minutes:seconds`

## Coordinators

Account coordinators are administrators for the account:

**Responsibilities:**
- Approve user access requests
- Monitor account resource usage
- Adjust project priorities
- Troubleshoot user issues
- Coordinate with cluster admins

**Permissions:**
- View all account jobs
- Manage account users (if admin)
- Set user priorities within account
- Request limit increases

## Account Hierarchy Benefits

### Organization
- Mirrors organizational structure
- Clear billing attribution
- Logical resource grouping

### Resource Management
- Allocate resources by department
- Sub-allocate to teams/projects
- Prevent single team monopolizing resources

### Access Control
- Grant/revoke access at any level
- Inherit permissions from parent
- Delegate admin to coordinators

**Example:**
```
research (Max CPUs: 5000)
├─ ml-team (Max CPUs: 3000)
│  └─ project-alpha (Max CPUs: 1000)
│     - Focused allocation for specific project
│     - Cannot exceed 1000 CPUs
│     - Counts against ml-team's 3000 limit
│     - Counts against research's 5000 limit
```

## Common Patterns

### Department-Level Structure
```
root
├── chemistry
├── physics
├── biology
└── engineering
```

### Project-Based Structure
```
root
├── research
│   ├── grant-NIH-12345
│   ├── grant-NSF-67890
│   └── grant-DOE-11111
└── instruction
    ├── course-cs101
    └── course-cs201
```

### Hybrid Structure
```
root
├── research
│   ├── chemistry
│   │   ├── project-a
│   │   └── project-b
│   └── physics
│       ├── project-c
│       └── project-d
└── operations
```

## Tips

- **Check hierarchy**: Use `H` to understand account relationships
- **Parent limits matter**: Child account limits don't help if parent is maxed out
- **Monitor usage**: Check account details to see who's consuming resources
- **Coordinate access**: Contact coordinators to request account access
- **Billing attribution**: Jobs bill to the specified account
- **Default account**: Users have a default, but can submit to any associated account
- **Filter by organization**: Use `F3` with `organization:name` to group by dept
- **Wall time inheritance**: Child accounts often inherit parent wall time limits
- **QoS interaction**: Account limits stack with QoS limits (most restrictive wins)

## Common Issues

### "Account limit exceeded"
- Account has reached max jobs/CPUs/nodes
- Check account usage vs. limits
- Wait for jobs to complete
- Contact coordinator to prioritize
- Request limit increase from admin

### "Cannot charge to account X"
- User not associated with account
- Check user's account associations
- Contact account coordinator

### "Parent account limit exceeded"
- Child account has resources available but parent doesn't
- Need to wait for other siblings' jobs to complete
- Coordinate with other teams using parent account

### Time limit exceeded partition/QoS limit
- Job wall time exceeds account max
- Reduce requested time
- Contact coordinator to request extension
- Use different account with higher limit
