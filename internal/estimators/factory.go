// Package estimators provides the factory pattern implementation for managing
// AWS resource cost estimators. It handles registration, routing, and execution
// of cost estimation for different AWS services.
//
// The factory supports the following AWS services:
//   - EC2: Elastic Compute Cloud instances
//   - ALB: Application Load Balancers
//   - RDS: Relational Database Service
//   - Lambda: Serverless functions
//   - S3: Simple Storage Service
//
// Usage:
//
//	factory := estimators.NewFactory(awsClient)
//	result, err := factory.EstimateFromConfig(ctx, config)
package estimators

import (
	"context"
	"fmt"
	"sort"
	"time"

	"shylock/internal/errors"
	"shylock/internal/estimators/alb"
	"shylock/internal/estimators/ec2"
	"shylock/internal/estimators/lambda"
	"shylock/internal/estimators/rds"
	"shylock/internal/estimators/s3"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// Factory creates and manages resource estimators for different AWS services.
// It implements the factory pattern to provide a unified interface for cost
// estimation across multiple AWS resource types.
type Factory struct {
	estimators map[string]interfaces.ResourceEstimator // Map of resource type to estimator
	awsClient  interfaces.AWSPricingClient             // AWS Pricing API client
}

// NewFactory creates a new estimator factory with all supported AWS service
// estimators pre-registered. The factory automatically registers estimators
// for EC2, ALB, RDS, Lambda, and S3 services.
//
// Parameters:
//   - awsClient: AWS Pricing API client for retrieving pricing data
//
// Returns:
//   - *Factory: Configured factory with all estimators registered
func NewFactory(awsClient interfaces.AWSPricingClient) *Factory {
	factory := &Factory{
		estimators: make(map[string]interfaces.ResourceEstimator),
		awsClient:  awsClient,
	}

	// Register built-in estimators
	factory.RegisterEstimator("ALB", alb.NewEstimator(awsClient))
	factory.RegisterEstimator("EC2", ec2.NewEstimator(awsClient))
	factory.RegisterEstimator("Lambda", lambda.NewEstimator(awsClient))
	factory.RegisterEstimator("RDS", rds.NewEstimator(awsClient))
	factory.RegisterEstimator("S3", s3.NewEstimator(awsClient))

	return factory
}

// RegisterEstimator registers a resource estimator for a specific resource type
func (f *Factory) RegisterEstimator(resourceType string, estimator interfaces.ResourceEstimator) error {
	if resourceType == "" {
		return errors.ValidationError("resource type cannot be empty").
			WithSuggestion("Provide a valid resource type (e.g., 'EC2', 'S3')")
	}

	if estimator == nil {
		return errors.ValidationError("estimator cannot be nil").
			WithContext("resourceType", resourceType).
			WithSuggestion("Provide a valid estimator implementation")
	}

	// Validate that the estimator supports the claimed resource type
	if estimator.SupportedResourceType() != resourceType {
		return errors.ValidationError("estimator resource type mismatch").
			WithContext("expectedType", resourceType).
			WithContext("actualType", estimator.SupportedResourceType()).
			WithSuggestion("Ensure the estimator supports the specified resource type")
	}

	f.estimators[resourceType] = estimator
	return nil
}

// GetEstimator returns the estimator for a specific resource type
func (f *Factory) GetEstimator(resourceType string) (interfaces.ResourceEstimator, error) {
	if resourceType == "" {
		return nil, errors.ValidationError("resource type cannot be empty").
			WithSuggestion("Specify a resource type (e.g., 'EC2', 'S3')")
	}

	estimator, exists := f.estimators[resourceType]
	if !exists {
		supportedTypes := f.GetSupportedResourceTypes()
		return nil, errors.ValidationError("unsupported resource type").
			WithContext("resourceType", resourceType).
			WithContext("supportedTypes", fmt.Sprintf("[%s]", joinStrings(supportedTypes, ", "))).
			WithSuggestion(fmt.Sprintf("Use one of the supported resource types: %s", joinStrings(supportedTypes, ", "))).
			WithSuggestion("Check if the resource type is spelled correctly")
	}

	return estimator, nil
}

// GetSupportedResourceTypes returns a list of supported resource types
func (f *Factory) GetSupportedResourceTypes() []string {
	var types []string
	for resourceType := range f.estimators {
		types = append(types, resourceType)
	}
	sort.Strings(types) // Return in consistent order
	return types
}

// EstimateResource estimates the cost for a single resource
func (f *Factory) EstimateResource(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error) {
	estimator, err := f.GetEstimator(resource.Type)
	if err != nil {
		return nil, errors.WrapError(err, errors.ValidationErrorType, "failed to get estimator for resource").
			WithContext("resourceName", resource.Name).
			WithContext("resourceType", resource.Type)
	}

	estimate, err := estimator.EstimateCost(ctx, resource)
	if err != nil {
		return nil, errors.WrapError(err, "", "failed to estimate cost for resource").
			WithContext("resourceName", resource.Name).
			WithContext("resourceType", resource.Type)
	}

	return estimate, nil
}

// EstimateFromConfig estimates costs for all resources in a configuration
func (f *Factory) EstimateFromConfig(ctx context.Context, config *models.EstimationConfig) (*models.EstimationResult, error) {
	if config == nil {
		return nil, errors.ValidationError("configuration cannot be nil").
			WithSuggestion("Provide a valid estimation configuration")
	}

	if len(config.Resources) == 0 {
		return nil, errors.ValidationError("no resources to estimate").
			WithSuggestion("Add at least one resource to the configuration")
	}

	result := &models.EstimationResult{
		ResourceCosts: make([]models.CostEstimate, 0, len(config.Resources)),
		Currency:      "USD", // Default currency
		GeneratedAt:   time.Now(),
	}

	// Apply configuration options
	if config.Options.Currency != "" {
		result.Currency = config.Options.Currency
	}

	var totalHourly, totalDaily, totalMonthly float64
	var estimationErrors []error

	// Estimate cost for each resource
	for i, resource := range config.Resources {
		estimate, err := f.EstimateResource(ctx, resource)
		if err != nil {
			// Collect errors but continue with other resources
			wrappedErr := errors.WrapError(err, "", fmt.Sprintf("failed to estimate resource %d", i+1)).
				WithContext("resourceIndex", i).
				WithContext("resourceName", resource.Name).
				WithContext("resourceType", resource.Type)
			estimationErrors = append(estimationErrors, wrappedErr)
			continue
		}

		// Add to results
		result.ResourceCosts = append(result.ResourceCosts, *estimate)

		// Accumulate totals
		totalHourly += estimate.HourlyCost
		totalDaily += estimate.DailyCost
		totalMonthly += estimate.MonthlyCost
	}

	// Check if we have any successful estimates
	if len(result.ResourceCosts) == 0 {
		if len(estimationErrors) > 0 {
			// Return the first error if no estimates succeeded
			return nil, estimationErrors[0]
		}
		return nil, errors.ValidationError("no resources could be estimated").
			WithSuggestion("Check that all resources have valid configurations")
	}

	// Set totals
	result.TotalHourlyCost = totalHourly
	result.TotalDailyCost = totalDaily
	result.TotalMonthlyCost = totalMonthly

	// If there were some errors but also some successes, we could optionally
	// include error information in the result or log warnings
	// For now, we'll return the partial results

	return result, nil
}

// ValidateResource validates a resource using the appropriate estimator
func (f *Factory) ValidateResource(resource models.ResourceSpec) error {
	estimator, err := f.GetEstimator(resource.Type)
	if err != nil {
		return errors.WrapError(err, errors.ValidationErrorType, "failed to get estimator for validation").
			WithContext("resourceName", resource.Name).
			WithContext("resourceType", resource.Type)
	}

	return estimator.ValidateResource(resource)
}

// ValidateConfig validates all resources in a configuration
func (f *Factory) ValidateConfig(config *models.EstimationConfig) error {
	if config == nil {
		return errors.ValidationError("configuration cannot be nil")
	}

	// Basic config validation
	if err := config.Validate(); err != nil {
		return errors.WrapError(err, errors.ValidationErrorType, "configuration validation failed")
	}

	// Validate each resource
	for i, resource := range config.Resources {
		if err := f.ValidateResource(resource); err != nil {
			return errors.WrapError(err, errors.ValidationErrorType, fmt.Sprintf("resource %d validation failed", i+1)).
				WithContext("resourceIndex", i).
				WithContext("resourceName", resource.Name).
				WithContext("resourceType", resource.Type)
		}
	}

	return nil
}

// GetEstimatorInfo returns information about a specific estimator
func (f *Factory) GetEstimatorInfo(resourceType string) (map[string]interface{}, error) {
	estimator, err := f.GetEstimator(resourceType)
	if err != nil {
		return nil, err
	}

	info := map[string]interface{}{
		"resourceType":  estimator.SupportedResourceType(),
		"estimatorType": fmt.Sprintf("%T", estimator),
	}

	// Add type-specific information
	switch resourceType {
	case "ALB":
		if albEstimator, ok := estimator.(*alb.Estimator); ok {
			info["supportedALBTypes"] = albEstimator.GetSupportedALBTypes()

			// Add ALB type descriptions
			albTypeInfo := make(map[string]string)
			for _, albType := range albEstimator.GetSupportedALBTypes() {
				albTypeInfo[albType] = albEstimator.GetALBTypeDescription(albType)
			}
			info["albTypeDescriptions"] = albTypeInfo
		}
	case "EC2":
		if ec2Estimator, ok := estimator.(*ec2.Estimator); ok {
			info["supportedInstanceFamilies"] = ec2Estimator.GetSupportedInstanceFamilies()
			info["commonInstanceTypes"] = ec2Estimator.GetCommonInstanceTypes()
		}
	case "Lambda":
		if lambdaEstimator, ok := estimator.(*lambda.Estimator); ok {
			info["supportedMemorySizes"] = lambdaEstimator.GetSupportedMemorySizes()
			info["supportedArchitectures"] = lambdaEstimator.GetSupportedArchitectures()
			info["typicalUseCases"] = lambdaEstimator.GetTypicalUseCases()

			// Add architecture descriptions
			archInfo := make(map[string]string)
			for _, arch := range lambdaEstimator.GetSupportedArchitectures() {
				archInfo[arch] = lambdaEstimator.GetArchitectureDescription(arch)
			}
			info["architectureDescriptions"] = archInfo
		}
	case "RDS":
		if rdsEstimator, ok := estimator.(*rds.Estimator); ok {
			info["supportedInstanceClasses"] = rdsEstimator.GetSupportedInstanceClasses()
			info["supportedEngines"] = rdsEstimator.GetSupportedEngines()

			// Add engine descriptions
			engineInfo := make(map[string]string)
			for _, engine := range rdsEstimator.GetSupportedEngines() {
				engineInfo[engine] = rdsEstimator.GetEngineDescription(engine)
			}
			info["engineDescriptions"] = engineInfo
		}
	case "S3":
		if s3Estimator, ok := estimator.(*s3.Estimator); ok {
			info["supportedStorageClasses"] = s3Estimator.GetSupportedStorageClasses()

			// Add storage class descriptions
			storageClassInfo := make(map[string]string)
			for _, class := range s3Estimator.GetSupportedStorageClasses() {
				storageClassInfo[class] = s3Estimator.GetStorageClassDescription(class)
			}
			info["storageClassDescriptions"] = storageClassInfo
		}
	}

	return info, nil
}

// GetAllEstimatorInfo returns information about all registered estimators
func (f *Factory) GetAllEstimatorInfo() map[string]map[string]interface{} {
	result := make(map[string]map[string]interface{})

	for resourceType := range f.estimators {
		if info, err := f.GetEstimatorInfo(resourceType); err == nil {
			result[resourceType] = info
		}
	}

	return result
}

// EstimateResourceConcurrently estimates costs for resources concurrently
func (f *Factory) EstimateResourcesConcurrently(ctx context.Context, resources []models.ResourceSpec) ([]models.CostEstimate, []error) {
	if len(resources) == 0 {
		return nil, nil
	}

	// Create channels for results and errors
	type result struct {
		index    int
		estimate *models.CostEstimate
		err      error
	}

	resultChan := make(chan result, len(resources))

	// Start goroutines for each resource
	for i, resource := range resources {
		go func(index int, res models.ResourceSpec) {
			estimate, err := f.EstimateResource(ctx, res)
			resultChan <- result{
				index:    index,
				estimate: estimate,
				err:      err,
			}
		}(i, resource)
	}

	// Collect results
	estimates := make([]models.CostEstimate, len(resources))
	errs := make([]error, len(resources))

	for i := 0; i < len(resources); i++ {
		res := <-resultChan
		if res.err != nil {
			errs[res.index] = res.err
		} else if res.estimate != nil {
			estimates[res.index] = *res.estimate
		}
	}

	return estimates, errs
}

// Helper function to join strings
func joinStrings(strs []string, separator string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += separator + strs[i]
	}
	return result
}
