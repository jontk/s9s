Require s9s

Output demos/output/health.gif
Output demos/output/health.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Switch to Health view
Type "8"
Sleep 2s

# Navigate through alerts
Down
Sleep 500ms
Down
Sleep 500ms

# Show health statistics (s key)
Type "s"
Sleep 3s
Escape
Sleep 1s

# Exit
Type "q"
Sleep 500ms
