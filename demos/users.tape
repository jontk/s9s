Require s9s

Output demos/output/users.gif
Output demos/output/users.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Switch to Users view (key 7)
Type "7"
Sleep 1500ms

# Navigate through users
Down
Sleep 300ms
Down
Sleep 300ms

# Show user details (Enter)
Enter
Sleep 2s
Escape
Sleep 500ms

# Toggle admin filter - show admins only (a key)
Type "a"
Sleep 1500ms

# Toggle back to show all users
Type "a"
Sleep 1s

# Quick filter by name
Type "/"
Sleep 300ms
Type "admin"
Sleep 1s
Escape
Sleep 500ms

# Sort by different columns
Type "2"
Sleep 1s
Type "3"
Sleep 1s
Type "1"
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
