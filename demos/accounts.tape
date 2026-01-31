Require s9s

Output demos/output/accounts.gif
Output demos/output/accounts.mp4

Source demos/common.tape

Hide
Type "export S9S_ENABLE_MOCK=dev && clear"
Enter
Sleep 500ms
Show

Type "./build/s9s"
Enter
Sleep 2s

# Switch to Accounts view
Type "5"
Sleep 2s

# Navigate through accounts
Down
Sleep 500ms
Down
Sleep 500ms

# Show account details
Enter
Sleep 2s
Escape
Sleep 1s

# Show account hierarchy/tree (H key)
Type "H"
Sleep 3s
Escape
Sleep 1s

# Exit
Type "q"
Sleep 500ms
