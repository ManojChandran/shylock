package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"shylock/internal/models"
)

// Helper function to create test estimation result
func createTestEstimationResult() *models.EstimationResult {
	timestamp := time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

	return &models.EstimationResult{
		TotalHourlyCost:  1.5,
		TotalDailyCost:   36.0,
		TotalMonthlyCost: 1080.0,
		Currency:         "USD",
		GeneratedAt:      timestamp,
		ResourceCosts: []models.CostEstimate{
			{
				ResourceName: "web-server",
				ResourceType: "EC2",
				Region:       "us-east-1",
				HourlyCost:   0.5,
				DailyCost:    12.0,
				MonthlyCost:  360.0,
				Currency:     "USD",
				Timestamp:    timestamp,
				Assumptions:  []string{"24/7 usage assumed", "On-demand pricing"},
				Details: map[string]string{
					"instanceType": "t3.micro",
					"count":        "2",
				},
			},
			{
				ResourceName: "database",
				ResourceType: "RDS",
				Region:       "us-east-1",
				HourlyCost:   1.0,
				DailyCost:    24.0,
				MonthlyCost:  720.0,
				Currency:     "USD",
				Timestamp:    timestamp,
				Assumptions:  []string{"Single AZ deployment", "GP2 storage"},
				Details: map[string]string{
					"instanceClass": "db.t3.micro",
					"engine":        "mysql",
					"storageGB":     "20",
				},
			},
		},
	}
}

func TestDefaultFormatOptions(t *testing.T) {
	options := DefaultFormatOptions()

	if options == nil {
		t.Fatal("Expected default options but got nil")
	}

	// Test default values
	if options.Verbose != false {
		t.Errorf("Expected Verbose to be false, got %t", options.Verbose)
	}

	if options.ShowDetails != true {
		t.Errorf("Expected ShowDetails to be true, got %t", options.ShowDetails)
	}

	if options.ShowAssumptions != true {
		t.Errorf("Expected ShowAssumptions to be true, got %t", options.ShowAssumptions)
	}

	if options.Currency != "USD" {
		t.Errorf("Expected Currency to be USD, got %s", options.Currency)
	}

	if options.Precision != 4 {
		t.Errorf("Expected Precision to be 4, got %d", options.Precision)
	}

	if options.SortBy != "name" {
		t.Errorf("Expected SortBy to be name, got %s", options.SortBy)
	}

	if options.GroupBy != "none" {
		t.Errorf("Expected GroupBy to be none, got %s", options.GroupBy)
	}
}

func TestTableFormatter_FormatType(t *testing.T) {
	formatter := NewTableFormatter()

	expected := "table"
	actual := formatter.FormatType()

	if actual != expected {
		t.Errorf("Expected format type %s, got %s", expected, actual)
	}
}

func TestTableFormatter_Format(t *testing.T) {
	formatter := &TableFormatter{}
	result := createTestEstimationResult()

	output, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output == "" {
		t.Fatal("Expected output but got empty string")
	}

	// Check for expected content
	expectedContent := []string{
		"AWS Cost Estimation Results",
		"Cost Summary",
		"Resource Breakdown",
		"web-server",
		"database",
		"EC2",
		"RDS",
		"us-east-1",
		"$1.5000",    // Total hourly cost
		"$36.0000",   // Total daily cost
		"$1080.0000", // Total monthly cost
	}

	for _, content := range expectedContent {
		if !strings.Contains(output, content) {
			t.Errorf("Expected output to contain '%s', but it didn't", content)
		}
	}
}

func TestTableFormatter_FormatWithOptions(t *testing.T) {
	formatter := &TableFormatter{}
	result := createTestEstimationResult()

	tests := []struct {
		name        string
		options     *FormatOptions
		contains    []string
		notContains []string
	}{
		{
			name: "verbose mode",
			options: &FormatOptions{
				Verbose:         true,
				ShowDetails:     true,
				ShowAssumptions: true,
				Currency:        "USD",
				Precision:       4,
				SortBy:          "name",
				GroupBy:         "none",
			},
			contains: []string{
				"Detailed Information",
				"Assumptions:",
				"Configuration:",
				"24/7 usage assumed",
				"Instance Type: t3.micro",
			},
		},
		{
			name: "sort by cost",
			options: &FormatOptions{
				Verbose:         false,
				ShowDetails:     true,
				ShowAssumptions: true,
				Currency:        "USD",
				Precision:       2,
				SortBy:          "cost",
				GroupBy:         "none",
			},
			contains: []string{
				"database", // Should appear first (higher cost)
				"web-server",
			},
		},
		{
			name: "group by type",
			options: &FormatOptions{
				Verbose:         false,
				ShowDetails:     true,
				ShowAssumptions: true,
				Currency:        "USD",
				Precision:       4,
				SortBy:          "name",
				GroupBy:         "type",
			},
			contains: []string{
				"Resource Breakdown (Grouped)",
				"EC2 (1 resources)",
				"RDS (1 resources)",
			},
		},
		{
			name: "no details",
			options: &FormatOptions{
				Verbose:         true,
				ShowDetails:     false,
				ShowAssumptions: true,
				Currency:        "USD",
				Precision:       4,
				SortBy:          "name",
				GroupBy:         "none",
			},
			contains: []string{
				"Detailed Information",
				"Assumptions:",
			},
			notContains: []string{
				"Configuration:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatter.FormatWithOptions(result, tt.options)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			for _, content := range tt.contains {
				if !strings.Contains(output, content) {
					t.Errorf("Expected output to contain '%s', but it didn't", content)
				}
			}

			for _, content := range tt.notContains {
				if strings.Contains(output, content) {
					t.Errorf("Expected output to NOT contain '%s', but it did", content)
				}
			}
		})
	}
}

func TestTableFormatter_EmptyResult(t *testing.T) {
	formatter := &TableFormatter{}
	result := &models.EstimationResult{
		TotalHourlyCost:  0,
		TotalDailyCost:   0,
		TotalMonthlyCost: 0,
		Currency:         "USD",
		GeneratedAt:      time.Now(),
		ResourceCosts:    []models.CostEstimate{},
	}

	output, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "No resources found in estimation") {
		t.Errorf("Expected message about no resources, but didn't find it")
	}
}

func TestJSONFormatter_FormatType(t *testing.T) {
	formatter := NewJSONFormatter()

	expected := "json"
	actual := formatter.FormatType()

	if actual != expected {
		t.Errorf("Expected format type %s, got %s", expected, actual)
	}
}

func TestJSONFormatter_Format(t *testing.T) {
	formatter := &JSONFormatter{}
	result := createTestEstimationResult()

	output, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output == "" {
		t.Fatal("Expected output but got empty string")
	}

	// Verify it's valid JSON
	var parsed models.EstimationResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify content
	if parsed.TotalHourlyCost != result.TotalHourlyCost {
		t.Errorf("Expected TotalHourlyCost %f, got %f", result.TotalHourlyCost, parsed.TotalHourlyCost)
	}

	if len(parsed.ResourceCosts) != len(result.ResourceCosts) {
		t.Errorf("Expected %d resource costs, got %d", len(result.ResourceCosts), len(parsed.ResourceCosts))
	}
}

func TestCSVFormatter_FormatType(t *testing.T) {
	formatter := NewCSVFormatter()

	expected := "csv"
	actual := formatter.FormatType()

	if actual != expected {
		t.Errorf("Expected format type %s, got %s", expected, actual)
	}
}

func TestCSVFormatter_Format(t *testing.T) {
	formatter := &CSVFormatter{}
	result := createTestEstimationResult()

	output, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output == "" {
		t.Fatal("Expected output but got empty string")
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + 2 resources + 1 total = 4 lines
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines, got %d", len(lines))
	}

	// Check header
	expectedHeader := "Resource Name,Resource Type,Region,Hourly Cost,Daily Cost,Monthly Cost,Currency,Generated At"
	if lines[0] != expectedHeader {
		t.Errorf("Expected header '%s', got '%s'", expectedHeader, lines[0])
	}

	// Check that resources are included
	csvContent := strings.Join(lines, "\n")
	if !strings.Contains(csvContent, "web-server") {
		t.Errorf("Expected CSV to contain 'web-server'")
	}

	if !strings.Contains(csvContent, "database") {
		t.Errorf("Expected CSV to contain 'database'")
	}

	if !strings.Contains(csvContent, "TOTAL") {
		t.Errorf("Expected CSV to contain 'TOTAL' row")
	}
}

func TestYAMLFormatter_FormatType(t *testing.T) {
	formatter := NewYAMLFormatter()

	expected := "yaml"
	actual := formatter.FormatType()

	if actual != expected {
		t.Errorf("Expected format type %s, got %s", expected, actual)
	}
}

func TestYAMLFormatter_Format(t *testing.T) {
	formatter := &YAMLFormatter{}
	result := createTestEstimationResult()

	output, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if output == "" {
		t.Fatal("Expected output but got empty string")
	}

	// Check for expected YAML structure
	expectedContent := []string{
		"estimationResult:",
		"totalHourlyCost: 1.5000",
		"totalDailyCost: 36.0000",
		"totalMonthlyCost: 1080.0000",
		"currency: USD",
		"resourceCosts:",
		"- resourceName: web-server",
		"resourceType: EC2",
		"region: us-east-1",
	}

	for _, content := range expectedContent {
		if !strings.Contains(output, content) {
			t.Errorf("Expected YAML to contain '%s', but it didn't", content)
		}
	}
}

func TestFormatterFactory_NewFormatterFactory(t *testing.T) {
	factory := NewFormatterFactory()

	if factory == nil {
		t.Fatal("Expected factory but got nil")
	}

	// Check that built-in formatters are registered
	supportedFormats := factory.GetSupportedFormats()
	expectedFormats := []string{"table", "json", "csv", "yaml"}

	if len(supportedFormats) != len(expectedFormats) {
		t.Errorf("Expected %d formats, got %d", len(expectedFormats), len(supportedFormats))
	}

	for _, expectedFormat := range expectedFormats {
		found := false
		for _, actualFormat := range supportedFormats {
			if actualFormat == expectedFormat {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected format %s not found in supported formats", expectedFormat)
		}
	}
}

func TestFormatterFactory_RegisterFormatter(t *testing.T) {
	factory := NewFormatterFactory()

	// Create a mock formatter
	mockFormatter := &MockFormatter{formatType: "mock"}

	factory.RegisterFormatter(mockFormatter)

	// Check that it's registered
	formatter, err := factory.GetFormatter("mock")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if formatter != mockFormatter {
		t.Errorf("Expected registered formatter, got different instance")
	}
}

func TestFormatterFactory_GetFormatter(t *testing.T) {
	factory := NewFormatterFactory()

	tests := []struct {
		name        string
		formatType  string
		expectError bool
	}{
		{
			name:        "existing formatter",
			formatType:  "table",
			expectError: false,
		},
		{
			name:        "non-existing formatter",
			formatType:  "xml",
			expectError: true,
		},
		{
			name:        "empty format type",
			formatType:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter, err := factory.GetFormatter(tt.formatType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if formatter != nil {
					t.Errorf("Expected nil formatter but got %T", formatter)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if formatter == nil {
					t.Errorf("Expected formatter but got nil")
				}
			}
		})
	}
}

func TestFormatterFactory_FormatResult(t *testing.T) {
	factory := NewFormatterFactory()
	result := createTestEstimationResult()

	tests := []struct {
		name        string
		formatType  string
		expectError bool
	}{
		{
			name:        "table format",
			formatType:  "table",
			expectError: false,
		},
		{
			name:        "json format",
			formatType:  "json",
			expectError: false,
		},
		{
			name:        "csv format",
			formatType:  "csv",
			expectError: false,
		},
		{
			name:        "yaml format",
			formatType:  "yaml",
			expectError: false,
		},
		{
			name:        "unsupported format",
			formatType:  "xml",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := factory.FormatResult(result, tt.formatType)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if output == "" {
					t.Errorf("Expected output but got empty string")
				}
			}
		})
	}
}

func TestFormatterFactory_WriteFormattedResult(t *testing.T) {
	factory := NewFormatterFactory()
	result := createTestEstimationResult()

	var buffer bytes.Buffer
	err := factory.WriteFormattedResult(&buffer, result, "json")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	output := buffer.String()
	if output == "" {
		t.Fatal("Expected output but got empty string")
	}

	// Verify it's valid JSON
	var parsed models.EstimationResult
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}
}

func TestTableFormatter_SortResources(t *testing.T) {
	formatter := &TableFormatter{}

	costs := []models.CostEstimate{
		{ResourceName: "zebra", ResourceType: "EC2", Region: "us-west-2", MonthlyCost: 100},
		{ResourceName: "alpha", ResourceType: "RDS", Region: "us-east-1", MonthlyCost: 200},
		{ResourceName: "beta", ResourceType: "EC2", Region: "us-east-1", MonthlyCost: 50},
	}

	tests := []struct {
		name     string
		sortBy   string
		expected []string // Expected order of resource names
	}{
		{
			name:     "sort by name",
			sortBy:   "name",
			expected: []string{"alpha", "beta", "zebra"},
		},
		{
			name:     "sort by cost",
			sortBy:   "cost",
			expected: []string{"alpha", "zebra", "beta"}, // Descending by cost
		},
		{
			name:     "sort by type",
			sortBy:   "type",
			expected: []string{"beta", "zebra", "alpha"}, // EC2 first, then RDS
		},
		{
			name:     "sort by region",
			sortBy:   "region",
			expected: []string{"alpha", "beta", "zebra"}, // us-east-1 first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original
			testCosts := make([]models.CostEstimate, len(costs))
			copy(testCosts, costs)

			formatter.sortResources(testCosts, tt.sortBy)

			for i, expectedName := range tt.expected {
				if testCosts[i].ResourceName != expectedName {
					t.Errorf("Expected resource %d to be %s, got %s", i, expectedName, testCosts[i].ResourceName)
				}
			}
		})
	}
}

func TestTableFormatter_GroupResources(t *testing.T) {
	formatter := &TableFormatter{}

	costs := []models.CostEstimate{
		{ResourceName: "web-server", ResourceType: "EC2", Region: "us-east-1"},
		{ResourceName: "database", ResourceType: "RDS", Region: "us-east-1"},
		{ResourceName: "backup-server", ResourceType: "EC2", Region: "us-west-2"},
	}

	tests := []struct {
		name           string
		groupBy        string
		expectedGroups []string
		expectedCounts map[string]int
	}{
		{
			name:           "group by type",
			groupBy:        "type",
			expectedGroups: []string{"EC2", "RDS"},
			expectedCounts: map[string]int{"EC2": 2, "RDS": 1},
		},
		{
			name:           "group by region",
			groupBy:        "region",
			expectedGroups: []string{"us-east-1", "us-west-2"},
			expectedCounts: map[string]int{"us-east-1": 2, "us-west-2": 1},
		},
		{
			name:           "no grouping",
			groupBy:        "none",
			expectedGroups: []string{"All Resources"},
			expectedCounts: map[string]int{"All Resources": 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groups := formatter.groupResources(costs, tt.groupBy)

			// Check that all expected groups exist
			for _, expectedGroup := range tt.expectedGroups {
				if _, exists := groups[expectedGroup]; !exists {
					t.Errorf("Expected group %s not found", expectedGroup)
				}
			}

			// Check group counts
			for group, expectedCount := range tt.expectedCounts {
				if actualCount := len(groups[group]); actualCount != expectedCount {
					t.Errorf("Expected group %s to have %d items, got %d", group, expectedCount, actualCount)
				}
			}
		})
	}
}

// MockFormatter for testing
type MockFormatter struct {
	formatType string
}

func (m *MockFormatter) Format(result *models.EstimationResult) (string, error) {
	return "mock output", nil
}

func (m *MockFormatter) FormatType() string {
	return m.formatType
}

func TestTableFormatter_CalculateColumnWidths(t *testing.T) {
	formatter := &TableFormatter{}

	costs := []models.CostEstimate{
		{ResourceName: "short", ResourceType: "EC2", Region: "us-east-1"},
		{ResourceName: "very-long-resource-name", ResourceType: "Lambda", Region: "us-west-2"},
	}

	nameWidth, typeWidth, regionWidth := formatter.calculateColumnWidths(costs)

	// Should accommodate the longest names plus padding
	if nameWidth < len("very-long-resource-name")+2 {
		t.Errorf("Name width too small: %d", nameWidth)
	}

	if typeWidth < len("Lambda")+2 {
		t.Errorf("Type width too small: %d", typeWidth)
	}

	if regionWidth < len("us-west-2")+2 {
		t.Errorf("Region width too small: %d", regionWidth)
	}
}

func TestTableFormatter_CalculateGroupTotal(t *testing.T) {
	formatter := &TableFormatter{}

	costs := []models.CostEstimate{
		{MonthlyCost: 100.0},
		{MonthlyCost: 200.0},
		{MonthlyCost: 50.0},
	}

	total := formatter.calculateGroupTotal(costs)
	expected := 350.0

	if total != expected {
		t.Errorf("Expected total %f, got %f", expected, total)
	}
}

func TestTableFormatter_IsImportantDetail(t *testing.T) {
	formatter := &TableFormatter{}

	tests := []struct {
		key      string
		expected bool
	}{
		{"instanceType", true},
		{"count", true},
		{"storageClass", true},
		{"randomKey", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := formatter.isImportantDetail(tt.key)
			if result != tt.expected {
				t.Errorf("Expected %t for key %s, got %t", tt.expected, tt.key, result)
			}
		})
	}
}

func TestTableFormatter_FormatDetailKey(t *testing.T) {
	formatter := &TableFormatter{}

	tests := []struct {
		key      string
		expected string
	}{
		{"instanceType", "Instance Type"},
		{"storageClass", "Storage Class"},
		{"requestsPerMonth", "Requests/Month"}, // This is the actual mapping
		{"unknownKey", "Unknown Key"},
		{"someNewKey", "Some New Key"}, // Test default formatting
		{"", ""},                       // Empty string edge case
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := formatter.formatDetailKey(tt.key)
			if result != tt.expected {
				t.Errorf("Expected %s for key %s, got %s", tt.expected, tt.key, result)
			}
		})
	}
}

func TestFormatterFactory_GetSupportedFormats(t *testing.T) {
	factory := NewFormatterFactory()

	formats := factory.GetSupportedFormats()

	if len(formats) == 0 {
		t.Fatal("Expected supported formats but got none")
	}

	// Should be sorted
	for i := 1; i < len(formats); i++ {
		if formats[i-1] > formats[i] {
			t.Errorf("Formats not sorted: %s > %s", formats[i-1], formats[i])
		}
	}
}

func TestCSVFormatter_EmptyResult(t *testing.T) {
	formatter := &CSVFormatter{}
	result := &models.EstimationResult{
		TotalHourlyCost:  0,
		TotalDailyCost:   0,
		TotalMonthlyCost: 0,
		Currency:         "USD",
		GeneratedAt:      time.Now(),
		ResourceCosts:    []models.CostEstimate{},
	}

	output, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + total = 2 lines
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines for empty result, got %d", len(lines))
	}

	if !strings.Contains(output, "TOTAL") {
		t.Errorf("Expected TOTAL row even for empty result")
	}
}

func TestYAMLFormatter_EmptyResult(t *testing.T) {
	formatter := &YAMLFormatter{}
	result := &models.EstimationResult{
		TotalHourlyCost:  0,
		TotalDailyCost:   0,
		TotalMonthlyCost: 0,
		Currency:         "USD",
		GeneratedAt:      time.Now(),
		ResourceCosts:    []models.CostEstimate{},
	}

	output, err := formatter.Format(result)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if !strings.Contains(output, "estimationResult:") {
		t.Errorf("Expected YAML structure even for empty result")
	}

	if !strings.Contains(output, "resourceCosts:") {
		t.Errorf("Expected resourceCosts section even for empty result")
	}
}

func TestFormatterFactory_WriteFormattedResult_Error(t *testing.T) {
	factory := NewFormatterFactory()
	result := createTestEstimationResult()

	var buffer bytes.Buffer
	err := factory.WriteFormattedResult(&buffer, result, "unsupported")

	if err == nil {
		t.Errorf("Expected error for unsupported format, but got none")
	}
}
