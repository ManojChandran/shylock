package rds

import (
	"context"
	"fmt"
	"time"

	"shylock/internal/aws"
	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// Estimator implements the ResourceEstimator interface for RDS instances
type Estimator struct {
	pricingService *aws.PricingService
}

// NewEstimator creates a new RDS cost estimator
func NewEstimator(awsClient interfaces.AWSPricingClient) interfaces.ResourceEstimator {
	return &Estimator{
		pricingService: aws.NewPricingService(awsClient),
	}
}

// SupportedResourceType returns the AWS resource type this estimator supports
func (e *Estimator) SupportedResourceType() string {
	return "RDS"
}

// ValidateResource validates that the resource specification is valid for RDS
func (e *Estimator) ValidateResource(resource models.ResourceSpec) error {
	if resource.Type != "RDS" {
		return errors.ValidationError("resource type must be 'RDS'").
			WithContext("actualType", resource.Type).
			WithSuggestion("Use 'RDS' as the resource type")
	}

	// Validate required properties
	requiredProps := []string{"instanceClass", "engine"}
	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return errors.ValidationError(fmt.Sprintf("missing required property '%s'", prop)).
				WithContext("resourceName", resource.Name).
				WithSuggestion(fmt.Sprintf("Add '%s' property to the resource configuration", prop))
		}
	}

	// Validate instance class format
	instanceClass, err := resource.GetStringProperty("instanceClass")
	if err != nil {
		return errors.ValidationErrorWithCause("invalid instanceClass property", err).
			WithContext("resourceName", resource.Name).
			WithSuggestion("Ensure instanceClass is a string (e.g., 'db.t3.micro', 'db.r5.large')")
	}

	if !e.isValidInstanceClass(instanceClass) {
		return errors.ValidationError("invalid RDS instance class format").
			WithContext("resourceName", resource.Name).
			WithContext("instanceClass", instanceClass).
			WithSuggestion("Use valid RDS instance class format (e.g., 'db.t3.micro', 'db.r5.large')").
			WithSuggestion("Check AWS documentation for available RDS instance classes")
	}

	// Validate engine
	engine, err := resource.GetStringProperty("engine")
	if err != nil {
		return errors.ValidationErrorWithCause("invalid engine property", err).
			WithContext("resourceName", resource.Name).
			WithSuggestion("Ensure engine is a string")
	}

	if !e.isValidEngine(engine) {
		return errors.ValidationError("invalid RDS engine").
			WithContext("resourceName", resource.Name).
			WithContext("engine", engine).
			WithSuggestion("Use valid RDS engine: mysql, postgres, mariadb, oracle-ee, sqlserver-ex, aurora-mysql, aurora-postgresql").
			WithSuggestion("Check AWS documentation for supported database engines")
	}

	// Validate optional storage properties
	if _, exists := resource.GetProperty("storageGB"); exists {
		storageGB, err := resource.GetIntProperty("storageGB")
		if err != nil {
			return errors.ValidationErrorWithCause("invalid storageGB property", err).
				WithContext("resourceName", resource.Name).
				WithSuggestion("Ensure storageGB is a positive integer")
		}
		if storageGB <= 0 {
			return errors.ValidationError("storageGB must be greater than 0").
				WithContext("resourceName", resource.Name).
				WithContext("storageGB", storageGB).
				WithSuggestion("Set storageGB to a positive integer (minimum varies by engine)")
		}
	}

	// Validate optional boolean properties
	boolProps := []string{"multiAZ", "encrypted"}
	for _, prop := range boolProps {
		if _, exists := resource.GetProperty(prop); exists {
			if _, ok := resource.Properties[prop].(bool); !ok {
				return errors.ValidationError(fmt.Sprintf("%s must be a boolean", prop)).
					WithContext("resourceName", resource.Name).
					WithContext(prop, resource.Properties[prop]).
					WithSuggestion(fmt.Sprintf("Set %s to true or false", prop))
			}
		}
	}

	// Validate region
	if err := e.pricingService.ValidateRegion(resource.Region); err != nil {
		return errors.WrapError(err, errors.ValidationErrorType, "invalid region for RDS resource").
			WithContext("resourceName", resource.Name)
	}

	return nil
}

// EstimateCost calculates the cost for an RDS instance
func (e *Estimator) EstimateCost(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error) {
	// Validate the resource first
	if err := e.ValidateResource(resource); err != nil {
		return nil, err
	}

	// Extract properties
	instanceClass, _ := resource.GetStringProperty("instanceClass")
	engine, _ := resource.GetStringProperty("engine")

	// Get optional properties with defaults
	storageGB := 20 // Default storage
	if _, exists := resource.GetProperty("storageGB"); exists {
		if sg, err := resource.GetIntProperty("storageGB"); err == nil {
			storageGB = sg
		}
	}

	multiAZ := false // Default single AZ
	if _, exists := resource.GetProperty("multiAZ"); exists {
		if maz, ok := resource.Properties["multiAZ"].(bool); ok {
			multiAZ = maz
		}
	}

	encrypted := false // Default not encrypted
	if _, exists := resource.GetProperty("encrypted"); exists {
		if enc, ok := resource.Properties["encrypted"].(bool); ok {
			encrypted = enc
		}
	}

	storageType := "gp2" // Default storage type
	if _, exists := resource.GetProperty("storageType"); exists {
		if st, err := resource.GetStringProperty("storageType"); err == nil {
			storageType = st
		}
	}

	// Get pricing data from AWS
	products, err := e.pricingService.GetRDSPricing(ctx, instanceClass, engine, resource.Region, multiAZ)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve RDS pricing data").
			WithContext("resourceName", resource.Name).
			WithContext("instanceClass", instanceClass).
			WithContext("engine", engine).
			WithContext("region", resource.Region)
	}

	// Calculate costs
	totalHourlyCost, costBreakdown, err := e.calculateRDSCosts(products, storageGB, storageType, encrypted)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to calculate RDS costs").
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
	estimate.AddAssumption("24/7 database operation assumed")
	estimate.AddAssumption("On-demand pricing (no reserved instances)")
	if !multiAZ {
		estimate.AddAssumption("Single AZ deployment (set multiAZ: true for Multi-AZ)")
	} else {
		estimate.AddAssumption("Multi-AZ deployment for high availability")
	}
	if !encrypted {
		estimate.AddAssumption("Storage not encrypted (set encrypted: true for encryption)")
	}
	if storageGB == 20 {
		estimate.AddAssumption("Default 20 GB storage (specify storageGB for accurate pricing)")
	}

	// Add details
	estimate.SetDetail("instanceClass", instanceClass)
	estimate.SetDetail("engine", engine)
	estimate.SetDetail("storageGB", fmt.Sprintf("%d", storageGB))
	estimate.SetDetail("storageType", storageType)
	estimate.SetDetail("multiAZ", fmt.Sprintf("%t", multiAZ))
	estimate.SetDetail("encrypted", fmt.Sprintf("%t", encrypted))

	// Add cost breakdown details
	for component, cost := range costBreakdown {
		estimate.SetDetail(fmt.Sprintf("%sCost", component), fmt.Sprintf("$%.4f/hour", cost))
	}

	return estimate, nil
}

// calculateRDSCosts calculates RDS costs based on instance and storage pricing
func (e *Estimator) calculateRDSCosts(products []interfaces.PricingProduct, storageGB int, storageType string, encrypted bool) (float64, map[string]float64, error) {
	costBreakdown := make(map[string]float64)

	// Find pricing components
	var instanceHourPrice, storageHourPrice float64

	for _, product := range products {
		// Extract pricing based on usage type
		if usageType, exists := product.Attributes["usageType"]; exists {
			price, err := e.pricingService.ExtractHourlyPrice(product)
			if err != nil {
				continue // Skip products we can't parse
			}

			if e.isInstanceUsage(usageType) {
				instanceHourPrice = price
				costBreakdown["instance"] = price
			} else if e.isStorageUsage(usageType) {
				// Storage pricing is typically per GB-month, convert to hourly
				storageMonthlyPrice := price * float64(storageGB)
				storageHourPrice = storageMonthlyPrice / (24 * 30)
				costBreakdown["storage"] = storageHourPrice
			}
		}
	}

	// Add encryption cost if applicable (typically small additional cost)
	if encrypted {
		encryptionCost := instanceHourPrice * 0.05 // Approximate 5% additional cost
		costBreakdown["encryption"] = encryptionCost
		instanceHourPrice += encryptionCost
	}

	totalCost := instanceHourPrice + storageHourPrice

	return totalCost, costBreakdown, nil
}

// Helper functions

func (e *Estimator) isValidInstanceClass(instanceClass string) bool {
	if instanceClass == "" {
		return false
	}

	// Basic validation: should start with "db." and have proper format
	if len(instanceClass) < 6 || instanceClass[:3] != "db." {
		return false
	}

	// Check for valid instance family patterns
	validFamilies := []string{"db.t3", "db.t4g", "db.r5", "db.r6g", "db.m5", "db.m6g", "db.x1e", "db.z1d"}
	for _, family := range validFamilies {
		if len(instanceClass) > len(family) && instanceClass[:len(family)] == family {
			return true
		}
	}

	return false
}

func (e *Estimator) isValidEngine(engine string) bool {
	validEngines := map[string]bool{
		"mysql":             true,
		"postgres":          true,
		"mariadb":           true,
		"oracle-ee":         true,
		"oracle-se2":        true,
		"sqlserver-ex":      true,
		"sqlserver-web":     true,
		"sqlserver-se":      true,
		"sqlserver-ee":      true,
		"aurora-mysql":      true,
		"aurora-postgresql": true,
	}
	return validEngines[engine]
}

func (e *Estimator) isInstanceUsage(usageType string) bool {
	// RDS instance usage types typically contain "InstanceUsage" or "Multi-AZ"
	return containsSubstring(usageType, "InstanceUsage") || containsSubstring(usageType, "Multi-AZ")
}

func (e *Estimator) isStorageUsage(usageType string) bool {
	// RDS storage usage types typically contain "Storage"
	return containsSubstring(usageType, "Storage")
}

// GetSupportedInstanceClasses returns common RDS instance classes
func (e *Estimator) GetSupportedInstanceClasses() []string {
	return []string{
		// Burstable performance
		"db.t3.micro", "db.t3.small", "db.t3.medium", "db.t3.large", "db.t3.xlarge",
		"db.t4g.micro", "db.t4g.small", "db.t4g.medium", "db.t4g.large",

		// Memory optimized
		"db.r5.large", "db.r5.xlarge", "db.r5.2xlarge", "db.r5.4xlarge",
		"db.r6g.large", "db.r6g.xlarge", "db.r6g.2xlarge", "db.r6g.4xlarge",

		// General purpose
		"db.m5.large", "db.m5.xlarge", "db.m5.2xlarge", "db.m5.4xlarge",
		"db.m6g.large", "db.m6g.xlarge", "db.m6g.2xlarge", "db.m6g.4xlarge",
	}
}

// GetSupportedEngines returns supported RDS engines
func (e *Estimator) GetSupportedEngines() []string {
	return []string{
		"mysql", "postgres", "mariadb",
		"oracle-ee", "oracle-se2",
		"sqlserver-ex", "sqlserver-web", "sqlserver-se", "sqlserver-ee",
		"aurora-mysql", "aurora-postgresql",
	}
}

// GetEngineDescription returns description for database engines
func (e *Estimator) GetEngineDescription(engine string) string {
	descriptions := map[string]string{
		"mysql":             "MySQL Community Edition",
		"postgres":          "PostgreSQL",
		"mariadb":           "MariaDB Community Edition",
		"oracle-ee":         "Oracle Database Enterprise Edition",
		"oracle-se2":        "Oracle Database Standard Edition 2",
		"sqlserver-ex":      "Microsoft SQL Server Express Edition",
		"sqlserver-web":     "Microsoft SQL Server Web Edition",
		"sqlserver-se":      "Microsoft SQL Server Standard Edition",
		"sqlserver-ee":      "Microsoft SQL Server Enterprise Edition",
		"aurora-mysql":      "Amazon Aurora MySQL-Compatible Edition",
		"aurora-postgresql": "Amazon Aurora PostgreSQL-Compatible Edition",
	}
	if desc, exists := descriptions[engine]; exists {
		return desc
	}
	return "Unknown database engine"
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
