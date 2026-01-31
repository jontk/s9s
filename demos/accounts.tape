Require s9s

Output demos/output/accounts.gif
Output demos/output/accounts.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./s9s"
Enter
Sleep 2s

# Switch to Accounts view (key 6)
Type "6"
Sleep 1500ms

# Navigate through accounts
Down
Sleep 300ms
Down
Sleep 300ms

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
Sleep 300ms
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
