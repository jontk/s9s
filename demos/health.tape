Require s9s

Output demos/output/health.gif
Output demos/output/health.mp4

Source demos/common.tape
Source demos/bootstrap.tape

# Switch to Health view (key 9)
Type "9"
Sleep 1500ms

# Navigate through alerts
Down
Sleep 500ms
Down
Sleep 500ms

# Show alert details (Enter)
Enter
Sleep 2s
Escape
Sleep 500ms

# Show health statistics (s key)
Type "s"
Sleep 3s
Escape
Sleep 500ms

# Show detailed health check information (H key)
Type "H"
Sleep 3s
Escape
Sleep 500ms

# Acknowledge an alert (a key)
Type "a"
Sleep 1500ms

# Resolve an alert (r key)
Type "r"
Sleep 1500ms

# Clear old resolved alerts (c key)
Type "c"
Sleep 1500ms

# Refresh health data
Type "R"
Sleep 1s

# Exit
Type "q"
Sleep 500ms
