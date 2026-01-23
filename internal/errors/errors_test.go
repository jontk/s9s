package errors

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	err := New(ErrorTypeValidation, "test error")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "test error", err.Message)
	assert.Nil(t, err.Cause)
	assert.NotEmpty(t, err.Stack)
}

func TestNewf(t *testing.T) {
	err := Newf(ErrorTypeInternal, "error %d: %s", 42, "test")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeInternal, err.Type)
	assert.Equal(t, "error 42: test", err.Message)
	assert.NotEmpty(t, err.Stack)
}

func TestS9sErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *S9sError
		expected string
	}{
		{
			name: "error without cause",
			err: &S9sError{
				Type:    ErrorTypeValidation,
				Message: "invalid input",
			},
			expected: "validation: invalid input",
		},
		{
			name: "error with cause",
			err: &S9sError{
				Type:    ErrorTypeInternal,
				Message: "processing failed",
				Cause:   fmt.Errorf("underlying error"),
			},
			expected: "internal: processing failed: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Error())
		})
	}
}

func TestS9sErrorUnwrap(t *testing.T) {
	cause := fmt.Errorf("underlying error")
	err := &S9sError{
		Type:    ErrorTypeInternal,
		Message: "wrapped error",
		Cause:   cause,
	}

	assert.Equal(t, cause, err.Unwrap())
}

func TestS9sErrorIs(t *testing.T) {
	baseErr := New(ErrorTypeValidation, "validation error")
	wrappedErr := Wrap(baseErr, ErrorTypeInternal, "internal error")

	tests := []struct {
		name     string
		err      *S9sError
		target   error
		expected bool
	}{
		{
			name:     "nil target returns false",
			err:      baseErr,
			target:   nil,
			expected: false,
		},
		{
			name:     "same type returns true",
			err:      baseErr,
			target:   New(ErrorTypeValidation, "another error"),
			expected: true,
		},
		{
			name:     "different type returns false",
			err:      baseErr,
			target:   New(ErrorTypeNotFound, "not found"),
			expected: false,
		},
		{
			name:     "wrapped error with same type as target",
			err:      wrappedErr,
			target:   New(ErrorTypeInternal, "test"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.err.Is(tt.target))
		})
	}
}

func TestS9sErrorWithContext(t *testing.T) {
	err := New(ErrorTypeValidation, "test error")

	err = err.WithContext("field", "username").WithContext("value", 42)

	require.NotNil(t, err.Context)
	assert.Equal(t, "username", err.Context["field"])
	assert.Equal(t, 42, err.Context["value"])
}

func TestWrap(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedNil  bool
		checkContext bool
	}{
		{
			name:        "wrapping nil returns nil",
			err:         nil,
			expectedNil: true,
		},
		{
			name:        "wrapping standard error",
			err:         fmt.Errorf("standard error"),
			expectedNil: false,
		},
		{
			name:         "wrapping S9sError preserves context",
			err:          New(ErrorTypeValidation, "original").WithContext("key", "value"),
			expectedNil:  false,
			checkContext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := Wrap(tt.err, ErrorTypeInternal, "wrapped message")

			if tt.expectedNil {
				assert.Nil(t, wrapped)
			} else {
				require.NotNil(t, wrapped)
				assert.Equal(t, ErrorTypeInternal, wrapped.Type)
				assert.Equal(t, "wrapped message", wrapped.Message)
				assert.Equal(t, tt.err, wrapped.Cause)

				if tt.checkContext {
					require.NotNil(t, wrapped.Context)
					assert.Equal(t, "value", wrapped.Context["key"])
				}
			}
		})
	}
}

func TestWrapf(t *testing.T) {
	baseErr := fmt.Errorf("base error")
	wrapped := Wrapf(baseErr, ErrorTypeInternal, "wrapped: %s", "formatted")

	require.NotNil(t, wrapped)
	assert.Equal(t, ErrorTypeInternal, wrapped.Type)
	assert.Equal(t, "wrapped: formatted", wrapped.Message)
	assert.Equal(t, baseErr, wrapped.Cause)
}

func TestWrapfWithNil(t *testing.T) {
	wrapped := Wrapf(nil, ErrorTypeInternal, "message")
	assert.Nil(t, wrapped)
}

func TestIsType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		errType  ErrorType
		expected bool
	}{
		{
			name:     "S9sError with matching type",
			err:      New(ErrorTypeValidation, "test"),
			errType:  ErrorTypeValidation,
			expected: true,
		},
		{
			name:     "S9sError with non-matching type",
			err:      New(ErrorTypeValidation, "test"),
			errType:  ErrorTypeNotFound,
			expected: false,
		},
		{
			name:     "standard error returns false",
			err:      fmt.Errorf("standard error"),
			errType:  ErrorTypeValidation,
			expected: false,
		},
		{
			name:     "wrapped S9sError",
			err:      Wrap(New(ErrorTypeValidation, "test"), ErrorTypeInternal, "wrapped"),
			errType:  ErrorTypeInternal,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsType(tt.err, tt.errType))
		})
	}
}

func TestGetType(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorType
	}{
		{
			name:     "S9sError returns its type",
			err:      New(ErrorTypeValidation, "test"),
			expected: ErrorTypeValidation,
		},
		{
			name:     "standard error returns unknown",
			err:      fmt.Errorf("standard error"),
			expected: ErrorTypeUnknown,
		},
		{
			name:     "wrapped error returns wrapper type",
			err:      Wrap(New(ErrorTypeValidation, "test"), ErrorTypeInternal, "wrapped"),
			expected: ErrorTypeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, GetType(tt.err))
		})
	}
}

func TestGetContext(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		expected    map[string]interface{}
		expectEmpty bool
	}{
		{
			name:     "S9sError with context",
			err:      New(ErrorTypeValidation, "test").WithContext("key", "value"),
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:        "S9sError without context returns nil",
			err:         New(ErrorTypeValidation, "test"),
			expectEmpty: true,
		},
		{
			name:        "standard error returns nil",
			err:         fmt.Errorf("standard error"),
			expectEmpty: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := GetContext(tt.err)
			if tt.expectEmpty {
				assert.Nil(t, ctx)
			} else {
				assert.Equal(t, tt.expected, ctx)
			}
		})
	}
}

func TestNotFound(t *testing.T) {
	err := NotFound("resource")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeNotFound, err.Type)
	assert.Equal(t, "resource not found", err.Message)
}

func TestNotFoundf(t *testing.T) {
	err := NotFoundf("resource %s with id %d", "user", 42)

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeNotFound, err.Type)
	assert.Equal(t, "resource user with id 42", err.Message)
}

func TestInvalid(t *testing.T) {
	err := Invalid("username", "too short")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "invalid username: too short", err.Message)
	assert.Equal(t, "username", err.Context["field"])
	assert.Equal(t, "too short", err.Context["reason"])
}

func TestInvalidf(t *testing.T) {
	err := Invalidf("field %s: %s", "email", "invalid format")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeValidation, err.Type)
	assert.Equal(t, "field email: invalid format", err.Message)
}

func TestInternal(t *testing.T) {
	err := Internal("something went wrong")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeInternal, err.Type)
	assert.Equal(t, "something went wrong", err.Message)
}

func TestInternalf(t *testing.T) {
	err := Internalf("error code: %d", 500)

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeInternal, err.Type)
	assert.Equal(t, "error code: 500", err.Message)
}

func TestTimeout(t *testing.T) {
	err := Timeout("database query")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeTimeout, err.Type)
	assert.Equal(t, "database query timed out", err.Message)
}

func TestUnauthorized(t *testing.T) {
	err := Unauthorized("invalid credentials")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeAuthentication, err.Type)
	assert.Equal(t, "invalid credentials", err.Message)
}

func TestForbidden(t *testing.T) {
	err := Forbidden("insufficient permissions")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeAuthorization, err.Type)
	assert.Equal(t, "insufficient permissions", err.Message)
}

func TestNetwork(t *testing.T) {
	err := Network("connection refused")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeNetwork, err.Type)
	assert.Equal(t, "connection refused", err.Message)
}

func TestConfig(t *testing.T) {
	err := Config("missing required field")

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeConfiguration, err.Type)
	assert.Equal(t, "missing required field", err.Message)
}

func TestConfigf(t *testing.T) {
	err := Configf("invalid value for %s: %v", "port", 99999)

	require.NotNil(t, err)
	assert.Equal(t, ErrorTypeConfiguration, err.Type)
	assert.Equal(t, "invalid value for port: 99999", err.Message)
}

func TestErrorTypeConstants(t *testing.T) {
	// Verify all error type constants are defined
	assert.Equal(t, ErrorType("unknown"), ErrorTypeUnknown)
	assert.Equal(t, ErrorType("validation"), ErrorTypeValidation)
	assert.Equal(t, ErrorType("not_found"), ErrorTypeNotFound)
	assert.Equal(t, ErrorType("conflict"), ErrorTypeConflict)
	assert.Equal(t, ErrorType("internal"), ErrorTypeInternal)
	assert.Equal(t, ErrorType("timeout"), ErrorTypeTimeout)
	assert.Equal(t, ErrorType("authentication"), ErrorTypeAuthentication)
	assert.Equal(t, ErrorType("authorization"), ErrorTypeAuthorization)
	assert.Equal(t, ErrorType("network"), ErrorTypeNetwork)
	assert.Equal(t, ErrorType("configuration"), ErrorTypeConfiguration)
}

func TestStackFrameCapture(t *testing.T) {
	err := New(ErrorTypeInternal, "test")

	require.NotNil(t, err.Stack)
	assert.Greater(t, len(err.Stack), 0)

	// Check that stack frames have required fields
	if len(err.Stack) > 0 {
		frame := err.Stack[0]
		assert.NotEmpty(t, frame.Function)
		assert.NotEmpty(t, frame.File)
		assert.Greater(t, frame.Line, 0)
	}
}

func TestErrorUnwrapChain(t *testing.T) {
	baseErr := fmt.Errorf("base error")
	wrapped1 := Wrap(baseErr, ErrorTypeInternal, "wrapped once")
	wrapped2 := Wrap(wrapped1, ErrorTypeNetwork, "wrapped twice")

	// Test unwrap chain
	assert.True(t, errors.Is(wrapped2, baseErr))
	assert.True(t, errors.Is(wrapped2, wrapped1))

	// Test type checking through chain
	var s9sErr *S9sError
	assert.True(t, errors.As(wrapped2, &s9sErr))
	assert.Equal(t, ErrorTypeNetwork, s9sErr.Type)
}
