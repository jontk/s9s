Require s9s

Output demos/output/partitions.gif
Output demos/output/partitions.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Switch to Partitions view (key 3)
Type "3"
Sleep 1500ms

# Navigate and show partition details
Down
Sleep 300ms
Enter
Sleep 2s
Escape
Sleep 500ms

# Show partition analytics (A key)
Type "A"
Sleep 3s
Escape
Sleep 500ms

# Show wait time analytics for all partitions (W key)
Type "W"
Sleep 3s
Escape
Sleep 500ms

# Jump to jobs for selected partition (J key)
Down
Sleep 300ms
Type "J"
Sleep 2s

# Go back to partitions
Type "3"
Sleep 1s

# Jump to nodes for partition (N key)
Type "N"
Sleep 2s

# Return to partitions
Type "3"
Sleep 1s

# Quick filter
Type "/"
Sleep 300ms
Type "gpu"
Sleep 1s
Escape
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
