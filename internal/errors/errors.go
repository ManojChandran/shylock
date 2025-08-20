package errors

import (
	"fmt"
	"strings"
)

// ErrorType represents the category of error
type ErrorType string

const (
	// ConfigErrorType represents configuration-related errors
	ConfigErrorType ErrorType = "CONFIG"
	// AuthErrorType represents authentication-related errors
	AuthErrorType ErrorType = "AUTH"
	// APIErrorType represents AWS API-related errors
	APIErrorType ErrorType = "API"
	// NetworkErrorType represents network-related errors
	NetworkErrorType ErrorType = "NETWORK"
	// ValidationErrorType represents validation-related errors
	ValidationErrorType ErrorType = "VALIDATION"
	// FileErrorType represents file system-related errors
	FileErrorType ErrorType = "FILE"
)

// EstimationError is the base error type for all application errors
type EstimationError struct {
	Type        ErrorType
	Message     string
	Context     map[string]interface{}
	Cause       error
	Suggestions []string
}

// Error implements the error interface
func (e *EstimationError) Error() string {
	var parts []string

	// Add error type prefix
	parts = append(parts, fmt.Sprintf("[%s]", e.Type))

	// Add main message
	parts = append(parts, e.Message)

	// Add context if available
	if len(e.Context) > 0 {
		var contextParts []string
		for key, value := range e.Context {
			contextParts = append(contextParts, fmt.Sprintf("%s=%v", key, value))
		}
		parts = append(parts, fmt.Sprintf("(%s)", strings.Join(contextParts, ", ")))
	}

	// Add cause if available
	if e.Cause != nil {
		parts = append(parts, fmt.Sprintf("caused by: %v", e.Cause))
	}

	return strings.Join(parts, " ")
}

// Unwrap returns the underlying cause error
func (e *EstimationError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error type
func (e *EstimationError) Is(target error) bool {
	if targetErr, ok := target.(*EstimationError); ok {
		return e.Type == targetErr.Type
	}
	return false
}

// WithContext adds context information to the error
func (e *EstimationError) WithContext(key string, value interface{}) *EstimationError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithSuggestion adds a suggestion to help resolve the error
func (e *EstimationError) WithSuggestion(suggestion string) *EstimationError {
	if e.Suggestions == nil {
		e.Suggestions = make([]string, 0)
	}
	e.Suggestions = append(e.Suggestions, suggestion)
	return e
}

// GetSuggestions returns formatted suggestions for resolving the error
func (e *EstimationError) GetSuggestions() string {
	if len(e.Suggestions) == 0 {
		return ""
	}

	var result strings.Builder
	result.WriteString("Suggestions:\n")
	for i, suggestion := range e.Suggestions {
		result.WriteString(fmt.Sprintf("  %d. %s\n", i+1, suggestion))
	}
	return result.String()
}

// ConfigError creates a new configuration error
func ConfigError(message string) *EstimationError {
	return &EstimationError{
		Type:    ConfigErrorType,
		Message: message,
	}
}

// ConfigErrorf creates a new configuration error with formatting
func ConfigErrorf(format string, args ...interface{}) *EstimationError {
	return &EstimationError{
		Type:    ConfigErrorType,
		Message: fmt.Sprintf(format, args...),
	}
}

// ConfigErrorWithCause creates a new configuration error with a cause
func ConfigErrorWithCause(message string, cause error) *EstimationError {
	return &EstimationError{
		Type:    ConfigErrorType,
		Message: message,
		Cause:   cause,
	}
}

// AuthError creates a new authentication error
func AuthError(message string) *EstimationError {
	return &EstimationError{
		Type:    AuthErrorType,
		Message: message,
	}
}

// AuthErrorf creates a new authentication error with formatting
func AuthErrorf(format string, args ...interface{}) *EstimationError {
	return &EstimationError{
		Type:    AuthErrorType,
		Message: fmt.Sprintf(format, args...),
	}
}

// AuthErrorWithCause creates a new authentication error with a cause
func AuthErrorWithCause(message string, cause error) *EstimationError {
	return &EstimationError{
		Type:    AuthErrorType,
		Message: message,
		Cause:   cause,
	}
}

// APIError creates a new AWS API error
func APIError(message string) *EstimationError {
	return &EstimationError{
		Type:    APIErrorType,
		Message: message,
	}
}

// APIErrorf creates a new AWS API error with formatting
func APIErrorf(format string, args ...interface{}) *EstimationError {
	return &EstimationError{
		Type:    APIErrorType,
		Message: fmt.Sprintf(format, args...),
	}
}

// APIErrorWithCause creates a new AWS API error with a cause
func APIErrorWithCause(message string, cause error) *EstimationError {
	return &EstimationError{
		Type:    APIErrorType,
		Message: message,
		Cause:   cause,
	}
}

// NetworkError creates a new network error
func NetworkError(message string) *EstimationError {
	return &EstimationError{
		Type:    NetworkErrorType,
		Message: message,
	}
}

// NetworkErrorf creates a new network error with formatting
func NetworkErrorf(format string, args ...interface{}) *EstimationError {
	return &EstimationError{
		Type:    NetworkErrorType,
		Message: fmt.Sprintf(format, args...),
	}
}

// NetworkErrorWithCause creates a new network error with a cause
func NetworkErrorWithCause(message string, cause error) *EstimationError {
	return &EstimationError{
		Type:    NetworkErrorType,
		Message: message,
		Cause:   cause,
	}
}

// ValidationError creates a new validation error
func ValidationError(message string) *EstimationError {
	return &EstimationError{
		Type:    ValidationErrorType,
		Message: message,
	}
}

// ValidationErrorf creates a new validation error with formatting
func ValidationErrorf(format string, args ...interface{}) *EstimationError {
	return &EstimationError{
		Type:    ValidationErrorType,
		Message: fmt.Sprintf(format, args...),
	}
}

// ValidationErrorWithCause creates a new validation error with a cause
func ValidationErrorWithCause(message string, cause error) *EstimationError {
	return &EstimationError{
		Type:    ValidationErrorType,
		Message: message,
		Cause:   cause,
	}
}

// FileError creates a new file system error
func FileError(message string) *EstimationError {
	return &EstimationError{
		Type:    FileErrorType,
		Message: message,
	}
}

// FileErrorf creates a new file system error with formatting
func FileErrorf(format string, args ...interface{}) *EstimationError {
	return &EstimationError{
		Type:    FileErrorType,
		Message: fmt.Sprintf(format, args...),
	}
}

// FileErrorWithCause creates a new file system error with a cause
func FileErrorWithCause(message string, cause error) *EstimationError {
	return &EstimationError{
		Type:    FileErrorType,
		Message: message,
		Cause:   cause,
	}
}

// WrapError wraps an existing error with additional context
func WrapError(err error, errorType ErrorType, message string) *EstimationError {
	if err == nil {
		return nil
	}

	// If it's already an EstimationError, preserve the original type unless explicitly overridden
	if estimationErr, ok := err.(*EstimationError); ok && errorType == "" {
		return &EstimationError{
			Type:        estimationErr.Type,
			Message:     message,
			Context:     estimationErr.Context,
			Cause:       estimationErr,
			Suggestions: estimationErr.Suggestions,
		}
	}

	return &EstimationError{
		Type:    errorType,
		Message: message,
		Cause:   err,
	}
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType ErrorType) bool {
	if estimationErr, ok := err.(*EstimationError); ok {
		return estimationErr.Type == errorType
	}
	return false
}

// GetErrorType returns the error type of an error, or empty string if not an EstimationError
func GetErrorType(err error) ErrorType {
	if estimationErr, ok := err.(*EstimationError); ok {
		return estimationErr.Type
	}
	return ""
}

// FormatErrorForUser formats an error in a user-friendly way
func FormatErrorForUser(err error) string {
	if err == nil {
		return ""
	}

	estimationErr, ok := err.(*EstimationError)
	if !ok {
		return fmt.Sprintf("Error: %v", err)
	}

	var result strings.Builder

	// Add error message
	result.WriteString(fmt.Sprintf("Error: %s\n", estimationErr.Message))

	// Add context if available
	if len(estimationErr.Context) > 0 {
		result.WriteString("Details:\n")
		for key, value := range estimationErr.Context {
			result.WriteString(fmt.Sprintf("  %s: %v\n", key, value))
		}
	}

	// Add suggestions if available
	if len(estimationErr.Suggestions) > 0 {
		result.WriteString("\n")
		result.WriteString(estimationErr.GetSuggestions())
	}

	return result.String()
}

// GetExitCode returns an appropriate exit code based on error type
func GetExitCode(err error) int {
	if err == nil {
		return 0
	}

	estimationErr, ok := err.(*EstimationError)
	if !ok {
		return 1 // Generic error
	}

	switch estimationErr.Type {
	case ConfigErrorType:
		return 2
	case AuthErrorType:
		return 3
	case APIErrorType:
		return 4
	case NetworkErrorType:
		return 5
	case ValidationErrorType:
		return 6
	case FileErrorType:
		return 7
	default:
		return 1
	}
}
