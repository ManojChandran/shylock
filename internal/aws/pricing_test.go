package aws

import (
	"context"
	"testing"

	"shylock/internal/errors"
	"shylock/internal/interfaces"
)

// MockAWSClient implements the AWSPricingClient interface for testing
type MockAWSClient struct {
	products       []interfaces.PricingProduct
	services       []interfaces.ServiceInfo
	regions        []string
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
	return m.services, nil
}

func (m *MockAWSClient) GetRegions(ctx context.Context, serviceCode string) ([]string, error) {
	return m.regions, nil
}

func TestNewPricingService(t *testing.T) {
	mockClient := &MockAWSClient{}
	service := NewPricingService(mockClient)

	if service == nil {
		t.Error("expected pricing service but got nil")
	}

	if service.client != mockClient {
		t.Error("pricing service should use the provided client")
	}
}

func TestGetEC2Pricing(t *testing.T) {
	tests := []struct {
		name            string
		instanceType    string
		region          string
		operatingSystem string
		mockProducts    []interfaces.PricingProduct
		shouldFail      bool
		expectError     bool
		errorType       errors.ErrorType
	}{
		{
			name:            "valid EC2 request",
			instanceType:    "t3.micro",
			region:          "us-east-1",
			operatingSystem: "Linux",
			mockProducts: []interfaces.PricingProduct{
				{
					SKU:           "TEST123",
					ProductFamily: "Compute Instance",
					ServiceCode:   "AmazonEC2",
					Attributes: map[string]string{
						"instanceType": "t3.micro",
						"location":     "US East (N. Virginia)",
					},
				},
			},
			expectError: false,
		},
		{
			name:         "empty instance type",
			instanceType: "",
			region:       "us-east-1",
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
		{
			name:         "empty region",
			instanceType: "t3.micro",
			region:       "",
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
		{
			name:         "unsupported region",
			instanceType: "t3.micro",
			region:       "invalid-region",
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
		{
			name:            "API failure",
			instanceType:    "t3.micro",
			region:          "us-east-1",
			operatingSystem: "Linux",
			shouldFail:      true,
			expectError:     true,
			errorType:       errors.APIErrorType,
		},
		{
			name:            "no products found",
			instanceType:    "t3.micro",
			region:          "us-east-1",
			operatingSystem: "Linux",
			mockProducts:    []interfaces.PricingProduct{}, // Empty results
			expectError:     true,
			errorType:       errors.APIErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAWSClient{
				products:      tt.mockProducts,
				shouldFailGet: tt.shouldFail,
			}
			service := NewPricingService(mockClient)

			products, err := service.GetEC2Pricing(context.Background(), tt.instanceType, tt.region, tt.operatingSystem)

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
				if len(products) == 0 {
					t.Error("expected products but got none")
				}
			}
		})
	}
}

func TestGetS3Pricing(t *testing.T) {
	tests := []struct {
		name         string
		storageClass string
		region       string
		mockProducts []interfaces.PricingProduct
		shouldFail   bool
		expectError  bool
		errorType    errors.ErrorType
	}{
		{
			name:         "valid S3 request",
			storageClass: "STANDARD",
			region:       "us-east-1",
			mockProducts: []interfaces.PricingProduct{
				{
					SKU:           "S3TEST123",
					ProductFamily: "Storage",
					ServiceCode:   "AmazonS3",
					Attributes: map[string]string{
						"storageClass": "STANDARD",
						"location":     "US East (N. Virginia)",
					},
				},
			},
			expectError: false,
		},
		{
			name:         "empty storage class",
			storageClass: "",
			region:       "us-east-1",
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
		{
			name:         "empty region",
			storageClass: "STANDARD",
			region:       "",
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
		{
			name:         "unsupported region",
			storageClass: "STANDARD",
			region:       "invalid-region",
			expectError:  true,
			errorType:    errors.ValidationErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &MockAWSClient{
				products:      tt.mockProducts,
				shouldFailGet: tt.shouldFail,
			}
			service := NewPricingService(mockClient)

			products, err := service.GetS3Pricing(context.Background(), tt.storageClass, tt.region)

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
				if len(products) == 0 {
					t.Error("expected products but got none")
				}
			}
		})
	}
}

func TestExtractHourlyPrice(t *testing.T) {
	tests := []struct {
		name          string
		product       interfaces.PricingProduct
		expectedPrice float64
		expectError   bool
	}{
		{
			name: "valid pricing structure",
			product: interfaces.PricingProduct{
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
			},
			expectedPrice: 0.0116,
			expectError:   false,
		},
		{
			name: "no terms",
			product: interfaces.PricingProduct{
				SKU:   "TEST123",
				Terms: nil,
			},
			expectError: true,
		},
		{
			name: "no OnDemand terms",
			product: interfaces.PricingProduct{
				SKU: "TEST123",
				Terms: map[string]interface{}{
					"Reserved": map[string]interface{}{},
				},
			},
			expectError: true,
		},
		{
			name: "invalid price format",
			product: interfaces.PricingProduct{
				SKU: "TEST123",
				Terms: map[string]interface{}{
					"OnDemand": map[string]interface{}{
						"TEST123.JRTCKXETXF": map[string]interface{}{
							"priceDimensions": map[string]interface{}{
								"TEST123.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
									"pricePerUnit": map[string]interface{}{
										"USD": "invalid-price",
									},
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &PricingService{}

			price, err := service.ExtractHourlyPrice(tt.product)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if price != tt.expectedPrice {
					t.Errorf("expected price %.4f, got %.4f", tt.expectedPrice, price)
				}
			}
		})
	}
}

func TestRegionToLocation(t *testing.T) {
	service := &PricingService{}

	tests := []struct {
		region   string
		expected string
	}{
		{"us-east-1", "US East (N. Virginia)"},
		{"us-west-2", "US West (Oregon)"},
		{"eu-west-1", "Europe (Ireland)"},
		{"ap-southeast-1", "Asia Pacific (Singapore)"},
		{"invalid-region", ""}, // Should return empty for unknown regions
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result := service.regionToLocation(tt.region)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestValidateRegion(t *testing.T) {
	service := &PricingService{}

	tests := []struct {
		region      string
		expectError bool
	}{
		{"us-east-1", false},
		{"us-west-2", false},
		{"eu-west-1", false},
		{"invalid-region", true},
		{"", true},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			err := service.ValidateRegion(tt.region)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for region '%s' but got none", tt.region)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for region '%s': %v", tt.region, err)
				}
			}
		})
	}
}

func TestGetSupportedRegions(t *testing.T) {
	service := &PricingService{}

	regions := service.GetSupportedRegions()

	if len(regions) == 0 {
		t.Error("expected supported regions but got none")
	}

	// Check that common regions are included
	expectedRegions := []string{"us-east-1", "us-west-2", "eu-west-1"}
	for _, expected := range expectedRegions {
		found := false
		for _, region := range regions {
			if region == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected region '%s' to be in supported regions", expected)
		}
	}
}

// Benchmark tests
func BenchmarkGetEC2Pricing(b *testing.B) {
	mockClient := &MockAWSClient{
		products: []interfaces.PricingProduct{
			{
				SKU:           "TEST123",
				ProductFamily: "Compute Instance",
				ServiceCode:   "AmazonEC2",
			},
		},
	}
	service := NewPricingService(mockClient)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetEC2Pricing(ctx, "t3.micro", "us-east-1", "Linux")
	}
}

func BenchmarkExtractHourlyPrice(b *testing.B) {
	service := &PricingService{}
	product := interfaces.PricingProduct{
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

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ExtractHourlyPrice(product)
	}
}
