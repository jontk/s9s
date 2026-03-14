Require s9s

Output demos/output/job-submission.gif
Output demos/output/job-submission.mp4

Source demos/common.tape
Source demos/bootstrap.tape

# Override height for job submission wizard (needs more space)
Set Height 900


# Open job submission wizard (s key)
Type "s"
Sleep 2s

# Template selection is shown first
# Select "Custom Job" (already highlighted as first item)
Enter
Sleep 1500ms

# Fill in job name
Type "simulation-run-01"
Tab
Sleep 500ms

# Add script content in the text area
Type "#!/bin/bash"
Enter
Type "srun ./my_simulation --input data.csv"
Sleep 1s

# Tab through remaining fields
Tab
Sleep 500ms
Tab
Sleep 500ms
Tab
Sleep 500ms
Tab
Sleep 500ms
Tab
Sleep 500ms
Tab
Sleep 500ms

# Pause to show the filled form
Sleep 2s

# Cancel this submission
Escape
Sleep 1s

# Open wizard again to show template selection
Type "s"
Sleep 2s

# Select "MPI Parallel Job" template (second item)
Down
Enter
Sleep 2s

# Show the pre-filled template form
Sleep 2s

# Cancel and return to jobs view
Escape
Sleep 1s

# Exit
Type "q"
Sleep 500ms
