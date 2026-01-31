Require s9s

Output demos/output/users.gif
Output demos/output/users.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Switch to Users view
Type "4"
Sleep 2s

# Navigate through users
Down
Sleep 500ms
Down
Sleep 500ms

# Show user details
Enter
Sleep 2s
Escape
Sleep 1s

# Toggle admin filter (a key)
Type "a"
Sleep 2s

# Toggle back
Type "a"
Sleep 1s

# Exit
Type "q"
Sleep 500ms
