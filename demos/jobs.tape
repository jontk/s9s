Require s9s

Output demos/output/jobs.gif
Output demos/output/jobs.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Jobs view is default - show the list
Sleep 2s

# Navigate down to select a job
Down
Sleep 500ms
Down
Sleep 500ms

# Show job details (Enter)
Enter
Sleep 2s
Escape
Sleep 1s

# Filter by state
Type "/"
Sleep 500ms
Type "state:PENDING"
Enter
Sleep 2s

# Clear filter
Escape
Sleep 1s

# Show job output (o key)
Type "o"
Sleep 3s
Escape
Sleep 1s

# Show batch operations (b key)
Type "b"
Sleep 2s
Escape
Sleep 1s

# Exit
Type "q"
Sleep 500ms
