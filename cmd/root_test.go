package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"shylock/internal/errors"
	"shylock/internal/models"
)

func TestValidateConfigFile(t *testing.T) {
	// Create a temporary directory for test files
	tempDir := t.TempDir()

	// Create a valid JSON file
	validFile := filepath.Join(tempDir, "valid.json")
	if err := os.WriteFile(validFile, []byte(`{"version": "1.0"}`), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create an invalid extension file
	invalidFile := filepath.Join(tempDir, "invalid.yaml")
	if err := os.WriteFile(invalidFile, []byte(`version: "1.0"`), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	tests := []struct {
		name        string
		configFile  string
		expectError bool
		errorType   errors.ErrorType
	}{
		{
			name:        "valid JSON file",
			configFile:  validFile,
			expectError: false,
		},
		{
			name:        "non-existent file",
			configFile:  filepath.Join(tempDir, "nonexistent.json"),
			expectError: true,
			errorType:   errors.FileErrorType,
		},
		{
			name:        "invalid extension",
			configFile:  invalidFile,
			expectError: true,
			errorType:   errors.FileErrorType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateConfigFile(tt.configFile)

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

func TestApplyCliOverrides(t *testing.T) {
	tests := []struct {
		name           string
		initialConfig  *models.EstimationConfig
		setupFlags     func()
		expectedRegion string
		expectError    bool
	}{
		{
			name: "override region",
			initialConfig: &models.EstimationConfig{
				Version: "1.0",
				Resources: []models.ResourceSpec{
					{Type: "EC2", Name: "test", Region: "us-east-1"},
					{Type: "S3", Name: "bucket", Region: "us-west-1"},
				},
			},
			setupFlags: func() {
				region = "eu-west-1"
				outputFormat = "table"
				currency = "USD"
			},
			expectedRegion: "eu-west-1",
			expectError:    false,
		},
		{
			name: "invalid output format",
			initialConfig: &models.EstimationConfig{
				Version: "1.0",
				Resources: []models.ResourceSpec{
					{Type: "EC2", Name: "test", Region: "us-east-1"},
				},
			},
			setupFlags: func() {
				region = ""
				outputFormat = "invalid"
				currency = "USD"
			},
			expectError: true,
		},
		{
			name: "override currency",
			initialConfig: &models.EstimationConfig{
				Version: "1.0",
				Resources: []models.ResourceSpec{
					{Type: "EC2", Name: "test", Region: "us-east-1"},
				},
				Options: models.ConfigOptions{Currency: "USD"},
			},
			setupFlags: func() {
				region = ""
				outputFormat = "json"
				currency = "EUR"
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			region = ""
			outputFormat = "table"
			currency = "USD"

			// Setup test flags
			tt.setupFlags()

			err := applyCliOverrides(tt.initialConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				// Check region override
				if tt.expectedRegion != "" {
					for _, resource := range tt.initialConfig.Resources {
						if resource.Region != tt.expectedRegion {
							t.Errorf("expected region '%s', got '%s'", tt.expectedRegion, resource.Region)
						}
					}
				}

				// Check currency override
				if currency != "USD" && currency != "" {
					if tt.initialConfig.Options.Currency != currency {
						t.Errorf("expected currency '%s', got '%s'", currency, tt.initialConfig.Options.Currency)
					}
				}
			}
		})
	}
}

func TestOutputFormats(t *testing.T) {
	// Create test result
	result := &models.EstimationResult{
		TotalHourlyCost:  1.5,
		TotalDailyCost:   36.0,
		TotalMonthlyCost: 1080.0,
		Currency:         "USD",
		ResourceCosts: []models.CostEstimate{
			{
				ResourceName: "test-instance",
				ResourceType: "EC2",
				Region:       "us-east-1",
				HourlyCost:   1.0,
				DailyCost:    24.0,
				MonthlyCost:  720.0,
				Currency:     "USD",
			},
			{
				ResourceName: "test-bucket",
				ResourceType: "S3",
				Region:       "us-east-1",
				HourlyCost:   0.5,
				DailyCost:    12.0,
				MonthlyCost:  360.0,
				Currency:     "USD",
			},
		},
	}

	tests := []struct {
		name         string
		format       string
		expectError  bool
		checkContent func(string) bool
	}{
		{
			name:        "table format",
			format:      "table",
			expectError: false,
			checkContent: func(output string) bool {
				return strings.Contains(output, "Cost Summary") &&
					strings.Contains(output, "Resource Breakdown") &&
					strings.Contains(output, "test-instance") &&
					strings.Contains(output, "test-bucket")
			},
		},
		{
			name:        "json format",
			format:      "json",
			expectError: false,
			checkContent: func(output string) bool {
				return strings.Contains(output, `"totalHourlyCost"`) &&
					strings.Contains(output, `"resourceCosts"`) &&
					strings.Contains(output, `"test-instance"`)
			},
		},
		{
			name:        "csv format",
			format:      "csv",
			expectError: false,
			checkContent: func(output string) bool {
				return strings.Contains(output, "Resource Name,Resource Type") &&
					strings.Contains(output, "test-instance,EC2") &&
					strings.Contains(output, "TOTAL,,")
			},
		},
		{
			name:        "invalid format",
			format:      "xml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := outputResults(result, tt.format)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			buf := make([]byte, 1024*10) // 10KB buffer
			n, _ := r.Read(buf)
			output := string(buf[:n])

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if tt.checkContent != nil && !tt.checkContent(output) {
					t.Errorf("output content validation failed. Output: %s", output)
				}
			}
		})
	}
}

func TestFormatDetailKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"instanceType", "Instance Type"},
		{"operatingSystem", "Operating System"},
		{"storageClass", "Storage Class"},
		{"sizeGB", "Size (GB)"},
		{"unknownKey", "Unknown Key"},
		{"camelCaseKey", "Camel Case Key"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := formatDetailKey(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIsImportantDetail(t *testing.T) {
	tests := []struct {
		key      string
		expected bool
	}{
		{"instanceType", true},
		{"count", true},
		{"storageClass", true},
		{"sku", false},
		{"productFamily", false},
		{"unknownKey", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := isImportantDetail(tt.key)
			if result != tt.expected {
				t.Errorf("expected %v for key '%s', got %v", tt.expected, tt.key, result)
			}
		})
	}
}

func TestCommandStructure(t *testing.T) {
	// Test that all expected commands are available
	expectedCommands := []string{"estimate", "list", "validate", "version"}

	for _, cmdName := range expectedCommands {
		t.Run("command_"+cmdName, func(t *testing.T) {
			cmd, _, err := rootCmd.Find([]string{cmdName})
			if err != nil {
				t.Errorf("command '%s' not found: %v", cmdName, err)
			}
			if cmd.Name() != cmdName {
				t.Errorf("expected command name '%s', got '%s'", cmdName, cmd.Name())
			}
		})
	}
}

func TestFlagDefinitions(t *testing.T) {
	// Test that all expected flags are defined
	expectedFlags := []string{"output", "region", "verbose", "currency"}

	for _, flagName := range expectedFlags {
		t.Run("flag_"+flagName, func(t *testing.T) {
			flag := rootCmd.PersistentFlags().Lookup(flagName)
			if flag == nil {
				t.Errorf("flag '%s' not found", flagName)
			}
		})
	}
}

// Test helper to capture command output
func executeCommand(args ...string) (string, error) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)

	err := rootCmd.Execute()
	return buf.String(), err
}

func TestVersionCommand(t *testing.T) {
	// Test that version command exists and can be found
	cmd, _, err := rootCmd.Find([]string{"version"})
	if err != nil {
		t.Errorf("version command not found: %v", err)
	}
	if cmd.Name() != "version" {
		t.Errorf("expected command name 'version', got '%s'", cmd.Name())
	}
}

func TestListCommand(t *testing.T) {
	// Test that list command exists and can be found
	cmd, _, err := rootCmd.Find([]string{"list"})
	if err != nil {
		t.Errorf("list command not found: %v", err)
	}
	if cmd.Name() != "list" {
		t.Errorf("expected command name 'list', got '%s'", cmd.Name())
	}
}

// Benchmark tests
func BenchmarkOutputTable(b *testing.B) {
	result := &models.EstimationResult{
		TotalHourlyCost:  10.0,
		TotalDailyCost:   240.0,
		TotalMonthlyCost: 7200.0,
		Currency:         "USD",
		ResourceCosts:    make([]models.CostEstimate, 10),
	}

	// Fill with test data
	for i := range result.ResourceCosts {
		result.ResourceCosts[i] = models.CostEstimate{
			ResourceName: fmt.Sprintf("resource-%d", i),
			ResourceType: "EC2",
			Region:       "us-east-1",
			HourlyCost:   1.0,
			DailyCost:    24.0,
			MonthlyCost:  720.0,
			Currency:     "USD",
		}
	}

	// Redirect output to discard
	oldStdout := os.Stdout
	os.Stdout, _ = os.Open(os.DevNull)
	defer func() { os.Stdout = oldStdout }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = outputTable(result)
	}
}
