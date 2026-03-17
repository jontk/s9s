# Mock Mode Guide

s9s includes environment variable gating for mock mode to prevent accidental use in production environments while maintaining flexibility for development and testing.

> **Note**: Mock mode is an internal development/testing feature. The `--mock`, `--no-mock` flags and `mock` subcommand are hidden from `s9s --help` to avoid confusing end users, but remain fully functional for developers.

## Security Features

### Environment Variable Gating

Mock mode requires the `S9S_ENABLE_MOCK` environment variable to be set. Any non-empty value enables mock mode:

```bash
# Any non-empty value works
export S9S_ENABLE_MOCK=1
export S9S_ENABLE_MOCK=true
export S9S_ENABLE_MOCK=yes
export S9S_ENABLE_MOCK=development
```

## Usage Examples

### Development Setup

```bash
# Set environment variable (permanent)
echo 'export S9S_ENABLE_MOCK=1' >> ~/.bashrc
source ~/.bashrc

# Use mock mode
s9s --mock
```

### One-time Usage

```bash
# Set for current session only
export S9S_ENABLE_MOCK=1
s9s --mock

# Or inline
S9S_ENABLE_MOCK=1 s9s --mock
```

### Check Mock Status

```bash
s9s mock status
```

## CLI Commands

### Primary Mock Usage

```bash
s9s --mock               # Use mock SLURM client (requires S9S_ENABLE_MOCK)
s9s --no-mock           # Force real SLURM client (override config)
```

### Mock Utilities

```bash
s9s mock status         # Show mock mode status and configuration
```

## Error Messages

### Mock Disabled

When mock mode is requested without the environment variable set, the following error is displayed:

```
mock mode disabled

To enable mock mode, set the S9S_ENABLE_MOCK environment variable:
  export S9S_ENABLE_MOCK=1
  s9s --mock
```

## Configuration File Behavior

If `useMockClient: true` is set in the config file but `S9S_ENABLE_MOCK` is not set:
- The application exits with an error message
- The error instructs the user to set the environment variable
- Setup suggestions are displayed via `SuggestMockSetup()`

## Implementation Details

### Mock Validator (`internal/mock/validator.go`)
- `IsMockEnabled()`: Returns true if `S9S_ENABLE_MOCK` environment variable is set to any non-empty value
- `ValidateMockUsage(useMockClient bool)`: Returns an error if mock is requested but not enabled via the environment variable
- `SuggestMockSetup()`: Prints setup instructions for enabling mock mode

### CLI Integration (`internal/cli/root.go`)
- Environment validation before application startup
- If validation fails, the error is displayed and `SuggestMockSetup()` is called
- The application returns an error and does not start

## Testing

Run the included test suite:
```bash
bash test_mock_validation.sh
```

Tests cover:
- Mock blocking without environment variable
- Mock allowance with valid environment variables

## Benefits

1. **Security**: Prevents accidental mock usage in production
2. **Flexibility**: Allows controlled testing/debugging when needed
3. **Simplicity**: Single binary works across all environments
4. **Visibility**: Clear status and configuration commands

## Related Guides

- [Configuration Reference](../reference/configuration.md)
- [Getting Started](../getting-started/quickstart.md)
