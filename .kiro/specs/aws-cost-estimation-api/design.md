# Design Document

## Overview

The AWS Cost Estimation CLI tool is a Go-based command line application that reads JSON configuration files containing AWS resource specifications and provides cost estimates using AWS Pricing APIs. The tool will leverage the AWS SDK for Go to interact with AWS Pricing API and potentially AWS Cost Explorer API to calculate infrastructure costs.

The primary AWS services we'll integrate with are:
- **AWS Pricing API**: Provides current pricing information for AWS services
- **AWS Cost Explorer API**: Offers cost and usage data (for more advanced scenarios)

## Architecture

The CLI tool follows a modular architecture with clear separation of concerns:

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   CLI Layer     │───▶│  Business Logic  │───▶│   AWS Client    │
│                 │    │                  │    │                 │
│ - Argument      │    │ - JSON Parser    │    │ - Pricing API   │
│   Parsing       │    │ - Cost Calculator│    │ - Authentication│
│ - Output        │    │ - Validation     │    │ - Rate Limiting │
│   Formatting    │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
         │                       │                       │
         ▼                       ▼                       ▼
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   File System   │    │   Data Models    │    │   External APIs │
│                 │    │                  │    │                 │
│ - JSON File     │    │ - Resource Specs │    │ - AWS Pricing   │
│   Reading       │    │ - Cost Results   │    │ - AWS Cost      │
│ - Output Files  │    │ - Error Types    │    │   Explorer      │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## Components and Interfaces

### 1. CLI Interface
- **Command Structure**: `aws-cost-estimator [flags] <json-file>`
- **Flags**:
  - `--output, -o`: Output format (table, json, csv)
  - `--region, -r`: AWS region for pricing (default: us-east-1)
  - `--verbose, -v`: Verbose output
  - `--help, -h`: Help information

### 2. JSON Configuration Schema
```json
{
  "resources": [
    {
      "type": "ec2",
      "specifications": {
        "instanceType": "t3.medium",
        "region": "us-east-1",
        "operatingSystem": "Linux",
        "tenancy": "Shared",
        "quantity": 2
      }
    },
    {
      "type": "s3",
      "specifications": {
        "storageClass": "Standard",
        "region": "us-east-1",
        "storageGB": 1000
      }
    }
  ],
  "estimationPeriod": "monthly"
}
```

### 3. Core Interfaces

```go
// ResourceEstimator defines the interface for cost estimation
type ResourceEstimator interface {
    EstimateCost(ctx context.Context, resource ResourceSpec) (*CostEstimate, error)
}

// AWSPricingClient wraps AWS Pricing API calls
type AWSPricingClient interface {
    GetProducts(ctx context.Context, params *GetProductsInput) (*GetProductsOutput, error)
    DescribeServices(ctx context.Context) (*DescribeServicesOutput, error)
}

// ConfigParser handles JSON configuration parsing
type ConfigParser interface {
    ParseConfig(filePath string) (*EstimationConfig, error)
    ValidateConfig(config *EstimationConfig) error
}
```

### 4. Data Models

```go
type EstimationConfig struct {
    Resources         []ResourceSpec `json:"resources"`
    EstimationPeriod  string        `json:"estimationPeriod"`
}

type ResourceSpec struct {
    Type           string                 `json:"type"`
    Specifications map[string]interface{} `json:"specifications"`
}

type CostEstimate struct {
    ResourceType   string  `json:"resourceType"`
    HourlyCost     float64 `json:"hourlyCost"`
    DailyCost      float64 `json:"dailyCost"`
    MonthlyCost    float64 `json:"monthlyCost"`
    Currency       string  `json:"currency"`
    Region         string  `json:"region"`
    Assumptions    []string `json:"assumptions"`
}
```

## Error Handling

### Error Categories
1. **Configuration Errors**: Invalid JSON, missing required fields
2. **Authentication Errors**: AWS credential issues, permission problems
3. **API Errors**: AWS service unavailable, rate limiting, invalid parameters
4. **Network Errors**: Connection timeouts, DNS resolution failures

### Error Handling Strategy
- Use structured error types with context information
- Implement retry logic with exponential backoff for transient errors
- Provide clear, actionable error messages to users
- Log detailed error information in verbose mode

```go
type EstimationError struct {
    Type    ErrorType
    Message string
    Cause   error
    Context map[string]interface{}
}

type ErrorType int

const (
    ConfigError ErrorType = iota
    AuthError
    APIError
    NetworkError
)
```

## Testing Strategy

### Unit Testing
- Test JSON parsing and validation logic
- Mock AWS API responses for cost calculation testing
- Test error handling scenarios
- Validate output formatting functions

### Integration Testing
- Test with real AWS Pricing API (using test credentials)
- Validate end-to-end workflows with sample JSON files
- Test authentication with different credential sources

### Test Data
- Create sample JSON configuration files for different resource types
- Mock AWS Pricing API responses for consistent testing
- Test edge cases: empty files, malformed JSON, unsupported resources

### Testing Tools
- Use Go's built-in testing framework
- Implement table-driven tests for multiple scenarios
- Use testify library for assertions and mocking
- Create integration test suite with Docker containers if needed

## Implementation Considerations

### AWS Pricing API Integration
- The AWS Pricing API provides detailed pricing information but requires careful filtering
- Pricing data is region-specific and service-specific
- API responses can be large; implement efficient parsing
- Consider caching pricing data for repeated queries

### Performance Optimization
- Implement concurrent API calls for multiple resources
- Cache pricing data during single execution
- Use connection pooling for AWS SDK clients
- Implement request batching where possible

### Security Considerations
- Use AWS SDK's default credential chain
- Never log or expose AWS credentials
- Validate all input data to prevent injection attacks
- Use HTTPS for all AWS API communications

### Extensibility
- Design plugin architecture for adding new resource types
- Support custom pricing calculators for complex scenarios
- Allow configuration of different AWS pricing models (On-Demand, Reserved, Spot)