Require s9s

Output demos/output/qos.gif
Output demos/output/qos.mp4

Source demos/common.tape
Source demos/bootstrap.tape


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

# Sort by priority (S key)
Type "S"
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1s

# Quick filter
Type "/"
Sleep 500ms
Type "high"
Sleep 1s
Escape
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
