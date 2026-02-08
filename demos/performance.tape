Require s9s

Output demos/output/performance.gif
Output demos/output/performance.mp4

Source demos/common.tape
Source demos/bootstrap.tape


# Switch to Performance view (key 9)
Type "9"
Sleep 2s

# Show cluster metrics updating
# The view auto-refreshes every 5 seconds
Sleep 3s

# Toggle auto-refresh off (R key)
Type "R"
Sleep 1s

# Manual refresh (F5)
Type@500ms "F5"
Sleep 1500ms

# Toggle auto-refresh back on
Type "R"
Sleep 2s

# Let metrics update one more time
Sleep 3s

# Exit
Type "q"
Sleep 500ms
