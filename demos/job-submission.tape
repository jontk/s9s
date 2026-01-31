Require s9s

Output demos/output/job-submission.gif
Output demos/output/job-submission.mp4

Source demos/common.tape

# Override height for job submission wizard (needs more space)
Set Height 900

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Open job submission directly (s key)
Type "s"
Sleep 2s

# Fill in job name
Type "simulation-job"
Tab
Sleep 500ms

# Skip to script content (Tab through fields)
Tab
Sleep 300ms

# Add script content
Type "#!/bin/bash"
Enter
Type "#SBATCH --nodes=4"
Enter
Type "#SBATCH --ntasks-per-node=16"
Enter
Enter
Type "srun ./my_simulation"
Sleep 1s

# Navigate through other fields
Tab
Sleep 300ms
Tab
Sleep 300ms
Tab
Sleep 300ms

# Show preview of the job script
Tab
Tab
Enter
Sleep 3s
Escape
Sleep 500ms

# Cancel this submission
Escape
Sleep 500ms

# Open job submission directly (s key)
Type "s"
Sleep 1500ms

# Quick submit without template
Type "quick-test"
Tab
Sleep 300ms
Tab
Sleep 300ms
Type "echo 'Hello World'"
Sleep 1s

# Cancel and exit
Escape
Sleep 500ms
Escape
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
