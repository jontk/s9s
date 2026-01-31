Require s9s

Output demos/output/nodes.gif
Output demos/output/nodes.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Switch to Nodes view (key 2)
Type "2"
Sleep 1500ms

# Navigate to a node and show details
Down
Sleep 300ms
Down
Sleep 300ms
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

# Group by partition
Type "g"
Sleep 500ms
Type "2"
Sleep 1500ms

# Toggle group expansion with Space
Down
Sleep 300ms
Space
Sleep 1s
Space
Sleep 500ms

# Group by state
Type "g"
Sleep 500ms
Type "3"
Sleep 1500ms

# Back to no grouping
Type "g"
Sleep 500ms
Type "1"
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

# Show SSH options (s key)
Type "s"
Sleep 2s
Escape
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
