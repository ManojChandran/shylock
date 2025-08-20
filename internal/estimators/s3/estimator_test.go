package s3

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

	if estimator.SupportedResourceType() != "S3" {
		t.Errorf("expected resource type 'S3', got '%s'", estimator.SupportedResourceType())
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
			name: "valid S3 resource",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
					"sizeGB":       100,
				},
			},
			expectError: false,
		},
		{
			name: "wrong resource type",
			resource: models.ResourceSpec{
				Type:   "EC2",
				Name:   "instance",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "missing storageClass",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"sizeGB": 100,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid storageClass",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "INVALID_CLASS",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid sizeGB",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
					"sizeGB":       -10,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid requestsPerMonth",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass":     "STANDARD",
					"requestsPerMonth": -100,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid region",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "invalid-region",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
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
	// Create mock pricing products
	storageProduct := interfaces.PricingProduct{
		SKU:           "S3STORAGE123",
		ProductFamily: "Storage",
		ServiceCode:   "AmazonS3",
		Attributes: map[string]string{
			"storageClass": "STANDARD",
			"location":     "US East (N. Virginia)",
			"usageType":    "TimedStorage-ByteHrs",
		},
		Terms: map[string]interface{}{
			"OnDemand": map[string]interface{}{
				"S3STORAGE123.JRTCKXETXF": map[string]interface{}{
					"priceDimensions": map[string]interface{}{
						"S3STORAGE123.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
							"pricePerUnit": map[string]interface{}{
								"USD": "0.023", // $0.023 per GB-month
							},
						},
					},
				},
			},
		},
	}

	requestProduct := interfaces.PricingProduct{
		SKU:           "S3REQUEST123",
		ProductFamily: "API Request",
		ServiceCode:   "AmazonS3",
		Attributes: map[string]string{
			"storageClass": "STANDARD",
			"location":     "US East (N. Virginia)",
			"usageType":    "Requests-Tier1",
		},
		Terms: map[string]interface{}{
			"OnDemand": map[string]interface{}{
				"S3REQUEST123.JRTCKXETXF": map[string]interface{}{
					"priceDimensions": map[string]interface{}{
						"S3REQUEST123.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
							"pricePerUnit": map[string]interface{}{
								"USD": "0.0004", // $0.0004 per 1000 requests
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
		expectedMonthly float64 // Expected monthly cost for validation
	}{
		{
			name: "valid S3 storage only",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
					"sizeGB":       100,
				},
			},
			mockProducts:    []interfaces.PricingProduct{storageProduct},
			expectError:     false,
			expectedMonthly: 0.023 * 100, // $0.023 per GB-month * 100 GB
		},
		{
			name: "S3 with requests",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "api-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass":     "STANDARD",
					"sizeGB":           50,
					"requestsPerMonth": 10000,
				},
			},
			mockProducts: []interfaces.PricingProduct{storageProduct, requestProduct},
			expectError:  false,
			// Storage: $0.023 * 50 = $1.15, Requests: (10000/1000) * $0.0004 = $0.004
			expectedMonthly: (0.023 * 50) + ((10000.0 / 1000.0) * 0.0004),
		},
		{
			name: "default size (1 GB)",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "small-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
				},
			},
			mockProducts:    []interfaces.PricingProduct{storageProduct},
			expectError:     false,
			expectedMonthly: 0.023 * 1, // Default 1 GB
		},
		{
			name: "invalid resource",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "invalid-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "INVALID_CLASS",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "API failure",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
				},
			},
			shouldFailAPI: true,
			expectError:   true,
			errorType:     errors.APIErrorType,
		},
		{
			name: "no products found",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
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

				// Validate cost calculations (with tolerance for floating point)
				tolerance := 0.0001
				if tt.expectedMonthly > 0 && abs(estimate.MonthlyCost-tt.expectedMonthly) > tolerance {
					t.Errorf("expected monthly cost %.6f, got %.6f", tt.expectedMonthly, estimate.MonthlyCost)
				}

				// Validate metadata
				if estimate.ResourceName != tt.resource.Name {
					t.Errorf("expected resource name '%s', got '%s'", tt.resource.Name, estimate.ResourceName)
				}
				if estimate.ResourceType != "S3" {
					t.Errorf("expected resource type 'S3', got '%s'", estimate.ResourceType)
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
				requiredDetails := []string{"storageClass", "sizeGB", "requestsPerMonth", "storagePrice", "storageSKU"}
				for _, detail := range requiredDetails {
					if _, exists := estimate.Details[detail]; !exists {
						t.Errorf("expected detail '%s' but not found", detail)
					}
				}
			}
		})
	}
}

func TestStorageClassValidation(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		storageClass string
		valid        bool
	}{
		{"STANDARD", true},
		{"STANDARD_IA", true},
		{"ONEZONE_IA", true},
		{"GLACIER", true},
		{"DEEP_ARCHIVE", true},
		{"INTELLIGENT_TIERING", true},
		{"REDUCED_REDUNDANCY", true},
		{"INVALID_CLASS", false},
		{"", false},
		{"standard", false}, // Case sensitive
	}

	for _, tt := range tests {
		t.Run(tt.storageClass, func(t *testing.T) {
			result := estimator.isValidStorageClass(tt.storageClass)
			if result != tt.valid {
				t.Errorf("expected %v for storage class '%s', got %v", tt.valid, tt.storageClass, result)
			}
		})
	}
}

func TestUsageTypeDetection(t *testing.T) {
	estimator := &Estimator{}

	storageUsageTypes := []string{
		"TimedStorage-ByteHrs",
		"StandardStorage",
		"IA-Storage",
		"GlacierStorage",
	}

	requestUsageTypes := []string{
		"Requests-Tier1",
		"Requests-Tier2",
		"GetRequests",
		"PutRequests",
	}

	for _, usageType := range storageUsageTypes {
		t.Run("storage_"+usageType, func(t *testing.T) {
			if !estimator.isStorageUsageType(usageType) {
				t.Errorf("expected '%s' to be detected as storage usage type", usageType)
			}
		})
	}

	for _, usageType := range requestUsageTypes {
		t.Run("request_"+usageType, func(t *testing.T) {
			if !estimator.isRequestUsageType(usageType) {
				t.Errorf("expected '%s' to be detected as request usage type", usageType)
			}
		})
	}
}

func TestGetSupportedStorageClasses(t *testing.T) {
	estimator := &Estimator{}
	classes := estimator.GetSupportedStorageClasses()

	if len(classes) == 0 {
		t.Error("expected storage classes but got none")
	}

	// Check for common classes
	expectedClasses := []string{"STANDARD", "STANDARD_IA", "GLACIER", "DEEP_ARCHIVE"}
	for _, expected := range expectedClasses {
		found := false
		for _, class := range classes {
			if class == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected storage class '%s' to be in supported classes", expected)
		}
	}
}

func TestGetStorageClassDescription(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		storageClass string
		expectDesc   bool
	}{
		{"STANDARD", true},
		{"STANDARD_IA", true},
		{"GLACIER", true},
		{"DEEP_ARCHIVE", true},
		{"INVALID_CLASS", false},
	}

	for _, tt := range tests {
		t.Run(tt.storageClass, func(t *testing.T) {
			desc := estimator.GetStorageClassDescription(tt.storageClass)

			if tt.expectDesc {
				if desc == "Unknown storage class" {
					t.Errorf("expected description for '%s' but got unknown", tt.storageClass)
				}
			} else {
				if desc != "Unknown storage class" {
					t.Errorf("expected unknown description for '%s' but got '%s'", tt.storageClass, desc)
				}
			}
		})
	}
}

func TestGetTypicalUseCases(t *testing.T) {
	estimator := &Estimator{}

	tests := []string{"STANDARD", "STANDARD_IA", "GLACIER", "DEEP_ARCHIVE", "INTELLIGENT_TIERING"}

	for _, storageClass := range tests {
		t.Run(storageClass, func(t *testing.T) {
			useCases := estimator.GetTypicalUseCases(storageClass)

			if len(useCases) == 0 {
				t.Errorf("expected use cases for '%s' but got none", storageClass)
			}
		})
	}

	// Test invalid class
	useCases := estimator.GetTypicalUseCases("INVALID")
	if len(useCases) == 0 {
		t.Error("expected default use cases for invalid class")
	}
}

func TestCostEstimateDetails(t *testing.T) {
	storageProduct := interfaces.PricingProduct{
		SKU:           "S3TEST123",
		ProductFamily: "Storage",
		ServiceCode:   "AmazonS3",
		Attributes: map[string]string{
			"storageClass": "STANDARD",
			"usageType":    "TimedStorage-ByteHrs",
		},
		Terms: map[string]interface{}{
			"OnDemand": map[string]interface{}{
				"S3TEST123.JRTCKXETXF": map[string]interface{}{
					"priceDimensions": map[string]interface{}{
						"S3TEST123.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
							"pricePerUnit": map[string]interface{}{
								"USD": "0.023",
							},
						},
					},
				},
			},
		},
	}

	mockClient := &MockAWSClient{
		products: []interfaces.PricingProduct{storageProduct},
	}
	estimator := NewEstimator(mockClient)

	resource := models.ResourceSpec{
		Type:   "S3",
		Name:   "test-bucket",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"storageClass":     "STANDARD",
			"sizeGB":           500,
			"requestsPerMonth": 25000,
		},
	}

	estimate, err := estimator.EstimateCost(context.Background(), resource)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check assumptions
	expectedAssumptions := []string{
		"Storage costs calculated",
		"Pricing based on standard S3 rates",
		"Request costs calculated for 25000 requests",
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
		"storageClass":     "STANDARD",
		"sizeGB":           "500",
		"requestsPerMonth": "25000",
		"storageSKU":       "S3TEST123",
		"productFamily":    "Storage",
	}

	for key, expectedValue := range expectedDetails {
		if actualValue, exists := estimate.Details[key]; !exists {
			t.Errorf("expected detail '%s' but not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("expected detail '%s' to be '%s', got '%s'", key, expectedValue, actualValue)
		}
	}

	// Check that storage price detail exists
	if storagePrice, exists := estimate.Details["storagePrice"]; !exists {
		t.Error("expected storagePrice detail but not found")
	} else if storagePrice != "$0.023000/GB-month" {
		t.Errorf("expected storagePrice '$0.023000/GB-month', got '%s'", storagePrice)
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

// Benchmark tests
func BenchmarkEstimateCost(b *testing.B) {
	storageProduct := interfaces.PricingProduct{
		SKU: "S3TEST123",
		Terms: map[string]interface{}{
			"OnDemand": map[string]interface{}{
				"S3TEST123.JRTCKXETXF": map[string]interface{}{
					"priceDimensions": map[string]interface{}{
						"S3TEST123.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
							"pricePerUnit": map[string]interface{}{
								"USD": "0.023",
							},
						},
					},
				},
			},
		},
	}

	mockClient := &MockAWSClient{
		products: []interfaces.PricingProduct{storageProduct},
	}
	estimator := NewEstimator(mockClient)

	resource := models.ResourceSpec{
		Type:   "S3",
		Name:   "benchmark-bucket",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"storageClass": "STANDARD",
			"sizeGB":       100,
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
		Type:   "S3",
		Name:   "benchmark-bucket",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"storageClass": "STANDARD",
			"sizeGB":       100,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = estimator.ValidateResource(resource)
	}
}
