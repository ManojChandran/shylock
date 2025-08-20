package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"shylock/internal/errors"
	"shylock/internal/interfaces"
	"shylock/internal/models"
)

// Parser implements the ConfigParser interface
type Parser struct {
	supportedResourceTypes map[string]bool
}

// NewParser creates a new configuration parser
func NewParser() interfaces.ConfigParser {
	return &Parser{
		supportedResourceTypes: map[string]bool{
			"EC2":    true,
			"S3":     true,
			"RDS":    true,
			"ALB":    true,
			"Lambda": true,
			// Add more supported types as they are implemented
		},
	}
}

// ParseConfig reads and parses a configuration file
func (p *Parser) ParseConfig(filePath string) (*models.EstimationConfig, error) {
	// Validate file path
	if filePath == "" {
		return nil, errors.FileError("file path cannot be empty").
			WithSuggestion("Provide a valid path to a JSON configuration file")
	}

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, errors.FileErrorWithCause("configuration file does not exist", err).
			WithContext("filePath", filePath).
			WithSuggestion("Check that the file path is correct").
			WithSuggestion("Ensure the file exists and is readable")
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(filePath))
	if ext != ".json" {
		return nil, errors.FileErrorf("unsupported file format: %s (only .json files are supported)", ext).
			WithContext("filePath", filePath).
			WithContext("extension", ext).
			WithSuggestion("Use a .json file for configuration").
			WithSuggestion("Convert your configuration to JSON format")
	}

	// Read file contents
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, errors.FileErrorWithCause("failed to read configuration file", err).
			WithContext("filePath", filePath).
			WithSuggestion("Check file permissions").
			WithSuggestion("Ensure the file is not locked by another process")
	}

	// Parse configuration from bytes
	config, err := p.ParseConfigFromBytes(data)
	if err != nil {
		return nil, errors.WrapError(err, errors.ConfigErrorType, "failed to parse configuration file").
			WithContext("filePath", filePath)
	}

	return config, nil
}

// ParseConfigFromBytes parses configuration from byte array
func (p *Parser) ParseConfigFromBytes(data []byte) (*models.EstimationConfig, error) {
	if len(data) == 0 {
		return nil, errors.ConfigError("configuration data is empty").
			WithSuggestion("Provide a valid JSON configuration").
			WithSuggestion("Check that the file is not empty")
	}

	var config models.EstimationConfig

	// Parse JSON
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.ConfigErrorWithCause("invalid JSON format", err).
			WithSuggestion("Validate your JSON syntax using a JSON validator").
			WithSuggestion("Check for missing commas, brackets, or quotes")
	}

	// Validate the parsed configuration
	if err := p.ValidateConfig(&config); err != nil {
		return nil, errors.WrapError(err, errors.ValidationErrorType, "configuration validation failed")
	}

	// Apply default values
	p.applyDefaults(&config)

	return &config, nil
}

// ValidateConfig validates the parsed configuration
func (p *Parser) ValidateConfig(config *models.EstimationConfig) error {
	if config == nil {
		return errors.ValidationError("configuration cannot be nil")
	}

	// Basic validation using the model's validation
	if err := config.Validate(); err != nil {
		return errors.ValidationErrorWithCause("basic configuration validation failed", err).
			WithSuggestion("Check that all required fields are present").
			WithSuggestion("Refer to the configuration schema documentation")
	}

	// Additional validation for supported resource types
	for i, resource := range config.Resources {
		if !p.supportedResourceTypes[resource.Type] {
			return errors.ValidationErrorf("resource %d: unsupported resource type '%s'", i, resource.Type).
				WithContext("resourceIndex", i).
				WithContext("resourceName", resource.Name).
				WithContext("resourceType", resource.Type).
				WithContext("supportedTypes", p.getSupportedTypesString()).
				WithSuggestion(fmt.Sprintf("Use one of the supported resource types: %s", p.getSupportedTypesString())).
				WithSuggestion("Check the documentation for supported AWS services")
		}

		// Validate resource-specific requirements
		if err := p.validateResourceSpecific(&resource); err != nil {
			return errors.WrapError(err, errors.ValidationErrorType, fmt.Sprintf("resource %d (%s) validation failed", i, resource.Name)).
				WithContext("resourceIndex", i).
				WithContext("resourceName", resource.Name).
				WithContext("resourceType", resource.Type)
		}
	}

	// Validate options if present
	if err := p.validateOptions(&config.Options); err != nil {
		return errors.WrapError(err, errors.ValidationErrorType, "configuration options validation failed")
	}

	return nil
}

// validateResourceSpecific performs resource-type-specific validation
func (p *Parser) validateResourceSpecific(resource *models.ResourceSpec) error {
	switch resource.Type {
	case "EC2":
		return p.validateEC2Resource(resource)
	case "S3":
		return p.validateS3Resource(resource)
	case "RDS":
		return p.validateRDSResource(resource)
	case "ALB":
		return p.validateALBResource(resource)
	case "Lambda":
		return p.validateLambdaResource(resource)
	default:
		return fmt.Errorf("validation not implemented for resource type: %s", resource.Type)
	}
}

// validateEC2Resource validates EC2-specific properties
func (p *Parser) validateEC2Resource(resource *models.ResourceSpec) error {
	// Required properties for EC2
	requiredProps := []string{"instanceType"}

	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return fmt.Errorf("missing required property '%s' for EC2 resource", prop)
		}
	}

	// Validate instance type format
	instanceType, err := resource.GetStringProperty("instanceType")
	if err != nil {
		return fmt.Errorf("instanceType must be a string: %w", err)
	}

	if !p.isValidInstanceType(instanceType) {
		return fmt.Errorf("invalid instance type format: %s", instanceType)
	}

	// Validate optional count property
	if _, exists := resource.GetProperty("count"); exists {
		count, err := resource.GetIntProperty("count")
		if err != nil {
			return fmt.Errorf("count must be a number: %w", err)
		}
		if count <= 0 {
			return fmt.Errorf("count must be greater than 0, got %d", count)
		}
	}

	return nil
}

// validateS3Resource validates S3-specific properties
func (p *Parser) validateS3Resource(resource *models.ResourceSpec) error {
	// Required properties for S3
	requiredProps := []string{"storageClass"}

	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return fmt.Errorf("missing required property '%s' for S3 resource", prop)
		}
	}

	// Validate storage class
	storageClass, err := resource.GetStringProperty("storageClass")
	if err != nil {
		return fmt.Errorf("storageClass must be a string: %w", err)
	}

	validStorageClasses := []string{"STANDARD", "STANDARD_IA", "ONEZONE_IA", "GLACIER", "DEEP_ARCHIVE"}
	if !p.contains(validStorageClasses, storageClass) {
		return fmt.Errorf("invalid storage class '%s'. Valid options: %s",
			storageClass, strings.Join(validStorageClasses, ", "))
	}

	// Validate optional sizeGB property
	if _, exists := resource.GetProperty("sizeGB"); exists {
		sizeGB, err := resource.GetIntProperty("sizeGB")
		if err != nil {
			return fmt.Errorf("sizeGB must be a number: %w", err)
		}
		if sizeGB <= 0 {
			return fmt.Errorf("sizeGB must be greater than 0, got %d", sizeGB)
		}
	}

	return nil
}

// validateRDSResource validates RDS-specific properties
func (p *Parser) validateRDSResource(resource *models.ResourceSpec) error {
	// Required properties for RDS
	requiredProps := []string{"instanceClass", "engine"}

	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return fmt.Errorf("missing required property '%s' for RDS resource", prop)
		}
	}

	// Validate engine
	engine, err := resource.GetStringProperty("engine")
	if err != nil {
		return fmt.Errorf("engine must be a string: %w", err)
	}

	validEngines := []string{"mysql", "postgres", "mariadb", "oracle-ee", "sqlserver-ex"}
	if !p.contains(validEngines, engine) {
		return fmt.Errorf("invalid engine '%s'. Valid options: %s",
			engine, strings.Join(validEngines, ", "))
	}

	return nil
}

// validateALBResource validates ALB-specific properties
func (p *Parser) validateALBResource(resource *models.ResourceSpec) error {
	// Required properties for ALB
	requiredProps := []string{"type"}

	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return fmt.Errorf("missing required property '%s' for ALB resource", prop)
		}
	}

	// Validate ALB type
	albType, err := resource.GetStringProperty("type")
	if err != nil {
		return fmt.Errorf("type must be a string: %w", err)
	}

	validTypes := []string{"application", "network"}
	if !p.contains(validTypes, albType) {
		return fmt.Errorf("invalid ALB type '%s'. Valid options: %s",
			albType, strings.Join(validTypes, ", "))
	}

	// Validate optional numeric properties
	numericProps := []string{"dataProcessingGB", "newConnectionsPerSecond", "activeConnectionsPerMinute", "ruleEvaluations"}
	for _, prop := range numericProps {
		if _, exists := resource.GetProperty(prop); exists {
			value, err := resource.GetIntProperty(prop)
			if err != nil {
				return fmt.Errorf("%s must be a number: %w", prop, err)
			}
			if value < 0 {
				return fmt.Errorf("%s must be non-negative, got %d", prop, value)
			}
		}
	}

	return nil
}

// validateLambdaResource validates Lambda-specific properties
func (p *Parser) validateLambdaResource(resource *models.ResourceSpec) error {
	// Required properties for Lambda
	requiredProps := []string{"memoryMB"}

	for _, prop := range requiredProps {
		if _, exists := resource.GetProperty(prop); !exists {
			return fmt.Errorf("missing required property '%s' for Lambda resource", prop)
		}
	}

	// Validate memory size
	memoryMB, err := resource.GetIntProperty("memoryMB")
	if err != nil {
		return fmt.Errorf("memoryMB must be a number: %w", err)
	}

	if memoryMB < 128 || memoryMB > 10240 {
		return fmt.Errorf("memoryMB must be between 128 and 10240, got %d", memoryMB)
	}

	// Validate optional architecture
	if _, exists := resource.GetProperty("architecture"); exists {
		architecture, err := resource.GetStringProperty("architecture")
		if err != nil {
			return fmt.Errorf("architecture must be a string: %w", err)
		}

		validArchitectures := []string{"x86_64", "arm64"}
		if !p.contains(validArchitectures, architecture) {
			return fmt.Errorf("invalid architecture '%s'. Valid options: %s",
				architecture, strings.Join(validArchitectures, ", "))
		}
	}

	// Validate optional numeric properties
	numericProps := []string{"requestsPerMonth", "averageDurationMs", "storageGB"}
	for _, prop := range numericProps {
		if _, exists := resource.GetProperty(prop); exists {
			value, err := resource.GetIntProperty(prop)
			if err != nil {
				return fmt.Errorf("%s must be a number: %w", prop, err)
			}
			if value < 0 {
				return fmt.Errorf("%s must be non-negative, got %d", prop, value)
			}
		}
	}

	return nil
}

// validateOptions validates configuration options
func (p *Parser) validateOptions(options *models.ConfigOptions) error {
	if options == nil {
		return nil // Options are optional
	}

	// Validate currency if specified
	if options.Currency != "" {
		validCurrencies := []string{"USD", "EUR", "GBP", "JPY"}
		if !p.contains(validCurrencies, options.Currency) {
			return fmt.Errorf("invalid currency '%s'. Valid options: %s",
				options.Currency, strings.Join(validCurrencies, ", "))
		}
	}

	// Validate timeFrame if specified
	if options.TimeFrame != "" {
		validTimeFrames := []string{"hourly", "daily", "monthly"}
		if !p.contains(validTimeFrames, options.TimeFrame) {
			return fmt.Errorf("invalid timeFrame '%s'. Valid options: %s",
				options.TimeFrame, strings.Join(validTimeFrames, ", "))
		}
	}

	return nil
}

// applyDefaults applies default values to the configuration
func (p *Parser) applyDefaults(config *models.EstimationConfig) {
	// Apply default options
	if config.Options.Currency == "" {
		config.Options.Currency = "USD"
	}
	if config.Options.TimeFrame == "" {
		config.Options.TimeFrame = "monthly"
	}

	// Apply default region to resources that don't have one
	for i := range config.Resources {
		if config.Resources[i].Region == "" && config.Options.DefaultRegion != "" {
			config.Resources[i].Region = config.Options.DefaultRegion
		}
	}
}

// Helper functions

func (p *Parser) getSupportedTypesString() string {
	var types []string
	for resourceType := range p.supportedResourceTypes {
		types = append(types, resourceType)
	}
	return strings.Join(types, ", ")
}

func (p *Parser) isValidInstanceType(instanceType string) bool {
	// Basic validation for EC2 instance type format (e.g., t3.micro, m5.large)
	parts := strings.Split(instanceType, ".")
	return len(parts) == 2 && len(parts[0]) > 0 && len(parts[1]) > 0
}

func (p *Parser) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
