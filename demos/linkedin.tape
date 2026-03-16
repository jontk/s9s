# LinkedIn demo — 12 seconds, 3 views, one "aha" moment
# Shows: jobs → node detail → dashboard. That's it.

Require s9s

Output demos/output/linkedin.gif
Output demos/output/linkedin.mp4

Source demos/common.tape

# Setup environment (hidden from recording)
Hide
Type "export S9S_ENABLE_MOCK=dev"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 500ms
Show

# Launch s9s — pause so viewer reads the command
Type "s9s"
Sleep 1500ms
Enter
Sleep 2s

# Jobs view (default) — let it breathe so viewer reads the table
Sleep 2s

# Select a job to show detail panel
Down
Sleep 300ms
Enter
Sleep 2s
Escape
Sleep 500ms

# Nodes view — resource bars are the visual hook
Type "2"
Sleep 2s

# Partitions view — queue analytics
Type "3"
Sleep 2s

# Dashboard view — cluster-wide summary, end here
Type "8"
Sleep 2s
