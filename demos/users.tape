Require s9s

Output demos/output/users.gif
Output demos/output/users.mp4

Source demos/common.tape
Source demos/bootstrap.tape


# Switch to Users view (key 7)
Type "7"
Sleep 1500ms

# Navigate through users
Down
Sleep 500ms
Down
Sleep 500ms

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
Sleep 500ms
Type "admin"
Sleep 1s
Escape
Sleep 500ms

# Sort by UID (S key)
Type "S"
Sleep 500ms
Down
Sleep 500ms
Enter
Sleep 1s

# Exit
Type "q"
Sleep 500ms
