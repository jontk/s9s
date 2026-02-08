Require s9s

Output demos/output/nodes.gif
Output demos/output/nodes.mp4

Source demos/common.tape
Source demos/bootstrap.tape


# Switch to Nodes view (key 2)
Type "2"
Sleep 1500ms

# Navigate to a node and show details
Down
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 2s
Escape
Sleep 500ms

# Filter by state - show idle nodes only
Type "i"
Sleep 1500ms

# Show mixed state nodes
Type "m"
Sleep 1500ms

# Show all states
Type "a"
Sleep 1s

# Filter by partition
Type "p"
Sleep 500ms
Type "compute"
Enter
Sleep 1500ms

# Clear partition filter
Type "p"
Sleep 500ms
Ctrl+U
Enter
Sleep 500ms

# Sort by column (S key)
Type "S"
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1s

# Show SSH options (s key)
Type "s"
Sleep 2s
Escape
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
