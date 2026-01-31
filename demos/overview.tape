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
Type "./s9s"
Sleep 500ms
Enter
Sleep 2s

# Jobs view (default - key 1) - show job details
Down
Sleep 200ms
Enter
Sleep 1500ms
Escape
Sleep 300ms

# Nodes view (key 2) - show resource bars
Type "2"
Sleep 1500ms

# Partitions view (key 3) - show queue analytics
Type "3"
Sleep 1500ms

# Reservations view (key 4)
Type "4"
Sleep 1200ms

# QoS view (key 5) - show priority colors
Type "5"
Sleep 1200ms

# Accounts view (key 6)
Type "6"
Sleep 1200ms

# Users view (key 7)
Type "7"
Sleep 1200ms

# Dashboard view (key 8) - cluster overview
Type "8"
Sleep 1500ms

# Health view (key 9) - alerts and monitoring
Type "9"
Sleep 1500ms

# Demonstrate Global Search (Ctrl+F)
Ctrl+F
Sleep 500ms
Type "compute"
Sleep 1s
Down
Sleep 200ms
Enter
Sleep 1500ms

# Back to Jobs for filtering demo
Type "1"
Sleep 800ms

# Quick filter (/)
Type "/"
Sleep 300ms
Type "RUNNING"
Sleep 800ms
Escape
Sleep 300ms

# Show help (?)
Type "?"
Sleep 2s
Escape
Sleep 300ms

# Exit
Type "q"
Sleep 500ms
