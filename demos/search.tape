Require s9s

Output demos/output/search.gif
Output demos/output/search.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Open global search
Ctrl+F
Sleep 1s

# Search for a job
Type "job"
Sleep 2s

# Clear and search for node
Ctrl+U
Type "node"
Sleep 2s

# Clear and search for user
Ctrl+U
Type "admin"
Sleep 2s

# Select a result
Down
Sleep 500ms
Enter
Sleep 2s

# Exit
Type "q"
Sleep 500ms
