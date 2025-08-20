package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestResourceSpecValidation(t *testing.T) {
	tests := []struct {
		name        string
		resource    ResourceSpec
		expectError bool
	}{
		{
			name: "valid resource spec",
			resource: ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
			expectError: false,
		},
		{
			name: "missing type",
			resource: ResourceSpec{
				Name:   "web-server",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
			expectError: true,
		},
		{
			name: "missing name",
			resource: ResourceSpec{
				Type:   "EC2",
				Region: "us-east-1",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
			expectError: true,
		},
		{
			name: "missing region",
			resource: ResourceSpec{
				Type: "EC2",
				Name: "web-server",
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
			expectError: true,
		},
		{
			name: "missing properties",
			resource: ResourceSpec{
				Type:   "EC2",
				Name:   "web-server",
				Region: "us-east-1",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.resource.Validate()
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestEstimationConfigValidation(t *testing.T) {
	validResource := ResourceSpec{
		Type:   "EC2",
		Name:   "web-server",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"instanceType": "t3.micro",
		},
	}

	tests := []struct {
		name        string
		config      EstimationConfig
		expectError bool
	}{
		{
			name: "valid config",
			config: EstimationConfig{
				Version:   "1.0",
				Resources: []ResourceSpec{validResource},
			},
			expectError: false,
		},
		{
			name: "missing version",
			config: EstimationConfig{
				Resources: []ResourceSpec{validResource},
			},
			expectError: true,
		},
		{
			name: "no resources",
			config: EstimationConfig{
				Version:   "1.0",
				Resources: []ResourceSpec{},
			},
			expectError: true,
		},
		{
			name: "invalid resource",
			config: EstimationConfig{
				Version: "1.0",
				Resources: []ResourceSpec{
					{
						Type: "EC2",
						// Missing required fields
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestResourceSpecJSONParsing(t *testing.T) {
	jsonData := `{
		"type": "EC2",
		"name": "web-server",
		"region": "us-east-1",
		"properties": {
			"instanceType": "t3.micro",
			"count": 2,
			"storage": 20
		}
	}`

	var resource ResourceSpec
	err := json.Unmarshal([]byte(jsonData), &resource)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if resource.Type != "EC2" {
		t.Errorf("expected type 'EC2', got '%s'", resource.Type)
	}
	if resource.Name != "web-server" {
		t.Errorf("expected name 'web-server', got '%s'", resource.Name)
	}
	if resource.Region != "us-east-1" {
		t.Errorf("expected region 'us-east-1', got '%s'", resource.Region)
	}

	// Test property access
	instanceType, err := resource.GetStringProperty("instanceType")
	if err != nil {
		t.Errorf("failed to get instanceType: %v", err)
	}
	if instanceType != "t3.micro" {
		t.Errorf("expected instanceType 't3.micro', got '%s'", instanceType)
	}

	count, err := resource.GetIntProperty("count")
	if err != nil {
		t.Errorf("failed to get count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}
}

func TestEstimationConfigJSONParsing(t *testing.T) {
	jsonData := `{
		"version": "1.0",
		"resources": [
			{
				"type": "EC2",
				"name": "web-server",
				"region": "us-east-1",
				"properties": {
					"instanceType": "t3.micro"
				}
			},
			{
				"type": "S3",
				"name": "data-bucket",
				"region": "us-west-2",
				"properties": {
					"storageClass": "STANDARD",
					"sizeGB": 100
				}
			}
		],
		"options": {
			"defaultRegion": "us-east-1",
			"currency": "USD",
			"timeFrame": "monthly"
		}
	}`

	var config EstimationConfig
	err := json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("expected version '1.0', got '%s'", config.Version)
	}

	if len(config.Resources) != 2 {
		t.Errorf("expected 2 resources, got %d", len(config.Resources))
	}

	if config.Options.Currency != "USD" {
		t.Errorf("expected currency 'USD', got '%s'", config.Options.Currency)
	}

	// Validate the parsed config
	err = config.Validate()
	if err != nil {
		t.Errorf("config validation failed: %v", err)
	}
}

func TestCostEstimateCalculations(t *testing.T) {
	estimate := CostEstimate{
		ResourceName: "test-resource",
		ResourceType: "EC2",
		Region:       "us-east-1",
		HourlyCost:   0.0116, // t3.micro hourly cost
		Currency:     "USD",
		Timestamp:    time.Now(),
	}

	estimate.CalculateCosts()

	expectedDaily := 0.0116 * 24
	expectedMonthly := 0.0116 * 24 * 30

	if estimate.DailyCost != expectedDaily {
		t.Errorf("expected daily cost %.4f, got %.4f", expectedDaily, estimate.DailyCost)
	}

	if estimate.MonthlyCost != expectedMonthly {
		t.Errorf("expected monthly cost %.4f, got %.4f", expectedMonthly, estimate.MonthlyCost)
	}
}

func TestCostEstimateAssumptions(t *testing.T) {
	estimate := CostEstimate{}

	estimate.AddAssumption("24/7 usage assumed")
	estimate.AddAssumption("On-demand pricing")

	if len(estimate.Assumptions) != 2 {
		t.Errorf("expected 2 assumptions, got %d", len(estimate.Assumptions))
	}

	if estimate.Assumptions[0] != "24/7 usage assumed" {
		t.Errorf("unexpected first assumption: %s", estimate.Assumptions[0])
	}
}

func TestCostEstimateDetails(t *testing.T) {
	estimate := CostEstimate{}

	estimate.SetDetail("instanceType", "t3.micro")
	estimate.SetDetail("operatingSystem", "Linux")

	if len(estimate.Details) != 2 {
		t.Errorf("expected 2 details, got %d", len(estimate.Details))
	}

	if estimate.Details["instanceType"] != "t3.micro" {
		t.Errorf("unexpected instanceType detail: %s", estimate.Details["instanceType"])
	}
}

func TestResourceSpecPropertyAccess(t *testing.T) {
	resource := ResourceSpec{
		Type:   "EC2",
		Name:   "test",
		Region: "us-east-1",
		Properties: map[string]interface{}{
			"stringProp": "value",
			"intProp":    42,
			"floatProp":  3.14,
		},
	}

	// Test string property
	str, err := resource.GetStringProperty("stringProp")
	if err != nil {
		t.Errorf("failed to get string property: %v", err)
	}
	if str != "value" {
		t.Errorf("expected 'value', got '%s'", str)
	}

	// Test int property
	intVal, err := resource.GetIntProperty("intProp")
	if err != nil {
		t.Errorf("failed to get int property: %v", err)
	}
	if intVal != 42 {
		t.Errorf("expected 42, got %d", intVal)
	}

	// Test float as int property
	floatAsInt, err := resource.GetIntProperty("floatProp")
	if err != nil {
		t.Errorf("failed to get float as int property: %v", err)
	}
	if floatAsInt != 3 {
		t.Errorf("expected 3, got %d", floatAsInt)
	}

	// Test non-existent property
	_, err = resource.GetStringProperty("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent property")
	}

	// Test wrong type
	_, err = resource.GetStringProperty("intProp")
	if err == nil {
		t.Error("expected error for wrong property type")
	}
}
