package interfaces

import (
	"context"

	"shylock/internal/models"
)

// ResourceEstimator defines the interface for estimating costs of AWS resources
type ResourceEstimator interface {
	// EstimateCost calculates the cost for a given resource specification
	EstimateCost(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error)

	// SupportedResourceType returns the AWS resource type this estimator supports
	SupportedResourceType() string

	// ValidateResource validates that the resource specification is valid for this estimator
	ValidateResource(resource models.ResourceSpec) error
}

// AWSPricingClient defines the interface for interacting with AWS Pricing API
type AWSPricingClient interface {
	// GetProducts retrieves pricing information for AWS services
	GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]PricingProduct, error)

	// DescribeServices lists available AWS services in the Pricing API
	DescribeServices(ctx context.Context) ([]ServiceInfo, error)

	// GetRegions returns available AWS regions for a service
	GetRegions(ctx context.Context, serviceCode string) ([]string, error)
}

// ConfigParser defines the interface for parsing configuration files
type ConfigParser interface {
	// ParseConfig reads and parses a configuration file
	ParseConfig(filePath string) (*models.EstimationConfig, error)

	// ValidateConfig validates the parsed configuration
	ValidateConfig(config *models.EstimationConfig) error

	// ParseConfigFromBytes parses configuration from byte array
	ParseConfigFromBytes(data []byte) (*models.EstimationConfig, error)
}

// OutputFormatter defines the interface for formatting estimation results
type OutputFormatter interface {
	// Format formats the estimation result according to the formatter's type
	Format(result *models.EstimationResult) (string, error)

	// FormatType returns the format type (e.g., "table", "json", "csv")
	FormatType() string
}

// EstimationEngine defines the main interface for the cost estimation engine
type EstimationEngine interface {
	// EstimateFromConfig performs cost estimation based on a configuration
	EstimateFromConfig(ctx context.Context, config *models.EstimationConfig) (*models.EstimationResult, error)

	// RegisterEstimator registers a resource estimator for a specific resource type
	RegisterEstimator(resourceType string, estimator ResourceEstimator) error

	// GetSupportedResourceTypes returns list of supported AWS resource types
	GetSupportedResourceTypes() []string
}

// PricingProduct represents a product from AWS Pricing API
type PricingProduct struct {
	SKU           string                 `json:"sku"`
	ProductFamily string                 `json:"productFamily"`
	ServiceCode   string                 `json:"serviceCode"`
	Attributes    map[string]string      `json:"attributes"`
	Terms         map[string]interface{} `json:"terms"`
}

// ServiceInfo represents information about an AWS service
type ServiceInfo struct {
	ServiceCode string            `json:"serviceCode"`
	ServiceName string            `json:"serviceName"`
	Attributes  map[string]string `json:"attributes"`
}

// PricingTerm represents pricing terms from AWS Pricing API
type PricingTerm struct {
	OfferTermCode   string                    `json:"offerTermCode"`
	SKU             string                    `json:"sku"`
	EffectiveDate   string                    `json:"effectiveDate"`
	PriceDimensions map[string]PriceDimension `json:"priceDimensions"`
	TermAttributes  map[string]string         `json:"termAttributes"`
}

// PriceDimension represents a price dimension from AWS Pricing API
type PriceDimension struct {
	RateCode     string            `json:"rateCode"`
	Description  string            `json:"description"`
	BeginRange   string            `json:"beginRange"`
	EndRange     string            `json:"endRange"`
	Unit         string            `json:"unit"`
	PricePerUnit map[string]string `json:"pricePerUnit"`
	AppliesTo    []string          `json:"appliesTo"`
}

// Logger defines the interface for logging within the application
type Logger interface {
	// Debug logs debug level messages
	Debug(msg string, fields ...interface{})

	// Info logs info level messages
	Info(msg string, fields ...interface{})

	// Warn logs warning level messages
	Warn(msg string, fields ...interface{})

	// Error logs error level messages
	Error(msg string, fields ...interface{})

	// Fatal logs fatal level messages and exits
	Fatal(msg string, fields ...interface{})
}
