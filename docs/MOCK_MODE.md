# Mock Mode Security Implementation

## Overview

s9s includes environment variable gating for mock mode to prevent accidental use in production environments while maintaining flexibility for development and testing.

> **Note**: Mock mode is an internal development/testing feature. The `--mock`, `--no-mock` flags and `mock` subcommand are hidden from `s9s --help` to avoid confusing end users, but remain fully functional for developers.

## Security Features

### Environment Variable Gating
Mock mode requires `S9S_ENABLE_MOCK` environment variable to be set to specific values:

```bash
# Allowed values (case insensitive)
S9S_ENABLE_MOCK=development  # Recommended for development
S9S_ENABLE_MOCK=testing      # For testing environments  
S9S_ENABLE_MOCK=debug        # For debugging purposes
S9S_ENABLE_MOCK=dev          # Short form for development
S9S_ENABLE_MOCK=local        # For local usage
S9S_ENABLE_MOCK=true         # Generic enablement
```

### Production Environment Detection
The system automatically detects production environments by checking:
- `ENVIRONMENT=production`
- `NODE_ENV=production`
- `GO_ENV=production`
- `RAILS_ENV=production`

### Production Safety Features
When mock mode is requested in a production environment:
1. **Warning Display**: Shows prominent warning about mock usage
2. **Interactive Confirmation**: Requires explicit user confirmation
3. **Non-Interactive Protection**: Automatically denies in non-interactive terminals

## Usage Examples

### Development Setup
```bash
# Set environment variable (permanent)
echo 'export S9S_ENABLE_MOCK=development' >> ~/.bashrc
source ~/.bashrc

# Use mock mode
s9s --mock
```

### One-time Usage
```bash
# Set for current session only
export S9S_ENABLE_MOCK=development
s9s --mock

# Or inline
S9S_ENABLE_MOCK=development s9s --mock
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
```
❌ mock mode disabled

To enable mock mode, set one of these environment variables:
  S9S_ENABLE_MOCK=development  # For development
  S9S_ENABLE_MOCK=testing      # For testing
  S9S_ENABLE_MOCK=debug        # For debugging
  S9S_ENABLE_MOCK=true         # Generic enable

Example:
  export S9S_ENABLE_MOCK=development
  s9s --mock
```

### Production Warning
```
🚨 WARNING: Mock SLURM client enabled in production environment!
   This should only be used for debugging purposes.
   Mock mode provides simulated data, not real cluster information.

Are you sure you want to continue with mock mode in production? (yes/no):
```

## Configuration File Behavior

If `useMockClient: true` is set in the config file but `S9S_ENABLE_MOCK` is not set:
- Mock mode is automatically disabled
- Real SLURM client mode is used instead
- Warning message is displayed about the environment override

## Implementation Details

### Mock Validator (`internal/mock/validator.go`)
- `IsMockEnabled()`: Checks if mock mode is allowed via environment variables
- `IsProductionEnvironment()`: Detects production environment indicators
- `ValidateMockUsage()`: Main validation logic with user prompts
- `GetMockStatusMessage()`: User-friendly status messages

### CLI Integration (`internal/cli/root.go`)
- Environment validation before application startup
- Graceful fallback from config-enabled mock to real mode
- Clear error messages and setup suggestions

## Testing

Run the included test suite:
```bash
bash test_mock_validation.sh
```

Tests cover:
- Mock blocking without environment variable
- Mock allowance with valid environment variables
- Production environment detection and warnings
- Invalid environment variable rejection

## Benefits

1. **Security**: Prevents accidental mock usage in production
2. **Flexibility**: Allows controlled testing/debugging when needed
3. **Simplicity**: Single binary works across all environments
4. **Visibility**: Clear status and configuration commands
5. **Safety**: Multiple layers of protection for production environments