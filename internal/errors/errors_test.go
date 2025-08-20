package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestEstimationErrorCreation(t *testing.T) {
	tests := []struct {
		name        string
		createError func() *EstimationError
		expectType  ErrorType
		expectMsg   string
	}{
		{
			name: "config error",
			createError: func() *EstimationError {
				return ConfigError("invalid configuration")
			},
			expectType: ConfigErrorType,
			expectMsg:  "invalid configuration",
		},
		{
			name: "config error with formatting",
			createError: func() *EstimationError {
				return ConfigErrorf("invalid value: %s", "test")
			},
			expectType: ConfigErrorType,
			expectMsg:  "invalid value: test",
		},
		{
			name: "auth error",
			createError: func() *EstimationError {
				return AuthError("authentication failed")
			},
			expectType: AuthErrorType,
			expectMsg:  "authentication failed",
		},
		{
			name: "API error",
			createError: func() *EstimationError {
				return APIError("API call failed")
			},
			expectType: APIErrorType,
			expectMsg:  "API call failed",
		},
		{
			name: "network error",
			createError: func() *EstimationError {
				return NetworkError("connection timeout")
			},
			expectType: NetworkErrorType,
			expectMsg:  "connection timeout",
		},
		{
			name: "validation error",
			createError: func() *EstimationError {
				return ValidationError("validation failed")
			},
			expectType: ValidationErrorType,
			expectMsg:  "validation failed",
		},
		{
			name: "file error",
			createError: func() *EstimationError {
				return FileError("file not found")
			},
			expectType: FileErrorType,
			expectMsg:  "file not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createError()

			if err.Type != tt.expectType {
				t.Errorf("expected type %s, got %s", tt.expectType, err.Type)
			}

			if err.Message != tt.expectMsg {
				t.Errorf("expected message '%s', got '%s'", tt.expectMsg, err.Message)
			}
		})
	}
}

func TestEstimationErrorWithCause(t *testing.T) {
	originalErr := errors.New("original error")

	tests := []struct {
		name        string
		createError func() *EstimationError
		expectType  ErrorType
	}{
		{
			name: "config error with cause",
			createError: func() *EstimationError {
				return ConfigErrorWithCause("config parsing failed", originalErr)
			},
			expectType: ConfigErrorType,
		},
		{
			name: "auth error with cause",
			createError: func() *EstimationError {
				return AuthErrorWithCause("authentication failed", originalErr)
			},
			expectType: AuthErrorType,
		},
		{
			name: "API error with cause",
			createError: func() *EstimationError {
				return APIErrorWithCause("API call failed", originalErr)
			},
			expectType: APIErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createError()

			if err.Type != tt.expectType {
				t.Errorf("expected type %s, got %s", tt.expectType, err.Type)
			}

			if err.Cause != originalErr {
				t.Errorf("expected cause to be original error")
			}

			if err.Unwrap() != originalErr {
				t.Errorf("Unwrap() should return the original error")
			}
		})
	}
}

func TestEstimationErrorContext(t *testing.T) {
	err := ConfigError("test error")

	// Add context
	err.WithContext("file", "config.json")
	err.WithContext("line", 42)

	if len(err.Context) != 2 {
		t.Errorf("expected 2 context items, got %d", len(err.Context))
	}

	if err.Context["file"] != "config.json" {
		t.Errorf("expected file context to be 'config.json', got %v", err.Context["file"])
	}

	if err.Context["line"] != 42 {
		t.Errorf("expected line context to be 42, got %v", err.Context["line"])
	}

	// Check error string includes context
	errorStr := err.Error()
	if !strings.Contains(errorStr, "file=config.json") {
		t.Errorf("error string should contain context: %s", errorStr)
	}
}

func TestEstimationErrorSuggestions(t *testing.T) {
	err := ValidationError("invalid input")

	// Add suggestions
	err.WithSuggestion("Check the input format")
	err.WithSuggestion("Refer to the documentation")

	if len(err.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(err.Suggestions))
	}

	suggestions := err.GetSuggestions()
	if !strings.Contains(suggestions, "1. Check the input format") {
		t.Errorf("suggestions should contain first suggestion: %s", suggestions)
	}

	if !strings.Contains(suggestions, "2. Refer to the documentation") {
		t.Errorf("suggestions should contain second suggestion: %s", suggestions)
	}
}

func TestErrorFormatting(t *testing.T) {
	tests := []struct {
		name        string
		createError func() *EstimationError
		expectParts []string
	}{
		{
			name: "simple error",
			createError: func() *EstimationError {
				return ConfigError("test message")
			},
			expectParts: []string{"[CONFIG]", "test message"},
		},
		{
			name: "error with context",
			createError: func() *EstimationError {
				return ConfigError("test message").WithContext("file", "test.json")
			},
			expectParts: []string{"[CONFIG]", "test message", "file=test.json"},
		},
		{
			name: "error with cause",
			createError: func() *EstimationError {
				return ConfigErrorWithCause("test message", errors.New("original"))
			},
			expectParts: []string{"[CONFIG]", "test message", "caused by: original"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createError()
			errorStr := err.Error()

			for _, part := range tt.expectParts {
				if !strings.Contains(errorStr, part) {
					t.Errorf("error string should contain '%s': %s", part, errorStr)
				}
			}
		})
	}
}

func TestErrorTypeChecking(t *testing.T) {
	configErr := ConfigError("config error")
	authErr := AuthError("auth error")

	// Test IsErrorType function
	if !IsErrorType(configErr, ConfigErrorType) {
		t.Error("IsErrorType should return true for matching type")
	}

	if IsErrorType(configErr, AuthErrorType) {
		t.Error("IsErrorType should return false for non-matching type")
	}

	if IsErrorType(errors.New("regular error"), ConfigErrorType) {
		t.Error("IsErrorType should return false for non-EstimationError")
	}

	// Test GetErrorType function
	if GetErrorType(configErr) != ConfigErrorType {
		t.Error("GetErrorType should return correct type")
	}

	if GetErrorType(errors.New("regular error")) != "" {
		t.Error("GetErrorType should return empty string for non-EstimationError")
	}

	// Test Is method
	if !configErr.Is(ConfigError("")) {
		t.Error("Is method should return true for same error type")
	}

	if configErr.Is(authErr) {
		t.Error("Is method should return false for different error type")
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")

	// Test wrapping regular error
	wrappedErr := WrapError(originalErr, ConfigErrorType, "wrapped message")
	if wrappedErr.Type != ConfigErrorType {
		t.Errorf("expected type %s, got %s", ConfigErrorType, wrappedErr.Type)
	}

	if wrappedErr.Message != "wrapped message" {
		t.Errorf("expected message 'wrapped message', got '%s'", wrappedErr.Message)
	}

	if wrappedErr.Cause != originalErr {
		t.Error("wrapped error should have original error as cause")
	}

	// Test wrapping EstimationError
	estimationErr := ConfigError("config error")
	wrappedEstimationErr := WrapError(estimationErr, "", "new message")
	if wrappedEstimationErr.Type != ConfigErrorType {
		t.Error("wrapped EstimationError should preserve original type")
	}

	// Test wrapping nil error
	nilWrapped := WrapError(nil, ConfigErrorType, "message")
	if nilWrapped != nil {
		t.Error("wrapping nil error should return nil")
	}
}

func TestFormatErrorForUser(t *testing.T) {
	// Test nil error
	if FormatErrorForUser(nil) != "" {
		t.Error("formatting nil error should return empty string")
	}

	// Test regular error
	regularErr := errors.New("regular error")
	formatted := FormatErrorForUser(regularErr)
	if !strings.Contains(formatted, "regular error") {
		t.Errorf("formatted error should contain original message: %s", formatted)
	}

	// Test EstimationError with context and suggestions
	err := ConfigError("test error").
		WithContext("file", "config.json").
		WithSuggestion("Check the file format")

	formatted = FormatErrorForUser(err)

	expectedParts := []string{
		"Error: test error",
		"Details:",
		"file: config.json",
		"Suggestions:",
		"1. Check the file format",
	}

	for _, part := range expectedParts {
		if !strings.Contains(formatted, part) {
			t.Errorf("formatted error should contain '%s': %s", part, formatted)
		}
	}
}

func TestGetExitCode(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode int
	}{
		{
			name:         "nil error",
			err:          nil,
			expectedCode: 0,
		},
		{
			name:         "regular error",
			err:          errors.New("regular error"),
			expectedCode: 1,
		},
		{
			name:         "config error",
			err:          ConfigError("config error"),
			expectedCode: 2,
		},
		{
			name:         "auth error",
			err:          AuthError("auth error"),
			expectedCode: 3,
		},
		{
			name:         "API error",
			err:          APIError("API error"),
			expectedCode: 4,
		},
		{
			name:         "network error",
			err:          NetworkError("network error"),
			expectedCode: 5,
		},
		{
			name:         "validation error",
			err:          ValidationError("validation error"),
			expectedCode: 6,
		},
		{
			name:         "file error",
			err:          FileError("file error"),
			expectedCode: 7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := GetExitCode(tt.err)
			if code != tt.expectedCode {
				t.Errorf("expected exit code %d, got %d", tt.expectedCode, code)
			}
		})
	}
}

func TestErrorChaining(t *testing.T) {
	// Create a chain of errors
	originalErr := errors.New("database connection failed")
	apiErr := APIErrorWithCause("failed to fetch pricing data", originalErr)
	configErr := ConfigErrorWithCause("configuration processing failed", apiErr)

	// Test error unwrapping
	if configErr.Unwrap() != apiErr {
		t.Error("first unwrap should return API error")
	}

	if apiErr.Unwrap() != originalErr {
		t.Error("second unwrap should return original error")
	}

	// Test error string contains all information
	errorStr := configErr.Error()
	if !strings.Contains(errorStr, "configuration processing failed") {
		t.Error("error string should contain top-level message")
	}

	if !strings.Contains(errorStr, "failed to fetch pricing data") {
		t.Error("error string should contain cause message")
	}
}

func TestErrorBuilderPattern(t *testing.T) {
	// Test method chaining
	err := ConfigError("test error").
		WithContext("file", "test.json").
		WithContext("line", 10).
		WithSuggestion("Check syntax").
		WithSuggestion("Validate JSON")

	if len(err.Context) != 2 {
		t.Errorf("expected 2 context items, got %d", len(err.Context))
	}

	if len(err.Suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(err.Suggestions))
	}

	// Verify the error can be used in error interfaces
	var testErr error = err
	if testErr.Error() == "" {
		t.Error("error should implement error interface correctly")
	}
}
