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