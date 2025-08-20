// Package ec2 provides cost estimation functionality for Amazon EC2 instances.
//
// This package implements the ResourceEstimator interface to calculate costs
// for EC2 instances based on instance type, operating system, tenancy, and
// instance count. It supports all major EC2 instance families and operating
// systems available through the AWS Pricing API.
//
// Supported features:
//   - 50+ instance types across all families (t3, m5, c5, r5, etc.)
//   - Multiple operating systems (Linux, Windows, RHEL, SUSE)
//   - Tenancy options (Shared, Dedicated, Host)
//   - Multiple instance counts
//   - Regional pricing variations
//
// Usage:
//
//	estimator := ec2.NewEstimator(awsClient)
//	estimate, err := estimator.EstimateCost(ctx, resourceSpec)
package ec2

import (
	"context"
	"fmt"
	"time"

	"shylock/internal/aws"
	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// Estimator implements the ResourceEstimator interface for EC2 instances.
// It provides cost estimation capabilities for various EC2 instance types,
// operating systems, and configurations using AWS Pricing API data.
type Estimator struct {
	pricingService *aws.PricingService // AWS Pricing API service client
}

// NewEstimator creates a new EC2 cost estimator with the provided AWS client.
//
// Parameters:
//   - awsClient: AWS Pricing API client for retrieving EC2 pricing data
//
// Returns:
//   - interfaces.ResourceEstimator: EC2 estimator implementing the interface
func NewEstimator(awsClient interfaces.AWSPricingClient) interfaces.ResourceEstimator {
	return &Estimator{
		pricingService: aws.NewPricingService(awsClient),
	}
}

// SupportedResourceType returns the AWS resource type this estimator supports
func (e *Estimator) SupportedResourceType() string {
	return "EC2"
}

// ValidateResource validates that the resource specification is valid for EC2
func (e *Estimator) ValidateResource(resource models.ResourceSpec) error {
	if resource.Type != "EC2" {
		return errors.ValidationError("resource type must be 'EC2'").
			WithContext("actualType", resource.Type).
			WithSuggestion("Use 'EC2' as the resource type")
	}

	// Validate required properties
	requiredProps := []string{"instanceType"}
	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return errors.ValidationError(fmt.Sprintf("missing required property '%s'", prop)).
				WithContext("resourceName", resource.Name).
				WithSuggestion(fmt.Sprintf("Add '%s' property to the resource configuration", prop))
		}
	}

	// Validate instance type format
	instanceType, err := resource.GetStringProperty("instanceType")
	if err != nil {
		return errors.ValidationErrorWithCause("invalid instanceType property", err).
			WithContext("resourceName", resource.Name).
			WithSuggestion("Ensure instanceType is a string (e.g., 't3.micro', 'm5.large')")
	}

	if !e.isValidInstanceType(instanceType) {
		return errors.ValidationError("invalid instance type format").
			WithContext("resourceName", resource.Name).
			WithContext("instanceType", instanceType).
			WithSuggestion("Use valid EC2 instance type format (e.g., 't3.micro', 'm5.large')").
			WithSuggestion("Check AWS documentation for available instance types")
	}

	// Validate optional count property
	if _, exists := resource.GetProperty("count"); exists {
		count, err := resource.GetIntProperty("count")
		if err != nil {
			return errors.ValidationErrorWithCause("invalid count property", err).
				WithContext("resourceName", resource.Name).
				WithSuggestion("Ensure count is a positive integer")
		}
		if count <= 0 {
			return errors.ValidationError("count must be greater than 0").
				WithContext("resourceName", resource.Name).
				WithContext("count", count).
				WithSuggestion("Set count to a positive integer (default is 1)")
		}
	}

	// Validate region
	if err := e.pricingService.ValidateRegion(resource.Region); err != nil {
		return errors.WrapError(err, errors.ValidationErrorType, "invalid region for EC2 resource").
			WithContext("resourceName", resource.Name)
	}

	return nil
}

// EstimateCost calculates the cost for an EC2 instance
func (e *Estimator) EstimateCost(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error) {
	// Validate the resource first
	if err := e.ValidateResource(resource); err != nil {
		return nil, err
	}

	// Extract properties
	instanceType, _ := resource.GetStringProperty("instanceType")

	// Get optional properties with defaults
	count := 1
	if _, exists := resource.GetProperty("count"); exists {
		if c, err := resource.GetIntProperty("count"); err == nil {
			count = c
		}
	}

	operatingSystem := "Linux" // Default
	if _, exists := resource.GetProperty("operatingSystem"); exists {
		if os, err := resource.GetStringProperty("operatingSystem"); err == nil {
			operatingSystem = os
		}
	}

	tenancy := "Shared" // Default
	if _, exists := resource.GetProperty("tenancy"); exists {
		if t, err := resource.GetStringProperty("tenancy"); err == nil {
			tenancy = t
		}
	}

	// Get pricing data from AWS
	products, err := e.pricingService.GetEC2Pricing(ctx, instanceType, resource.Region, operatingSystem)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve EC2 pricing data").
			WithContext("resourceName", resource.Name).
			WithContext("instanceType", instanceType).
			WithContext("region", resource.Region)
	}

	// Find the best matching product (prefer shared tenancy, on-demand)
	var selectedProduct *interfaces.PricingProduct
	for _, product := range products {
		// Check tenancy preference
		if productTenancy, exists := product.Attributes["tenancy"]; exists {
			if productTenancy == tenancy {
				selectedProduct = &product
				break
			}
		}
		// Fallback to first product if no exact tenancy match
		if selectedProduct == nil {
			selectedProduct = &product
		}
	}

	if selectedProduct == nil {
		return nil, errors.APIError("no suitable pricing product found").
			WithContext("resourceName", resource.Name).
			WithContext("instanceType", instanceType).
			WithContext("region", resource.Region).
			WithContext("operatingSystem", operatingSystem).
			WithSuggestion("Check that the instance type is available in the specified region").
			WithSuggestion("Verify the operating system is supported")
	}

	// Extract hourly price
	hourlyPrice, err := e.pricingService.ExtractHourlyPrice(*selectedProduct)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to extract pricing information").
			WithContext("resourceName", resource.Name).
			WithContext("sku", selectedProduct.SKU)
	}

	// Calculate total cost for all instances
	totalHourlyPrice := hourlyPrice * float64(count)

	// Create cost estimate
	estimate := &models.CostEstimate{
		ResourceName: resource.Name,
		ResourceType: resource.Type,
		Region:       resource.Region,
		HourlyCost:   totalHourlyPrice,
		Currency:     "USD",
		Timestamp:    time.Now(),
	}

	// Calculate daily and monthly costs
	estimate.CalculateCosts()

	// Add assumptions
	estimate.AddAssumption("24/7 usage assumed (8760 hours per year)")
	estimate.AddAssumption("On-demand pricing (no reserved instances or savings plans)")
	if count > 1 {
		estimate.AddAssumption(fmt.Sprintf("Cost calculated for %d instances", count))
	}
	if operatingSystem != "Linux" {
		estimate.AddAssumption(fmt.Sprintf("Operating system: %s", operatingSystem))
	}
	if tenancy != "Shared" {
		estimate.AddAssumption(fmt.Sprintf("Tenancy: %s", tenancy))
	}

	// Add details
	estimate.SetDetail("instanceType", instanceType)
	estimate.SetDetail("operatingSystem", operatingSystem)
	estimate.SetDetail("tenancy", tenancy)
	estimate.SetDetail("count", fmt.Sprintf("%d", count))
	estimate.SetDetail("pricePerInstance", fmt.Sprintf("$%.4f/hour", hourlyPrice))
	estimate.SetDetail("sku", selectedProduct.SKU)

	// Add product family and service info
	if selectedProduct.ProductFamily != "" {
		estimate.SetDetail("productFamily", selectedProduct.ProductFamily)
	}

	return estimate, nil
}

// isValidInstanceType validates EC2 instance type format
func (e *Estimator) isValidInstanceType(instanceType string) bool {
	if instanceType == "" {
		return false
	}

	// Basic validation: should be in format like "t3.micro", "m5.large", etc.
	// More comprehensive validation could check against known instance families
	parts := []rune(instanceType)
	if len(parts) < 4 { // Minimum: "t3.x"
		return false
	}

	// Should start with letter(s) for instance family
	familyEnd := 0
	for i, r := range parts {
		if r >= '0' && r <= '9' {
			familyEnd = i
			break
		}
		if i > 5 { // Instance family shouldn't be too long
			return false
		}
	}

	if familyEnd == 0 {
		return false // No numbers found
	}

	// Should have a dot after the generation number
	dotFound := false
	for i := familyEnd; i < len(parts); i++ {
		if parts[i] == '.' {
			dotFound = true
			break
		}
		if i-familyEnd > 2 { // Generation number shouldn't be too long
			return false
		}
	}

	return dotFound && len(parts) > familyEnd+2 // Should have size after dot
}

// GetSupportedInstanceFamilies returns common EC2 instance families
func (e *Estimator) GetSupportedInstanceFamilies() []string {
	return []string{
		"t3", "t3a", "t4g", // Burstable performance
		"m5", "m5a", "m5n", "m6i", "m6a", // General purpose
		"c5", "c5n", "c6i", "c6a", // Compute optimized
		"r5", "r5a", "r5n", "r6i", "r6a", // Memory optimized
		"i3", "i4i", // Storage optimized
		"p3", "p4", "g4", // Accelerated computing
	}
}

// GetCommonInstanceTypes returns a list of commonly used instance types
func (e *Estimator) GetCommonInstanceTypes() []string {
	return []string{
		// Burstable performance
		"t3.nano", "t3.micro", "t3.small", "t3.medium", "t3.large",
		"t3a.nano", "t3a.micro", "t3a.small", "t3a.medium", "t3a.large",

		// General purpose
		"m5.large", "m5.xlarge", "m5.2xlarge", "m5.4xlarge",
		"m6i.large", "m6i.xlarge", "m6i.2xlarge", "m6i.4xlarge",

		// Compute optimized
		"c5.large", "c5.xlarge", "c5.2xlarge", "c5.4xlarge",
		"c6i.large", "c6i.xlarge", "c6i.2xlarge", "c6i.4xlarge",

		// Memory optimized
		"r5.large", "r5.xlarge", "r5.2xlarge", "r5.4xlarge",
		"r6i.large", "r6i.xlarge", "r6i.2xlarge", "r6i.4xlarge",
	}
}
