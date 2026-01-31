Require s9s

Output demos/output/jobs.gif
Output demos/output/jobs.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Jobs view is default - show the list
Sleep 1s

# Navigate and show job details (Enter)
Down
Sleep 300ms
Down
Sleep 300ms
Enter
Sleep 2s
Escape
Sleep 500ms

# Filter jobs by text
Type "/"
Sleep 300ms
Type "RUNNING"
Sleep 500ms
Enter
Sleep 1500ms
Escape
Sleep 500ms

# Show all jobs
Type "a"
Sleep 1s

# Show pending only
Type "p"
Sleep 1500ms

# Show all again
Type "a"
Sleep 500ms

# View job output (o key)
Down
Sleep 300ms
Type "o"
Sleep 2s
Escape
Sleep 500ms

# Show job dependencies (d key)
Type "d"
Sleep 2s
Escape
Sleep 500ms

# Enable multi-select mode (v key)
Type "v"
Sleep 500ms

# Select multiple jobs with Space
Space
Sleep 300ms
Down
Sleep 300ms
Space
Sleep 300ms
Down
Sleep 300ms
Space
Sleep 1s

# Show batch operations (b key)
Type "b"
Sleep 2s
Escape
Sleep 500ms

# Exit multi-select mode
Type "v"
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
