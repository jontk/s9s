Require s9s

Output demos/output/search.gif
Output demos/output/search.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Open global search (Ctrl+F)
Ctrl+F
Sleep 1s

# Search for jobs
Type "job"
Sleep 1500ms

# Clear and search for a node
Ctrl+U
Type "node"
Sleep 1500ms

# Clear and search for compute resources
Ctrl+U
Type "compute"
Sleep 1500ms

# Navigate through results with Tab to focus results list
Tab
Sleep 500ms

# Navigate through search results
Down
Sleep 300ms
Down
Sleep 300ms
Down
Sleep 300ms

# Select a result - Enter navigates to the item
Enter
Sleep 2s

# Now we're in the target view showing the selected item
Sleep 1s

# Open search again from this view
Ctrl+F
Sleep 500ms
Type "admin"
Sleep 1500ms

# Select first result directly with Enter
Enter
Sleep 2s

# Exit
Type "q"
Sleep 500ms
