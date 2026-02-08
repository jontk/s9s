Require s9s

Output demos/output/partitions.gif
Output demos/output/partitions.mp4

Source demos/common.tape
Source demos/bootstrap.tape


# Switch to Partitions view (key 3)
Type "3"
Sleep 1500ms

# Navigate and show partition details
Down
Sleep 500ms
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
Sleep 500ms
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

# Sort by column (S key)
Type "S"
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1s

# Quick filter
Type "/"
Sleep 500ms
Type "gpu"
Sleep 1s
Escape
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
