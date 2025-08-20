package output

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// Formatter defines the interface for output formatters
type Formatter interface {
	Format(result *models.EstimationResult, options *FormatOptions) (string, error)
	FormatType() string
}

// FormatOptions contains options for formatting output
type FormatOptions struct {
	Verbose         bool
	ShowDetails     bool
	ShowAssumptions bool
	Currency        string
	Precision       int
	SortBy          string // "name", "type", "cost", "region"
	GroupBy         string // "type", "region", "none"
}

// DefaultFormatOptions returns default formatting options
func DefaultFormatOptions() *FormatOptions {
	return &FormatOptions{
		Verbose:         false,
		ShowDetails:     true,
		ShowAssumptions: true,
		Currency:        "USD",
		Precision:       4,
		SortBy:          "name",
		GroupBy:         "none",
	}
}

// TableFormatter formats output as a human-readable table
type TableFormatter struct{}

// NewTableFormatter creates a new table formatter
func NewTableFormatter() interfaces.OutputFormatter {
	return &TableFormatter{}
}

// FormatType returns the format type
func (f *TableFormatter) FormatType() string {
	return "table"
}

// Format formats the estimation result as a table
func (f *TableFormatter) Format(result *models.EstimationResult) (string, error) {
	options := DefaultFormatOptions()
	return f.FormatWithOptions(result, options)
}

// FormatWithOptions formats with specific options
func (f *TableFormatter) FormatWithOptions(result *models.EstimationResult, options *FormatOptions) (string, error) {
	var output strings.Builder

	// Header
	output.WriteString("AWS Cost Estimation Results\n")
	output.WriteString("===========================\n")
	output.WriteString(fmt.Sprintf("Generated: %s\n", result.GeneratedAt.Format("2006-01-02 15:04:05 MST")))
	output.WriteString(fmt.Sprintf("Currency: %s\n\n", result.Currency))

	// Summary section
	output.WriteString("💰 Cost Summary\n")
	output.WriteString("---------------\n")
	output.WriteString(fmt.Sprintf("Hourly Cost:  $%.*f\n", options.Precision, result.TotalHourlyCost))
	output.WriteString(fmt.Sprintf("Daily Cost:   $%.*f\n", options.Precision, result.TotalDailyCost))
	output.WriteString(fmt.Sprintf("Monthly Cost: $%.*f\n\n", options.Precision, result.TotalMonthlyCost))

	if len(result.ResourceCosts) == 0 {
		output.WriteString("No resources found in estimation.\n")
		return output.String(), nil
	}

	// Sort resources if requested
	sortedCosts := make([]models.CostEstimate, len(result.ResourceCosts))
	copy(sortedCosts, result.ResourceCosts)
	f.sortResources(sortedCosts, options.SortBy)

	// Group resources if requested
	if options.GroupBy != "none" {
		return f.formatGrouped(result, sortedCosts, options)
	}

	// Resource breakdown
	output.WriteString("📊 Resource Breakdown\n")
	output.WriteString("--------------------\n")

	// Calculate column widths
	maxNameWidth, maxTypeWidth, maxRegionWidth := f.calculateColumnWidths(sortedCosts)

	// Print header
	headerFormat := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%12s %%12s %%12s\n",
		maxNameWidth, maxTypeWidth, maxRegionWidth)
	output.WriteString(fmt.Sprintf(headerFormat, "Resource Name", "Type", "Region", "Hourly", "Daily", "Monthly"))

	// Print separator
	separator := strings.Repeat("-", maxNameWidth+maxTypeWidth+maxRegionWidth+36+6)
	output.WriteString(separator + "\n")

	// Print resource rows
	rowFormat := fmt.Sprintf("%%-%ds %%-%ds %%-%ds $%%10.*f $%%10.*f $%%10.*f\n",
		maxNameWidth, maxTypeWidth, maxRegionWidth)

	for _, cost := range sortedCosts {
		output.WriteString(fmt.Sprintf(rowFormat,
			cost.ResourceName,
			cost.ResourceType,
			cost.Region,
			options.Precision, cost.HourlyCost,
			options.Precision, cost.DailyCost,
			options.Precision, cost.MonthlyCost))
	}

	// Show detailed information if verbose
	if options.Verbose {
		output.WriteString(f.formatDetailedInfo(sortedCosts, options))
	}

	return output.String(), nil
}

// formatGrouped formats resources grouped by the specified field
func (f *TableFormatter) formatGrouped(result *models.EstimationResult, costs []models.CostEstimate, options *FormatOptions) (string, error) {
	var output strings.Builder

	// Header (same as regular format)
	output.WriteString("AWS Cost Estimation Results\n")
	output.WriteString("===========================\n")
	output.WriteString(fmt.Sprintf("Generated: %s\n", result.GeneratedAt.Format("2006-01-02 15:04:05 MST")))
	output.WriteString(fmt.Sprintf("Currency: %s\n\n", result.Currency))

	// Summary section
	output.WriteString("💰 Cost Summary\n")
	output.WriteString("---------------\n")
	output.WriteString(fmt.Sprintf("Hourly Cost:  $%.*f\n", options.Precision, result.TotalHourlyCost))
	output.WriteString(fmt.Sprintf("Daily Cost:   $%.*f\n", options.Precision, result.TotalDailyCost))
	output.WriteString(fmt.Sprintf("Monthly Cost: $%.*f\n\n", options.Precision, result.TotalMonthlyCost))

	// Group resources
	groups := f.groupResources(costs, options.GroupBy)

	output.WriteString("📊 Resource Breakdown (Grouped)\n")
	output.WriteString("-------------------------------\n")

	for groupName, groupCosts := range groups {
		// Group header
		groupTotal := f.calculateGroupTotal(groupCosts)
		output.WriteString(fmt.Sprintf("\n🏷️  %s (%d resources) - Monthly: $%.*f\n",
			groupName, len(groupCosts), options.Precision, groupTotal))
		output.WriteString(strings.Repeat("-", 50) + "\n")

		// Calculate column widths for this group
		maxNameWidth, maxTypeWidth, maxRegionWidth := f.calculateColumnWidths(groupCosts)

		// Print group resources
		rowFormat := fmt.Sprintf("  %%-%ds %%-%ds %%-%ds $%%8.*f $%%8.*f $%%8.*f\n",
			maxNameWidth-2, maxTypeWidth, maxRegionWidth)

		for _, cost := range groupCosts {
			output.WriteString(fmt.Sprintf(rowFormat,
				cost.ResourceName,
				cost.ResourceType,
				cost.Region,
				options.Precision, cost.HourlyCost,
				options.Precision, cost.DailyCost,
				options.Precision, cost.MonthlyCost))
		}
	}

	return output.String(), nil
}

// formatDetailedInfo formats detailed information for verbose mode
func (f *TableFormatter) formatDetailedInfo(costs []models.CostEstimate, options *FormatOptions) string {
	var output strings.Builder

	output.WriteString("\n🔍 Detailed Information\n")
	output.WriteString("----------------------\n")

	for i, cost := range costs {
		output.WriteString(fmt.Sprintf("\n%d. %s (%s)\n", i+1, cost.ResourceName, cost.ResourceType))

		// Show assumptions if enabled
		if options.ShowAssumptions && len(cost.Assumptions) > 0 {
			output.WriteString("   Assumptions:\n")
			for _, assumption := range cost.Assumptions {
				output.WriteString(fmt.Sprintf("   • %s\n", assumption))
			}
		}

		// Show key details if enabled
		if options.ShowDetails && len(cost.Details) > 0 {
			output.WriteString("   Configuration:\n")
			for key, value := range cost.Details {
				if f.isImportantDetail(key) {
					output.WriteString(fmt.Sprintf("   • %s: %s\n", f.formatDetailKey(key), value))
				}
			}
		}
	}

	return output.String()
}

// Helper methods for TableFormatter

func (f *TableFormatter) calculateColumnWidths(costs []models.CostEstimate) (int, int, int) {
	maxNameWidth := 12  // "Resource Name"
	maxTypeWidth := 4   // "Type"
	maxRegionWidth := 6 // "Region"

	for _, cost := range costs {
		if len(cost.ResourceName) > maxNameWidth {
			maxNameWidth = len(cost.ResourceName)
		}
		if len(cost.ResourceType) > maxTypeWidth {
			maxTypeWidth = len(cost.ResourceType)
		}
		if len(cost.Region) > maxRegionWidth {
			maxRegionWidth = len(cost.Region)
		}
	}

	// Add padding
	return maxNameWidth + 2, maxTypeWidth + 2, maxRegionWidth + 2
}

func (f *TableFormatter) sortResources(costs []models.CostEstimate, sortBy string) {
	switch sortBy {
	case "name":
		sort.Slice(costs, func(i, j int) bool {
			return costs[i].ResourceName < costs[j].ResourceName
		})
	case "type":
		sort.Slice(costs, func(i, j int) bool {
			if costs[i].ResourceType == costs[j].ResourceType {
				return costs[i].ResourceName < costs[j].ResourceName
			}
			return costs[i].ResourceType < costs[j].ResourceType
		})
	case "cost":
		sort.Slice(costs, func(i, j int) bool {
			return costs[i].MonthlyCost > costs[j].MonthlyCost // Descending by cost
		})
	case "region":
		sort.Slice(costs, func(i, j int) bool {
			if costs[i].Region == costs[j].Region {
				return costs[i].ResourceName < costs[j].ResourceName
			}
			return costs[i].Region < costs[j].Region
		})
	}
}

func (f *TableFormatter) groupResources(costs []models.CostEstimate, groupBy string) map[string][]models.CostEstimate {
	groups := make(map[string][]models.CostEstimate)

	for _, cost := range costs {
		var key string
		switch groupBy {
		case "type":
			key = cost.ResourceType
		case "region":
			key = cost.Region
		default:
			key = "All Resources"
		}

		groups[key] = append(groups[key], cost)
	}

	// Sort within each group
	for _, groupCosts := range groups {
		sort.Slice(groupCosts, func(i, j int) bool {
			return groupCosts[i].ResourceName < groupCosts[j].ResourceName
		})
	}

	return groups
}

func (f *TableFormatter) calculateGroupTotal(costs []models.CostEstimate) float64 {
	total := 0.0
	for _, cost := range costs {
		total += cost.MonthlyCost
	}
	return total
}

func (f *TableFormatter) isImportantDetail(key string) bool {
	importantKeys := map[string]bool{
		"instanceType":     true,
		"count":            true,
		"operatingSystem":  true,
		"storageClass":     true,
		"sizeGB":           true,
		"requestsPerMonth": true,
		"tenancy":          true,
		"pricePerInstance": true,
		"storagePrice":     true,
	}
	return importantKeys[key]
}

func (f *TableFormatter) formatDetailKey(key string) string {
	keyMap := map[string]string{
		"instanceType":     "Instance Type",
		"count":            "Count",
		"operatingSystem":  "Operating System",
		"storageClass":     "Storage Class",
		"sizeGB":           "Size (GB)",
		"requestsPerMonth": "Requests/Month",
		"tenancy":          "Tenancy",
		"pricePerInstance": "Price per Instance",
		"storagePrice":     "Storage Price",
	}

	if formatted, exists := keyMap[key]; exists {
		return formatted
	}

	// Default formatting: capitalize first letter and add spaces before capitals
	if len(key) == 0 {
		return ""
	}
	result := strings.ToUpper(string(key[0])) + key[1:]
	for i := 1; i < len(result); i++ {
		if result[i] >= 'A' && result[i] <= 'Z' {
			result = result[:i] + " " + result[i:]
			i++ // Skip the inserted space
		}
	}
	return result
}

// JSONFormatter formats output as JSON
type JSONFormatter struct{}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() interfaces.OutputFormatter {
	return &JSONFormatter{}
}

// FormatType returns the format type
func (f *JSONFormatter) FormatType() string {
	return "json"
}

// Format formats the estimation result as JSON
func (f *JSONFormatter) Format(result *models.EstimationResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// CSVFormatter formats output as CSV
type CSVFormatter struct{}

// NewCSVFormatter creates a new CSV formatter
func NewCSVFormatter() interfaces.OutputFormatter {
	return &CSVFormatter{}
}

// FormatType returns the format type
func (f *CSVFormatter) FormatType() string {
	return "csv"
}

// Format formats the estimation result as CSV
func (f *CSVFormatter) Format(result *models.EstimationResult) (string, error) {
	var output strings.Builder
	writer := csv.NewWriter(&output)

	// Write header
	header := []string{
		"Resource Name",
		"Resource Type",
		"Region",
		"Hourly Cost",
		"Daily Cost",
		"Monthly Cost",
		"Currency",
		"Generated At",
	}
	if err := writer.Write(header); err != nil {
		return "", fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write resource rows
	for _, cost := range result.ResourceCosts {
		row := []string{
			cost.ResourceName,
			cost.ResourceType,
			cost.Region,
			fmt.Sprintf("%.4f", cost.HourlyCost),
			fmt.Sprintf("%.4f", cost.DailyCost),
			fmt.Sprintf("%.4f", cost.MonthlyCost),
			cost.Currency,
			cost.Timestamp.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return "", fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	// Write summary row
	summaryRow := []string{
		"TOTAL",
		"",
		"",
		fmt.Sprintf("%.4f", result.TotalHourlyCost),
		fmt.Sprintf("%.4f", result.TotalDailyCost),
		fmt.Sprintf("%.4f", result.TotalMonthlyCost),
		result.Currency,
		result.GeneratedAt.Format(time.RFC3339),
	}
	if err := writer.Write(summaryRow); err != nil {
		return "", fmt.Errorf("failed to write CSV summary: %w", err)
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", fmt.Errorf("CSV writer error: %w", err)
	}

	return output.String(), nil
}

// YAMLFormatter formats output as YAML
type YAMLFormatter struct{}

// NewYAMLFormatter creates a new YAML formatter
func NewYAMLFormatter() interfaces.OutputFormatter {
	return &YAMLFormatter{}
}

// FormatType returns the format type
func (f *YAMLFormatter) FormatType() string {
	return "yaml"
}

// Format formats the estimation result as YAML
func (f *YAMLFormatter) Format(result *models.EstimationResult) (string, error) {
	// Simple YAML implementation without external dependencies
	var output strings.Builder

	output.WriteString("estimationResult:\n")
	output.WriteString(fmt.Sprintf("  totalHourlyCost: %.4f\n", result.TotalHourlyCost))
	output.WriteString(fmt.Sprintf("  totalDailyCost: %.4f\n", result.TotalDailyCost))
	output.WriteString(fmt.Sprintf("  totalMonthlyCost: %.4f\n", result.TotalMonthlyCost))
	output.WriteString(fmt.Sprintf("  currency: %s\n", result.Currency))
	output.WriteString(fmt.Sprintf("  generatedAt: %s\n", result.GeneratedAt.Format(time.RFC3339)))
	output.WriteString("  resourceCosts:\n")

	for _, cost := range result.ResourceCosts {
		output.WriteString(fmt.Sprintf("    - resourceName: %s\n", cost.ResourceName))
		output.WriteString(fmt.Sprintf("      resourceType: %s\n", cost.ResourceType))
		output.WriteString(fmt.Sprintf("      region: %s\n", cost.Region))
		output.WriteString(fmt.Sprintf("      hourlyCost: %.4f\n", cost.HourlyCost))
		output.WriteString(fmt.Sprintf("      dailyCost: %.4f\n", cost.DailyCost))
		output.WriteString(fmt.Sprintf("      monthlyCost: %.4f\n", cost.MonthlyCost))
		output.WriteString(fmt.Sprintf("      currency: %s\n", cost.Currency))
		output.WriteString(fmt.Sprintf("      timestamp: %s\n", cost.Timestamp.Format(time.RFC3339)))

		if len(cost.Assumptions) > 0 {
			output.WriteString("      assumptions:\n")
			for _, assumption := range cost.Assumptions {
				output.WriteString(fmt.Sprintf("        - %s\n", assumption))
			}
		}
	}

	return output.String(), nil
}

// FormatterFactory creates formatters based on type
type FormatterFactory struct {
	formatters map[string]interfaces.OutputFormatter
}

// NewFormatterFactory creates a new formatter factory
func NewFormatterFactory() *FormatterFactory {
	factory := &FormatterFactory{
		formatters: make(map[string]interfaces.OutputFormatter),
	}

	// Register built-in formatters
	factory.RegisterFormatter(NewTableFormatter())
	factory.RegisterFormatter(NewJSONFormatter())
	factory.RegisterFormatter(NewCSVFormatter())
	factory.RegisterFormatter(NewYAMLFormatter())

	return factory
}

// RegisterFormatter registers a new formatter
func (f *FormatterFactory) RegisterFormatter(formatter interfaces.OutputFormatter) {
	f.formatters[formatter.FormatType()] = formatter
}

// GetFormatter returns a formatter by type
func (f *FormatterFactory) GetFormatter(formatType string) (interfaces.OutputFormatter, error) {
	formatter, exists := f.formatters[formatType]
	if !exists {
		available := make([]string, 0, len(f.formatters))
		for fType := range f.formatters {
			available = append(available, fType)
		}
		return nil, fmt.Errorf("unsupported format type '%s'. Available formats: %s",
			formatType, strings.Join(available, ", "))
	}
	return formatter, nil
}

// GetSupportedFormats returns a list of supported format types
func (f *FormatterFactory) GetSupportedFormats() []string {
	formats := make([]string, 0, len(f.formatters))
	for formatType := range f.formatters {
		formats = append(formats, formatType)
	}
	sort.Strings(formats)
	return formats
}

// FormatResult formats a result using the specified formatter
func (f *FormatterFactory) FormatResult(result *models.EstimationResult, formatType string) (string, error) {
	formatter, err := f.GetFormatter(formatType)
	if err != nil {
		return "", err
	}
	return formatter.Format(result)
}

// WriteFormattedResult writes formatted result to a writer
func (f *FormatterFactory) WriteFormattedResult(writer io.Writer, result *models.EstimationResult, formatType string) error {
	formatted, err := f.FormatResult(result, formatType)
	if err != nil {
		return err
	}

	_, err = writer.Write([]byte(formatted))
	return err
}
