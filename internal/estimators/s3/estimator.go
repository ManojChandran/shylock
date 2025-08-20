package s3

import (
	"context"
	"fmt"
	"time"

	"shylock/internal/aws"
	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// Estimator implements the ResourceEstimator interface for S3 storage
type Estimator struct {
	pricingService *aws.PricingService
}

// NewEstimator creates a new S3 cost estimator
func NewEstimator(awsClient interfaces.AWSPricingClient) interfaces.ResourceEstimator {
	return &Estimator{
		pricingService: aws.NewPricingService(awsClient),
	}
}

// SupportedResourceType returns the AWS resource type this estimator supports
func (e *Estimator) SupportedResourceType() string {
	return "S3"
}

// ValidateResource validates that the resource specification is valid for S3
func (e *Estimator) ValidateResource(resource models.ResourceSpec) error {
	if resource.Type != "S3" {
		return errors.ValidationError("resource type must be 'S3'").
			WithContext("actualType", resource.Type).
			WithSuggestion("Use 'S3' as the resource type")
	}

	// Validate required properties
	requiredProps := []string{"storageClass"}
	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return errors.ValidationError(fmt.Sprintf("missing required property '%s'", prop)).
				WithContext("resourceName", resource.Name).
				WithSuggestion(fmt.Sprintf("Add '%s' property to the resource configuration", prop))
		}
	}

	// Validate storage class
	storageClass, err := resource.GetStringProperty("storageClass")
	if err != nil {
		return errors.ValidationErrorWithCause("invalid storageClass property", err).
			WithContext("resourceName", resource.Name).
			WithSuggestion("Ensure storageClass is a string (e.g., 'STANDARD', 'STANDARD_IA')")
	}

	if !e.isValidStorageClass(storageClass) {
		return errors.ValidationError("invalid storage class").
			WithContext("resourceName", resource.Name).
			WithContext("storageClass", storageClass).
			WithSuggestion("Use valid S3 storage class: STANDARD, STANDARD_IA, ONEZONE_IA, GLACIER, DEEP_ARCHIVE").
			WithSuggestion("Check AWS S3 documentation for available storage classes")
	}

	// Validate optional sizeGB property
	if _, exists := resource.GetProperty("sizeGB"); exists {
		sizeGB, err := resource.GetIntProperty("sizeGB")
		if err != nil {
			return errors.ValidationErrorWithCause("invalid sizeGB property", err).
				WithContext("resourceName", resource.Name).
				WithSuggestion("Ensure sizeGB is a positive integer")
		}
		if sizeGB <= 0 {
			return errors.ValidationError("sizeGB must be greater than 0").
				WithContext("resourceName", resource.Name).
				WithContext("sizeGB", sizeGB).
				WithSuggestion("Set sizeGB to a positive integer (in GB)")
		}
	}

	// Validate optional requestsPerMonth property
	if _, exists := resource.GetProperty("requestsPerMonth"); exists {
		requests, err := resource.GetIntProperty("requestsPerMonth")
		if err != nil {
			return errors.ValidationErrorWithCause("invalid requestsPerMonth property", err).
				WithContext("resourceName", resource.Name).
				WithSuggestion("Ensure requestsPerMonth is a positive integer")
		}
		if requests < 0 {
			return errors.ValidationError("requestsPerMonth cannot be negative").
				WithContext("resourceName", resource.Name).
				WithContext("requestsPerMonth", requests).
				WithSuggestion("Set requestsPerMonth to a non-negative integer")
		}
	}

	// Validate region
	if err := e.pricingService.ValidateRegion(resource.Region); err != nil {
		return errors.WrapError(err, errors.ValidationErrorType, "invalid region for S3 resource").
			WithContext("resourceName", resource.Name)
	}

	return nil
}

// EstimateCost calculates the cost for S3 storage
func (e *Estimator) EstimateCost(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error) {
	// Validate the resource first
	if err := e.ValidateResource(resource); err != nil {
		return nil, err
	}

	// Extract properties
	storageClass, _ := resource.GetStringProperty("storageClass")

	// Get optional properties with defaults
	sizeGB := 1 // Default to 1 GB if not specified
	if _, exists := resource.GetProperty("sizeGB"); exists {
		if size, err := resource.GetIntProperty("sizeGB"); err == nil {
			sizeGB = size
		}
	}

	requestsPerMonth := 0 // Default to 0 requests
	if _, exists := resource.GetProperty("requestsPerMonth"); exists {
		if requests, err := resource.GetIntProperty("requestsPerMonth"); err == nil {
			requestsPerMonth = requests
		}
	}

	// Get pricing data from AWS
	products, err := e.pricingService.GetS3Pricing(ctx, storageClass, resource.Region)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve S3 pricing data").
			WithContext("resourceName", resource.Name).
			WithContext("storageClass", storageClass).
			WithContext("region", resource.Region)
	}

	// Find storage pricing (per GB-month)
	var storageProduct *interfaces.PricingProduct
	var requestProduct *interfaces.PricingProduct

	for _, product := range products {
		// Look for storage pricing
		if usageType, exists := product.Attributes["usageType"]; exists {
			if e.isStorageUsageType(usageType) {
				storageProduct = &product
			} else if e.isRequestUsageType(usageType) && requestsPerMonth > 0 {
				requestProduct = &product
			}
		}
		// If we don't have storage product yet, use the first one as fallback
		if storageProduct == nil {
			storageProduct = &product
		}
	}

	if storageProduct == nil {
		return nil, errors.APIError("no suitable S3 storage pricing found").
			WithContext("resourceName", resource.Name).
			WithContext("storageClass", storageClass).
			WithContext("region", resource.Region).
			WithSuggestion("Check that the storage class is available in the specified region").
			WithSuggestion("Verify the storage class name is correct")
	}

	// Extract storage price (per GB-month)
	storagePrice, err := e.pricingService.ExtractHourlyPrice(*storageProduct)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to extract S3 storage pricing").
			WithContext("resourceName", resource.Name).
			WithContext("sku", storageProduct.SKU)
	}

	// Calculate storage costs
	// Note: S3 pricing is typically per GB-month, but we'll convert to hourly for consistency
	monthlyStorageCost := storagePrice * float64(sizeGB)
	hourlyStorageCost := monthlyStorageCost / (24 * 30) // Convert monthly to hourly

	// Calculate request costs if applicable
	var hourlyRequestCost float64
	if requestProduct != nil && requestsPerMonth > 0 {
		requestPrice, err := e.pricingService.ExtractHourlyPrice(*requestProduct)
		if err == nil {
			// Request pricing is typically per 1000 requests per month
			monthlyRequestCost := (float64(requestsPerMonth) / 1000.0) * requestPrice
			hourlyRequestCost = monthlyRequestCost / (24 * 30)
		}
	}

	// Total hourly cost
	totalHourlyCost := hourlyStorageCost + hourlyRequestCost

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
	estimate.AddAssumption("Storage costs calculated based on allocated size")
	estimate.AddAssumption("Pricing based on standard S3 rates (no volume discounts)")
	if requestsPerMonth > 0 {
		estimate.AddAssumption(fmt.Sprintf("Request costs calculated for %d requests per month", requestsPerMonth))
	} else {
		estimate.AddAssumption("Request costs not included (set requestsPerMonth for request pricing)")
	}
	if sizeGB == 1 {
		estimate.AddAssumption("Default size of 1 GB used (specify sizeGB for accurate pricing)")
	}

	// Add details
	estimate.SetDetail("storageClass", storageClass)
	estimate.SetDetail("sizeGB", fmt.Sprintf("%d", sizeGB))
	estimate.SetDetail("requestsPerMonth", fmt.Sprintf("%d", requestsPerMonth))
	estimate.SetDetail("storagePrice", fmt.Sprintf("$%.6f/GB-month", storagePrice))
	estimate.SetDetail("storageSKU", storageProduct.SKU)

	if requestProduct != nil {
		requestPrice, _ := e.pricingService.ExtractHourlyPrice(*requestProduct)
		estimate.SetDetail("requestPrice", fmt.Sprintf("$%.6f/1000 requests", requestPrice))
		estimate.SetDetail("requestSKU", requestProduct.SKU)
	}

	// Add product family info
	if storageProduct.ProductFamily != "" {
		estimate.SetDetail("productFamily", storageProduct.ProductFamily)
	}

	// Break down costs
	estimate.SetDetail("monthlyStorageCost", fmt.Sprintf("$%.4f", monthlyStorageCost))
	if hourlyRequestCost > 0 {
		monthlyRequestCost := hourlyRequestCost * 24 * 30
		estimate.SetDetail("monthlyRequestCost", fmt.Sprintf("$%.4f", monthlyRequestCost))
	}

	return estimate, nil
}

// isValidStorageClass validates S3 storage class
func (e *Estimator) isValidStorageClass(storageClass string) bool {
	validClasses := map[string]bool{
		"STANDARD":            true,
		"STANDARD_IA":         true,
		"ONEZONE_IA":          true,
		"GLACIER":             true,
		"DEEP_ARCHIVE":        true,
		"INTELLIGENT_TIERING": true,
		"REDUCED_REDUNDANCY":  true, // Legacy but still valid
	}
	return validClasses[storageClass]
}

// isStorageUsageType checks if the usage type is for storage
func (e *Estimator) isStorageUsageType(usageType string) bool {
	// S3 storage usage types typically contain "Storage" or "TimedStorage"
	return containsAny(usageType, []string{"Storage", "TimedStorage"})
}

// isRequestUsageType checks if the usage type is for requests
func (e *Estimator) isRequestUsageType(usageType string) bool {
	// S3 request usage types typically contain "Requests" or "Request"
	return containsAny(usageType, []string{"Requests", "Request"})
}

// GetSupportedStorageClasses returns supported S3 storage classes
func (e *Estimator) GetSupportedStorageClasses() []string {
	return []string{
		"STANDARD",
		"STANDARD_IA",
		"ONEZONE_IA",
		"GLACIER",
		"DEEP_ARCHIVE",
		"INTELLIGENT_TIERING",
		"REDUCED_REDUNDANCY",
	}
}

// GetStorageClassDescription returns description for storage classes
func (e *Estimator) GetStorageClassDescription(storageClass string) string {
	descriptions := map[string]string{
		"STANDARD":            "General purpose storage for frequently accessed data",
		"STANDARD_IA":         "Infrequently accessed data with rapid access when needed",
		"ONEZONE_IA":          "Infrequently accessed data stored in a single AZ",
		"GLACIER":             "Long-term archive with retrieval times from minutes to hours",
		"DEEP_ARCHIVE":        "Lowest cost storage for long-term retention (12+ hours retrieval)",
		"INTELLIGENT_TIERING": "Automatic cost optimization by moving data between access tiers",
		"REDUCED_REDUNDANCY":  "Legacy storage class with reduced durability",
	}
	if desc, exists := descriptions[storageClass]; exists {
		return desc
	}
	return "Unknown storage class"
}

// GetTypicalUseCases returns typical use cases for storage classes
func (e *Estimator) GetTypicalUseCases(storageClass string) []string {
	useCases := map[string][]string{
		"STANDARD": {
			"Frequently accessed data",
			"Content distribution",
			"Big data analytics",
			"Mobile and gaming applications",
		},
		"STANDARD_IA": {
			"Backups",
			"Disaster recovery",
			"Long-term storage with occasional access",
		},
		"ONEZONE_IA": {
			"Secondary backup copies",
			"Easily re-creatable data",
			"Cross-region replication destinations",
		},
		"GLACIER": {
			"Data archiving",
			"Backup and restore",
			"Media asset workflows",
		},
		"DEEP_ARCHIVE": {
			"Compliance archives",
			"Digital preservation",
			"Long-term data retention",
		},
		"INTELLIGENT_TIERING": {
			"Data with unknown or changing access patterns",
			"Cost optimization without performance impact",
		},
	}
	if cases, exists := useCases[storageClass]; exists {
		return cases
	}
	return []string{"General storage use cases"}
}

// Helper function
func containsAny(s string, substrings []string) bool {
	for _, substr := range substrings {
		if len(substr) > 0 && len(s) >= len(substr) {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}
