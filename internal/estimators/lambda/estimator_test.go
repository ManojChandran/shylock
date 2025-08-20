package lambda

import (
	"context"
	"fmt"
	"testing"

	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// MockAWSClient for testing
type MockAWSClient struct {
	products []interfaces.PricingProduct
	err      error
}

func (m *MockAWSClient) GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]interfaces.PricingProduct, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.products, nil
}

func (m *MockAWSClient) DescribeServices(ctx context.Context) ([]interfaces.ServiceInfo, error) {
	return []interfaces.ServiceInfo{}, nil
}

func (m *MockAWSClient) GetRegions(ctx context.Context, serviceCode string) ([]string, error) {
	return []string{"us-east-1", "us-west-2"}, nil
}

func TestLambdaEstimator_SupportedResourceType(t *testing.T) {
	estimator := NewEstimator(&MockAWSClient{})

	expected := "Lambda"
	actual := estimator.SupportedResourceType()

	if actual != expected {
		t.Errorf("Expected resource type %s, got %s", expected, actual)
	}
}

func TestLambdaEstimator_ValidateResource(t *testing.T) {
	estimator := NewEstimator(&MockAWSClient{})

	tests := []struct {
		name        string
		resource    models.ResourceSpec
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name: "valid Lambda function",
			resource: models.ResourceSpec{
				Type:   "Lambda",
				Name:   "api-handler",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"memoryMB":          512,
					"requestsPerMonth":  1000000,
					"averageDurationMs": 200,
					"architecture":      "arm64",
				},
			},
			expectError: false,
		},
		{
			name: "minimal valid configuration",
			resource: models.ResourceSpec{
				Type:   "Lambda",
				Name:   "minimal-function",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"memoryMB": 128,
				},
			},
			expectError: false,
		},
		{
			name: "wrong resource type",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "wrong-type",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"memoryMB": 512,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "missing memoryMB",
			resource: models.ResourceSpec{
				Type:       "Lambda",
				Name:       "no-memory",
				Region:     "us-east-1",
				Properties: map[string]interface{}{},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid memory size - too low",
			resource: models.ResourceSpec{
				Type:   "Lambda",
				Name:   "low-memory",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"memoryMB": 64,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid memory size - too high",
			resource: models.ResourceSpec{
				Type:   "Lambda",
				Name:   "high-memory",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"memoryMB": 20480,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid architecture",
			resource: models.ResourceSpec{
				Type:   "Lambda",
				Name:   "invalid-arch",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"memoryMB":     512,
					"architecture": "sparc",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "negative requests",
			resource: models.ResourceSpec{
				Type:   "Lambda",
				Name:   "negative-requests",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"memoryMB":         512,
					"requestsPerMonth": -1000,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := estimator.ValidateResource(tt.resource)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}

				if estimationErr, ok := err.(*errors.EstimationError); ok {
					if estimationErr.Type != tt.errorType {
						t.Errorf("Expected error type %s, got %s", tt.errorType, estimationErr.Type)
					}
				} else {
					t.Errorf("Expected EstimationError, got %T", err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestLambdaEstimator_GetSupportedMemorySizes(t *testing.T) {
	estimator := &Estimator{}

	sizes := estimator.GetSupportedMemorySizes()

	if len(sizes) == 0 {
		t.Errorf("Expected supported memory sizes but got none")
	}

	// Check that all returned sizes are valid
	for _, size := range sizes {
		if !estimator.isValidMemorySize(size) {
			t.Errorf("Invalid memory size returned: %d", size)
		}
	}

	// Check specific expected sizes
	expectedSizes := []int{128, 256, 512, 1024, 2048}
	for _, expectedSize := range expectedSizes {
		found := false
		for _, actualSize := range sizes {
			if actualSize == expectedSize {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected memory size %d not found in supported sizes", expectedSize)
		}
	}
}

func TestLambdaEstimator_GetSupportedArchitectures(t *testing.T) {
	estimator := &Estimator{}

	architectures := estimator.GetSupportedArchitectures()

	expectedArchitectures := []string{"x86_64", "arm64"}
	if len(architectures) != len(expectedArchitectures) {
		t.Errorf("Expected %d architectures, got %d", len(expectedArchitectures), len(architectures))
	}

	for _, expectedArch := range expectedArchitectures {
		found := false
		for _, actualArch := range architectures {
			if actualArch == expectedArch {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected architecture %s not found in supported architectures", expectedArch)
		}
	}
}

func TestLambdaEstimator_GetArchitectureDescription(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		architecture string
		expectEmpty  bool
	}{
		{"x86_64", false},
		{"arm64", false},
		{"unknown", false}, // Should return "Unknown architecture"
	}

	for _, tt := range tests {
		t.Run(tt.architecture, func(t *testing.T) {
			desc := estimator.GetArchitectureDescription(tt.architecture)

			if tt.expectEmpty && desc != "" {
				t.Errorf("Expected empty description, got %s", desc)
			}

			if !tt.expectEmpty && desc == "" {
				t.Errorf("Expected non-empty description, got empty")
			}
		})
	}
}

func TestLambdaEstimator_IsValidMemorySize(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		memoryMB int
		expected bool
	}{
		{128, true},
		{512, true},
		{1024, true},
		{10240, true},
		{64, false},    // Too low
		{20480, false}, // Too high
		{0, false},     // Invalid
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("memory_%d", tt.memoryMB), func(t *testing.T) {
			result := estimator.isValidMemorySize(tt.memoryMB)
			if result != tt.expected {
				t.Errorf("Expected %t for memory size %d, got %t", tt.expected, tt.memoryMB, result)
			}
		})
	}
}

func TestLambdaEstimator_IsValidArchitecture(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		architecture string
		expected     bool
	}{
		{"x86_64", true},
		{"arm64", true},
		{"sparc", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.architecture, func(t *testing.T) {
			result := estimator.isValidArchitecture(tt.architecture)
			if result != tt.expected {
				t.Errorf("Expected %t for architecture %s, got %t", tt.expected, tt.architecture, result)
			}
		})
	}
}

func TestLambdaEstimator_GetTypicalUseCases(t *testing.T) {
	estimator := &Estimator{}

	useCases := estimator.GetTypicalUseCases()

	if len(useCases) == 0 {
		t.Errorf("Expected typical use cases but got none")
	}

	// Check for some expected use cases
	expectedUseCases := []string{"API backends", "Event-driven", "Real-time"}
	for _, expectedUseCase := range expectedUseCases {
		found := false
		for _, actualUseCase := range useCases {
			if len(actualUseCase) > len(expectedUseCase) &&
				actualUseCase[:len(expectedUseCase)] == expectedUseCase {
				found = true
				break
			}
		}
		if !found {
			t.Logf("Expected use case pattern '%s' not found in: %v", expectedUseCase, useCases)
		}
	}
}
