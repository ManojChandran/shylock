package estimators

import (
	"context"
	"testing"
	"time"

	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// MockEstimator for testing
type MockEstimator struct {
	resourceType    string
	shouldFailCost  bool
	shouldFailValid bool
	costEstimate    *models.CostEstimate
}

func (m *MockEstimator) SupportedResourceType() string {
	return m.resourceType
}

func (m *MockEstimator) ValidateResource(resource models.ResourceSpec) error {
	if m.shouldFailValid {
		return errors.ValidationError("mock validation failure")
	}
	return nil
}

func (m *MockEstimator) EstimateCost(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error) {
	if m.shouldFailCost {
		return nil, errors.APIError("mock cost estimation failure")
	}

	if m.costEstimate != nil {
		// Return a copy with the resource name
		estimate := *m.costEstimate
		estimate.ResourceName = resource.Name
		estimate.ResourceType = resource.Type
		return &estimate, nil
	}

	// Default mock estimate
	return &models.CostEstimate{
		ResourceName: resource.Name,
		ResourceType: resource.Type,
		Region:       resource.Region,
		HourlyCost:   1.0,
		DailyCost:    24.0,
		MonthlyCost:  720.0,
		Currency:     "USD",
		Timestamp:    time.Now(),
	}, nil
}

// MockAWSClient for testing
type MockAWSClient struct{}

func (m *MockAWSClient) GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]interfaces.PricingProduct, error) {
	return []interfaces.PricingProduct{}, nil
}

func (m *MockAWSClient) DescribeServices(ctx context.Context) ([]interfaces.ServiceInfo, error) {
	return []interfaces.ServiceInfo{}, nil
}

func (m *MockAWSClient) GetRegions(ctx context.Context, serviceCode string) ([]string, error) {
	return []string{"us-east-1"}, nil
}

func TestNewFactory(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	if factory == nil {
		t.Error("expected factory but got nil")
	}

	// Check that built-in estimators are registered
	supportedTypes := factory.GetSupportedResourceTypes()
	expectedTypes := []string{"ALB", "EC2", "Lambda", "RDS", "S3"}

	if len(supportedTypes) != len(expectedTypes) {
		t.Errorf("expected %d supported types, got %d", len(expectedTypes), len(supportedTypes))
	}

	for _, expectedType := range expectedTypes {
		found := false
		for _, supportedType := range supportedTypes {
			if supportedType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected resource type '%s' to be supported", expectedType)
		}
	}
}

func TestRegisterEstimator(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	tests := []struct {
		name         string
		resourceType string
		estimator    interfaces.ResourceEstimator
		expectError  bool
		errorType    errors.ErrorType
	}{
		{
			name:         "valid estimator",
			resourceType: "RDS",
			estimator:    &MockEstimator{resourceType: "RDS"},
			expectError:  false,
		},
		{
			name:         "empty resource type",
			resourceType: "",
			estimator:    &MockEstimator{resourceType: "RDS"},
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
		{
			name:         "nil estimator",
			resourceType: "RDS",
			estimator:    nil,
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
		{
			name:         "resource type mismatch",
			resourceType: "RDS",
			estimator:    &MockEstimator{resourceType: "Lambda"},
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.RegisterEstimator(tt.resourceType, tt.estimator)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorType != "" && !errors.IsErrorType(err, tt.errorType) {
					t.Errorf("expected error type %s, got %s", tt.errorType, errors.GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Verify the estimator was registered
				estimator, err := factory.GetEstimator(tt.resourceType)
				if err != nil {
					t.Errorf("failed to get registered estimator: %v", err)
				}
				if estimator != tt.estimator {
					t.Error("registered estimator does not match")
				}
			}
		})
	}
}

func TestGetEstimator(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	tests := []struct {
		name         string
		resourceType string
		expectError  bool
		errorType    errors.ErrorType
	}{
		{
			name:         "existing estimator",
			resourceType: "EC2",
			expectError:  false,
		},
		{
			name:         "non-existing estimator",
			resourceType: "DynamoDB",
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
		{
			name:         "empty resource type",
			resourceType: "",
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimator, err := factory.GetEstimator(tt.resourceType)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorType != "" && !errors.IsErrorType(err, tt.errorType) {
					t.Errorf("expected error type %s, got %s", tt.errorType, errors.GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if estimator == nil {
					t.Error("expected estimator but got nil")
				}
			}
		})
	}
}

func TestEstimateResource(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	// Register a mock estimator
	mockEstimator := &MockEstimator{resourceType: "TEST"}
	factory.RegisterEstimator("TEST", mockEstimator)

	tests := []struct {
		name        string
		resource    models.ResourceSpec
		setupMock   func(*MockEstimator)
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name: "successful estimation",
			resource: models.ResourceSpec{
				Type:   "TEST",
				Name:   "test-resource",
				Region: "us-east-1",
			},
			setupMock:   func(m *MockEstimator) { m.shouldFailCost = false },
			expectError: false,
		},
		{
			name: "unsupported resource type",
			resource: models.ResourceSpec{
				Type:   "UNSUPPORTED",
				Name:   "test-resource",
				Region: "us-east-1",
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "estimation failure",
			resource: models.ResourceSpec{
				Type:   "TEST",
				Name:   "test-resource",
				Region: "us-east-1",
			},
			setupMock:   func(m *MockEstimator) { m.shouldFailCost = true },
			expectError: true,
			errorType:   errors.APIErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(mockEstimator)
			}

			estimate, err := factory.EstimateResource(context.Background(), tt.resource)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorType != "" && !errors.IsErrorType(err, tt.errorType) {
					t.Errorf("expected error type %s, got %s", tt.errorType, errors.GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if estimate == nil {
					t.Error("expected estimate but got nil")
				}
				if estimate.ResourceName != tt.resource.Name {
					t.Errorf("expected resource name '%s', got '%s'", tt.resource.Name, estimate.ResourceName)
				}
			}
		})
	}
}

func TestEstimateFromConfig(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	// Register mock estimators
	mockEstimator1 := &MockEstimator{
		resourceType: "TEST1",
		costEstimate: &models.CostEstimate{
			HourlyCost:  1.0,
			DailyCost:   24.0,
			MonthlyCost: 720.0,
			Currency:    "USD",
		},
	}
	mockEstimator2 := &MockEstimator{
		resourceType: "TEST2",
		costEstimate: &models.CostEstimate{
			HourlyCost:  2.0,
			DailyCost:   48.0,
			MonthlyCost: 1440.0,
			Currency:    "USD",
		},
	}

	factory.RegisterEstimator("TEST1", mockEstimator1)
	factory.RegisterEstimator("TEST2", mockEstimator2)

	tests := []struct {
		name            string
		config          *models.EstimationConfig
		setupMocks      func()
		expectError     bool
		errorType       errors.ErrorType
		expectedHourly  float64
		expectedDaily   float64
		expectedMonthly float64
	}{
		{
			name: "successful estimation",
			config: &models.EstimationConfig{
				Version: "1.0",
				Resources: []models.ResourceSpec{
					{Type: "TEST1", Name: "resource1", Region: "us-east-1"},
					{Type: "TEST2", Name: "resource2", Region: "us-east-1"},
				},
			},
			setupMocks: func() {
				mockEstimator1.shouldFailCost = false
				mockEstimator2.shouldFailCost = false
			},
			expectError:     false,
			expectedHourly:  3.0,    // 1.0 + 2.0
			expectedDaily:   72.0,   // 24.0 + 48.0
			expectedMonthly: 2160.0, // 720.0 + 1440.0
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "empty resources",
			config: &models.EstimationConfig{
				Version:   "1.0",
				Resources: []models.ResourceSpec{},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "partial failure",
			config: &models.EstimationConfig{
				Version: "1.0",
				Resources: []models.ResourceSpec{
					{Type: "TEST1", Name: "resource1", Region: "us-east-1"},
					{Type: "TEST2", Name: "resource2", Region: "us-east-1"},
				},
			},
			setupMocks: func() {
				mockEstimator1.shouldFailCost = false
				mockEstimator2.shouldFailCost = true // This one fails
			},
			expectError:     false, // Should succeed with partial results
			expectedHourly:  1.0,   // Only first resource
			expectedDaily:   24.0,
			expectedMonthly: 720.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMocks != nil {
				tt.setupMocks()
			}

			result, err := factory.EstimateFromConfig(context.Background(), tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorType != "" && !errors.IsErrorType(err, tt.errorType) {
					t.Errorf("expected error type %s, got %s", tt.errorType, errors.GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result == nil {
					t.Error("expected result but got nil")
					return
				}

				// Check totals
				tolerance := 0.0001
				if abs(result.TotalHourlyCost-tt.expectedHourly) > tolerance {
					t.Errorf("expected hourly cost %.2f, got %.2f", tt.expectedHourly, result.TotalHourlyCost)
				}
				if abs(result.TotalDailyCost-tt.expectedDaily) > tolerance {
					t.Errorf("expected daily cost %.2f, got %.2f", tt.expectedDaily, result.TotalDailyCost)
				}
				if abs(result.TotalMonthlyCost-tt.expectedMonthly) > tolerance {
					t.Errorf("expected monthly cost %.2f, got %.2f", tt.expectedMonthly, result.TotalMonthlyCost)
				}

				// Check metadata
				if result.Currency != "USD" {
					t.Errorf("expected currency 'USD', got '%s'", result.Currency)
				}
				if result.GeneratedAt.IsZero() {
					t.Error("expected GeneratedAt to be set")
				}
			}
		})
	}
}

func TestValidateResource(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	mockEstimator := &MockEstimator{resourceType: "TEST"}
	factory.RegisterEstimator("TEST", mockEstimator)

	tests := []struct {
		name        string
		resource    models.ResourceSpec
		setupMock   func(*MockEstimator)
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name: "successful validation",
			resource: models.ResourceSpec{
				Type:   "TEST",
				Name:   "test-resource",
				Region: "us-east-1",
			},
			setupMock:   func(m *MockEstimator) { m.shouldFailValid = false },
			expectError: false,
		},
		{
			name: "validation failure",
			resource: models.ResourceSpec{
				Type:   "TEST",
				Name:   "test-resource",
				Region: "us-east-1",
			},
			setupMock:   func(m *MockEstimator) { m.shouldFailValid = true },
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "unsupported resource type",
			resource: models.ResourceSpec{
				Type:   "UNSUPPORTED",
				Name:   "test-resource",
				Region: "us-east-1",
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(mockEstimator)
			}

			err := factory.ValidateResource(tt.resource)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorType != "" && !errors.IsErrorType(err, tt.errorType) {
					t.Errorf("expected error type %s, got %s", tt.errorType, errors.GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	mockEstimator := &MockEstimator{resourceType: "TEST"}
	factory.RegisterEstimator("TEST", mockEstimator)

	tests := []struct {
		name        string
		config      *models.EstimationConfig
		setupMock   func(*MockEstimator)
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name: "valid config",
			config: &models.EstimationConfig{
				Version: "1.0",
				Resources: []models.ResourceSpec{
					{Type: "TEST", Name: "resource1", Region: "us-east-1", Properties: map[string]interface{}{"key": "value"}},
				},
			},
			setupMock:   func(m *MockEstimator) { m.shouldFailValid = false },
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid config structure",
			config: &models.EstimationConfig{
				Version:   "", // Missing version
				Resources: []models.ResourceSpec{},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "resource validation failure",
			config: &models.EstimationConfig{
				Version: "1.0",
				Resources: []models.ResourceSpec{
					{Type: "TEST", Name: "resource1", Region: "us-east-1", Properties: map[string]interface{}{"key": "value"}},
				},
			},
			setupMock:   func(m *MockEstimator) { m.shouldFailValid = true },
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setupMock != nil {
				tt.setupMock(mockEstimator)
			}

			err := factory.ValidateConfig(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				if tt.errorType != "" && !errors.IsErrorType(err, tt.errorType) {
					t.Errorf("expected error type %s, got %s", tt.errorType, errors.GetErrorType(err))
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGetEstimatorInfo(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	tests := []struct {
		name         string
		resourceType string
		expectError  bool
	}{
		{
			name:         "EC2 estimator info",
			resourceType: "EC2",
			expectError:  false,
		},
		{
			name:         "S3 estimator info",
			resourceType: "S3",
			expectError:  false,
		},
		{
			name:         "Lambda estimator info",
			resourceType: "Lambda",
			expectError:  false,
		},
		{
			name:         "unsupported estimator",
			resourceType: "DynamoDB",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := factory.GetEstimatorInfo(tt.resourceType)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if info == nil {
					t.Error("expected info but got nil")
					return
				}

				// Check basic info
				if resourceType, exists := info["resourceType"]; !exists || resourceType != tt.resourceType {
					t.Errorf("expected resourceType '%s', got '%v'", tt.resourceType, resourceType)
				}

				// Check type-specific info
				switch tt.resourceType {
				case "EC2":
					if _, exists := info["supportedInstanceFamilies"]; !exists {
						t.Error("expected supportedInstanceFamilies for EC2")
					}
					if _, exists := info["commonInstanceTypes"]; !exists {
						t.Error("expected commonInstanceTypes for EC2")
					}
				case "S3":
					if _, exists := info["supportedStorageClasses"]; !exists {
						t.Error("expected supportedStorageClasses for S3")
					}
					if _, exists := info["storageClassDescriptions"]; !exists {
						t.Error("expected storageClassDescriptions for S3")
					}
				}
			}
		})
	}
}

func TestGetAllEstimatorInfo(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	allInfo := factory.GetAllEstimatorInfo()

	if len(allInfo) == 0 {
		t.Error("expected estimator info but got none")
	}

	// Check that we have info for built-in estimators
	expectedTypes := []string{"EC2", "S3"}
	for _, expectedType := range expectedTypes {
		if _, exists := allInfo[expectedType]; !exists {
			t.Errorf("expected info for '%s' estimator", expectedType)
		}
	}
}

func TestEstimateResourcesConcurrently(t *testing.T) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	// Register mock estimators
	mockEstimator1 := &MockEstimator{resourceType: "TEST1"}
	mockEstimator2 := &MockEstimator{resourceType: "TEST2"}
	factory.RegisterEstimator("TEST1", mockEstimator1)
	factory.RegisterEstimator("TEST2", mockEstimator2)

	resources := []models.ResourceSpec{
		{Type: "TEST1", Name: "resource1", Region: "us-east-1"},
		{Type: "TEST2", Name: "resource2", Region: "us-east-1"},
		{Type: "TEST1", Name: "resource3", Region: "us-east-1"},
	}

	estimates, errs := factory.EstimateResourcesConcurrently(context.Background(), resources)

	if len(estimates) != len(resources) {
		t.Errorf("expected %d estimates, got %d", len(resources), len(estimates))
	}

	if len(errs) != len(resources) {
		t.Errorf("expected %d error slots, got %d", len(resources), len(errs))
	}

	// Check that estimates have correct resource names
	for i, estimate := range estimates {
		if estimate.ResourceName != resources[i].Name {
			t.Errorf("estimate %d: expected name '%s', got '%s'", i, resources[i].Name, estimate.ResourceName)
		}
	}
}

// Helper function
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Benchmark tests
func BenchmarkEstimateResource(b *testing.B) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	resource := models.ResourceSpec{
		Type:   "EC2",
		Name:   "benchmark-instance",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"instanceType": "t3.micro",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = factory.EstimateResource(context.Background(), resource)
	}
}

func BenchmarkValidateResource(b *testing.B) {
	mockClient := &MockAWSClient{}
	factory := NewFactory(mockClient)

	resource := models.ResourceSpec{
		Type:   "EC2",
		Name:   "benchmark-instance",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"instanceType": "t3.micro",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = factory.ValidateResource(resource)
	}
}
