# S9S VHS Demo Recordings

This directory contains VHS tape recordings for demonstrating the s9s SLURM TUI application. The demos are used for marketing materials, documentation, and showcasing the application's features.

## Prerequisites

1. **Install VHS**: The charmbracelet VHS tool is required to generate demos.
   ```bash
   go install github.com/charmbracelet/vhs@latest
   ```

2. **Build s9s**: Demos require a built binary.
   ```bash
   make build
   ```

3. **Mock Mode**: All demos use mock mode (`S9S_ENABLE_MOCK=dev`) to ensure consistent, reproducible recordings without requiring a live SLURM cluster.

## Demo Files

### Common Settings
- **`common.tape`**: Shared VHS settings for all demos (fonts, colors, timing, etc.)

### Main Demos

| Demo | File | Duration | Description |
|------|------|----------|-------------|
| **Overview** | `overview.tape` | ~90s | Comprehensive tour of all features |
| **Jobs** | `jobs.tape` | ~45s | Job management, filtering, details |
| **Nodes** | `nodes.tape` | ~45s | Node operations, SSH, grouping |
| **Partitions** | `partitions.tape` | ~45s | Queue analytics, wait times |
| **Users** | `users.tape` | ~30s | User management, admin filters |
| **Accounts** | `accounts.tape` | ~30s | Account hierarchy, tree view |
| **QoS** | `qos.tape` | ~30s | Quality of Service policies |
| **Reservations** | `reservations.tape` | ~30s | Resource reservations |
| **Health** | `health.tape` | ~30s | Alerts, health monitoring |
| **Job Submission** | `job-submission.tape` | ~45s | Job wizard, templates |
| **Dashboard** | `dashboard.tape` | ~30s | Cluster overview |
| **Search** | `search.tape` | ~30s | Global search functionality |

## Generating Demos

### Generate All Demos
```bash
make demos
```

This will:
1. Build the s9s binary
2. Generate GIF and MP4 outputs for all demos
3. Save outputs to `demos/output/`

### Generate Single Demo
```bash
# Overview demo only
make demo-overview

# Or any specific demo
vhs demos/jobs.tape
vhs demos/nodes.tape
```

### Clean Outputs
```bash
make demo-clean
```

## Output Files

Generated demos are saved in `demos/output/`:
- `*.gif` - Animated GIF format (for web, GitHub, etc.)
- `*.mp4` - MP4 video format (for presentations, social media)

## Customizing Demos

### Visual Settings
Edit `common.tape` to change:
- Font family and size
- Window dimensions
- Color theme
- Typing speed
- Frame rate

### Demo Content
Edit individual `.tape` files to:
- Adjust timing with `Sleep` commands
- Change navigation sequences
- Add or remove features to showcase
- Modify mock data queries

## Best Practices

1. **Keep it concise**: Demos should be under 90 seconds
2. **Show, don't tell**: Let the UI speak for itself
3. **Realistic timing**: Allow viewers time to read the screen
4. **Smooth navigation**: Avoid jarring transitions
5. **Mock data**: Use mock mode for consistency

## VHS Commands Reference

Common VHS commands used in these demos:

| Command | Purpose | Example |
|---------|---------|---------|
| `Type` | Type text | `Type "hello"` |
| `Enter` | Press Enter key | `Enter` |
| `Sleep` | Pause execution | `Sleep 2s` |
| `Escape` | Press Escape key | `Escape` |
| `Ctrl+F` | Keyboard combo | `Ctrl+F` |
| `Down` | Arrow key down | `Down` |
| `Tab` | Tab key | `Tab` |
| `Hide` | Hide terminal output | `Hide` |
| `Show` | Show terminal output | `Show` |

## Troubleshooting

### VHS not found
```bash
go install github.com/charmbracelet/vhs@latest
# Ensure $GOPATH/bin is in your PATH
export PATH="$PATH:$(go env GOPATH)/bin"
```

### s9s binary not found
```bash
make build
# Or specify full path in tape files
```

### Mock data issues
Ensure the environment variable is set in tape files:
```tape
Type "export S9S_ENABLE_MOCK=dev"
```

### Slow generation
- Reduce `Framerate` in `common.tape`
- Decrease window size
- Use fewer demos

## File Size Optimization

GIFs can be large. To optimize:

1. **Reduce dimensions**: Edit `Width` and `Height` in `common.tape`
2. **Lower framerate**: Reduce `Framerate` (try 15-20 fps)
3. **Shorter duration**: Keep demos under 60 seconds
4. **Use MP4**: MP4 files are typically smaller than GIFs

## Using Demos

### In Documentation
```markdown
![S9S Overview](demos/output/overview.gif)
```

### In README
```markdown
## Demo
![Job Management](demos/output/jobs.gif)
```

### Social Media
- Use MP4 format for Twitter, LinkedIn
- Use GIF format for GitHub, Reddit

### Presentations
- MP4 provides better quality for slides
- Can be embedded in PowerPoint, Keynote

## Contributing

When adding new demos:

1. Create a new `.tape` file in `demos/`
2. Source `common.tape` for consistent styling
3. Use mock mode (`S9S_ENABLE_MOCK=dev`)
4. Keep duration under 60 seconds
5. Test the recording before committing
6. Update this README with the new demo

## License

These demo recordings are part of the s9s project and follow the same license.
