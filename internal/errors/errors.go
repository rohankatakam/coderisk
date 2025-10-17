package errors

import (
	"fmt"
	"runtime"
	"strings"
)

// ErrorType represents the category of error
type ErrorType int

const (
	// Configuration errors - missing or invalid configuration
	ErrorTypeConfig ErrorType = iota
	// Validation errors - invalid input data
	ErrorTypeValidation
	// Database errors - database connection or query failures
	ErrorTypeDatabase
	// Network errors - network connectivity issues
	ErrorTypeNetwork
	// FileSystem errors - file I/O failures
	ErrorTypeFileSystem
	// External errors - external service failures
	ErrorTypeExternal
	// Internal errors - unexpected internal state
	ErrorTypeInternal
	// Security errors - security-related failures
	ErrorTypeSecurity
)

// Severity represents how critical an error is
type Severity int

const (
	// SeverityLow - can continue with degraded functionality
	SeverityLow Severity = iota
	// SeverityMedium - should be addressed but not fatal
	SeverityMedium
	// SeverityHigh - significant issue, may impact functionality
	SeverityHigh
	// SeverityCritical - must be addressed, stops execution
	SeverityCritical
)

// Error represents a structured error with context
type Error struct {
	Type       ErrorType
	Severity   Severity
	Message    string
	Cause      error
	Context    map[string]interface{}
	StackTrace string
	Timestamp  string
}

// Error implements the error interface
func (e *Error) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause
func (e *Error) Unwrap() error {
	return e.Cause
}

// WithContext adds context to the error
func (e *Error) WithContext(key string, value interface{}) *Error {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// Is checks if this error matches the target error type
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Type == t.Type
}

// IsFatal returns true if this error should stop execution
func (e *Error) IsFatal() bool {
	return e.Severity == SeverityCritical
}

// DetailedString returns a detailed error message with context
func (e *Error) DetailedString() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] [%s] %s\n",
		severityString(e.Severity),
		typeString(e.Type),
		e.Message))

	if e.Cause != nil {
		sb.WriteString(fmt.Sprintf("Caused by: %v\n", e.Cause))
	}

	if len(e.Context) > 0 {
		sb.WriteString("Context:\n")
		for k, v := range e.Context {
			sb.WriteString(fmt.Sprintf("  %s: %v\n", k, v))
		}
	}

	if e.StackTrace != "" {
		sb.WriteString(fmt.Sprintf("Stack trace:\n%s\n", e.StackTrace))
	}

	return sb.String()
}

func typeString(t ErrorType) string {
	switch t {
	case ErrorTypeConfig:
		return "CONFIG"
	case ErrorTypeValidation:
		return "VALIDATION"
	case ErrorTypeDatabase:
		return "DATABASE"
	case ErrorTypeNetwork:
		return "NETWORK"
	case ErrorTypeFileSystem:
		return "FILESYSTEM"
	case ErrorTypeExternal:
		return "EXTERNAL"
	case ErrorTypeInternal:
		return "INTERNAL"
	case ErrorTypeSecurity:
		return "SECURITY"
	default:
		return "UNKNOWN"
	}
}

func severityString(s Severity) string {
	switch s {
	case SeverityLow:
		return "LOW"
	case SeverityMedium:
		return "MEDIUM"
	case SeverityHigh:
		return "HIGH"
	case SeverityCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// captureStackTrace captures the current stack trace
func captureStackTrace(skip int) string {
	var sb strings.Builder
	for i := skip; i < skip+10; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			break
		}
		sb.WriteString(fmt.Sprintf("  %s:%d %s\n", file, line, fn.Name()))
	}
	return sb.String()
}

// New creates a new error with the given type, severity, and message
func New(errType ErrorType, severity Severity, message string) *Error {
	return &Error{
		Type:       errType,
		Severity:   severity,
		Message:    message,
		Context:    make(map[string]interface{}),
		StackTrace: captureStackTrace(2),
	}
}

// Wrap wraps an existing error with additional context
func Wrap(err error, errType ErrorType, severity Severity, message string) *Error {
	if err == nil {
		return nil
	}

	return &Error{
		Type:       errType,
		Severity:   severity,
		Message:    message,
		Cause:      err,
		Context:    make(map[string]interface{}),
		StackTrace: captureStackTrace(2),
	}
}

// Convenience constructors for common error types

// ConfigError creates a configuration error
func ConfigError(message string) *Error {
	return New(ErrorTypeConfig, SeverityCritical, message)
}

// ConfigErrorf creates a configuration error with formatting
func ConfigErrorf(format string, args ...interface{}) *Error {
	return New(ErrorTypeConfig, SeverityCritical, fmt.Sprintf(format, args...))
}

// ValidationError creates a validation error
func ValidationError(message string) *Error {
	return New(ErrorTypeValidation, SeverityHigh, message)
}

// ValidationErrorf creates a validation error with formatting
func ValidationErrorf(format string, args ...interface{}) *Error {
	return New(ErrorTypeValidation, SeverityHigh, fmt.Sprintf(format, args...))
}

// DatabaseError wraps a database error
func DatabaseError(err error, message string) *Error {
	return Wrap(err, ErrorTypeDatabase, SeverityCritical, message)
}

// DatabaseErrorf wraps a database error with formatting
func DatabaseErrorf(err error, format string, args ...interface{}) *Error {
	return Wrap(err, ErrorTypeDatabase, SeverityCritical, fmt.Sprintf(format, args...))
}

// NetworkError wraps a network error
func NetworkError(err error, message string) *Error {
	return Wrap(err, ErrorTypeNetwork, SeverityHigh, message)
}

// NetworkErrorf wraps a network error with formatting
func NetworkErrorf(err error, format string, args ...interface{}) *Error {
	return Wrap(err, ErrorTypeNetwork, SeverityHigh, fmt.Sprintf(format, args...))
}

// FileSystemError wraps a filesystem error
func FileSystemError(err error, message string) *Error {
	return Wrap(err, ErrorTypeFileSystem, SeverityHigh, message)
}

// FileSystemErrorf wraps a filesystem error with formatting
func FileSystemErrorf(err error, format string, args ...interface{}) *Error {
	return Wrap(err, ErrorTypeFileSystem, SeverityHigh, fmt.Sprintf(format, args...))
}

// ExternalError wraps an external service error
func ExternalError(err error, message string) *Error {
	return Wrap(err, ErrorTypeExternal, SeverityMedium, message)
}

// ExternalErrorf wraps an external service error with formatting
func ExternalErrorf(err error, format string, args ...interface{}) *Error {
	return Wrap(err, ErrorTypeExternal, SeverityMedium, fmt.Sprintf(format, args...))
}

// InternalError creates an internal error
func InternalError(message string) *Error {
	return New(ErrorTypeInternal, SeverityCritical, message)
}

// InternalErrorf creates an internal error with formatting
func InternalErrorf(format string, args ...interface{}) *Error {
	return New(ErrorTypeInternal, SeverityCritical, fmt.Sprintf(format, args...))
}

// SecurityError creates a security error
func SecurityError(message string) *Error {
	return New(ErrorTypeSecurity, SeverityCritical, message)
}

// SecurityErrorf creates a security error with formatting
func SecurityErrorf(format string, args ...interface{}) *Error {
	return New(ErrorTypeSecurity, SeverityCritical, fmt.Sprintf(format, args...))
}

// IsFatal checks if an error is fatal (should stop execution)
func IsFatal(err error) bool {
	if err == nil {
		return false
	}

	if e, ok := err.(*Error); ok {
		return e.IsFatal()
	}

	return false
}

// GetSeverity returns the severity of an error
func GetSeverity(err error) Severity {
	if err == nil {
		return SeverityLow
	}

	if e, ok := err.(*Error); ok {
		return e.Severity
	}

	return SeverityMedium
}

// GetType returns the type of an error
func GetType(err error) ErrorType {
	if err == nil {
		return ErrorTypeInternal
	}

	if e, ok := err.(*Error); ok {
		return e.Type
	}

	return ErrorTypeInternal
}
