package alb

import (
	"context"
	"fmt"
	"time"

	"shylock/internal/aws"
	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// Estimator implements the ResourceEstimator interface for ALB (Application Load Balancer)
type Estimator struct {
	pricingService *aws.PricingService
}

// NewEstimator creates a new ALB cost estimator
func NewEstimator(awsClient interfaces.AWSPricingClient) interfaces.ResourceEstimator {
	return &Estimator{
		pricingService: aws.NewPricingService(awsClient),
	}
}

// SupportedResourceType returns the AWS resource type this estimator supports
func (e *Estimator) SupportedResourceType() string {
	return "ALB"
}

// ValidateResource validates that the resource specification is valid for ALB
func (e *Estimator) ValidateResource(resource models.ResourceSpec) error {
	if resource.Type != "ALB" {
		return errors.ValidationError("resource type must be 'ALB'").
			WithContext("actualType", resource.Type).
			WithSuggestion("Use 'ALB' as the resource type")
	}

	// Validate required properties
	requiredProps := []string{"type"}
	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return errors.ValidationError(fmt.Sprintf("missing required property '%s'", prop)).
				WithContext("resourceName", resource.Name).
				WithSuggestion(fmt.Sprintf("Add '%s' property to the resource configuration", prop))
		}
	}

	// Validate ALB type
	albType, err := resource.GetStringProperty("type")
	if err != nil {
		return errors.ValidationErrorWithCause("invalid type property", err).
			WithContext("resourceName", resource.Name).
			WithSuggestion("Ensure type is a string ('application' or 'network')")
	}

	if !e.isValidALBType(albType) {
		return errors.ValidationError("invalid ALB type").
			WithContext("resourceName", resource.Name).
			WithContext("albType", albType).
			WithSuggestion("Use 'application' or 'network' as ALB type").
			WithSuggestion("Check AWS documentation for supported load balancer types")
	}

	// Validate optional numeric properties
	numericProps := []string{"dataProcessingGB", "newConnectionsPerSecond", "activeConnectionsPerMinute", "ruleEvaluations"}
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

	// Validate region
	if err := e.pricingService.ValidateRegion(resource.Region); err != nil {
		return errors.WrapError(err, errors.ValidationErrorType, "invalid region for ALB resource").
			WithContext("resourceName", resource.Name)
	}

	return nil
}

// EstimateCost calculates the cost for an ALB
func (e *Estimator) EstimateCost(ctx context.Context, resource models.ResourceSpec) (*models.CostEstimate, error) {
	// Validate the resource first
	if err := e.ValidateResource(resource); err != nil {
		return nil, err
	}

	// Extract properties
	albType, _ := resource.GetStringProperty("type")

	// Get optional properties with defaults
	dataProcessingGB := 0
	if _, exists := resource.GetProperty("dataProcessingGB"); exists {
		if dp, err := resource.GetIntProperty("dataProcessingGB"); err == nil {
			dataProcessingGB = dp
		}
	}

	newConnectionsPerSecond := 0
	if _, exists := resource.GetProperty("newConnectionsPerSecond"); exists {
		if nc, err := resource.GetIntProperty("newConnectionsPerSecond"); err == nil {
			newConnectionsPerSecond = nc
		}
	}

	activeConnectionsPerMinute := 0
	if _, exists := resource.GetProperty("activeConnectionsPerMinute"); exists {
		if ac, err := resource.GetIntProperty("activeConnectionsPerMinute"); err == nil {
			activeConnectionsPerMinute = ac
		}
	}

	ruleEvaluations := 0
	if _, exists := resource.GetProperty("ruleEvaluations"); exists {
		if re, err := resource.GetIntProperty("ruleEvaluations"); err == nil {
			ruleEvaluations = re
		}
	}

	// Get pricing data from AWS
	products, err := e.pricingService.GetALBPricing(ctx, albType, resource.Region)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to retrieve ALB pricing data").
			WithContext("resourceName", resource.Name).
			WithContext("albType", albType).
			WithContext("region", resource.Region)
	}

	// Calculate costs based on ALB pricing model
	totalHourlyCost, costBreakdown, err := e.calculateALBCosts(products, albType, dataProcessingGB, newConnectionsPerSecond, activeConnectionsPerMinute, ruleEvaluations)
	if err != nil {
		return nil, errors.WrapError(err, errors.APIErrorType, "failed to calculate ALB costs").
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
	estimate.AddAssumption("24/7 load balancer operation assumed")
	estimate.AddAssumption("Pricing based on Load Balancer Capacity Units (LCU)")
	if dataProcessingGB == 0 {
		estimate.AddAssumption("No data processing costs included (set dataProcessingGB for data transfer pricing)")
	}
	if newConnectionsPerSecond == 0 && activeConnectionsPerMinute == 0 {
		estimate.AddAssumption("Minimal connection costs assumed (set connection parameters for accurate pricing)")
	}
	if ruleEvaluations == 0 {
		estimate.AddAssumption("Basic rule evaluation costs assumed (set ruleEvaluations for complex routing)")
	}

	// Add details
	estimate.SetDetail("albType", albType)
	estimate.SetDetail("dataProcessingGB", fmt.Sprintf("%d", dataProcessingGB))
	estimate.SetDetail("newConnectionsPerSecond", fmt.Sprintf("%d", newConnectionsPerSecond))
	estimate.SetDetail("activeConnectionsPerMinute", fmt.Sprintf("%d", activeConnectionsPerMinute))
	estimate.SetDetail("ruleEvaluations", fmt.Sprintf("%d", ruleEvaluations))

	// Add cost breakdown details
	for component, cost := range costBreakdown {
		estimate.SetDetail(fmt.Sprintf("%sCost", component), fmt.Sprintf("$%.4f/hour", cost))
	}

	return estimate, nil
}

// calculateALBCosts calculates ALB costs based on the LCU pricing model
func (e *Estimator) calculateALBCosts(products []interfaces.PricingProduct, albType string, dataProcessingGB, newConnectionsPerSecond, activeConnectionsPerMinute, ruleEvaluations int) (float64, map[string]float64, error) {
	costBreakdown := make(map[string]float64)

	// Find pricing components
	var loadBalancerHourPrice, lcuHourPrice float64

	for _, product := range products {
		// Extract pricing based on usage type
		if usageType, exists := product.Attributes["usageType"]; exists {
			price, err := e.pricingService.ExtractHourlyPrice(product)
			if err != nil {
				continue // Skip products we can't parse
			}

			if e.isLoadBalancerHourUsage(usageType) {
				loadBalancerHourPrice = price
				costBreakdown["loadBalancerHour"] = price
			} else if e.isLCUUsage(usageType) {
				lcuHourPrice = price
			}
		}
	}

	// Calculate LCU consumption based on the highest dimension
	lcuPerHour := e.calculateLCUConsumption(albType, dataProcessingGB, newConnectionsPerSecond, activeConnectionsPerMinute, ruleEvaluations)
	lcuCost := lcuPerHour * lcuHourPrice
	costBreakdown["lcu"] = lcuCost

	totalCost := loadBalancerHourPrice + lcuCost

	return totalCost, costBreakdown, nil
}

// calculateLCUConsumption calculates LCU consumption based on ALB metrics
func (e *Estimator) calculateLCUConsumption(albType string, dataProcessingGB, newConnectionsPerSecond, activeConnectionsPerMinute, ruleEvaluations int) float64 {
	if albType == "application" {
		return e.calculateApplicationLCU(dataProcessingGB, newConnectionsPerSecond, activeConnectionsPerMinute, ruleEvaluations)
	} else if albType == "network" {
		return e.calculateNetworkLCU(dataProcessingGB, newConnectionsPerSecond, activeConnectionsPerMinute)
	}
	return 1.0 // Default minimum LCU
}

// calculateApplicationLCU calculates LCU for Application Load Balancer
func (e *Estimator) calculateApplicationLCU(dataProcessingGB, newConnectionsPerSecond, activeConnectionsPerMinute, ruleEvaluations int) float64 {
	// ALB LCU dimensions (per hour):
	// - 1 GB of data processed
	// - 25 new connections per second (averaged over the hour)
	// - 3,000 active connections per minute (averaged over the hour)
	// - 1,000 rule evaluations per second (averaged over the hour)

	dataLCU := float64(dataProcessingGB) / 1.0
	connectionLCU := float64(newConnectionsPerSecond) / 25.0
	activeLCU := float64(activeConnectionsPerMinute) / 3000.0
	ruleLCU := float64(ruleEvaluations) / 1000.0

	// Take the maximum dimension (ALB charges for the highest consuming dimension)
	maxLCU := dataLCU
	if connectionLCU > maxLCU {
		maxLCU = connectionLCU
	}
	if activeLCU > maxLCU {
		maxLCU = activeLCU
	}
	if ruleLCU > maxLCU {
		maxLCU = ruleLCU
	}

	// Minimum 1 LCU per hour
	if maxLCU < 1.0 {
		maxLCU = 1.0
	}

	return maxLCU
}

// calculateNetworkLCU calculates LCU for Network Load Balancer
func (e *Estimator) calculateNetworkLCU(dataProcessingGB, newConnectionsPerSecond, activeConnectionsPerMinute int) float64 {
	// NLB LCU dimensions (per hour):
	// - 1 GB of data processed
	// - 800 new connections per second (averaged over the hour)
	// - 100,000 active connections per minute (averaged over the hour)

	dataLCU := float64(dataProcessingGB) / 1.0
	connectionLCU := float64(newConnectionsPerSecond) / 800.0
	activeLCU := float64(activeConnectionsPerMinute) / 100000.0

	// Take the maximum dimension
	maxLCU := dataLCU
	if connectionLCU > maxLCU {
		maxLCU = connectionLCU
	}
	if activeLCU > maxLCU {
		maxLCU = activeLCU
	}

	// Minimum 1 LCU per hour
	if maxLCU < 1.0 {
		maxLCU = 1.0
	}

	return maxLCU
}

// Helper functions

func (e *Estimator) isValidALBType(albType string) bool {
	validTypes := map[string]bool{
		"application": true,
		"network":     true,
	}
	return validTypes[albType]
}

func (e *Estimator) isLoadBalancerHourUsage(usageType string) bool {
	// Usage types for load balancer hours typically contain "LoadBalancerUsage"
	return containsSubstring(usageType, "LoadBalancerUsage")
}

func (e *Estimator) isLCUUsage(usageType string) bool {
	// Usage types for LCU typically contain "LCUUsage"
	return containsSubstring(usageType, "LCUUsage")
}

// GetSupportedALBTypes returns supported ALB types
func (e *Estimator) GetSupportedALBTypes() []string {
	return []string{"application", "network"}
}

// GetALBTypeDescription returns description for ALB types
func (e *Estimator) GetALBTypeDescription(albType string) string {
	descriptions := map[string]string{
		"application": "Application Load Balancer - Layer 7 load balancing with advanced routing",
		"network":     "Network Load Balancer - Layer 4 load balancing with ultra-high performance",
	}
	if desc, exists := descriptions[albType]; exists {
		return desc
	}
	return "Unknown ALB type"
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
