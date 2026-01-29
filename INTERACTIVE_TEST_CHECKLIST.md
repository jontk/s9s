# S9S Interactive TUI Testing Checklist

## Configuration Status
✅ **Config File**: `~/.s9s/config.yaml`
✅ **Cluster URL**: `http://rocky9.ar.jontk.com:6820`
✅ **API Version**: `v0.0.44`
✅ **Token**: Fresh JWT obtained
✅ **Config Validation**: PASSED

---

## Test Session Commands

### Start s9s TUI (Real Cluster)
```bash
# Connect to real cluster
./s9s --no-mock

# Or with debug logging
./s9s --no-mock --debug
```

### Start s9s TUI (Mock Mode)
```bash
# Test with mock data
export S9S_ENABLE_MOCK=development
./s9s --mock
```

---

## Interactive Features Test Checklist

### 1. Navigation & Views
- [ ] **Launch s9s** - Application starts without errors
- [ ] **Jobs View (j)** - Shows all jobs in queue
- [ ] **Nodes View (n)** - Shows cluster nodes
- [ ] **Partitions View (p)** - Shows normal, debug, long partitions
- [ ] **Users View (u)** - Shows user accounts
- [ ] **QoS View (q)** - Shows Quality of Service settings
- [ ] **Tab Key** - Cycles through views
- [ ] **Arrow Keys** - Navigate up/down in lists
- [ ] **Help (?)** - Shows help screen

### 2. Search & Filter
- [ ] **Search (/)** - Opens search prompt
- [ ] **Search by Job ID** - Filters to specific job
- [ ] **Search by User** - Filters by username
- [ ] **Search by Status** - Filters RUNNING, PENDING, etc.
- [ ] **Clear Filter (Esc)** - Returns to full view

### 3. Job Management
- [ ] **View Job Details (d)** - Shows detailed job info
- [ ] **View Job Output (o)** - Shows job stdout/stderr
- [ ] **Cancel Job (c)** - Prompts to cancel selected job
- [ ] **Hold Job (h)** - Places job on hold
- [ ] **Release Job (r)** - Releases held job
- [ ] **Batch Operations (b)** - Multi-select jobs

### 4. Node Operations
- [ ] **SSH to Node (s)** - Opens SSH connection to node
- [ ] **Node Details (d)** - Shows detailed node info
- [ ] **Drain Node** - Marks node as draining
- [ ] **Resume Node** - Returns node to idle

### 5. Refresh & Updates
- [ ] **Manual Refresh (Ctrl+r)** - Forces immediate refresh
- [ ] **Auto-refresh** - Updates every 5 seconds (per config)
- [ ] **Toggle Auto-refresh (m)** - Pause/resume updates

### 6. Export Functionality
- [ ] **Export to CSV** - Exports current view to CSV
- [ ] **Export to JSON** - Exports to JSON format
- [ ] **Export to Markdown** - Exports to Markdown table
- [ ] **Export to HTML** - Exports to HTML table

### 7. Display & UI
- [ ] **Dark Theme** - Proper colors and contrast
- [ ] **Column Sorting** - Click headers to sort
- [ ] **Column Resize** - Adjust column widths
- [ ] **Status Bar** - Shows connection status
- [ ] **Job Count** - Displays total jobs
- [ ] **Last Update Time** - Shows refresh timestamp

### 8. Error Handling
- [ ] **Network Error** - Graceful handling if API down
- [ ] **Auth Error** - Clear message if token expired
- [ ] **Invalid Operation** - Proper error messages
- [ ] **Connection Retry** - Attempts to reconnect

### 9. Performance
- [ ] **Fast Startup** - Launches in <2 seconds
- [ ] **Responsive UI** - No lag when scrolling
- [ ] **Memory Usage** - Stable memory footprint
- [ ] **Refresh Performance** - Quick updates

### 10. Advanced Features
- [ ] **Plugin System** - Can load plugins if any
- [ ] **Custom Views** - Configuration-based views work
- [ ] **Command Mode (:)** - Enter custom commands
- [ ] **Configuration Reload** - Hot reload config changes

---

## Specific Tests for rocky9 Cluster

### Expected Data to Verify

**Jobs** (should see):
- Job #358 - Running
- Job #359-365 - Pending (various states)
- Job #366 - Test job we submitted
- Job #257 - airflow_ml_pipeline (Pending)

**Partitions** (should see):
- normal* (unlimited time)
- debug (3:30:00 limit)
- long (7-day limit)

**Nodes** (should see):
- rocky9.ar.jontk.com
- Status: idle or allocated
- CPUs: 2 cores
- Memory: 1771 MB

**Users** (should see):
- root (admin)

---

## Test Scenarios

### Scenario 1: View Current Jobs
1. Start s9s
2. Default view should show jobs
3. Verify all 9 jobs are visible
4. Check job states: RUNNING, PENDING
5. Verify job IDs match squeue output

### Scenario 2: Search for Specific Job
1. Press `/` to search
2. Enter job ID (e.g., "358")
3. Verify filtered to that job only
4. Press ESC to clear filter

### Scenario 3: View Job Details
1. Navigate to a running job
2. Press `d` for details
3. Verify shows:
   - Job ID, name, user
   - Partition, state, time
   - Node assignment
   - Resources allocated

### Scenario 4: Navigate Between Views
1. Start in jobs view (j)
2. Press `n` for nodes view
3. Press `p` for partitions view
4. Press `Tab` to cycle through
5. Verify each view loads correctly

### Scenario 5: Monitor Job Updates
1. Select a running job
2. Watch time elapsed update
3. Press `m` to toggle auto-refresh
4. Press `Ctrl+r` to force refresh
5. Verify updates occur

### Scenario 6: Test with Mock Mode
1. Exit if in real mode (q)
2. Set: `export S9S_ENABLE_MOCK=development`
3. Run: `./s9s --mock`
4. Verify mock data loads
5. Test all navigation features
6. Compare with real cluster view

---

## Expected Issues to Watch For

⚠️ **Token Expiration**
- JWT tokens expire after ~30 minutes
- Should show clear error message
- Solution: Regenerate token with `scontrol token`

⚠️ **Network Latency**
- Remote connection may have delays
- Should handle gracefully
- Auto-refresh may need adjustment

⚠️ **Large Job Queues**
- 9+ jobs currently in queue
- UI should handle scrolling well
- Performance should remain good

⚠️ **Held Jobs**
- Jobs #361, #362 are held (JobHeldAdmin)
- Should show correct state
- Release operation should work

---

## Test Results Template

```
Date: _____________
Tester: _____________
s9s Version: dev (Go 1.24.5)
Cluster: rocky9.ar.jontk.com
SLURM: 25.11.1
API: v0.0.44

✅ PASSED: __/40 tests
⚠️  ISSUES: __
❌ FAILED: __

Notes:
_______________________________________
_______________________________________
_______________________________________
```

---

## Quick Reference

**Launch Commands**:
```bash
# Real cluster
./s9s --no-mock

# Mock mode
S9S_ENABLE_MOCK=development ./s9s --mock

# With debug
./s9s --no-mock --debug

# Show help
./s9s --help
```

**Keyboard Shortcuts**:
```
?     Help
q     Quit
j     Jobs view
n     Nodes view
p     Partitions view
u     Users view
Tab   Next view
/     Search
d     Details
c     Cancel
h     Hold
r     Release
m     Toggle refresh
Ctrl+r Force refresh
```

**Configuration**:
```bash
# View config
./s9s config show

# Validate config
./s9s config validate

# Edit config
vim ~/.s9s/config.yaml
```

---

## Post-Testing

After completing tests:

1. **Document Issues**: Note any bugs or unexpected behavior
2. **Performance Notes**: Record any lag or slowness
3. **Feature Requests**: List desired improvements
4. **Screenshots**: Capture key views if possible
5. **Update Checklist**: Mark completed items

---

**Ready to Test!** 🚀

Start with: `./s9s --no-mock`
