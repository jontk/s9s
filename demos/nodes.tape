Require s9s

Output demos/output/nodes.gif
Output demos/output/nodes.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Switch to Nodes view
Type "2"
Sleep 2s

# Navigate to a node
Down
Sleep 500ms
Down
Sleep 500ms

# Show node details
Enter
Sleep 2s
Escape
Sleep 1s

# Show grouping options
Type "g"
Sleep 2s

# Filter by partition
Type "/"
Sleep 500ms
Type "p:compute"
Enter
Sleep 2s

# Clear and exit
Escape
Sleep 1s
Type "q"
Sleep 500ms
