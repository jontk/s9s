Require s9s

Output demos/output/qos.gif
Output demos/output/qos.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Switch to QoS view (key 5)
Type "5"
Sleep 1500ms

# Navigate through QoS entries (note priority color-coding)
Down
Sleep 500ms
Down
Sleep 500ms
Down
Sleep 500ms

# Show QoS details (Enter) - displays all limits
Enter
Sleep 3s
Escape
Sleep 500ms

# Sort by priority
Type "2"
Sleep 1s

# Sort by preemption mode
Type "3"
Sleep 1s

# Back to name
Type "1"
Sleep 500ms

# Quick filter
Type "/"
Sleep 300ms
Type "high"
Sleep 1s
Escape
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
