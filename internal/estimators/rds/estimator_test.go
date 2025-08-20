package rds

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

func TestRDSEstimator_SupportedResourceType(t *testing.T) {
	estimator := NewEstimator(&MockAWSClient{})

	expected := "RDS"
	actual := estimator.SupportedResourceType()

	if actual != expected {
		t.Errorf("Expected resource type %s, got %s", expected, actual)
	}
}

func TestRDSEstimator_ValidateResource(t *testing.T) {
	estimator := NewEstimator(&MockAWSClient{})

	tests := []struct {
		name        string
		resource    models.ResourceSpec
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name: "valid MySQL RDS instance",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "mysql-db",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "db.t3.micro",
					"engine":        "mysql",
					"storageGB":     20,
					"multiAZ":       false,
					"encrypted":     false,
				},
			},
			expectError: false,
		},
		{
			name: "valid PostgreSQL RDS instance with Multi-AZ",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "postgres-db",
				Region: "us-west-2",
				Properties: map[string]interface{}{
					"instanceClass": "db.r5.large",
					"engine":        "postgres",
					"storageGB":     100,
					"multiAZ":       true,
					"encrypted":     true,
				},
			},
			expectError: false,
		},
		{
			name: "minimal valid configuration",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "minimal-db",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "db.t3.micro",
					"engine":        "mysql",
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
					"instanceClass": "db.t3.micro",
					"engine":        "mysql",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "missing instanceClass",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "no-instance-class",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"engine": "mysql",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "missing engine",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "no-engine",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "db.t3.micro",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid instance class format",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "invalid-class",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "t3.micro",
					"engine":        "mysql",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid engine",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "invalid-engine",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "db.t3.micro",
					"engine":        "mongodb",
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "negative storage",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "negative-storage",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "db.t3.micro",
					"engine":        "mysql",
					"storageGB":     -10,
				},
			},
			expectError: true,
			errorType:   errors.ValidationErrorType,
		},
		{
			name: "invalid multiAZ type",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "invalid-multiaz",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "db.t3.micro",
					"engine":        "mysql",
					"multiAZ":       "yes",
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

func TestRDSEstimator_EstimateCost(t *testing.T) {
	// Mock pricing data
	mockProducts := []interfaces.PricingProduct{
		{
			SKU:           "RDS001",
			ProductFamily: "Database Instance",
			ServiceCode:   "AmazonRDS",
			Attributes: map[string]string{
				"usageType":        "InstanceUsage:db.t3.micro",
				"location":         "US East (N. Virginia)",
				"instanceType":     "db.t3.micro",
				"databaseEngine":   "MySQL",
				"deploymentOption": "Single-AZ",
			},
			Terms: map[string]interface{}{
				"OnDemand": map[string]interface{}{
					"RDS001.JRTCKXETXF": map[string]interface{}{
						"priceDimensions": map[string]interface{}{
							"RDS001.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
								"pricePerUnit": map[string]interface{}{
									"USD": "0.017",
								},
							},
						},
					},
				},
			},
		},
		{
			SKU:           "RDS002",
			ProductFamily: "Database Storage",
			ServiceCode:   "AmazonRDS",
			Attributes: map[string]string{
				"usageType": "GP2-Storage",
				"location":  "US East (N. Virginia)",
			},
			Terms: map[string]interface{}{
				"OnDemand": map[string]interface{}{
					"RDS002.JRTCKXETXF": map[string]interface{}{
						"priceDimensions": map[string]interface{}{
							"RDS002.JRTCKXETXF.6YS6EN2CT7": map[string]interface{}{
								"pricePerUnit": map[string]interface{}{
									"USD": "0.115",
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
			name: "MySQL RDS instance",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "mysql-db",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "db.t3.micro",
					"engine":        "mysql",
					"storageGB":     20,
					"multiAZ":       false,
				},
			},
			expectError:     false,
			expectedMinCost: 0.01, // Should have some cost
		},
		{
			name: "PostgreSQL RDS with Multi-AZ",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "postgres-db",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "db.t3.micro",
					"engine":        "postgres",
					"storageGB":     100,
					"multiAZ":       true,
					"encrypted":     true,
				},
			},
			expectError:     false,
			expectedMinCost: 0.01, // Should have some cost
		},
		{
			name: "minimal configuration",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "minimal-db",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "db.t3.micro",
					"engine":        "mysql",
				},
			},
			expectError:     false,
			expectedMinCost: 0.01, // Should have base cost
		},
		{
			name: "invalid resource",
			resource: models.ResourceSpec{
				Type:   "RDS",
				Name:   "invalid",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceClass": "invalid",
					"engine":        "mysql",
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

func TestRDSEstimator_GetSupportedInstanceClasses(t *testing.T) {
	estimator := &Estimator{}

	classes := estimator.GetSupportedInstanceClasses()

	if len(classes) == 0 {
		t.Errorf("Expected supported instance classes but got none")
	}

	// Check that all returned classes are valid
	for _, class := range classes {
		if !estimator.isValidInstanceClass(class) {
			t.Errorf("Invalid instance class returned: %s", class)
		}
	}
}

func TestRDSEstimator_GetSupportedEngines(t *testing.T) {
	estimator := &Estimator{}

	engines := estimator.GetSupportedEngines()

	expectedEngines := []string{"mysql", "postgres", "mariadb"}
	for _, expectedEngine := range expectedEngines {
		found := false
		for _, actualEngine := range engines {
			if actualEngine == expectedEngine {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected engine %s not found in supported engines", expectedEngine)
		}
	}
}

func TestRDSEstimator_GetEngineDescription(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		engine      string
		expectEmpty bool
	}{
		{"mysql", false},
		{"postgres", false},
		{"oracle-ee", false},
		{"unknown", false}, // Should return "Unknown database engine"
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			desc := estimator.GetEngineDescription(tt.engine)

			if tt.expectEmpty && desc != "" {
				t.Errorf("Expected empty description, got %s", desc)
			}

			if !tt.expectEmpty && desc == "" {
				t.Errorf("Expected non-empty description, got empty")
			}
		})
	}
}

func TestRDSEstimator_IsValidInstanceClass(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		instanceClass string
		expected      bool
	}{
		{"db.t3.micro", true},
		{"db.r5.large", true},
		{"db.m5.xlarge", true},
		{"t3.micro", false},        // Missing "db." prefix
		{"db.invalid.size", false}, // Invalid family
		{"", false},                // Empty string
	}

	for _, tt := range tests {
		t.Run(tt.instanceClass, func(t *testing.T) {
			result := estimator.isValidInstanceClass(tt.instanceClass)
			if result != tt.expected {
				t.Errorf("Expected %t for instance class %s, got %t", tt.expected, tt.instanceClass, result)
			}
		})
	}
}

func TestRDSEstimator_IsValidEngine(t *testing.T) {
	estimator := &Estimator{}

	tests := []struct {
		engine   string
		expected bool
	}{
		{"mysql", true},
		{"postgres", true},
		{"oracle-ee", true},
		{"aurora-mysql", true},
		{"mongodb", false}, // Not supported by RDS
		{"", false},        // Empty string
	}

	for _, tt := range tests {
		t.Run(tt.engine, func(t *testing.T) {
			result := estimator.isValidEngine(tt.engine)
			if result != tt.expected {
				t.Errorf("Expected %t for engine %s, got %t", tt.expected, tt.engine, result)
			}
		})
	}
}
