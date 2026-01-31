Require s9s

Output demos/output/dashboard.gif
Output demos/output/dashboard.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Switch to Dashboard
Type "9"
Sleep 3s

# Show analytics (A key)
Type "A"
Sleep 3s
Escape
Sleep 1s

# Show health report (H key)
Type "H"
Sleep 3s
Escape
Sleep 1s

# Refresh
Type "R"
Sleep 2s

# Exit
Type "q"
Sleep 500ms
