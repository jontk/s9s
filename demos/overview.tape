Require s9s

Output demos/output/overview.gif
Output demos/output/overview.mp4

Source demos/common.tape

# Setup environment
Hide
Type "export S9S_ENABLE_MOCK=dev"
Enter
Sleep 500ms
Type "clear"
Enter
Sleep 500ms
Show

# Launch s9s
Type "./build/s9s"
Sleep 500ms
Enter
Sleep 2s

# Show Jobs view (default)
Sleep 2s

# Navigate to Nodes (press 2)
Type "2"
Sleep 2s

# Navigate to Partitions (press 3)
Type "3"
Sleep 2s

# Navigate to Dashboard (press 9)
Type "9"
Sleep 2s

# Show Global Search
Ctrl+F
Sleep 1s
Type "compute"
Sleep 2s
Escape
Sleep 1s

# Back to Jobs
Type "1"
Sleep 2s

# Show filtering
Type "/"
Sleep 500ms
Type "state:RUNNING"
Sleep 1s
Enter
Sleep 2s

# Clear filter and exit
Escape
Sleep 1s
Type "q"
Sleep 1s
