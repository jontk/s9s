Require s9s

Output demos/output/partitions.gif
Output demos/output/partitions.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Switch to Partitions view
Type "3"
Sleep 2s

# Select a partition
Down
Sleep 500ms

# Show analytics (A key - uppercase)
Type "A"
Sleep 3s
Escape
Sleep 1s

# Show wait time analytics (W key)
Type "W"
Sleep 3s
Escape
Sleep 1s

# Jump to partition's jobs (J key)
Type "J"
Sleep 2s

# Exit
Type "q"
Sleep 500ms
