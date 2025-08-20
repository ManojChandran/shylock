package models

import (
	"fmt"
	"time"
)

// ResourceSpec represents a single AWS resource configuration
type ResourceSpec struct {
	Type       string                 `json:"type" validate:"required"`
	Name       string                 `json:"name" validate:"required"`
	Region     string                 `json:"region" validate:"required"`
	Properties map[string]interface{} `json:"properties" validate:"required"`
}

// EstimationConfig represents the complete configuration for cost estimation
type EstimationConfig struct {
	Version   string         `json:"version" validate:"required"`
	Resources []ResourceSpec `json:"resources" validate:"required,min=1"`
	Options   ConfigOptions  `json:"options,omitempty"`
}

// ConfigOptions represents optional configuration settings
type ConfigOptions struct {
	DefaultRegion string `json:"defaultRegion,omitempty"`
	Currency      string `json:"currency,omitempty"`
	TimeFrame     string `json:"timeFrame,omitempty"`
}

// CostEstimate represents the cost estimation result for a resource
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

// EstimationResult represents the complete estimation result
type EstimationResult struct {
	TotalHourlyCost  float64        `json:"totalHourlyCost"`
	TotalDailyCost   float64        `json:"totalDailyCost"`
	TotalMonthlyCost float64        `json:"totalMonthlyCost"`
	Currency         string         `json:"currency"`
	ResourceCosts    []CostEstimate `json:"resourceCosts"`
	GeneratedAt      time.Time      `json:"generatedAt"`
}

// Validate performs basic validation on ResourceSpec
func (r *ResourceSpec) Validate() error {
	if r.Type == "" {
		return fmt.Errorf("resource type is required")
	}
	if r.Name == "" {
		return fmt.Errorf("resource name is required")
	}
	if r.Region == "" {
		return fmt.Errorf("resource region is required")
	}
	if r.Properties == nil || len(r.Properties) == 0 {
		return fmt.Errorf("resource properties are required")
	}
	return nil
}

// Validate performs basic validation on EstimationConfig
func (c *EstimationConfig) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("configuration version is required")
	}
	if len(c.Resources) == 0 {
		return fmt.Errorf("at least one resource must be specified")
	}

	for i, resource := range c.Resources {
		if err := resource.Validate(); err != nil {
			return fmt.Errorf("resource %d validation failed: %w", i, err)
		}
	}

	return nil
}

// GetProperty safely retrieves a property from ResourceSpec
func (r *ResourceSpec) GetProperty(key string) (interface{}, bool) {
	value, exists := r.Properties[key]
	return value, exists
}

// GetStringProperty retrieves a string property with type checking
func (r *ResourceSpec) GetStringProperty(key string) (string, error) {
	value, exists := r.Properties[key]
	if !exists {
		return "", fmt.Errorf("property '%s' not found", key)
	}

	str, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("property '%s' is not a string", key)
	}

	return str, nil
}

// GetIntProperty retrieves an integer property with type checking
func (r *ResourceSpec) GetIntProperty(key string) (int, error) {
	value, exists := r.Properties[key]
	if !exists {
		return 0, fmt.Errorf("property '%s' not found", key)
	}

	// Handle both int and float64 (JSON numbers are float64 by default)
	switch v := value.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("property '%s' is not a number", key)
	}
}

// CalculateCosts calculates daily and monthly costs from hourly cost
func (c *CostEstimate) CalculateCosts() {
	c.DailyCost = c.HourlyCost * 24
	c.MonthlyCost = c.HourlyCost * 24 * 30 // Approximate month
}

// AddAssumption adds an assumption to the cost estimate
func (c *CostEstimate) AddAssumption(assumption string) {
	if c.Assumptions == nil {
		c.Assumptions = make([]string, 0)
	}
	c.Assumptions = append(c.Assumptions, assumption)
}

// SetDetail sets a detail key-value pair
func (c *CostEstimate) SetDetail(key, value string) {
	if c.Details == nil {
		c.Details = make(map[string]string)
	}
	c.Details[key] = value
}
