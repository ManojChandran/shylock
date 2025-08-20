package config

import (
	"os"
	"testing"

	"shylock/internal/models"
)

func TestParseConfigFromBytes(t *testing.T) {
	parser := NewParser()

	tests := []struct {
		name        string
		jsonData    string
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid simple config",
			jsonData: `{
				"version": "1.0",
				"resources": [
					{
						"type": "EC2",
						"name": "web-server",
						"region": "us-east-1",
						"properties": {
							"instanceType": "t3.micro"
						}
					}
				]
			}`,
			expectError: false,
		},
		{
			name: "valid config with options",
			jsonData: `{
				"version": "1.0",
				"resources": [
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
					"currency": "USD",
					"timeFrame": "monthly"
				}
			}`,
			expectError: false,
		},
		{
			name:        "empty data",
			jsonData:    "",
			expectError: true,
			errorMsg:    "configuration data is empty",
		},
		{
			name:        "invalid JSON",
			jsonData:    `{"version": "1.0", "resources": [}`,
			expectError: true,
			errorMsg:    "invalid JSON format",
		},
		{
			name: "missing version",
			jsonData: `{
				"resources": [
					{
						"type": "EC2",
						"name": "web-server",
						"region": "us-east-1",
						"properties": {
							"instanceType": "t3.micro"
						}
					}
				]
			}`,
			expectError: true,
			errorMsg:    "configuration version is required",
		},
		{
			name: "unsupported resource type",
			jsonData: `{
				"version": "1.0",
				"resources": [
					{
						"type": "LAMBDA",
						"name": "function",
						"region": "us-east-1",
						"properties": {
							"runtime": "nodejs"
						}
					}
				]
			}`,
			expectError: true,
			errorMsg:    "unsupported resource type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parser.ParseConfigFromBytes([]byte(tt.jsonData))

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if config == nil {
					t.Error("expected config but got nil")
				}
			}
		})
	}
}

func TestValidateEC2Resource(t *testing.T) {
	parser := &Parser{
		supportedResourceTypes: map[string]bool{"EC2": true},
	}

	tests := []struct {
		name        string
		resource    models.ResourceSpec
		expectError bool
		errorMsg    string
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
			errorMsg:    "missing required property 'instanceType'",
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
			errorMsg:    "invalid instance type format",
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
			errorMsg:    "count must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validateEC2Resource(&tt.resource)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateS3Resource(t *testing.T) {
	parser := &Parser{
		supportedResourceTypes: map[string]bool{"S3": true},
	}

	tests := []struct {
		name        string
		resource    models.ResourceSpec
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid S3 resource",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-west-2",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
					"sizeGB":       100,
				},
			},
			expectError: false,
		},
		{
			name: "missing storageClass",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-west-2",
				Properties: map[string]interface{}{
					"sizeGB": 100,
				},
			},
			expectError: true,
			errorMsg:    "missing required property 'storageClass'",
		},
		{
			name: "invalid storageClass",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-west-2",
				Properties: map[string]interface{}{
					"storageClass": "INVALID_CLASS",
				},
			},
			expectError: true,
			errorMsg:    "invalid storage class",
		},
		{
			name: "invalid sizeGB",
			resource: models.ResourceSpec{
				Type:   "S3",
				Name:   "data-bucket",
				Region: "us-west-2",
				Properties: map[string]interface{}{
					"storageClass": "STANDARD",
					"sizeGB":       -10,
				},
			},
			expectError: true,
			errorMsg:    "sizeGB must be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validateS3Resource(&tt.resource)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidateOptions(t *testing.T) {
	parser := &Parser{}

	tests := []struct {
		name        string
		options     models.ConfigOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid options",
			options: models.ConfigOptions{
				Currency:  "USD",
				TimeFrame: "monthly",
			},
			expectError: false,
		},
		{
			name: "invalid currency",
			options: models.ConfigOptions{
				Currency: "INVALID",
			},
			expectError: true,
			errorMsg:    "invalid currency",
		},
		{
			name: "invalid timeFrame",
			options: models.ConfigOptions{
				TimeFrame: "yearly",
			},
			expectError: true,
			errorMsg:    "invalid timeFrame",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := parser.validateOptions(&tt.options)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestParseConfigFile(t *testing.T) {
	parser := NewParser()

	// Create a temporary valid config file
	validConfig := `{
		"version": "1.0",
		"resources": [
			{
				"type": "EC2",
				"name": "test-server",
				"region": "us-east-1",
				"properties": {
					"instanceType": "t3.micro"
				}
			}
		]
	}`

	// Create temporary JSON file
	tmpFile, err := os.CreateTemp("", "test-config-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(validConfig); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	tmpFile.Close()

	// Create temporary YAML file for extension test
	tmpYamlFile, err := os.CreateTemp("", "test-config-*.yaml")
	if err != nil {
		t.Fatalf("failed to create temp yaml file: %v", err)
	}
	defer os.Remove(tmpYamlFile.Name())
	tmpYamlFile.Close()

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid file",
			filePath:    tmpFile.Name(),
			expectError: false,
		},
		{
			name:        "empty path",
			filePath:    "",
			expectError: true,
			errorMsg:    "file path cannot be empty",
		},
		{
			name:        "non-existent file",
			filePath:    "/non/existent/file.json",
			expectError: true,
			errorMsg:    "configuration file does not exist",
		},
		{
			name:        "unsupported extension",
			filePath:    tmpYamlFile.Name(),
			expectError: true,
			errorMsg:    "unsupported file format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := parser.ParseConfig(tt.filePath)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				} else if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if config == nil {
					t.Error("expected config but got nil")
				}
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	parser := &Parser{}

	config := &models.EstimationConfig{
		Version: "1.0",
		Resources: []models.ResourceSpec{
			{
				Type:   "EC2",
				Name:   "test",
				Region: "", // Empty region
				Properties: map[string]interface{}{
					"instanceType": "t3.micro",
				},
			},
		},
		Options: models.ConfigOptions{
			DefaultRegion: "us-west-1",
			// Currency and TimeFrame are empty
		},
	}

	parser.applyDefaults(config)

	// Check that defaults were applied
	if config.Options.Currency != "USD" {
		t.Errorf("expected default currency 'USD', got '%s'", config.Options.Currency)
	}

	if config.Options.TimeFrame != "monthly" {
		t.Errorf("expected default timeFrame 'monthly', got '%s'", config.Options.TimeFrame)
	}

	if config.Resources[0].Region != "us-west-1" {
		t.Errorf("expected resource region to be set to default 'us-west-1', got '%s'", config.Resources[0].Region)
	}
}

func TestInstanceTypeValidation(t *testing.T) {
	parser := &Parser{}

	tests := []struct {
		instanceType string
		valid        bool
	}{
		{"t3.micro", true},
		{"m5.large", true},
		{"c5.xlarge", true},
		{"r5.2xlarge", true},
		{"invalid", false},
		{"t3", false},
		{".micro", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.instanceType, func(t *testing.T) {
			result := parser.isValidInstanceType(tt.instanceType)
			if result != tt.valid {
				t.Errorf("expected %v for instance type '%s', got %v", tt.valid, tt.instanceType, result)
			}
		})
	}
}

// Helper function to check if a string contains a substring
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
