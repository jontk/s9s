# Job Output Streaming

s9s supports real-time job output streaming, allowing you to watch stdout and stderr as a job writes to its output files. This works similar to `tail -f` but is integrated directly into the TUI.

## Quick Start

1. Open s9s and navigate to the **Jobs** view.
2. Select a running job.
3. Press `o` to open the Job Output Viewer.
4. Press `t` to start real-time streaming.

Output appears in real time as the job writes to its log files.

## Enabling Streaming

Streaming is enabled by default. You can toggle it in your configuration file:

```yaml
features:
  streaming: true
```

Or set the default in `config.example.yaml`. When disabled, the Job Output Viewer still works for loading output on demand (press `r` to refresh), but real-time streaming is unavailable.

## Job Output Viewer

The Job Output Viewer is a modal that opens over the Jobs view. It displays the contents of a single job's stdout or stderr file.

### Key Bindings

| Key   | Action                                |
|-------|---------------------------------------|
| `o`   | Open Job Output Viewer (from Jobs)    |
| `t`   | Toggle real-time streaming on/off     |
| `a`   | Toggle auto-scroll                    |
| `s`   | Switch between stdout and stderr      |
| `f`   | Follow output (scroll to end + auto-refresh) |
| `r`   | Refresh output manually               |
| `e`   | Export output to file                 |
| `Esc` | Close the viewer                      |

### Streaming Status

When streaming is active, the title bar shows a **LIVE** indicator. The status bar at the bottom displays:

- **Streaming ACTIVE** with buffer usage when a stream is running.
- **Streaming stopped** with a hint to press `t` when idle.
- **Streaming not available** if the `StreamManager` is not configured (for example, when the feature flag is off).

### Auto-Scroll

Auto-scroll is enabled by default. When new output arrives, the viewer automatically scrolls to the bottom. Press `a` to toggle this behavior if you need to review earlier output without being interrupted.

## How Streaming Works

Behind the scenes, the `StreamManager` in `internal/streaming/` coordinates output streaming for a single job at a time.

### Path Resolution

When you start a stream, the `PathResolver` determines where the output file lives:

1. It queries the SLURM API (via `slurmrestd`) for the job's `StdOut` and `StdErr` paths.
2. If the API does not provide a path, it falls back to the job's working directory or the configured SLURM spool directory.
3. It checks whether the file is on the local filesystem or on a remote compute node.

### Local File Streaming

For files on a shared filesystem (the common HPC setup with NFS or Lustre), the `StreamManager` uses `fsnotify` to watch for file changes. When the output file is written to, new content is read from the last known offset and pushed to the viewer.

### Remote File Streaming (SSH)

When the output file is on a remote compute node that is not accessible via a shared filesystem, the `StreamManager` falls back to SSH-based polling:

- It connects to the compute node using the configured SSH credentials.
- It periodically runs `tail -c +<offset>` to fetch new content since the last read.
- The default polling interval is 3 seconds.

If a local file watch fails because the file does not exist locally but a compute node is assigned to the job, the manager automatically falls back to SSH polling.

### Circular Buffer

Each active stream maintains a `CircularBuffer` (default capacity: 10,000 lines) to store recent output efficiently. When the buffer reaches capacity, the oldest lines are discarded. This keeps memory usage bounded regardless of how much output a job produces.

### Event Bus

The streaming system uses an internal `EventBus` for pub/sub event delivery. The Job Output Viewer subscribes to events for the active job and output type. Events include:

- `NEW_OUTPUT` -- new content available.
- `JOB_COMPLETE` -- the job has finished.
- `ERROR` -- a streaming error occurred.
- `FILE_ROTATED` -- the output file was rotated or truncated.
- `STREAM_START` / `STREAM_STOP` -- lifecycle events.

## Export

From the Job Output Viewer, press `e` to export the currently displayed output. Supported formats:

- **Text** -- plain text with a header.
- **JSON** -- structured output with metadata.
- **CSV** -- line-by-line for analysis.
- **Markdown** -- output in fenced code blocks.

Exported files are saved to `~/slurm_exports/` by default.

## Troubleshooting

### No output appears

- Verify the job is running and has started writing output. Check with `scontrol show job <id> | grep StdOut`.
- Confirm the output file exists and is readable: `ls -la <path>`.
- If the file is on a remote node, verify SSH connectivity: `ssh <node> cat <path>`.

### Streaming says "not available"

- Check that `features.streaming: true` is set in your configuration.
- Ensure the `StreamManager` was initialized at startup (check debug logs with `s9s --debug`).

### SSH streaming is slow or failing

- The default SSH polling interval is 3 seconds. This is intentional to avoid overloading compute nodes.
- Verify SSH key-based authentication is configured for the compute nodes.
- Check that the SSH user has read access to the output file.

## Related Guides

- [Configuration Guide](../getting-started/configuration.md)
