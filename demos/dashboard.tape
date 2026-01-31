Require s9s

Output demos/output/dashboard.gif
Output demos/output/dashboard.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Switch to Dashboard (key 8)
Type "8"
Sleep 2s

# Show advanced analytics (A key)
Type "A"
Sleep 3s
Escape
Sleep 500ms

# Show health check details (H key)
Type "H"
Sleep 3s
Escape
Sleep 500ms

# Jump to Jobs view (J key)
Type "J"
Sleep 1500ms

# Back to Dashboard
Type "8"
Sleep 1s

# Jump to Nodes view (N key)
Type "N"
Sleep 1500ms

# Back to Dashboard
Type "8"
Sleep 1s

# Jump to Partitions view (P key)
Type "P"
Sleep 1500ms

# Back to Dashboard
Type "8"
Sleep 1s

# Refresh dashboard
Type "R"
Sleep 1s

# Exit
Type "q"
Sleep 500ms
