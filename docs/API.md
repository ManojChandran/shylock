# Shylock API Documentation

This document describes the internal API structure and interfaces for developers who want to extend or integrate with Shylock.

## Table of Contents

1. [Core Interfaces](#core-interfaces)
2. [Resource Estimators](#resource-estimators)
3. [Configuration Parser](#configuration-parser)
4. [Output Formatters](#output-formatters)
5. [Error Handling](#error-handling)
6. [Performance Optimizations](#performance-optimizations)
7. [Extending Shylock](#extending-shylock)

## Core Interfaces

### ResourceEstimator Interface

The `ResourceEstimator` interface defines how cost estimators work:

```go
type ResourceEstimator interface {
    EstimateCost(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error)
    SupportedResourceType() string
    ValidateResource(resource models.ResourceSpec) error
}
```

**Methods**:
- `EstimateCost`: Calculates cost for a resource specification
- `SupportedResourceType`: Returns the AWS service type (e.g., "EC2", "RDS")
- `ValidateResource`: Validates resource configuration before estimation

### AWSPricingClient Interface

Defines interaction with AWS Pricing API:

```go
type AWSPricingClient interface {
    GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]PricingProduct, error)
    DescribeServices(ctx context.Context) ([]ServiceInfo, error)
    GetRegions(ctx context.Context, serviceCode string) ([]string, error)
}
```

### OutputFormatter Interface

Defines output formatting capabilities:

```go
type OutputFormatter interface {
    Format(result *models.EstimationResult) (string, error)
    FormatType() string
}
```

## Resource Estimators

### Creating a New Estimator

To add support for a new AWS service, implement the `ResourceEstimator` interface:

```go
package myservice

import (
    "context"
    "shylock/internal/interfaces"
    "shylock/internal/models"
)

type Estimator struct {
    pricingService *aws.PricingService
}

func NewEstimator(awsClient interfaces.AWSPricingClient) interfaces.ResourceEstimator {
    return &Estimator{
        pricingService: aws.NewPricingService(awsClient),
    }
}

func (e *Estimator) SupportedResourceType() string {
    return "MyService"
}

func (e *Estimator) ValidateResource(resource models.ResourceSpec) error {
    // Implement validation logic
    return nil
}

func (e *Estimator) EstimateCost(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error) {
    // Implement cost calculation logic
    return &models.CostEstimate{}, nil
}
```

### Registering an Estimator

Add your estimator to the factory:

```go
// In internal/estimators/factory.go
import "shylock/internal/estimators/myservice"

func NewFactory(awsClient interfaces.AWSPricingClient) *Factory {
    factory := &Factory{
        estimators: make(map[string]interfaces.ResourceEstimator),
        awsClient:  awsClient,
    }

    // Register your estimator
    factory.RegisterEstimator("MyService", myservice.NewEstimator(awsClient))
    
    return factory
}
```

### Estimator Best Practices

1. **Validation First**: Always validate resources before estimation
2. **Error Context**: Provide detailed error messages with context
3. **Assumptions**: Document pricing assumptions clearly
4. **Details**: Include relevant configuration details in results
5. **Testing**: Write comprehensive unit tests

## Configuration Parser

### Adding New Resource Types

1. **Update Supported Types**:
   ```go
   // In internal/config/parser.go
   supportedResourceTypes: map[string]bool{
       "EC2":       true,
       "MyService": true, // Add your service
   }
   ```

2. **Add Validation Function**:
   ```go
   func (p *Parser) validateMyServiceResource(resource *models.ResourceSpec) error {
       // Implement service-specific validation
       return nil
   }
   ```

3. **Update Switch Statement**:
   ```go
   func (p *Parser) validateResourceSpecific(resource *models.ResourceSpec) error {
       switch resource.Type {
       case "MyService":
           return p.validateMyServiceResource(resource)
       // ... other cases
       }
   }
   ```

## Output Formatters

### Creating Custom Formatters

Implement the `OutputFormatter` interface:

```go
type MyFormatter struct{}

func (f *MyFormatter) FormatType() string {
    return "myformat"
}

func (f *MyFormatter) Format(result *models.EstimationResult) (string, error) {
    // Implement custom formatting logic
    return "formatted output", nil
}
```

### Registering Formatters

```go
// In your application code
factory := output.NewFormatterFactory()
factory.RegisterFormatter(&MyFormatter{})
```

## Error Handling

### Error Types

Shylock uses structured error handling with specific error types:

```go
const (
    ConfigErrorType     ErrorType = "CONFIG"
    AuthErrorType       ErrorType = "AUTH"
    APIErrorType        ErrorType = "API"
    NetworkErrorType    ErrorType = "NETWORK"
    ValidationErrorType ErrorType = "VALIDATION"
    FileErrorType       ErrorType = "FILE"
)
```

### Creating Errors

```go
// Validation error with context and suggestions
return errors.ValidationError("invalid instance type").
    WithContext("instanceType", instanceType).
    WithContext("resourceName", resource.Name).
    WithSuggestion("Use valid EC2 instance type format").
    WithSuggestion("Check AWS documentation for available types")

// API error with cause
return errors.APIErrorWithCause("failed to retrieve pricing", err).
    WithContext("serviceCode", serviceCode)
```

### Error Best Practices

1. **Use Appropriate Types**: Choose the correct error type
2. **Add Context**: Include relevant information
3. **Provide Suggestions**: Help users resolve issues
4. **Wrap Errors**: Preserve original error information

## Performance Optimizations

### Using Optimized Factory

```go
import "shylock/internal/performance"

// Create optimized factory
config := &performance.OptimizedFactoryConfig{
    MaxConcurrency: 8,
    EnableCaching:  true,
    EnableBatching: true,
    BatchSize:      10,
    CacheTTL:       15 * time.Minute,
    MaxCacheSize:   1000,
}

factory := performance.NewOptimizedFactory(awsClient, config)

// Use optimized estimation
result, err := factory.EstimateFromConfigOptimized(ctx, estimationConfig)
```

### Cache Management

```go
// Get cache statistics
stats := factory.GetCacheStats()
fmt.Printf("Cache hits: %d, entries: %d\n", stats.TotalHits, stats.TotalEntries)

// Clear cache
factory.ClearCache()

// Cleanup expired entries
removed := factory.CleanupExpiredCache()
```

### Performance Configuration

```go
type OptimizedFactoryConfig struct {
    MaxConcurrency int           // Concurrent goroutines (default: 2x CPU)
    EnableCaching  bool          // Enable pricing cache (default: true)
    EnableBatching bool          // Enable batch processing (default: true)
    BatchSize      int           // Resources per batch (default: 10)
    CacheTTL       time.Duration // Cache expiration (default: 15min)
    MaxCacheSize   int           // Max cache entries (default: 1000)
}
```

## Extending Shylock

### Adding a New AWS Service

1. **Create Estimator Package**:
   ```
   internal/estimators/myservice/
   ├── estimator.go
   └── estimator_test.go
   ```

2. **Implement Pricing Service Method**:
   ```go
   // In internal/aws/pricing.go
   func (p *PricingService) GetMyServicePricing(ctx context.Context, params...) ([]interfaces.PricingProduct, error) {
       // Implementation
   }
   ```

3. **Add Configuration Validation**:
   ```go
   // In internal/config/parser.go
   func (p *Parser) validateMyServiceResource(resource *models.ResourceSpec) error {
       // Implementation
   }
   ```

4. **Register in Factory**:
   ```go
   // In internal/estimators/factory.go
   factory.RegisterEstimator("MyService", myservice.NewEstimator(awsClient))
   ```

5. **Add CLI Support**:
   ```go
   // In cmd/root.go - update list command
   case "MyService":
       // Add service-specific information display
   ```

6. **Create Examples**:
   ```
   examples/myservice-example.json
   ```

7. **Write Tests**:
   - Unit tests for estimator
   - Integration tests
   - Update factory tests

### Testing New Services

```go
func TestMyServiceEstimator_EstimateCost(t *testing.T) {
    mockClient := &MockAWSClient{
        products: createMockProducts(),
    }
    
    estimator := NewEstimator(mockClient)
    
    resource := models.ResourceSpec{
        Type:   "MyService",
        Name:   "test-resource",
        Region: "us-east-1",
        Properties: map[string]interface{}{
            "requiredProperty": "value",
        },
    }
    
    estimate, err := estimator.EstimateCost(context.Background(), resource)
    
    // Assertions
    assert.NoError(t, err)
    assert.NotNil(t, estimate)
    assert.Equal(t, "MyService", estimate.ResourceType)
}
```

### Integration Checklist

When adding a new service, ensure:

- [ ] Estimator implements `ResourceEstimator` interface
- [ ] Pricing service method added
- [ ] Configuration validation implemented
- [ ] Factory registration completed
- [ ] CLI list command updated
- [ ] Example configurations created
- [ ] Comprehensive tests written
- [ ] Documentation updated

## Data Models

### ResourceSpec

Represents a single AWS resource configuration:

```go
type ResourceSpec struct {
    Type       string                 `json:"type"`
    Name       string                 `json:"name"`
    Region     string                 `json:"region"`
    Properties map[string]interface{} `json:"properties"`
}
```

### CostEstimate

Represents the cost estimation result:

```go
type CostEstimate struct {
    ResourceName string            `json:"resourceName"`
    ResourceType string            `json:"resourceType"`
    Region       string            `json:"region"`
    HourlyCost   float64           `json:"hourlyCost"`
    DailyCost    float64           `json:"dailyCost"`
    MonthlyCost  float64           `json:"monthlyCost"`
    Currency     string            `json:"currency"`
    Assumptions  []string          `json:"assumptions,omitempty"`
    Details      map[string]string `json:"details,omitempty"`
    Timestamp    time.Time         `json:"timestamp"`
}
```

### EstimationResult

Complete estimation result with totals:

```go
type EstimationResult struct {
    TotalHourlyCost  float64        `json:"totalHourlyCost"`
    TotalDailyCost   float64        `json:"totalDailyCost"`
    TotalMonthlyCost float64        `json:"totalMonthlyCost"`
    Currency         string         `json:"currency"`
    ResourceCosts    []CostEstimate `json:"resourceCosts"`
    GeneratedAt      time.Time      `json:"generatedAt"`
}
```

## Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI Layer     │    │  Configuration   │    │  Output Layer   │
│   (cmd/)        │    │  Parser          │    │  (internal/     │
│                 │    │  (internal/      │    │   output/)      │
│  - Commands     │    │   config/)       │    │                 │
│  - Flags        │    │                  │    │  - Table        │
│  - Help         │    │  - JSON parsing  │    │  - JSON         │
│                 │    │  - Validation    │    │  - CSV          │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         └───────────────────────┼───────────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │   Estimation Engine     │
                    │   (internal/estimators) │
                    │                         │
                    │  - Factory Pattern      │
                    │  - Resource Routing     │
                    │  - Cost Aggregation     │
                    └─────────────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │   Service Estimators    │
                    │                         │
                    │  ┌─────┐ ┌─────┐ ┌─────┐│
                    │  │ EC2 │ │ RDS │ │ S3  ││
                    │  └─────┘ └─────┘ └─────┘│
                    │  ┌─────┐ ┌─────┐        │
                    │  │ ALB │ │Lambda│       │
                    │  └─────┘ └─────┘        │
                    └─────────────────────────┘
                                 │
                    ┌─────────────────────────┐
                    │   AWS Integration       │
                    │   (internal/aws)        │
                    │                         │
                    │  - Pricing API Client   │
                    │  - Authentication       │
                    │  - Region Mapping       │
                    │  - Error Handling       │
                    └─────────────────────────┘
```

---

This API documentation provides the foundation for extending Shylock with additional AWS services or integrating it into larger systems.