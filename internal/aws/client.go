package aws

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/pricing"
	"github.com/aws/aws-sdk-go-v2/service/pricing/types"

	"shylock/internal/errors"
	"shylock/internal/interfaces"
)

// Client implements the AWSPricingClient interface
type Client struct {
	pricingClient *pricing.Client
	region        string
	maxRetries    int
}

// ClientConfig holds configuration for the AWS client
type ClientConfig struct {
	Region     string
	MaxRetries int
}

// NewClient creates a new AWS pricing client with authentication and retry configuration
func NewClient(ctx context.Context, clientConfig *ClientConfig) (interfaces.AWSPricingClient, error) {
	if clientConfig == nil {
		clientConfig = &ClientConfig{
			Region:     "us-east-1", // Pricing API is only available in us-east-1
			MaxRetries: 3,
		}
	}

	// Ensure we use us-east-1 for pricing API (it's only available there)
	if clientConfig.Region != "us-east-1" {
		clientConfig.Region = "us-east-1"
	}

	// Load AWS configuration with retry settings
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(clientConfig.Region),
		config.WithRetryer(func() aws.Retryer {
			return retry.AddWithMaxAttempts(
				retry.NewStandard(func(so *retry.StandardOptions) {
					so.MaxAttempts = clientConfig.MaxRetries
					so.Backoff = retry.BackoffDelayerFunc(func(attempt int, err error) (time.Duration, error) {
						// Exponential backoff: 1s, 2s, 4s, 8s, etc.
						delay := time.Duration(math.Pow(2, float64(attempt))) * time.Second
						if delay > 30*time.Second {
							delay = 30 * time.Second // Cap at 30 seconds
						}
						return delay, nil
					})
				}),
				clientConfig.MaxRetries,
			)
		}),
	)
	if err != nil {
		return nil, errors.AuthErrorWithCause("failed to load AWS configuration", err).
			WithSuggestion("Ensure AWS credentials are configured (AWS CLI, environment variables, or IAM role)").
			WithSuggestion("Check that your AWS credentials have the necessary permissions").
			WithSuggestion("Verify your AWS region is accessible")
	}

	// Create pricing client
	pricingClient := pricing.NewFromConfig(cfg)

	client := &Client{
		pricingClient: pricingClient,
		region:        clientConfig.Region,
		maxRetries:    clientConfig.MaxRetries,
	}

	// Test the connection by calling DescribeServices
	if err := client.testConnection(ctx); err != nil {
		return nil, errors.WrapError(err, errors.AuthErrorType, "AWS client connection test failed")
	}

	return client, nil
}

// testConnection verifies that the AWS client can make API calls
func (c *Client) testConnection(ctx context.Context) error {
	// Create a context with timeout for the test
	testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Try to call DescribeServices with a limit to test connectivity
	input := &pricing.DescribeServicesInput{
		MaxResults: aws.Int32(1),
	}

	_, err := c.pricingClient.DescribeServices(testCtx, input)
	if err != nil {
		return errors.APIErrorWithCause("failed to connect to AWS Pricing API", err).
			WithSuggestion("Check your internet connection").
			WithSuggestion("Verify AWS credentials have pricing:DescribeServices permission").
			WithSuggestion("Ensure the AWS Pricing API is accessible from your network")
	}

	return nil
}

// GetProducts retrieves pricing information for AWS services
func (c *Client) GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]interfaces.PricingProduct, error) {
	if serviceCode == "" {
		return nil, errors.ValidationError("service code cannot be empty").
			WithSuggestion("Provide a valid AWS service code (e.g., 'AmazonEC2', 'AmazonS3')")
	}

	var allProducts []interfaces.PricingProduct
	var nextToken *string

	// Build filters for the API call
	var apiFilters []types.Filter
	for key, value := range filters {
		apiFilters = append(apiFilters, types.Filter{
			Field: aws.String(key),
			Value: aws.String(value),
			Type:  types.FilterTypeTermMatch,
		})
	}

	// Paginate through all results
	for {
		input := &pricing.GetProductsInput{
			ServiceCode: aws.String(serviceCode),
			Filters:     apiFilters,
			MaxResults:  aws.Int32(100), // Maximum allowed by API
		}

		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.pricingClient.GetProducts(ctx, input)
		if err != nil {
			return nil, errors.APIErrorWithCause("failed to retrieve pricing products", err).
				WithContext("serviceCode", serviceCode).
				WithContext("filtersCount", len(filters)).
				WithSuggestion("Check that the service code is valid").
				WithSuggestion("Verify your AWS credentials have pricing:GetProducts permission").
				WithSuggestion("Try reducing the number of filters if the request is too complex")
		}

		// Convert AWS SDK types to our interface types
		for _, product := range result.PriceList {
			convertedProduct, err := c.convertProduct(product)
			if err != nil {
				return nil, errors.APIErrorWithCause("failed to parse pricing product", err).
					WithContext("serviceCode", serviceCode)
			}
			allProducts = append(allProducts, convertedProduct)
		}

		// Check if there are more results
		nextToken = result.NextToken
		if nextToken == nil {
			break
		}

		// Prevent infinite loops
		if len(allProducts) > 10000 {
			return nil, errors.APIError("too many pricing products returned, consider adding more specific filters").
				WithContext("serviceCode", serviceCode).
				WithContext("productCount", len(allProducts)).
				WithSuggestion("Add more specific filters to narrow down the results").
				WithSuggestion("Consider querying for specific instance types or regions")
		}
	}

	return allProducts, nil
}

// DescribeServices lists available AWS services in the Pricing API
func (c *Client) DescribeServices(ctx context.Context) ([]interfaces.ServiceInfo, error) {
	var allServices []interfaces.ServiceInfo
	var nextToken *string

	// Paginate through all results
	for {
		input := &pricing.DescribeServicesInput{
			MaxResults: aws.Int32(100), // Maximum allowed by API
		}

		if nextToken != nil {
			input.NextToken = nextToken
		}

		result, err := c.pricingClient.DescribeServices(ctx, input)
		if err != nil {
			return nil, errors.APIErrorWithCause("failed to describe AWS services", err).
				WithSuggestion("Check your AWS credentials have pricing:DescribeServices permission").
				WithSuggestion("Verify your internet connection")
		}

		// Convert AWS SDK types to our interface types
		for _, service := range result.Services {
			serviceInfo := interfaces.ServiceInfo{
				ServiceCode: aws.ToString(service.ServiceCode),
				ServiceName: aws.ToString(service.ServiceCode), // Use ServiceCode as name if not available
			}

			// Add attributes if available
			if len(service.AttributeNames) > 0 {
				serviceInfo.Attributes = make(map[string]string)
				for _, attr := range service.AttributeNames {
					serviceInfo.Attributes[attr] = ""
				}
			}

			allServices = append(allServices, serviceInfo)
		}

		// Check if there are more results
		nextToken = result.NextToken
		if nextToken == nil {
			break
		}
	}

	return allServices, nil
}

// GetRegions returns available AWS regions for a service
func (c *Client) GetRegions(ctx context.Context, serviceCode string) ([]string, error) {
	if serviceCode == "" {
		return nil, errors.ValidationError("service code cannot be empty").
			WithSuggestion("Provide a valid AWS service code")
	}

	// Get products with location filter to find available regions
	filters := map[string]string{
		"servicecode": serviceCode,
	}

	products, err := c.GetProducts(ctx, serviceCode, filters)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to get regions for service").
			WithContext("serviceCode", serviceCode)
	}

	// Extract unique regions from product attributes
	regionSet := make(map[string]bool)
	for _, product := range products {
		if location, exists := product.Attributes["location"]; exists && location != "" {
			regionSet[location] = true
		}
		if region, exists := product.Attributes["regionCode"]; exists && region != "" {
			regionSet[region] = true
		}
	}

	// Convert set to slice
	var regions []string
	for region := range regionSet {
		regions = append(regions, region)
	}

	if len(regions) == 0 {
		return nil, errors.APIError("no regions found for service").
			WithContext("serviceCode", serviceCode).
			WithSuggestion("Check that the service code is valid").
			WithSuggestion("Some services may not have regional pricing data")
	}

	return regions, nil
}

// convertProduct converts AWS SDK pricing product to our interface type
func (c *Client) convertProduct(product string) (interfaces.PricingProduct, error) {
	// Parse the JSON product string
	var productData map[string]interface{}
	if err := json.Unmarshal([]byte(product), &productData); err != nil {
		return interfaces.PricingProduct{}, errors.APIErrorWithCause("failed to parse product JSON", err).
			WithContext("productData", product[:min(len(product), 200)]) // Limit context size
	}

	result := interfaces.PricingProduct{
		Attributes: make(map[string]string),
		Terms:      make(map[string]interface{}),
	}

	// Extract basic product information
	if sku, ok := productData["sku"].(string); ok {
		result.SKU = sku
	}

	if productFamily, ok := productData["productFamily"].(string); ok {
		result.ProductFamily = productFamily
	}

	if serviceCode, ok := productData["serviceCode"].(string); ok {
		result.ServiceCode = serviceCode
	}

	// Extract attributes
	if attributes, ok := productData["attributes"].(map[string]interface{}); ok {
		for key, value := range attributes {
			if strValue, ok := value.(string); ok {
				result.Attributes[key] = strValue
			}
		}
	}

	// Extract terms (pricing information)
	if terms, ok := productData["terms"].(map[string]interface{}); ok {
		result.Terms = terms
	}

	return result, nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetClientRegion returns the region the client is configured for
func (c *Client) GetClientRegion() string {
	return c.region
}

// GetMaxRetries returns the maximum number of retries configured
func (c *Client) GetMaxRetries() int {
	return c.maxRetries
}
