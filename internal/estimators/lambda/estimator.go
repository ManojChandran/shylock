package lambda

import (
	"context"
	"fmt"
	"time"

	"shylock/internal/aws"
	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// Estimator implements the ResourceEstimator interface for AWS Lambda functions
type Estimator struct {
	pricingService *aws.PricingService
}

// NewEstimator creates a new Lambda cost estimator
func NewEstimator(awsClient interfaces.AWSPricingClient) interfaces.ResourceEstimator {
	return &Estimator{
		pricingService: aws.NewPricingService(awsClient),
	}
}

// SupportedResourceType returns the AWS resource type this estimator supports
func (e *Estimator) SupportedResourceType() string {
	return "Lambda"
}

// ValidateResource validates that the resource specification is valid for Lambda
func (e *Estimator) ValidateResource(resource models.ResourceSpec) error {
	if resource.Type != "Lambda" {
		return errors.ValidationError("resource type must be 'Lambda'").
			WithContext("actualType", resource.Type).
			WithSuggestion("Use 'Lambda' as the resource type")
	}

	// Validate required properties
	requiredProps := []string{"memoryMB"}
	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return errors.ValidationError(fmt.Sprintf("missing required property '%s'", prop)).
				WithContext("resourceName", resource.Name).
				WithSuggestion(fmt.Sprintf("Add '%s' property to the resource configuration", prop))
		}
	}

	// Validate memory configuration
	memoryMB, err := resource.GetIntProperty("memoryMB")
	if err != nil {
		return errors.ValidationErrorWithCause("invalid memoryMB property", err).
			WithContext("resourceName", resource.Name).
			WithSuggestion("Ensure memoryMB is an integer between 128 and 10240")
	}

	if !e.isValidMemorySize(memoryMB) {
		return errors.ValidationError("invalid Lambda memory size").
			WithContext("resourceName", resource.Name).
			WithContext("memoryMB", memoryMB).
			WithSuggestion("Memory must be between 128 MB and 10,240 MB (10 GB)").
			WithSuggestion("Memory must be in 1 MB increments")
	}

	// Validate optional numeric properties
	numericProps := []string{"requestsPerMonth", "averageDurationMs", "storageGB"}
	for _, prop := range numericProps {
		if _, exists := resource.GetProperty(prop); exists {
			value, err := resource.GetIntProperty(prop)
			if err != nil {
				return errors.ValidationErrorWithCause(fmt.Sprintf("invalid %s property", prop), err).
					WithContext("resourceName", resource.Name).
					WithSuggestion(fmt.Sprintf("Ensure %s is a non-negative integer", prop))
			}
			if value < 0 {
				return errors.ValidationError(fmt.Sprintf("%s cannot be negative", prop)).
					WithContext("resourceName", resource.Name).
					WithContext(prop, value).
					WithSuggestion(fmt.Sprintf("Set %s to a non-negative integer", prop))
			}
		}
	}

	// Validate architecture if specified
	if _, exists := resource.GetProperty("architecture"); exists {
		architecture, err := resource.GetStringProperty("architecture")
		if err != nil {
			return errors.ValidationErrorWithCause("invalid architecture property", err).
				WithContext("resourceName", resource.Name).
				WithSuggestion("Ensure architecture is a string ('x86_64' or 'arm64')")
		}

		if !e.isValidArchitecture(architecture) {
			return errors.ValidationError("invalid Lambda architecture").
				WithContext("resourceName", resource.Name).
				WithContext("architecture", architecture).
				WithSuggestion("Use 'x86_64' or 'arm64' as architecture")
		}
	}

	// Validate region
	if err := e.pricingService.ValidateRegion(resource.Region); err != nil {
		return errors.WrapError(err, errors.ValidationErrorType, "invalid region for Lambda resource").
			WithContext("resourceName", resource.Name)
	}

	return nil
}

// EstimateCost calculates the cost for a Lambda function
func (e *Estimator) EstimateCost(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error) {
	// Validate the resource first
	if err := e.ValidateResource(resource); err != nil {
		return nil, err
	}

	// Extract properties
	memoryMB, _ := resource.GetIntProperty("memoryMB")

	// Get optional properties with defaults
	requestsPerMonth := 1000000 // Default 1M requests per month
	if _, exists := resource.GetProperty("requestsPerMonth"); exists {
		if rpm, err := resource.GetIntProperty("requestsPerMonth"); err == nil {
			requestsPerMonth = rpm
		}
	}

	averageDurationMs := 100 // Default 100ms duration
	if _, exists := resource.GetProperty("averageDurationMs"); exists {
		if adm, err := resource.GetIntProperty("averageDurationMs"); err == nil {
			averageDurationMs = adm
		}
	}

	storageGB := 0 // Default no additional storage
	if _, exists := resource.GetProperty("storageGB"); exists {
		if sg, err := resource.GetIntProperty("storageGB"); err == nil {
			storageGB = sg
		}
	}

	architecture := "x86_64" // Default architecture
	if _, exists := resource.GetProperty("architecture"); exists {
		if arch, err := resource.GetStringProperty("architecture"); err == nil {
			architecture = arch
		}
	}

	// Get pricing data from AWS
	products, err := e.pricingService.GetLambdaPricing(ctx, resource.Region, architecture)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve Lambda pricing data").
			WithContext("resourceName", resource.Name).
			WithContext("region", resource.Region).
			WithContext("architecture", architecture)
	}

	// Calculate costs
	totalHourlyCost, costBreakdown, err := e.calculateLambdaCosts(products, memoryMB, requestsPerMonth, averageDurationMs, storageGB)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to calculate Lambda costs").
			WithContext("resourceName", resource.Name)
	}

	// Create cost estimate
	estimate := &models.CostEstimate{
		ResourceName: resource.Name,
		ResourceType: resource.Type,
		Region:       resource.Region,
		HourlyCost:   totalHourlyCost,
		Currency:     "USD",
		Timestamp:    time.Now(),
	}

	// Calculate daily and monthly costs
	estimate.CalculateCosts()

	// Add assumptions
	estimate.AddAssumption("Pay-per-use pricing model")
	estimate.AddAssumption("Costs calculated based on requests, duration, and memory allocation")
	if requestsPerMonth == 1000000 {
		estimate.AddAssumption("Default 1M requests per month (set requestsPerMonth for accurate pricing)")
	}
	if averageDurationMs == 100 {
		estimate.AddAssumption("Default 100ms average duration (set averageDurationMs for accurate pricing)")
	}
	if storageGB == 0 {
		estimate.AddAssumption("No additional storage costs (set storageGB for EFS/S3 storage)")
	}

	// Add details
	estimate.SetDetail("memoryMB", fmt.Sprintf("%d", memoryMB))
	estimate.SetDetail("requestsPerMonth", fmt.Sprintf("%d", requestsPerMonth))
	estimate.SetDetail("averageDurationMs", fmt.Sprintf("%d", averageDurationMs))
	estimate.SetDetail("storageGB", fmt.Sprintf("%d", storageGB))
	estimate.SetDetail("architecture", architecture)

	// Add cost breakdown details
	for component, cost := range costBreakdown {
		estimate.SetDetail(fmt.Sprintf("%sCost", component), fmt.Sprintf("$%.6f/hour", cost))
	}

	return estimate, nil
}

// calculateLambdaCosts calculates Lambda costs based on requests, duration, and memory
func (e *Estimator) calculateLambdaCosts(products []interfaces.PricingProduct, memoryMB, requestsPerMonth, averageDurationMs, storageGB int) (float64, map[string]float64, error) {
	costBreakdown := make(map[string]float64)

	// Find pricing components
	var requestPrice, computePrice, storagePrice float64

	for _, product := range products {
		// Extract pricing based on usage type
		if usageType, exists := product.Attributes["usageType"]; exists {
			price, err := e.pricingService.ExtractHourlyPrice(product)
			if err != nil {
				continue // Skip products we can't parse
			}

			if e.isRequestUsage(usageType) {
				requestPrice = price
			} else if e.isComputeUsage(usageType) {
				computePrice = price
			} else if e.isStorageUsage(usageType) {
				storagePrice = price
			}
		}
	}

	// Calculate request costs (per million requests)
	requestsPerHour := float64(requestsPerMonth) / (24 * 30)
	requestCostPerHour := (requestsPerHour / 1000000) * requestPrice
	costBreakdown["request"] = requestCostPerHour

	// Calculate compute costs (GB-seconds)
	memoryGB := float64(memoryMB) / 1024
	durationSeconds := float64(averageDurationMs) / 1000
	gbSecondsPerRequest := memoryGB * durationSeconds
	gbSecondsPerHour := gbSecondsPerRequest * requestsPerHour
	computeCostPerHour := gbSecondsPerHour * computePrice
	costBreakdown["compute"] = computeCostPerHour

	// Calculate storage costs if applicable
	var storageCostPerHour float64
	if storageGB > 0 {
		// Storage pricing is typically per GB-month, convert to hourly
		storageMonthlyPrice := storagePrice * float64(storageGB)
		storageCostPerHour = storageMonthlyPrice / (24 * 30)
		costBreakdown["storage"] = storageCostPerHour
	}

	totalCost := requestCostPerHour + computeCostPerHour + storageCostPerHour

	return totalCost, costBreakdown, nil
}

// Helper functions

func (e *Estimator) isValidMemorySize(memoryMB int) bool {
	// Lambda memory must be between 128 MB and 10,240 MB (10 GB)
	// and must be in 1 MB increments
	return memoryMB >= 128 && memoryMB <= 10240
}

func (e *Estimator) isValidArchitecture(architecture string) bool {
	validArchitectures := map[string]bool{
		"x86_64": true,
		"arm64":  true,
	}
	return validArchitectures[architecture]
}

func (e *Estimator) isRequestUsage(usageType string) bool {
	// Lambda request usage types typically contain "Request"
	return containsSubstring(usageType, "Request")
}

func (e *Estimator) isComputeUsage(usageType string) bool {
	// Lambda compute usage types typically contain "Duration" or "GB-Second"
	return containsSubstring(usageType, "Duration") || containsSubstring(usageType, "GB-Second")
}

func (e *Estimator) isStorageUsage(usageType string) bool {
	// Lambda storage usage types typically contain "Storage"
	return containsSubstring(usageType, "Storage")
}

// GetSupportedMemorySizes returns common Lambda memory configurations
func (e *Estimator) GetSupportedMemorySizes() []int {
	return []int{
		128, 256, 512, 1024, 1536, 2048, 3008, 4096, 5120, 6144, 7168, 8192, 9216, 10240,
	}
}

// GetSupportedArchitectures returns supported Lambda architectures
func (e *Estimator) GetSupportedArchitectures() []string {
	return []string{"x86_64", "arm64"}
}

// GetArchitectureDescription returns description for Lambda architectures
func (e *Estimator) GetArchitectureDescription(architecture string) string {
	descriptions := map[string]string{
		"x86_64": "Intel/AMD 64-bit architecture (default)",
		"arm64":  "ARM-based Graviton2 processors (up to 34% better price performance)",
	}
	if desc, exists := descriptions[architecture]; exists {
		return desc
	}
	return "Unknown architecture"
}

// GetTypicalUseCases returns typical use cases for Lambda
func (e *Estimator) GetTypicalUseCases() []string {
	return []string{
		"API backends and microservices",
		"Event-driven data processing",
		"Real-time file processing",
		"Scheduled tasks and cron jobs",
		"IoT data processing",
		"Image and video processing",
		"ETL operations",
		"Chatbots and voice assistants",
	}
}

// Helper function
func containsSubstring(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
