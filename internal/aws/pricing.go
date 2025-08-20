package aws

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"shylock/internal/errors"
	"shylock/internal/interfaces"
)

// PricingService handles AWS pricing data retrieval and filtering
type PricingService struct {
	client interfaces.AWSPricingClient
}

// NewPricingService creates a new pricing service
func NewPricingService(client interfaces.AWSPricingClient) *PricingService {
	return &PricingService{
		client: client,
	}
}

// GetEC2Pricing retrieves EC2 pricing information for specific instance types
func (p *PricingService) GetEC2Pricing(ctx context.Context, instanceType, region, operatingSystem string) ([]interfaces.PricingProduct, error) {
	if instanceType == "" {
		return nil, errors.ValidationError("instance type cannot be empty").
			WithSuggestion("Provide a valid EC2 instance type (e.g., 't3.micro', 'm5.large')")
	}

	if region == "" {
		return nil, errors.ValidationError("region cannot be empty").
			WithSuggestion("Provide a valid AWS region (e.g., 'us-east-1', 'us-west-2')")
	}

	// Convert region to location format used by pricing API
	location := p.regionToLocation(region)
	if location == "" {
		return nil, errors.ValidationError("unsupported region").
			WithContext("region", region).
			WithSuggestion("Use a standard AWS region code")
	}

	// Set default operating system if not provided
	if operatingSystem == "" {
		operatingSystem = "Linux"
	}

	// Build filters for EC2 pricing query
	filters := map[string]string{
		"servicecode":     "AmazonEC2",
		"instanceType":    instanceType,
		"location":        location,
		"operatingSystem": operatingSystem,
		"tenancy":         "Shared", // Default to shared tenancy
		"preInstalledSw":  "NA",     // No pre-installed software
	}

	products, err := p.client.GetProducts(ctx, "AmazonEC2", filters)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve EC2 pricing").
			WithContext("instanceType", instanceType).
			WithContext("region", region).
			WithContext("operatingSystem", operatingSystem)
	}

	if len(products) == 0 {
		return nil, errors.APIError("no pricing data found for EC2 instance").
			WithContext("instanceType", instanceType).
			WithContext("region", region).
			WithContext("operatingSystem", operatingSystem).
			WithSuggestion("Check that the instance type is available in the specified region").
			WithSuggestion("Verify the operating system is supported")
	}

	return products, nil
}

// GetALBPricing retrieves ALB pricing information for specific load balancer type and region
func (p *PricingService) GetALBPricing(ctx context.Context, albType, region string) ([]interfaces.PricingProduct, error) {
	if albType == "" {
		return nil, errors.ValidationError("ALB type cannot be empty").
			WithSuggestion("Provide a valid ALB type ('application' or 'network')")
	}

	if region == "" {
		return nil, errors.ValidationError("region cannot be empty").
			WithSuggestion("Provide a valid AWS region")
	}

	// Convert region to location format
	location := p.regionToLocation(region)
	if location == "" {
		return nil, errors.ValidationError("unsupported region").
			WithContext("region", region).
			WithSuggestion("Use a standard AWS region code")
	}

	// Build filters for ALB pricing query
	// ALB pricing is under "AWSELB" service code
	filters := map[string]string{
		"servicecode": "AWSELB",
		"location":    location,
	}

	// Add type-specific filters
	if albType == "application" {
		filters["productFamily"] = "Load Balancer-Application"
	} else if albType == "network" {
		filters["productFamily"] = "Load Balancer-Network"
	}

	products, err := p.client.GetProducts(ctx, "AWSELB", filters)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve ALB pricing").
			WithContext("albType", albType).
			WithContext("region", region)
	}

	if len(products) == 0 {
		return nil, errors.APIError("no ALB pricing data found").
			WithContext("albType", albType).
			WithContext("region", region).
			WithSuggestion("Check that ALB is available in the specified region").
			WithSuggestion("Verify the ALB type is supported")
	}

	return products, nil
}

// GetRDSPricing retrieves RDS pricing information for specific instance class, engine, and region
func (p *PricingService) GetRDSPricing(ctx context.Context, instanceClass, engine, region string, multiAZ bool) ([]interfaces.PricingProduct, error) {
	if instanceClass == "" {
		return nil, errors.ValidationError("instance class cannot be empty").
			WithSuggestion("Provide a valid RDS instance class (e.g., 'db.t3.micro', 'db.r5.large')")
	}

	if engine == "" {
		return nil, errors.ValidationError("engine cannot be empty").
			WithSuggestion("Provide a valid RDS engine (e.g., 'mysql', 'postgres')")
	}

	if region == "" {
		return nil, errors.ValidationError("region cannot be empty").
			WithSuggestion("Provide a valid AWS region")
	}

	// Convert region to location format
	location := p.regionToLocation(region)
	if location == "" {
		return nil, errors.ValidationError("unsupported region").
			WithContext("region", region).
			WithSuggestion("Use a standard AWS region code")
	}

	// Build filters for RDS pricing query
	filters := map[string]string{
		"servicecode":    "AmazonRDS",
		"location":       location,
		"instanceType":   instanceClass,
		"databaseEngine": p.normalizeEngine(engine),
	}

	// Add deployment option filter
	if multiAZ {
		filters["deploymentOption"] = "Multi-AZ"
	} else {
		filters["deploymentOption"] = "Single-AZ"
	}

	products, err := p.client.GetProducts(ctx, "AmazonRDS", filters)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve RDS pricing").
			WithContext("instanceClass", instanceClass).
			WithContext("engine", engine).
			WithContext("region", region)
	}

	if len(products) == 0 {
		return nil, errors.APIError("no RDS pricing data found").
			WithContext("instanceClass", instanceClass).
			WithContext("engine", engine).
			WithContext("region", region).
			WithSuggestion("Check that the instance class is available in the specified region").
			WithSuggestion("Verify the database engine is supported")
	}

	return products, nil
}

// normalizeEngine converts engine names to AWS Pricing API format
func (p *PricingService) normalizeEngine(engine string) string {
	engineMap := map[string]string{
		"mysql":             "MySQL",
		"postgres":          "PostgreSQL",
		"mariadb":           "MariaDB",
		"oracle-ee":         "Oracle",
		"oracle-se2":        "Oracle",
		"sqlserver-ex":      "SQL Server",
		"sqlserver-web":     "SQL Server",
		"sqlserver-se":      "SQL Server",
		"sqlserver-ee":      "SQL Server",
		"aurora-mysql":      "Aurora MySQL",
		"aurora-postgresql": "Aurora PostgreSQL",
	}

	if normalized, exists := engineMap[engine]; exists {
		return normalized
	}
	return engine // Fallback to original
}

// GetLambdaPricing retrieves Lambda pricing information for specific region and architecture
func (p *PricingService) GetLambdaPricing(ctx context.Context, region, architecture string) ([]interfaces.PricingProduct, error) {
	if region == "" {
		return nil, errors.ValidationError("region cannot be empty").
			WithSuggestion("Provide a valid AWS region")
	}

	// Convert region to location format
	location := p.regionToLocation(region)
	if location == "" {
		return nil, errors.ValidationError("unsupported region").
			WithContext("region", region).
			WithSuggestion("Use a standard AWS region code")
	}

	// Build filters for Lambda pricing query
	filters := map[string]string{
		"servicecode": "AWSLambda",
		"location":    location,
	}

	// Add architecture filter if specified
	if architecture != "" {
		if architecture == "arm64" {
			filters["processorFeatures"] = "ARM"
		} else {
			filters["processorFeatures"] = "Intel"
		}
	}

	products, err := p.client.GetProducts(ctx, "AWSLambda", filters)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve Lambda pricing").
			WithContext("region", region).
			WithContext("architecture", architecture)
	}

	if len(products) == 0 {
		return nil, errors.APIError("no Lambda pricing data found").
			WithContext("region", region).
			WithContext("architecture", architecture).
			WithSuggestion("Check that Lambda is available in the specified region").
			WithSuggestion("Verify the architecture is supported")
	}

	return products, nil
}

// GetS3Pricing retrieves S3 pricing information for specific storage class and region
func (p *PricingService) GetS3Pricing(ctx context.Context, storageClass, region string) ([]interfaces.PricingProduct, error) {
	if storageClass == "" {
		return nil, errors.ValidationError("storage class cannot be empty").
			WithSuggestion("Provide a valid S3 storage class (e.g., 'STANDARD', 'STANDARD_IA')")
	}

	if region == "" {
		return nil, errors.ValidationError("region cannot be empty").
			WithSuggestion("Provide a valid AWS region")
	}

	// Convert region to location format
	location := p.regionToLocation(region)
	if location == "" {
		return nil, errors.ValidationError("unsupported region").
			WithContext("region", region).
			WithSuggestion("Use a standard AWS region code")
	}

	// Build filters for S3 pricing query
	filters := map[string]string{
		"servicecode":  "AmazonS3",
		"location":     location,
		"storageClass": storageClass,
		"volumeType":   "Standard", // Default volume type
	}

	products, err := p.client.GetProducts(ctx, "AmazonS3", filters)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve S3 pricing").
			WithContext("storageClass", storageClass).
			WithContext("region", region)
	}

	if len(products) == 0 {
		return nil, errors.APIError("no pricing data found for S3 storage").
			WithContext("storageClass", storageClass).
			WithContext("region", region).
			WithSuggestion("Check that the storage class is available in the specified region").
			WithSuggestion("Verify the storage class name is correct")
	}

	return products, nil
}

// ExtractHourlyPrice extracts the hourly price from pricing terms
func (p *PricingService) ExtractHourlyPrice(product interfaces.PricingProduct) (float64, error) {
	if product.Terms == nil {
		return 0, errors.APIError("no pricing terms found in product").
			WithContext("sku", product.SKU).
			WithSuggestion("The product may not have on-demand pricing available")
	}

	// Look for OnDemand terms
	onDemandTerms, ok := product.Terms["OnDemand"]
	if !ok {
		return 0, errors.APIError("no on-demand pricing terms found").
			WithContext("sku", product.SKU).
			WithSuggestion("The product may only have reserved or spot pricing")
	}

	// Navigate through the nested pricing structure
	termsMap, ok := onDemandTerms.(map[string]interface{})
	if !ok {
		return 0, errors.APIError("invalid on-demand terms structure").
			WithContext("sku", product.SKU)
	}

	// Find the first term (there's usually only one for on-demand)
	for _, termData := range termsMap {
		termInfo, ok := termData.(map[string]interface{})
		if !ok {
			continue
		}

		// Look for priceDimensions
		priceDimensions, ok := termInfo["priceDimensions"]
		if !ok {
			continue
		}

		dimensionsMap, ok := priceDimensions.(map[string]interface{})
		if !ok {
			continue
		}

		// Find the first price dimension
		for _, dimensionData := range dimensionsMap {
			dimension, ok := dimensionData.(map[string]interface{})
			if !ok {
				continue
			}

			// Look for pricePerUnit
			pricePerUnit, ok := dimension["pricePerUnit"]
			if !ok {
				continue
			}

			priceMap, ok := pricePerUnit.(map[string]interface{})
			if !ok {
				continue
			}

			// Look for USD price
			if usdPrice, ok := priceMap["USD"]; ok {
				if priceStr, ok := usdPrice.(string); ok {
					price, err := strconv.ParseFloat(priceStr, 64)
					if err != nil {
						return 0, errors.APIErrorWithCause("failed to parse price", err).
							WithContext("sku", product.SKU).
							WithContext("priceString", priceStr)
					}
					return price, nil
				}
			}
		}
	}

	return 0, errors.APIError("no USD pricing found in product").
		WithContext("sku", product.SKU).
		WithSuggestion("The product may not have USD pricing available")
}

// regionToLocation converts AWS region codes to pricing API location names
func (p *PricingService) regionToLocation(region string) string {
	// Map of common AWS regions to their pricing API location names
	regionMap := map[string]string{
		"us-east-1":      "US East (N. Virginia)",
		"us-east-2":      "US East (Ohio)",
		"us-west-1":      "US West (N. California)",
		"us-west-2":      "US West (Oregon)",
		"eu-west-1":      "Europe (Ireland)",
		"eu-west-2":      "Europe (London)",
		"eu-west-3":      "Europe (Paris)",
		"eu-central-1":   "Europe (Frankfurt)",
		"eu-north-1":     "Europe (Stockholm)",
		"ap-southeast-1": "Asia Pacific (Singapore)",
		"ap-southeast-2": "Asia Pacific (Sydney)",
		"ap-northeast-1": "Asia Pacific (Tokyo)",
		"ap-northeast-2": "Asia Pacific (Seoul)",
		"ap-south-1":     "Asia Pacific (Mumbai)",
		"ca-central-1":   "Canada (Central)",
		"sa-east-1":      "South America (Sao Paulo)",
	}

	if location, exists := regionMap[region]; exists {
		return location
	}

	// If not found in map, try to construct a reasonable location name
	// This is a fallback for newer regions
	parts := strings.Split(region, "-")
	if len(parts) >= 3 {
		switch parts[0] {
		case "us":
			if parts[1] == "east" {
				return fmt.Sprintf("US East (%s)", strings.Title(parts[1]))
			} else if parts[1] == "west" {
				return fmt.Sprintf("US West (%s)", strings.Title(parts[1]))
			}
		case "eu":
			return fmt.Sprintf("Europe (%s)", strings.Title(parts[1]))
		case "ap":
			return fmt.Sprintf("Asia Pacific (%s)", strings.Title(parts[1]))
		}
	}

	return "" // Unknown region
}

// GetSupportedRegions returns a list of regions supported by the pricing service
func (p *PricingService) GetSupportedRegions() []string {
	return []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2",
		"eu-west-1", "eu-west-2", "eu-west-3", "eu-central-1", "eu-north-1",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ap-northeast-2", "ap-south-1",
		"ca-central-1", "sa-east-1",
	}
}

// ValidateRegion checks if a region is supported
func (p *PricingService) ValidateRegion(region string) error {
	location := p.regionToLocation(region)
	if location == "" {
		supportedRegions := p.GetSupportedRegions()
		return errors.ValidationError("unsupported region").
			WithContext("region", region).
			WithContext("supportedRegions", strings.Join(supportedRegions, ", ")).
			WithSuggestion("Use one of the supported AWS regions").
			WithSuggestion("Check the AWS documentation for available regions")
	}
	return nil
}
