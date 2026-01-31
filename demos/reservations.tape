Require s9s

Output demos/output/reservations.gif
Output demos/output/reservations.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Switch to Reservations view
Type "7"
Sleep 2s

# Navigate through reservations
Down
Sleep 500ms

# Show reservation details
Enter
Sleep 2s
Escape
Sleep 1s

# Toggle active filter (a key)
Type "a"
Sleep 2s

# Toggle future filter (f key)
Type "f"
Sleep 2s

# Exit
Type "q"
Sleep 500ms
