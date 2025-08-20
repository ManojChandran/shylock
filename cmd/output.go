package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"shylock/internal/models"
)

// outputTable formats results as a human-readable table
func outputTable(result *models.EstimationResult) error {
	fmt.Println("AWS Cost Estimation Results")
	fmt.Println("===========================")
	fmt.Printf("Generated: %s\n", result.GeneratedAt.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Currency: %s\n\n", result.Currency)

	// Summary section
	fmt.Println("💰 Cost Summary")
	fmt.Println("---------------")
	fmt.Printf("Hourly Cost:  $%.4f\n", result.TotalHourlyCost)
	fmt.Printf("Daily Cost:   $%.4f\n", result.TotalDailyCost)
	fmt.Printf("Monthly Cost: $%.4f\n\n", result.TotalMonthlyCost)

	if len(result.ResourceCosts) == 0 {
		fmt.Println("No resources found in estimation.")
		return nil
	}

	// Resource breakdown
	fmt.Println("📊 Resource Breakdown")
	fmt.Println("--------------------")

	// Calculate column widths
	maxNameWidth := 12  // "Resource Name"
	maxTypeWidth := 4   // "Type"
	maxRegionWidth := 6 // "Region"

	for _, cost := range result.ResourceCosts {
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
	maxNameWidth += 2
	maxTypeWidth += 2
	maxRegionWidth += 2

	// Print header
	headerFormat := fmt.Sprintf("%%-%ds %%-%ds %%-%ds %%12s %%12s %%12s\n",
		maxNameWidth, maxTypeWidth, maxRegionWidth)
	fmt.Printf(headerFormat, "Resource Name", "Type", "Region", "Hourly", "Daily", "Monthly")

	// Print separator
	separator := strings.Repeat("-", maxNameWidth+maxTypeWidth+maxRegionWidth+36+6)
	fmt.Println(separator)

	// Print resource rows
	rowFormat := fmt.Sprintf("%%-%ds %%-%ds %%-%ds $%%10.4f $%%10.4f $%%10.4f\n",
		maxNameWidth, maxTypeWidth, maxRegionWidth)

	for _, cost := range result.ResourceCosts {
		fmt.Printf(rowFormat,
			cost.ResourceName,
			cost.ResourceType,
			cost.Region,
			cost.HourlyCost,
			cost.DailyCost,
			cost.MonthlyCost)
	}

	// Show detailed information if verbose
	if verbose {
		fmt.Println("\n🔍 Detailed Information")
		fmt.Println("----------------------")

		for i, cost := range result.ResourceCosts {
			fmt.Printf("\n%d. %s (%s)\n", i+1, cost.ResourceName, cost.ResourceType)

			// Show assumptions
			if len(cost.Assumptions) > 0 {
				fmt.Println("   Assumptions:")
				for _, assumption := range cost.Assumptions {
					fmt.Printf("   • %s\n", assumption)
				}
			}

			// Show key details
			if len(cost.Details) > 0 {
				fmt.Println("   Configuration:")
				for key, value := range cost.Details {
					if isImportantDetail(key) {
						fmt.Printf("   • %s: %s\n", formatDetailKey(key), value)
					}
				}
			}
		}
	}

	return nil
}

// outputJSON formats results as JSON
func outputJSON(result *models.EstimationResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// outputCSV formats results as CSV
func outputCSV(result *models.EstimationResult) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

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
		return fmt.Errorf("failed to write CSV header: %w", err)
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
			return fmt.Errorf("failed to write CSV row: %w", err)
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
		return fmt.Errorf("failed to write CSV summary: %w", err)
	}

	return nil
}

// Helper functions for table formatting

func isImportantDetail(key string) bool {
	importantKeys := map[string]bool{
		"instanceType":     true,
		"count":            true,
		"operatingSystem":  true,
		"storageClass":     true,
		"sizeGB":           true,
		"requestsPerMonth": true,
		"tenancy":          true,
	}
	return importantKeys[key]
}

func formatDetailKey(key string) string {
	keyMap := map[string]string{
		"instanceType":     "Instance Type",
		"count":            "Count",
		"operatingSystem":  "Operating System",
		"storageClass":     "Storage Class",
		"sizeGB":           "Size (GB)",
		"requestsPerMonth": "Requests/Month",
		"tenancy":          "Tenancy",
	}

	if formatted, exists := keyMap[key]; exists {
		return formatted
	}

	// Default formatting: capitalize first letter and add spaces before capitals
	result := strings.ToUpper(string(key[0])) + key[1:]
	for i := 1; i < len(result); i++ {
		if result[i] >= 'A' && result[i] <= 'Z' {
			result = result[:i] + " " + result[i:]
			i++ // Skip the inserted space
		}
	}
	return result
}
