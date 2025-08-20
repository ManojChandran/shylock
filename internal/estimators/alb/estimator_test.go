package alb

import (
	"context"
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

func TestALBEstimator_SupportedResourceType(t *testing.T) {
	estimator := NewEstimator(&MockAWSClient{})

	expected := "ALB"
	actual := estimator.SupportedResourceType()

	if actual != expected {
		t.Errorf("Expected resource type %s, got %s", expected, actual)
	}
}

func TestALBEstimator_ValidateResource(t *testing.T) {
	estimator := NewEstimator(&MockAWSClient{})

	tests := []struct {
		name        string
		resource    models.ResourceSpec
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name: "valid application load balancer",
			resource: models.ResourceSpec{
				Type:   "ALB",
				Name:   "app-lb",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"type":                       "application",
					"dataProcessingGB":           100,
					"newConnectionsPerSecond":    50,
					"activeConnectionsPerMinute": 1000,
					"ruleEvaluations":            500,
				},
			},
			expectError: false,
		},
		{
			name: "valid network load balancer",
			resource: models.ResourceSpec{
				Type:   "ALB",
				Name:   "net-lb",
				Region: "us-west-2",
				Properties: map[string]interface{}{
					"type":                       "network",
					"dataProcessingGB":           50,
					"newConnectionsPerSecond":    100,
					"activeConnectionsPerMinute": 5000,
				},
			},
			expectError: false,
		},
		{
			name: "minimal valid configuration",
			resource: models.ResourceSpec{
				Type:   "ALB",
				Name:   "minimal-lb",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"type": "application",
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
					"type": "application",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "missing type property",
			resource: models.ResourceSpec{
				Type:       "ALB",
				Name:       "no-type",
				Region:     "us-east-1",
				Properties: map[string]interface{}{},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid ALB type",
			resource: models.ResourceSpec{
				Type:   "ALB",
				Name:   "invalid-type",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"type": "classic",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "negative data processing",
			resource: models.ResourceSpec{
				Type:   "ALB",
				Name:   "negative-data",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"type":             "application",
					"dataProcessingGB": -10,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid data type for connections",
			resource: models.ResourceSpec{
				Type:   "ALB",
				Name:   "invalid-connections",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"type":                    "application",
					"newConnectionsPerSecond": "invalid",
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

func TestALBEstimator_EstimateCost(t *testing.T) {
	// Mock pricing data
	mockProducts := []interfaces.PricingProduct{
		{
			SKU:           "ALB001",
			ProductFamily: "Load Balancer-Application",
			ServiceCode:   "AWSELB",
			Attributes: map[string]string{
				"usageType": "LoadBalancerUsage",
				"location":  "US East (N. Virginia)",
			},
			Terms: map[string]interface{}{
				"OnDemand": map[string]interface{}{
					"ALB001.JRTCKXETXF": map[string]interface{}{
						"priceDimensions": map[string]interface{}{
							"ALB001.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
								"pricePerUnit": map[string]interface{}{
									"USD": "0.0225",
								},
							},
						},
					},
				},
			},
		},
		{
			SKU:           "ALB002",
			ProductFamily: "Load Balancer-Application",
			ServiceCode:   "AWSELB",
			Attributes: map[string]string{
				"usageType": "LCUUsage",
				"location":  "US East (N. Virginia)",
			},
			Terms: map[string]interface{}{
				"OnDemand": map[string]interface{}{
					"ALB002.JRTCKXETXF": map[string]interface{}{
						"priceDimensions": map[string]interface{}{
							"ALB002.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
								"pricePerUnit": map[string]interface{}{
									"USD": "0.008",
								},
							},
						},
					},
				},
			},
		},
	}

	mockClient := &MockAWSClient{
		products: mockProducts,
	}

	estimator := NewEstimator(mockClient)
	ctx := context.Background()

	tests := []struct {
		name            string
		resource        models.ResourceSpec
		expectError     bool
		expectedMinCost float64 // Minimum expected cost
	}{
		{
			name: "application load balancer with traffic",
			resource: models.ResourceSpec{
				Type:   "ALB",
				Name:   "app-lb",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"type":                       "application",
					"dataProcessingGB":           100,
					"newConnectionsPerSecond":    50,
					"activeConnectionsPerMinute": 1000,
					"ruleEvaluations":            500,
				},
			},
			expectError:     false,
			expectedMinCost: 0.02, // Should have some cost
		},
		{
			name: "network load balancer minimal",
			resource: models.ResourceSpec{
				Type:   "ALB",
				Name:   "net-lb",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"type": "network",
				},
			},
			expectError:     false,
			expectedMinCost: 0.02, // Should have base cost
		},
		{
			name: "invalid resource",
			resource: models.ResourceSpec{
				Type:   "ALB",
				Name:   "invalid",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"type": "invalid",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimate, err := estimator.EstimateCost(ctx, tt.resource)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if estimate == nil {
				t.Errorf("Expected estimate but got nil")
				return
			}

			// Validate estimate structure
			if estimate.ResourceName != tt.resource.Name {
				t.Errorf("Expected resource name %s, got %s", tt.resource.Name, estimate.ResourceName)
			}

			if estimate.ResourceType != tt.resource.Type {
				t.Errorf("Expected resource type %s, got %s", tt.resource.Type, estimate.ResourceType)
			}

			if estimate.Region != tt.resource.Region {
				t.Errorf("Expected region %s, got %s", tt.resource.Region, estimate.Region)
			}

			if estimate.HourlyCost < tt.expectedMinCost {
				t.Errorf("Expected hourly cost >= %f, got %f", tt.expectedMinCost, estimate.HourlyCost)
			}

			// Validate cost calculations
			expectedDaily := estimate.HourlyCost * 24
			if estimate.DailyCost != expectedDaily {
				t.Errorf("Expected daily cost %f, got %f", expectedDaily, estimate.DailyCost)
			}

			expectedMonthly := estimate.HourlyCost * 24 * 30
			if estimate.MonthlyCost != expectedMonthly {
				t.Errorf("Expected monthly cost %f, got %f", expectedMonthly, estimate.MonthlyCost)
			}

			// Validate assumptions and details
			if len(estimate.Assumptions) == 0 {
				t.Errorf("Expected assumptions but got none")
			}

			if len(estimate.Details) == 0 {
				t.Errorf("Expected details but got none")
			}
		})
	}
}

func TestALBEstimator_CalculateLCUConsumption(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		name                       string
		albType                    string
		dataProcessingGB           int
		newConnectionsPerSecond    int
		activeConnectionsPerMinute int
		ruleEvaluations            int
		expectedMinLCU             float64
	}{
		{
			name:                       "application LB high data processing",
			albType:                    "application",
			dataProcessingGB:           100,
			newConnectionsPerSecond:    10,
			activeConnectionsPerMinute: 1000,
			ruleEvaluations:            100,
			expectedMinLCU:             100.0, // 100 GB / 1 GB per LCU
		},
		{
			name:                       "application LB high connections",
			albType:                    "application",
			dataProcessingGB:           1,
			newConnectionsPerSecond:    100,
			activeConnectionsPerMinute: 1000,
			ruleEvaluations:            100,
			expectedMinLCU:             4.0, // 100 connections / 25 per LCU
		},
		{
			name:                       "network LB high connections",
			albType:                    "network",
			dataProcessingGB:           1,
			newConnectionsPerSecond:    1600,
			activeConnectionsPerMinute: 50000,
			expectedMinLCU:             2.0, // 1600 connections / 800 per LCU
		},
		{
			name:                       "minimal usage",
			albType:                    "application",
			dataProcessingGB:           0,
			newConnectionsPerSecond:    0,
			activeConnectionsPerMinute: 0,
			ruleEvaluations:            0,
			expectedMinLCU:             1.0, // Minimum 1 LCU
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lcu := estimator.calculateLCUConsumption(tt.albType, tt.dataProcessingGB, tt.newConnectionsPerSecond, tt.activeConnectionsPerMinute, tt.ruleEvaluations)

			if lcu < tt.expectedMinLCU {
				t.Errorf("Expected LCU >= %f, got %f", tt.expectedMinLCU, lcu)
			}
		})
	}
}

func TestALBEstimator_GetSupportedALBTypes(t *testing.T) {
	estimator := &Estimator{}

	types := estimator.GetSupportedALBTypes()

	expectedTypes := []string{"application", "network"}
	if len(types) != len(expectedTypes) {
		t.Errorf("Expected %d types, got %d", len(expectedTypes), len(types))
	}

	for _, expectedType := range expectedTypes {
		found := false
		for _, actualType := range types {
			if actualType == expectedType {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected type %s not found in supported types", expectedType)
		}
	}
}

func TestALBEstimator_GetALBTypeDescription(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		albType     string
		expectEmpty bool
	}{
		{"application", false},
		{"network", false},
		{"unknown", false}, // Should return "Unknown ALB type"
	}

	for _, tt := range tests {
		t.Run(tt.albType, func(t *testing.T) {
			desc := estimator.GetALBTypeDescription(tt.albType)

			if tt.expectEmpty && desc != "" {
				t.Errorf("Expected empty description, got %s", desc)
			}

			if !tt.expectEmpty && desc == "" {
				t.Errorf("Expected non-empty description, got empty")
			}
		})
	}
}
