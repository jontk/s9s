# Job Management Guide

Master job management in S9S with this comprehensive guide covering submission, monitoring, and advanced job operations.

## Job Overview

S9S provides a powerful interface for managing SLURM jobs with features that go beyond traditional command-line tools:

- **Real-time Monitoring**: Live job status updates
- **Batch Operations**: Manage multiple jobs simultaneously
- **Advanced Filtering**: Find jobs quickly
- **Direct Output Access**: View logs without leaving S9S
- **Job Templates**: Reusable job configurations
- **Dependency Management**: Visual dependency tracking

## Submitting Jobs

### Quick Submit

Press `s` in Jobs view to open the submission wizard:

1. **Choose Method**:
   - New job from scratch
   - From template
   - Copy existing job
   - Import script file

2. **Configure Resources**:
   ```yaml
   Job Name: my_analysis
   Partition: compute
   Nodes: 2
   Tasks per Node: 28
   Memory: 64GB
   Time Limit: 24:00:00
   ```

3. **Set Script**:
   ```bash
   #!/bin/bash

   module load python/3.9
   python analyze.py
   ```

### Job Submission Workflow

![Job Submission Demo](/assets/demos/job-submission.gif)

*Job submission wizard with step-by-step configuration*

The submission process guides you through all necessary options with helpful defaults and validation.

### Submission Wizard Fields

The wizard supports 86 sbatch fields across the full SLURM OpenAPI spec. Fields are organized into three visibility tiers so the form stays manageable while still exposing every option when needed.

**Always visible** -- these 7 core fields appear on every new job form:

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| name | name | --job-name | Job name |
| script | script | (script body) | Batch script content |
| partition | partition | --partition | Target partition |
| timeLimit | time_limit | --time | Wall-clock time limit (HH:MM:SS or D-HH:MM:SS) |
| nodes | nodes | --nodes | Number of nodes |
| cpus | cpus | --cpus-per-task | CPUs per task |
| memory | memory | --mem | Memory per node (e.g., 4G, 1024M) |

**Visible by default** -- these 8 fields are shown on the form unless explicitly hidden:

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| gpus | gpus | --gres=gpu:N | Number of GPUs |
| qos | qos | --qos | Quality of service |
| account | account | --account | Charge account |
| workingDir | working_directory | --chdir | Working directory |
| outputFile | output_file | --output | Stdout file path |
| errorFile | error_file | --error | Stderr file path |
| emailNotify | email_notify | --mail-type | Enable email notifications |
| email | email | --mail-user | Notification email address |

**Hidden by default** -- these 71 fields become visible when a template sets a value for them, when they are removed from the `hiddenFields` list in config, or when a per-template `hiddenFields` override brings them into view.

*Job arrays & dependencies:*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| arraySpec | array | --array | Array job index spec (e.g., 1-100%10) |
| dependencies | dependencies | --dependency | Job dependencies (list of job IDs, submitted as afterok:id1:id2) |

*Resource controls:*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| exclusive | exclusive | --exclusive | Exclusive node access |
| requeue | requeue | --requeue | Requeue on failure |
| gres | gres | --gres | Generic resources (e.g., gpu:a100:2) |
| constraints | constraints | --constraint | Required node features |
| ntasks | ntasks | --ntasks | Total number of tasks |
| ntasksPerNode | ntasks_per_node | --ntasks-per-node | Tasks per node |
| memoryPerCPU | memory_per_cpu | --mem-per-cpu | Memory per CPU (e.g., 4G) |
| minimumCPUs | minimum_cpus | (API only) | Minimum total CPUs |
| minimumCPUsPerNode | minimum_cpus_per_node | --mincpus | Minimum CPUs per node |
| maximumNodes | maximum_nodes | --nodes (max) | Maximum node count |
| maximumCPUs | maximum_cpus | (API only) | Maximum CPU count |
| tmpDiskPerNode | tmp_disk_per_node | --tmp | Temporary disk per node (MB) |
| overcommit | overcommit | --overcommit | Overcommit resources |
| contiguous | contiguous | --contiguous | Require contiguous nodes |

*Scheduling:*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| hold | hold | --hold | Submit in held state |
| priority | priority | --priority | Job priority |
| nice | nice | --nice | Priority adjustment |
| beginTime | begin_time | --begin | Deferred start (see [Begin Time Formats](#begin-time-formats) below) |
| deadline | deadline | --deadline | Latest acceptable start time |
| immediate | immediate | --immediate | Fail if resources not available now |
| timeMinimum | time_minimum | --time-min | Minimum time for backfill scheduling |

*Placement & topology:*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| distribution | distribution | --distribution | Task distribution (block, cyclic, etc.) |
| threadsPerCore | threads_per_core | --threads-per-core | Threads per core |
| tasksPerCore | tasks_per_core | --ntasks-per-core | Tasks per core |
| tasksPerSocket | tasks_per_socket | --ntasks-per-socket | Tasks per socket |
| socketsPerNode | sockets_per_node | --sockets-per-node | Sockets per node |
| cpuBinding | cpu_binding | --cpu-bind | CPU binding method |
| cpuBindingFlags | cpu_binding_flags | --cpu-bind (flags) | CPU binding flags (VERBOSE, etc.) |
| memoryBinding | memory_binding | --mem-bind | Memory NUMA binding |
| memoryBindingType | memory_binding_type | --mem-bind (type) | Memory binding type (LOCAL, RANK) |
| requiredNodes | required_nodes | --nodelist | Required specific nodes |
| excludeNodes | exclude_nodes | --exclude | Excluded nodes |

*TRES (Trackable Resources):*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| cpusPerTRES | cpus_per_tres | --cpus-per-gpu | CPUs per GPU/TRES |
| memoryPerTRES | memory_per_tres | --mem-per-gpu | Memory per GPU/TRES |
| ntasksPerTRES | ntasks_per_tres | --ntasks-per-gpu | Tasks per GPU/TRES |
| tresPerTask | tres_per_task | --tres-per-task | TRES per task |
| tresPerSocket | tres_per_socket | --tres-per-socket | TRES per socket |
| tresPerJob | tres_per_job | --tres-per-job | TRES per job |
| tresBind | tres_bind | --tres-bind | TRES binding (e.g., gres/gpu:closest) |
| tresFreq | tres_freq | --tres-freq | TRES frequency control |

*Signals & notifications:*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| signal | signal | --signal | Pre-termination signal (e.g., B:USR1@300) |
| killOnNodeFail | kill_on_node_fail | --no-kill | Kill job on node failure |
| waitAllNodes | wait_all_nodes | --wait-all-nodes | Wait for all nodes to boot |

*Accounting & admin:*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| reservation | reservation | --reservation | Reservation name |
| licenses | licenses | --licenses | Required licenses |
| wckey | wckey | --wckey | Workload characterization key |
| comment | comment | --comment | Job comment |
| prefer | prefer | --prefer | Preferred (not required) features |

*I/O & environment:*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| standardInput | standard_input | --input | Stdin file path |
| openMode | open_mode | --open-mode | Output file mode (append or truncate) |
| container | container | --container | OCI container path |

*Advanced:*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| cpuFrequency | cpu_frequency | --cpu-freq | CPU frequency (low, medium, high, or KHz) |
| network | network | --network | Network specs |
| x11 | x11 | --x11 | X11 forwarding (batch, first, last, all) |
| burstBuffer | burst_buffer | --bb | Burst buffer specification |
| batchFeatures | batch_features | --batch | Batch node features |
| coreSpecification | core_specification | --core-spec | Reserved system cores |
| threadSpecification | thread_specification | --thread-spec | Reserved system threads |
| argv | argv | (script arguments) | Script arguments (space-separated) |
| flags | flags | --spread-job, etc. | Job flags (comma-separated) |
| profile | profile | --profile | Profiling (ENERGY, NETWORK, TASK) |

*Cluster federation:*

| Config Key | JSON Key | sbatch Flag | Description |
|---|---|---|---|
| requiredSwitches | required_switches | --switches | Required network switch count |
| waitForSwitch | wait_for_switch | --switches (timeout) | Switch wait timeout (seconds) |
| clusterConstraint | cluster_constraint | --cluster-constraint | Federation cluster constraint |
| clusters | clusters | --clusters | Target clusters (federation) |

### Begin Time Formats

The `beginTime` field accepts all formats supported by sbatch `--begin`:

| Format | Example | Description |
|---|---|---|
| Named time | `now`, `today`, `tomorrow` | Current time, midnight today, midnight tomorrow |
| Named hour | `midnight`, `noon`, `teatime` | 00:00, 12:00, 16:00 (next occurrence) |
| Relative | `now+1hour`, `now+30minutes` | Offset from current time |
| Relative (seconds) | `now+3600` | Bare number = seconds |
| ISO date | `2024-12-31` | Midnight on date |
| ISO datetime | `2024-12-31T14:30` | Specific date and time |
| US date | `12/31/24`, `123124` | US date format |
| Time of day | `16:00`, `4:00PM` | Next occurrence of time |

Named times: `midnight` (00:00), `noon` (12:00), `elevenses` (11:00), `fika` (15:00), `teatime` (16:00).

### Template-Based Submission

Select a template from the wizard's template selector to pre-fill the form. Templates can set any field value and control which fields are visible.

```bash
# List all templates from every source
s9s templates list

# Submit from template in the wizard
# Press s -> choose "From template" -> pick a template
```

### Config-Driven Customization

All submission defaults and field visibility can be configured under `views.jobs.submission` in your config file (`~/.s9s/config.yaml`):

```yaml
views:
  jobs:
    submission:
      # Global defaults applied to every new job
      formDefaults:
        partition: "compute"
        timeLimit: "04:00:00"
        nodes: 1
        cpus: 4
        memory: "8G"
        workingDir: "/scratch/%u"  # %u = username (SLURM substitution)
        outputFile: "slurm_%j.out"
        errorFile: "slurm_%j.err"

      # Fields to hide globally from the form
      hiddenFields:
        - arraySpec
        - exclusive
        - requeue

      # Restrict dropdown values (filters cluster-fetched values)
      fieldOptions:
        partition: ["compute", "gpu", "highmem"]
        qos: ["normal", "high"]
        account: ["research-a", "research-b"]

      # Control which template sources are loaded (default: all three)
      # Options: "builtin", "config", "saved"
      templateSources: ["builtin", "config", "saved"]

      # Define custom config templates (see Job Templates section below)
      templates:
        - name: "GPU Training Job"
          description: "PyTorch training on GPU partition"
          defaults:
            partition: "gpu"
            timeLimit: "24:00:00"
            cpus: 8
            memory: "32G"
            gpus: 2
            script: |
              #!/bin/bash
              module load cuda pytorch
              python train.py
          hiddenFields: ["arraySpec"]
```

### Command Line Submit

```bash
# Submit with S9S command mode
:submit --partition=gpu --nodes=4 --time=2:00:00 myscript.sh

# Submit with dependencies
:submit --dependency=afterok:12345 --array=1-100 array_job.sh
```

## Monitoring Jobs

### Job States

S9S color-codes job states for quick identification:

| State | Color | Description |
|-------|-------|-------------|
| PENDING | Yellow | Waiting for resources |
| RUNNING | Green | Currently executing |
| COMPLETED | Blue | Finished successfully |
| FAILED | Red | Exited with error |
| CANCELLED | Gray | Cancelled by user/admin |
| TIMEOUT | Orange | Exceeded time limit |
| SUSPENDED | Purple | Temporarily suspended |

### Job Details

Press `Enter` or `d` on any job to view details:

- **Summary**: ID, name, user, submission time
- **Resources**: Nodes, CPUs, memory, GPUs
- **Timing**: Start, elapsed, remaining time
- **Performance**: CPU/memory efficiency
- **Output**: Stdout/stderr file paths
- **Dependencies**: Parent/child jobs

### Live Output Monitoring

View job output in real-time:

1. Select job and press `o`
2. Choose output type:
   - Standard output
   - Error output
   - Both (split view)
3. Options:
   - `f` - Follow/tail output
   - `/` - Search in output
   - `s` - Save to file
   - `Esc` - Exit viewer

For more details, see the [Jobs View Guide](./views/jobs.md).

## Job Operations

### Single Job Actions

| Key | Action | Description |
|-----|--------|-------------|
| `c` | Cancel | Cancel job (with confirmation) |
| `C` | Force Cancel | Cancel without confirmation |
| `h` | Hold | Prevent job from starting |
| `r` | Release | Release held job |
| `R` | Requeue | Resubmit failed job |
| `p` | Priority | Modify job priority |
| `e` | Edit | Modify pending job |
| `m` | Move | Move to different partition |

### Batch Operations

Select multiple jobs with `Space`, then press `b`:

1. **Selection Methods**:
   - Manual: `Space` on each job
   - All visible: `V`
   - By filter: `/state:PENDING` then `V`
   - By pattern: `:select pattern "analysis_*"`

2. **Batch Actions**:
   - Cancel selected
   - Hold/Release selected
   - Change priority
   - Move partition
   - Add dependency
   - Export data

### Advanced Operations

#### Job Arrays

Manage array jobs efficiently:

```bash
# View array summary
:array-summary 12345

# Expand all array tasks
:expand-array 12345

# Cancel specific tasks
:cancel 12345_[1-10,20,30-40]

# Hold array subset
:hold 12345_[50-100]
```

#### Dependencies

Visualize and manage job dependencies:

```bash
# View dependency tree
:deps tree 12345

# Add dependency
:deps add 12346 --after 12345

# Remove dependency
:deps remove 12346

# View dependency graph
:deps graph --format=dot | dot -Tpng > deps.png
```

## Advanced Filtering

### Filter Syntax

S9S supports powerful job filtering:

```bash
# Basic filters
/RUNNING                    # Running jobs
/gpu                       # Jobs with 'gpu' in name
/user:alice               # Alice's jobs

# State filters
/state:PENDING            # Pending jobs
/state:RUNNING,COMPLETED  # Multiple states
/state:!FAILED           # Not failed

# Resource filters
/nodes:>4                # More than 4 nodes
/memory:>=32GB          # 32GB or more memory
/gpus:>0                # GPU jobs

# Time filters
/runtime:>1h            # Running over 1 hour
/runtime:30m-2h        # Between 30min and 2h
/submitted:<1d         # Submitted within 1 day
/started:today         # Started today

# Complex filters
/user:bob state:RUNNING partition:gpu    # Bob's GPU jobs
/name:~"analysis.*2023" nodes:>10       # Regex + resource
```

### Saved Filters

Save frequently used filters:

```bash
# Save current filter
:filter save gpu-queue "/partition:gpu state:PENDING"

# Load saved filter
:filter load gpu-queue

# List saved filters
:filter list

# Delete filter
:filter delete old-filter
```

## Job Performance

### Efficiency Metrics

S9S calculates job efficiency:

- **CPU Efficiency**: Actual vs allocated CPU usage
- **Memory Efficiency**: Peak vs allocated memory
- **GPU Utilization**: GPU usage percentage
- **I/O Performance**: Read/write statistics

View metrics:
1. Select job and press `i`
2. Navigate to "Performance" tab
3. View graphs and statistics

### Performance Alerts

Set up alerts for inefficient jobs:

```yaml
# In config.yaml
alerts:
  lowEfficiency:
    threshold: 0.5
    metric: cpu
    action: notify

  highMemory:
    threshold: 90%
    metric: memory
    action: email
```

## Job Templates

### Template System Overview

S9S uses a three-tier merge system to assemble the list of available templates. When two or more sources define a template with the same name, the higher-priority source wins.

| Priority | Source | Location |
|----------|--------|----------|
| 1 (highest) | User-saved templates | `~/.s9s/templates/*.json` |
| 2 | Config YAML templates | `views.jobs.submission.templates` in config |
| 3 (lowest) | Built-in templates | Hardcoded in S9S |

### Built-in Templates

S9S ships with 8 built-in templates covering common job patterns:

| Template | Description |
|----------|-------------|
| Basic Batch Job | Simple single-node batch job |
| MPI Parallel Job | Parallel job using MPI across multiple nodes |
| GPU Job | Job requiring GPU resources |
| Array Job | Array job for processing multiple similar tasks |
| Interactive Job | Interactive session for development and testing |
| Long-Running Job | Extended wall-time job |
| High Memory Job | Job requesting large memory allocation |
| Development/Debug Job | Short debug session with verbose output |

### Config YAML Templates

Define custom templates in your config file under `views.jobs.submission.templates`. Each template can set default values for any form field and optionally hide irrelevant fields:

```yaml
views:
  jobs:
    submission:
      templates:
        - name: "GPU Training Job"
          description: "PyTorch training on GPU partition"
          defaults:
            partition: "gpu"
            timeLimit: "24:00:00"
            cpus: 8
            memory: "32G"
            gpus: 2
            script: |
              #!/bin/bash
              module load cuda pytorch
              python train.py
          hiddenFields: ["arraySpec"]

        - name: "Genomics Pipeline"
          description: "High-memory genomics analysis"
          defaults:
            partition: "highmem"
            timeLimit: "48:00:00"
            memory: "256G"
            cpus: 32
          hiddenFields: ["gpus", "arraySpec"]
```

### User-Saved Templates

User-saved templates are stored as individual JSON files in `~/.s9s/templates/` and have the highest priority in the merge order.

#### Saving from the Wizard

After configuring a job in the submission wizard, use the "Save as Template" flow to save the current form state as a new template in `~/.s9s/templates/`.

#### Template JSON Format

Each saved template is a JSON file with the following structure:

```json
{
  "name": "My Custom Template",
  "description": "Description of this template",
  "job_submission": {
    "name": "my_job",
    "partition": "compute",
    "time_limit": "04:00:00",
    "nodes": 2,
    "cpus": 8,
    "memory": "16G",
    "working_directory": "/scratch/%u",
    "output_file": "job_%j.out",
    "error_file": "job_%j.err",
    "script": "#!/bin/bash\nmodule load python\npython run.py"
  }
}
```

> **Naming conventions:** Saved JSON templates use snake_case field names (matching Go struct JSON tags): `time_limit`, `output_file`, `error_file`, `working_directory`. Config YAML templates use camelCase: `timeLimit`, `outputFile`, `errorFile`, `workingDir`. Using the wrong convention will silently ignore the field. Script arguments (`argv`) are split on whitespace in both formats — quoted arguments with spaces are not supported.

### Job Script Preview

Before submitting, press **Preview** to see the complete sbatch script that will be generated from your form values. The preview shows all `#SBATCH` directives derived from the form fields followed by your script body.

| Key | Action |
|-----|--------|
| `ESC` | Close preview |
| `Ctrl+Y` | Copy script to clipboard (via OSC 52) |

The clipboard copy produces a clean plain-text script with no color formatting, ready to paste into a file or terminal. OSC 52 clipboard support works in most modern terminals (iTerm2, kitty, tmux, Windows Terminal, Alacritty).

### CLI Commands

Manage templates from the command line:

```bash
# List all templates from all sources with source indicator (builtin/config/saved)
s9s templates list

# Export all built-in and config templates to ~/.s9s/templates/ as editable JSON
s9s templates export

# Export a single template by name
s9s templates export "GPU Job"

# Overwrite existing files during export
s9s templates export --force

# Export to a custom directory
s9s templates export --dir /path/to/templates
```

### Template Workflow

A typical workflow for customizing templates:

1. **Export the built-ins** to get editable copies:
   ```bash
   s9s templates export
   ```
   This writes all 8 built-in templates (plus any config templates) to `~/.s9s/templates/` as JSON files. Existing files are skipped unless `--force` is used.

2. **Edit the JSON files** in `~/.s9s/templates/` to match your environment (change partitions, modules, default resources, etc.). The exported files use the same format that `JobTemplateManager` loads — changes take effect on the next wizard open without restarting s9s.

3. **Verify your templates** are loaded:
   ```bash
   s9s templates list
   ```
   Edited templates show as `saved` source and override any built-in with the same name.

4. **Optionally restrict sources** so only your edited templates appear in the wizard:
   ```yaml
   views:
     jobs:
       submission:
         templateSources: ["saved"]
   ```

5. **Use templates** in the wizard — press `s` to open the submission wizard, then pick from the template selector. Your custom templates appear with the values you set.

### Controlling Template Sources

By default all three sources are loaded. Use the `templateSources` config option to control which sources appear:

```yaml
views:
  jobs:
    submission:
      # Show only user-saved and config templates (hide built-ins)
      templateSources: ["config", "saved"]
```

Valid values: `"builtin"`, `"config"`, `"saved"`.

## Job Workflows

### Job Chains

Create dependent job workflows:

```bash
# Submit job chain
:chain submit \
  --job1 preprocess.sh \
  --job2 analyze.sh --after job1 \
  --job3 cleanup.sh --after job2

# View chain status
:chain status my-workflow

# Cancel entire chain
:chain cancel my-workflow
```

### Recurring Jobs

Set up recurring job submissions:

```bash
# Daily job
:schedule add daily-backup \
  --script backup.sh \
  --time "02:00" \
  --repeat daily

# Weekly analysis
:schedule add weekly-report \
  --script report.sh \
  --day monday \
  --time "09:00" \
  --repeat weekly
```

## Job Reporting

### Export Job Data

Export job information for analysis:

```bash
# Export current view
:export csv jobs.csv

# Export with filters
:export json --filter "state:COMPLETED user:${USER}" my-jobs.json

# Export with specific columns
:export markdown --columns "JobID,Name,State,Runtime,Efficiency" report.md
```

### Generate Reports

Create job reports:

```bash
# User summary report
:report user-summary --period month

# Efficiency report
:report efficiency --threshold 0.7

# Failed jobs analysis
:report failures --period week --format html > failures.html
```

## Tips & Best Practices

### Efficiency Tips

1. **Use templates** for repetitive jobs
2. **Set up filters** for your common queries
3. **Monitor efficiency** to optimize resource requests
4. **Use batch operations** for multiple similar jobs
5. **Enable notifications** for long-running jobs

### Common Workflows

#### Debug Failed Jobs
```bash
/state:FAILED          # Filter failed jobs
Enter                  # View job details
o                      # Check output/errors
R                      # Requeue if needed
```

#### Monitor GPU Usage
```bash
/partition:gpu state:RUNNING    # Filter GPU jobs
i                              # View job info
Tab → Performance             # Check GPU utilization
```

#### Bulk Cancel User Jobs
```bash
/user:username                # Filter by user
V                            # Select all visible
b                           # Batch operations
c                          # Cancel selected
```

## Troubleshooting

### Common Issues

**Job Stuck in PENDING**
- Check reason code in job details
- View partition limits
- Check dependencies
- Verify resource availability

**Low Efficiency**
- Review resource requests
- Check for I/O bottlenecks
- Verify correct partition
- Consider job profiling

**Output Not Found**
- Verify output paths in job script
- Check working directory
- Ensure write permissions
- Look for redirected output

## Next Steps

- Learn about [Jobs View Details](./views/jobs.md)
- Explore [Performance Monitoring](../PERFORMANCE_ANALYSIS.md)
- Set up [Keyboard Shortcuts](./keyboard-shortcuts.md)
- Master [Advanced Filtering](./views/jobs.md#advanced-filter)
