# S9S E2E Tests

End-to-end tests for the s9s SLURM TUI using Playwright and ttyd web terminal.

## Prerequisites

1. **Build s9s binary**:
   ```bash
   cd ../..
   CGO_ENABLED=0 go build -o s9s cmd/s9s/main.go
   ```

2. **Install ttyd** (web terminal):
   - macOS: `brew install ttyd`
   - Linux: `apt-get install ttyd` or download from [releases](https://github.com/tsl0922/ttyd/releases)

3. **Install test dependencies**:
   ```bash
   npm install
   ```

## Running Tests

### Start the ttyd server

In one terminal:
```bash
# From project root
./run-with-ttyd.sh

# Or use the gotty-compatible script
./run-with-gotty.sh  # This now uses ttyd internally

# Or from test directory
npm run server:ttyd
```

The TUI will be available at http://localhost:7681 (ttyd default) or http://localhost:8080 (gotty-compatible).

### Run the tests

In another terminal:
```bash
# Basic test
npm test

# Advanced test with screenshots and video
npm run test:advanced

# Headless mode (for CI/CD)
npm run test:headless

# Use custom URL
TTYD_URL=http://192.168.1.100:7681 npm test
```

## Test Files

- `playwright-ttyd-test.js` - Basic navigation and functionality tests
- `playwright-ttyd-advanced.js` - Comprehensive tests with screenshots and video recording
- `playwright-gotty-test.js` - Legacy gotty tests (now uses ttyd)
- `playwright-gotty-advanced.js` - Legacy gotty advanced tests (now uses ttyd)

## Migration from gotty to ttyd

We've migrated from gotty to ttyd for better compatibility and stability:
- ttyd works reliably on modern macOS versions
- Better WebSocket implementation
- Active maintenance and development
- Compatible command-line options

The gotty scripts have been updated to use ttyd internally for backward compatibility.

## Environment Variables

- `TTYD_URL` - Override the default ttyd URL (default: http://localhost:7681)
- `TTYD_PORT` - Set the port for ttyd server (default: 7681)

## Output

Test results will create:
- Screenshots in `test/e2e/screenshots/`
- Videos in `test/e2e/videos/`
- Error screenshots on test failure

## Troubleshooting

1. If ttyd fails to start, ensure the s9s binary is built
2. For "device not configured" errors, ensure ttyd is running
3. Check that the port is not already in use: `lsof -i :7681`