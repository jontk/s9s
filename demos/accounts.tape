Require s9s

Output demos/output/accounts.gif
Output demos/output/accounts.mp4

Source demos/common.tape
Source demos/bootstrap.tape

# Switch to Accounts view (key 6)
Type "6"
Sleep 1500ms

# Navigate through accounts
Down
Sleep 500ms
Down
Sleep 500ms

# Show account details (Enter)
Enter
Sleep 2s
Escape
Sleep 500ms

# Show account hierarchy tree (H key)
Type "H"
Sleep 3s
Escape
Sleep 500ms

# Quick filter
Type "/"
Sleep 500ms
Type "research"
Sleep 1s
Escape
Sleep 500ms

# Sort by organization
Type "2"
Sleep 1s

# Sort by default QoS
Type "4"
Sleep 1s

# Back to name sort
Type "1"
Sleep 500ms

# Global search (Ctrl+F)
Ctrl+F
Sleep 500ms
Type "root"
Sleep 1s
Escape
Sleep 500ms

# Exit
Type "q"
Sleep 500ms
