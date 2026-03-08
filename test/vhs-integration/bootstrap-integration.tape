# Bootstrap for integration tests against real Slurm clusters
# Expects SLURM_REST_URL and SLURM_JWT to be set in the environment

Hide
Type "clear"
Enter
Sleep 500ms
Show

# Launch s9s without mock mode, with auto-discovery disabled
Type "./s9s --no-discovery"
Enter
Sleep 3s
