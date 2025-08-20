package ec2

import (
	"context"
	"testing"

	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// MockAWSClient for testing
type MockAWSClient struct {
	products       []interfaces.PricingProduct
	shouldFailGet  bool
	shouldFailDesc bool
}

func (m *MockAWSClient) GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]interfaces.PricingProduct, error) {
	if m.shouldFailGet {
		return nil, errors.APIError("mock API failure")
	}
	return m.products, nil
}

func (m *MockAWSClient) DescribeServices(ctx context.Context) ([]interfaces.ServiceInfo, error) {
	if m.shouldFailDesc {
		return nil, errors.APIError("mock API failure")
	}
	return []interfaces.ServiceInfo{}, nil
}

func (m *MockAWSClient) GetRegions(ctx context.Context, serviceCode string) ([]string, error) {
	return []string{"us-east-1", "us-west-2"}, nil
}

func TestNewEstimator(t *testing.T) {
	mockClient := &MockAWSClient{}
	estimator := NewEstimator(mockClient)

	if estimator == nil {
		t.Error("expected estimator but got nil")
	}

	if estimator.SupportedResourceType() != "EC2" {
		t.Errorf("expected resource type 'EC2', got '%s'", estimator.SupportedResourceType())
	}
}

func TestValidateResource(t *testing.T) {
	mockClient := &MockAWSClient{}
	estimator := NewEstimator(mockClient)

	tests := []struct {
		name        string
		resource    models.ResourceSpec
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name: "valid EC2 resource",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
					"count":        1,
				},
			},
			expectError: false,
		},
		{
			name: "wrong resource type",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "missing instanceType",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"count": 1,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid instanceType format",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "invalid-type",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid count",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
					"count":        0,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid region",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "invalid-region",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
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

func TestEstimateCost(t *testing.T) {
	// Create mock pricing product
	mockProduct := interfaces.PricingProduct{
		SKU:           "TEST123",
		ProductFamily: "Compute Instance",
		ServiceCode:   "AmazonEC2",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
			"location":     "US East (N. Virginia)",
			"tenancy":      "Shared",
		},
		Terms: map[string]interface{}{
			"OnDemand": map[string]interface{}{
				"TEST123.JRTCKXETXF": map[string]interface{}{
					"priceDimensions": map[string]interface{}{
						"TEST123.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
							"pricePerUnit": map[string]interface{}{
								"USD": "0.0116",
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name            string
		resource        models.ResourceSpec
		mockProducts    []interfaces.PricingProduct
		shouldFailAPI   bool
		expectError     bool
		errorType       errors.ErrorType
		expectedHourly  float64
		expectedDaily   float64
		expectedMonthly float64
	}{
		{
			name: "valid single instance",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
			mockProducts:    []interfaces.PricingProduct{mockProduct},
			expectError:     false,
			expectedHourly:  0.0116,
			expectedDaily:   0.0116 * 24,
			expectedMonthly: 0.0116 * 24 * 30,
		},
		{
			name: "multiple instances",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "web-servers",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
					"count":        3,
				},
			},
			mockProducts:    []interfaces.PricingProduct{mockProduct},
			expectError:     false,
			expectedHourly:  0.0116 * 3,
			expectedDaily:   0.0116 * 3 * 24,
			expectedMonthly: 0.0116 * 3 * 24 * 30,
		},
		{
			name: "with operating system",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "windows-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType":    "t3.micro",
					"operatingSystem": "Windows",
				},
			},
			mockProducts:   []interfaces.PricingProduct{mockProduct},
			expectError:    false,
			expectedHourly: 0.0116,
		},
		{
			name: "invalid resource",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "invalid-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "invalid-type",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "API failure",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
			shouldFailAPI: true,
			expectError:   true,
			errorType:     errors.APIErrorType,
		},
		{
			name: "no products found",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
			mockProducts: []interfaces.PricingProduct{}, // Empty
			expectError:  true,
			errorType:    errors.APIErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAWSClient{
				products:      tt.mockProducts,
				shouldFailGet: tt.shouldFailAPI,
			}
			estimator := NewEstimator(mockClient)

			estimate, err := estimator.EstimateCost(context.Background(), tt.resource)

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
					t.Error("expected cost estimate but got nil")
					return
				}

				// Validate cost calculations (with small tolerance for floating point)
				tolerance := 0.0001
				if abs(estimate.HourlyCost-tt.expectedHourly) > tolerance {
					t.Errorf("expected hourly cost %.4f, got %.4f", tt.expectedHourly, estimate.HourlyCost)
				}
				if tt.expectedDaily > 0 && abs(estimate.DailyCost-tt.expectedDaily) > tolerance {
					t.Errorf("expected daily cost %.4f, got %.4f", tt.expectedDaily, estimate.DailyCost)
				}
				if tt.expectedMonthly > 0 && abs(estimate.MonthlyCost-tt.expectedMonthly) > tolerance {
					t.Errorf("expected monthly cost %.4f, got %.4f", tt.expectedMonthly, estimate.MonthlyCost)
				}

				// Validate metadata
				if estimate.ResourceName != tt.resource.Name {
					t.Errorf("expected resource name '%s', got '%s'", tt.resource.Name, estimate.ResourceName)
				}
				if estimate.ResourceType != "EC2" {
					t.Errorf("expected resource type 'EC2', got '%s'", estimate.ResourceType)
				}
				if estimate.Currency != "USD" {
					t.Errorf("expected currency 'USD', got '%s'", estimate.Currency)
				}

				// Validate assumptions are present
				if len(estimate.Assumptions) == 0 {
					t.Error("expected assumptions but got none")
				}

				// Validate details are present
				if len(estimate.Details) == 0 {
					t.Error("expected details but got none")
				}

				// Check for required details
				requiredDetails := []string{"instanceType", "operatingSystem", "tenancy", "count", "pricePerInstance", "sku"}
				for _, detail := range requiredDetails {
					if _, exists := estimate.Details[detail]; !exists {
						t.Errorf("expected detail '%s' but not found", detail)
					}
				}
			}
		})
	}
}

func TestInstanceTypeValidation(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		instanceType string
		valid        bool
	}{
		{"t3.micro", true},
		{"t3a.small", true},
		{"m5.large", true},
		{"c5.xlarge", true},
		{"r5.2xlarge", true},
		{"m6i.4xlarge", true},
		{"p3.8xlarge", true},
		{"invalid", false},
		{"t3", false},
		{".micro", false},
		{"", false},
		{"t", false},
		{"t3.", false},
		{"123.micro", false},
		{"t3.micro.extra", true}, // This should be valid (some instance types have extra parts)
	}

	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			result := estimator.isValidInstanceType(tt.instanceType)
			if result != tt.valid {
				t.Errorf("expected %v for instance type '%s', got %v", tt.valid, tt.instanceType, result)
			}
		})
	}
}

func TestGetSupportedInstanceFamilies(t *testing.T) {
	estimator := &Estimator{}
	families := estimator.GetSupportedInstanceFamilies()

	if len(families) == 0 {
		t.Error("expected instance families but got none")
	}

	// Check for common families
	expectedFamilies := []string{"t3", "m5", "c5", "r5"}
	for _, expected := range expectedFamilies {
		found := false
		for _, family := range families {
			if family == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected instance family '%s' to be in supported families", expected)
		}
	}
}

func TestGetCommonInstanceTypes(t *testing.T) {
	estimator := &Estimator{}
	instanceTypes := estimator.GetCommonInstanceTypes()

	if len(instanceTypes) == 0 {
		t.Error("expected instance types but got none")
	}

	// Check for common types
	expectedTypes := []string{"t3.micro", "t3.small", "m5.large", "c5.large"}
	for _, expected := range expectedTypes {
		found := false
		for _, instanceType := range instanceTypes {
			if instanceType == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected instance type '%s' to be in common types", expected)
		}
	}

	// Validate all returned types
	estimator2 := &Estimator{}
	for _, instanceType := range instanceTypes {
		if !estimator2.isValidInstanceType(instanceType) {
			t.Errorf("common instance type '%s' failed validation", instanceType)
		}
	}
}

func TestCostEstimateDetails(t *testing.T) {
	mockProduct := interfaces.PricingProduct{
		SKU:           "TEST123",
		ProductFamily: "Compute Instance",
		ServiceCode:   "AmazonEC2",
		Attributes: map[string]string{
			"instanceType": "t3.micro",
			"tenancy":      "Shared",
		},
		Terms: map[string]interface{}{
			"OnDemand": map[string]interface{}{
				"TEST123.JRTCKXETXF": map[string]interface{}{
					"priceDimensions": map[string]interface{}{
						"TEST123.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
							"pricePerUnit": map[string]interface{}{
								"USD": "0.0116",
							},
						},
					},
				},
			},
		},
	}

	mockClient := &MockAWSClient{
		products: []interfaces.PricingProduct{mockProduct},
	}
	estimator := NewEstimator(mockClient)

	resource := models.ResourceSpec{
		Type:   "EC2",
		Name:   "test-server",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"instanceType":    "t3.micro",
			"count":           2,
			"operatingSystem": "Windows",
			"tenancy":         "Dedicated",
		},
	}

	estimate, err := estimator.EstimateCost(context.Background(), resource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check assumptions
	expectedAssumptions := []string{
		"24/7 usage assumed",
		"On-demand pricing",
		"Cost calculated for 2 instances",
		"Operating system: Windows",
		"Tenancy: Dedicated",
	}

	for _, expected := range expectedAssumptions {
		found := false
		for _, assumption := range estimate.Assumptions {
			if containsString(assumption, expected) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected assumption containing '%s' but not found in: %v", expected, estimate.Assumptions)
		}
	}

	// Check details
	expectedDetails := map[string]string{
		"instanceType":    "t3.micro",
		"operatingSystem": "Windows",
		"tenancy":         "Dedicated",
		"count":           "2",
		"sku":             "TEST123",
		"productFamily":   "Compute Instance",
	}

	for key, expectedValue := range expectedDetails {
		if actualValue, exists := estimate.Details[key]; !exists {
			t.Errorf("expected detail '%s' but not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected detail '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
		}
	}

	// Check that pricePerInstance detail exists and is reasonable
	if pricePerInstance, exists := estimate.Details["pricePerInstance"]; !exists {
		t.Error("expected pricePerInstance detail but not found")
	} else if pricePerInstance != "$0.0116/hour" {
		t.Errorf("expected pricePerInstance '$0.0116/hour', got '%s'", pricePerInstance)
	}
}

// Benchmark tests
func BenchmarkEstimateCost(b *testing.B) {
	mockProduct := interfaces.PricingProduct{
		SKU: "TEST123",
		Terms: map[string]interface{}{
			"OnDemand": map[string]interface{}{
				"TEST123.JRTCKXETXF": map[string]interface{}{
					"priceDimensions": map[string]interface{}{
						"TEST123.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
							"pricePerUnit": map[string]interface{}{
								"USD": "0.0116",
							},
						},
					},
				},
			},
		},
	}

	mockClient := &MockAWSClient{
		products: []interfaces.PricingProduct{mockProduct},
	}
	estimator := NewEstimator(mockClient)

	resource := models.ResourceSpec{
		Type:   "EC2",
		Name:   "benchmark-server",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"instanceType": "t3.micro",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = estimator.EstimateCost(context.Background(), resource)
	}
}

func BenchmarkValidateResource(b *testing.B) {
	mockClient := &MockAWSClient{}
	estimator := NewEstimator(mockClient)

	resource := models.ResourceSpec{
		Type:   "EC2",
		Name:   "benchmark-server",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"instanceType": "t3.micro",
			"count":        1,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = estimator.ValidateResource(resource)
	}
}

// Helper functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func containsString(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) &&
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}
