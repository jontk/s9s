Require s9s

Output demos/output/reservations.gif
Output demos/output/reservations.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Switch to Reservations view (key 4)
Type "4"
Sleep 1500ms

# Navigate through reservations
Down
Sleep 300ms
Down
Sleep 300ms

# Show reservation details (Enter)
Enter
Sleep 2s
Escape
Sleep 500ms

# Toggle active only filter (a key)
Type "a"
Sleep 1500ms

# Toggle future only filter (f key)
Type "f"
Sleep 1500ms

# Show all reservations (toggle both off)
Type "a"
Sleep 500ms
Type "f"
Sleep 1s

# Sort by start time
Type "3"
Sleep 1s

# Sort by duration
Type "4"
Sleep 1s

# Back to name
Type "1"
Sleep 500ms

# Quick filter
Type "/"
Sleep 300ms
Type "maint"
Sleep 1s
Escape
Sleep 500ms

# Refresh the view
Type "R"
Sleep 1s

# Exit
Type "q"
Sleep 500ms
