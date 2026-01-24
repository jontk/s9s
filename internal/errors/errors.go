// Package errors provides structured error types and handling for s9s.
//nolint:revive // var-naming: Package name is intentional for error type organization
package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// ErrorType represents the type of error
type ErrorType string

const (
	// ErrorTypeUnknown is for unknown errors
	ErrorTypeUnknown ErrorType = "unknown"

	// ErrorTypeValidation is for validation errors
	ErrorTypeValidation ErrorType = "validation"

	// ErrorTypeNotFound is for resource not found errors
	ErrorTypeNotFound ErrorType = "not_found"

	// ErrorTypeConflict is for conflict errors
	ErrorTypeConflict ErrorType = "conflict"

	// ErrorTypeInternal is for internal server errors
	ErrorTypeInternal ErrorType = "internal"

	// ErrorTypeTimeout is for timeout errors
	ErrorTypeTimeout ErrorType = "timeout"

	// ErrorTypeAuthentication is for authentication errors
	ErrorTypeAuthentication ErrorType = "authentication"

	// ErrorTypeAuthorization is for authorization errors
	ErrorTypeAuthorization ErrorType = "authorization"

	// ErrorTypeNetwork is for network-related errors
	ErrorTypeNetwork ErrorType = "network"

	// ErrorTypeConfiguration is for configuration errors
	ErrorTypeConfiguration ErrorType = "configuration"
)

// S9sError represents a structured error with additional context
type S9sError struct {
	Type    ErrorType
	Message string
	Cause   error
	Context map[string]interface{}
	Stack   []StackFrame
}

// StackFrame represents a single frame in the call stack
type StackFrame struct {
	Function string
	File     string
	Line     int
}

// Error implements the error interface
func (e *S9sError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap returns the underlying error
func (e *S9sError) Unwrap() error {
	return e.Cause
}

// Is checks if the error is of a specific type
func (e *S9sError) Is(target error) bool {
	if target == nil {
		return false
	}

	var targetErr *S9sError
	if errors.As(target, &targetErr) {
		return e.Type == targetErr.Type
	}

	return errors.Is(e.Cause, target)
}

// WithContext adds context to the error
func (e *S9sError) WithContext(key string, value interface{}) *S9sError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// New creates a new S9sError
func New(errType ErrorType, message string) *S9sError {
	return &S9sError{
		Type:    errType,
		Message: message,
		Stack:   captureStack(2),
	}
}

// Newf creates a new S9sError with formatted message
func Newf(errType ErrorType, format string, args ...interface{}) *S9sError {
	return &S9sError{
		Type:    errType,
		Message: fmt.Sprintf(format, args...),
		Stack:   captureStack(2),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, message string) *S9sError {
	if err == nil {
		return nil
	}

	// If already a S9sError, preserve the original stack
	var s9sErr *S9sError
	if errors.As(err, &s9sErr) {
		return &S9sError{
			Type:    errType,
			Message: message,
			Cause:   err,
			Context: s9sErr.Context,
			Stack:   s9sErr.Stack,
		}
	}

	return &S9sError{
		Type:    errType,
		Message: message,
		Cause:   err,
		Stack:   captureStack(2),
	}
}

// Wrapf wraps an existing error with formatted message
func Wrapf(err error, errType ErrorType, format string, args ...interface{}) *S9sError {
	if err == nil {
		return nil
	}

	return Wrap(err, errType, fmt.Sprintf(format, args...))
}

// captureStack captures the current call stack
func captureStack(skip int) []StackFrame {
	var frames []StackFrame

	for i := skip; ; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		fn := runtime.FuncForPC(pc)
		if fn == nil {
			continue
		}

		// Skip runtime and testing functions
		fnName := fn.Name()
		if strings.Contains(fnName, "runtime.") ||
			strings.Contains(fnName, "testing.") {
			continue
		}

		frames = append(frames, StackFrame{
			Function: fnName,
			File:     file,
			Line:     line,
		})

		// Limit stack depth
		if len(frames) >= 10 {
			break
		}
	}

	return frames
}

// IsType checks if an error is of a specific type
func IsType(err error, errType ErrorType) bool {
	var s9sErr *S9sError
	if errors.As(err, &s9sErr) {
		return s9sErr.Type == errType
	}
	return false
}

// GetType returns the error type
func GetType(err error) ErrorType {
	var s9sErr *S9sError
	if errors.As(err, &s9sErr) {
		return s9sErr.Type
	}
	return ErrorTypeUnknown
}

// GetContext returns the error context
func GetContext(err error) map[string]interface{} {
	var s9sErr *S9sError
	if errors.As(err, &s9sErr) {
		return s9sErr.Context
	}
	return nil
}

// Common error constructors

// NotFound creates a not found error
func NotFound(resource string) *S9sError {
	return Newf(ErrorTypeNotFound, "%s not found", resource)
}

// NotFoundf creates a not found error with formatted message
func NotFoundf(format string, args ...interface{}) *S9sError {
	return Newf(ErrorTypeNotFound, format, args...)
}

// Invalid creates a validation error
func Invalid(field, reason string) *S9sError {
	err := Newf(ErrorTypeValidation, "invalid %s: %s", field, reason)
	return err.WithContext("field", field).WithContext("reason", reason)
}

// Invalidf creates a validation error with formatted message
func Invalidf(format string, args ...interface{}) *S9sError {
	return Newf(ErrorTypeValidation, format, args...)
}

// Internal creates an internal error
func Internal(message string) *S9sError {
	return New(ErrorTypeInternal, message)
}

// Internalf creates an internal error with formatted message
func Internalf(format string, args ...interface{}) *S9sError {
	return Newf(ErrorTypeInternal, format, args...)
}

// Timeout creates a timeout error
func Timeout(operation string) *S9sError {
	return Newf(ErrorTypeTimeout, "%s timed out", operation)
}

// Unauthorized creates an authentication error
func Unauthorized(message string) *S9sError {
	return New(ErrorTypeAuthentication, message)
}

// Forbidden creates an authorization error
func Forbidden(message string) *S9sError {
	return New(ErrorTypeAuthorization, message)
}

// Network creates a network error
func Network(message string) *S9sError {
	return New(ErrorTypeNetwork, message)
}

// Config creates a configuration error
func Config(message string) *S9sError {
	return New(ErrorTypeConfiguration, message)
}

// Configf creates a configuration error with formatted message
func Configf(format string, args ...interface{}) *S9sError {
	return Newf(ErrorTypeConfiguration, format, args...)
}

// Domain-specific error constructors for s9s

// SlurmConnection creates an error for SLURM connection failures
func SlurmConnection(message string) *S9sError {
	err := New(ErrorTypeNetwork, message)
	return err.WithContext("component", "slurm").WithContext("operation", "connect")
}

// SlurmConnectionf creates a SLURM connection error with formatted message
func SlurmConnectionf(format string, args ...interface{}) *S9sError {
	err := Newf(ErrorTypeNetwork, format, args...)
	return err.WithContext("component", "slurm").WithContext("operation", "connect")
}

// SlurmAPI wraps a SLURM API error with operation context
func SlurmAPI(operation string, err error) *S9sError {
	wrapped := Wrapf(err, ErrorTypeInternal, "SLURM API error during %s", operation)
	return wrapped.WithContext("component", "slurm").WithContext("operation", operation)
}

// SlurmAPIf creates a SLURM API error with formatted message
func SlurmAPIf(operation, format string, args ...interface{}) *S9sError {
	err := Newf(ErrorTypeInternal, format, args...)
	return err.WithContext("component", "slurm").WithContext("operation", operation)
}

// ConfigLoad wraps a configuration loading error
func ConfigLoad(path string, err error) *S9sError {
	wrapped := Wrapf(err, ErrorTypeConfiguration, "failed to load configuration from %s", path)
	return wrapped.WithContext("config_path", path)
}

// ConfigLoadf creates a configuration load error with formatted message
func ConfigLoadf(path, format string, args ...interface{}) *S9sError {
	err := Newf(ErrorTypeConfiguration, format, args...)
	return err.WithContext("config_path", path)
}

// DAOError creates a data access error
func DAOError(operation, resource string, err error) *S9sError {
	wrapped := Wrapf(err, ErrorTypeInternal, "data access error during %s of %s", operation, resource)
	return wrapped.WithContext("component", "dao").
		WithContext("operation", operation).
		WithContext("resource", resource)
}

// SSHError creates an SSH-related error
func SSHError(operation string, err error) *S9sError {
	wrapped := Wrapf(err, ErrorTypeNetwork, "SSH error during %s", operation)
	return wrapped.WithContext("component", "ssh").WithContext("operation", operation)
}

// SSHErrorf creates an SSH error with formatted message
func SSHErrorf(operation, format string, args ...interface{}) *S9sError {
	err := Newf(ErrorTypeNetwork, format, args...)
	return err.WithContext("component", "ssh").WithContext("operation", operation)
}

// PluginError creates a plugin-related error
func PluginError(pluginName, operation string, err error) *S9sError {
	wrapped := Wrapf(err, ErrorTypeInternal, "plugin '%s' error during %s", pluginName, operation)
	return wrapped.WithContext("component", "plugin").
		WithContext("plugin", pluginName).
		WithContext("operation", operation)
}

// AuthError creates an authentication/authorization error
func AuthError(operation string, err error) *S9sError {
	wrapped := Wrapf(err, ErrorTypeAuthentication, "authentication error during %s", operation)
	return wrapped.WithContext("component", "auth").WithContext("operation", operation)
}

// ViewError creates a view-related error
func ViewError(viewName, operation string, err error) *S9sError {
	wrapped := Wrapf(err, ErrorTypeInternal, "view '%s' error during %s", viewName, operation)
	return wrapped.WithContext("component", "view").
		WithContext("view", viewName).
		WithContext("operation", operation)
}
