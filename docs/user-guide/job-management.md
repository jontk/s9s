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
   #SBATCH --job-name=my_analysis
   #SBATCH --output=output_%j.log

   module load python/3.9
   python analyze.py
   ```

### Job Submission Workflow

![Job Submission Demo](/assets/demos/job-submission.gif)

*Job submission wizard with step-by-step configuration*

The submission process guides you through all necessary options with helpful defaults and validation.

### Template-Based Submission

Use templates for common job types:

```bash
# List available templates
:templates list

# Submit from template
:submit template gpu-training

# Create new template
:template save current-job my-template
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

### Creating Templates

Save job configurations as templates:

1. Configure job in submission wizard
2. Instead of submitting, press `Ctrl+S`
3. Name your template
4. Template saved to `~/.s9s/templates/`

### Using Templates

```bash
# List templates
:template list

# View template
:template show gpu-analysis

# Submit from template
:template submit gpu-analysis

# Edit template
:template edit gpu-analysis

# Share template
:template export gpu-analysis > gpu-template.yaml
```

### Template Variables

Templates support variables:

```yaml
# ~/.s9s/templates/parametric.yaml
name: "parametric_${INDEX}"
script: |
  #!/bin/bash
  #SBATCH --array=1-${ARRAY_SIZE}
  #SBATCH --mem=${MEMORY}GB

  python process.py --index=$SLURM_ARRAY_TASK_ID

variables:
  ARRAY_SIZE: 100
  MEMORY: 32
```

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
