Require s9s

Output demos/output/qos.gif
Output demos/output/qos.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Switch to QoS view
Type "6"
Sleep 2s

# Navigate through QoS entries
Down
Sleep 500ms
Down
Sleep 500ms

# Show QoS details
Enter
Sleep 3s
Escape
Sleep 1s

# Exit
Type "q"
Sleep 500ms
