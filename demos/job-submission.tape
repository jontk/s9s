Require s9s

Output demos/output/job-submission.gif
Output demos/output/job-submission.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Open job submission (s key)
Type "s"
Sleep 2s

# Select template (e.g., Basic Batch Job)
Down
Sleep 500ms
Enter
Sleep 2s

# Fill in job name
Type "my-demo-job"
Tab
Sleep 500ms

# Add script content
Type "#!/bin/bash"
Enter
Type "echo 'Hello SLURM'"
Tab
Sleep 1s

# Navigate through fields
Tab
Sleep 500ms
Tab
Sleep 500ms

# Preview the job script
Tab
Tab
Tab
Enter
Sleep 3s
Escape
Sleep 1s

# Cancel and exit
Escape
Sleep 1s
Type "q"
Sleep 500ms
