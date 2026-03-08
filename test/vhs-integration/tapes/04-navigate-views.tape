Require s9s

Source test/vhs-integration/common-integration.tape
Source test/vhs-integration/bootstrap-integration.tape

# Cycle through all views to verify they handle real data
Type "1"
Sleep 1s
Type "2"
Sleep 1s
Type "3"
Sleep 1s
Type "4"
Sleep 1s
Type "5"
Sleep 1s
Type "6"
Sleep 1s
Type "7"
Sleep 1s

# Exit
Type "q"
Sleep 500ms
