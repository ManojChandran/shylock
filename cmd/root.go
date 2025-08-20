package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"shylock/internal/aws"
	"shylock/internal/config"
	"shylock/internal/errors"
	"shylock/internal/estimators"
	"shylock/internal/interfaces"
	"shylock/internal/models"
	"shylock/internal/version"
)

var (
	// Global flags
	outputFormat string
	region       string
	verbose      bool
	currency     string

	// Root command
	rootCmd = &cobra.Command{
		Use:   "shylock",
		Short: "AWS cost estimation tool",
		Long: `Shylock is a CLI tool for estimating AWS costs based on resource configurations 
defined in JSON files. It supports multiple AWS services and provides detailed 
cost breakdowns with assumptions and recommendations.`,
		Example: `  # Estimate costs from a configuration file
  shylock estimate config.json

  # Estimate with specific output format
  shylock estimate config.json --output table

  # Estimate with verbose output
  shylock estimate config.json --verbose

  # Estimate for specific region
  shylock estimate config.json --region us-west-2`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Estimate command
	estimateCmd = &cobra.Command{
		Use:   "estimate [config-file]",
		Short: "Estimate AWS costs from configuration file",
		Long: `Estimate AWS costs based on resource configurations defined in a JSON file.
The configuration file should contain resource specifications including instance types,
storage classes, and other AWS service parameters.`,
		Example: `  # Basic cost estimation
  shylock estimate examples/simple-ec2.json

  # Output as JSON
  shylock estimate config.json --output json

  # Output as CSV
  shylock estimate config.json --output csv

  # Verbose output with detailed assumptions
  shylock estimate config.json --verbose

  # Override region for all resources
  shylock estimate config.json --region eu-west-1`,
		Args: cobra.ExactArgs(1),
		RunE: runEstimate,
	}

	// List command
	listCmd = &cobra.Command{
		Use:   "list",
		Short: "List supported AWS services and resource types",
		Long: `List all supported AWS services and resource types that can be used 
in configuration files. Also shows available options for each service.`,
		RunE: runList,
	}

	// Validate command
	validateCmd = &cobra.Command{
		Use:   "validate [config-file]",
		Short: "Validate configuration file without estimating costs",
		Long: `Validate a configuration file to check for syntax errors, missing required 
fields, and unsupported resource types without performing actual cost estimation.`,
		Args: cobra.ExactArgs(1),
		RunE: runValidate,
	}

	// Version command
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Display version information for Shylock.`,
		Run: func(cmd *cobra.Command, args []string) {
			// Import version package
			fmt.Print(getVersionInfo())
		},
	}
)

func init() {
	// Add persistent flags to root command
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "Output format (table, json, csv)")
	rootCmd.PersistentFlags().StringVarP(&region, "region", "r", "", "Override AWS region for all resources")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output with detailed information")
	rootCmd.PersistentFlags().StringVarP(&currency, "currency", "c", "USD", "Currency for cost display (USD, EUR, GBP)")

	// Add subcommands
	rootCmd.AddCommand(estimateCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(versionCmd)

	// Set help template
	rootCmd.SetHelpTemplate(getHelpTemplate())
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}

// getVersionInfo returns formatted version information
func getVersionInfo() string {
	return version.GetFullVersionString()
}

// runEstimate handles the estimate command
func runEstimate(cmd *cobra.Command, args []string) error {
	configFile := args[0]

	if verbose {
		fmt.Printf("🔍 Loading configuration from: %s\n", configFile)
	}

	// Validate file exists and has correct extension
	if err := validateConfigFile(configFile); err != nil {
		return err
	}

	// Parse configuration
	parser := config.NewParser()
	cfg, err := parser.ParseConfig(configFile)
	if err != nil {
		return errors.WrapError(err, errors.ConfigErrorType, "failed to parse configuration file").
			WithContext("configFile", configFile).
			WithSuggestion("Check the JSON syntax and required fields").
			WithSuggestion("Use 'shylock validate' to check for configuration errors")
	}

	// Apply CLI overrides
	if err := applyCliOverrides(cfg); err != nil {
		return err
	}

	if verbose {
		fmt.Printf("✅ Configuration loaded successfully (%d resources)\n", len(cfg.Resources))
	}

	// Create AWS client
	ctx := context.Background()
	awsClient, err := aws.NewClient(ctx, nil)
	if err != nil {
		return errors.WrapError(err, errors.AuthErrorType, "failed to create AWS client").
			WithSuggestion("Ensure AWS credentials are configured").
			WithSuggestion("Check AWS CLI configuration with 'aws configure list'")
	}

	if verbose {
		fmt.Println("🔗 Connected to AWS Pricing API")
	}

	// Create estimator factory
	factory := estimators.NewFactory(awsClient)

	// Validate configuration
	if err := factory.ValidateConfig(cfg); err != nil {
		return errors.WrapError(err, errors.ValidationErrorType, "configuration validation failed").
			WithSuggestion("Use 'shylock validate' to check for specific validation errors")
	}

	if verbose {
		fmt.Println("✅ Configuration validation passed")
		fmt.Println("💰 Estimating costs...")
	}

	// Estimate costs
	result, err := factory.EstimateFromConfig(ctx, cfg)
	if err != nil {
		return errors.WrapError(err, "", "cost estimation failed").
			WithSuggestion("Check AWS credentials and network connectivity").
			WithSuggestion("Verify that all resource types are supported in the specified regions")
	}

	if verbose {
		fmt.Printf("✅ Cost estimation completed (%d resources processed)\n\n", len(result.ResourceCosts))
	}

	// Output results
	return outputResults(result, outputFormat)
}

// runList handles the list command
func runList(cmd *cobra.Command, args []string) error {
	// Create a mock AWS client for listing (doesn't need real credentials)
	mockClient := &MockAWSClient{}
	factory := estimators.NewFactory(mockClient)

	fmt.Println("Supported AWS Services and Resource Types:")
	fmt.Println("=========================================")

	supportedTypes := factory.GetSupportedResourceTypes()
	allInfo := factory.GetAllEstimatorInfo()

	for _, resourceType := range supportedTypes {
		fmt.Printf("\n📦 %s\n", resourceType)

		if info, exists := allInfo[resourceType]; exists {
			switch resourceType {
			case "ALB":
				if types, ok := info["supportedALBTypes"].([]string); ok {
					fmt.Printf("   Load Balancer Types: %s\n", strings.Join(types, ", "))
				}
				if descriptions, ok := info["albTypeDescriptions"].(map[string]string); ok {
					for albType, desc := range descriptions {
						fmt.Printf("   • %s: %s\n", albType, desc)
					}
				}
			case "EC2":
				if families, ok := info["supportedInstanceFamilies"].([]string); ok {
					fmt.Printf("   Instance Families: %s\n", strings.Join(families, ", "))
				}
				if types, ok := info["commonInstanceTypes"].([]string); ok && len(types) > 0 {
					fmt.Printf("   Common Types: %s\n", strings.Join(types[:min(5, len(types))], ", "))
					if len(types) > 5 {
						fmt.Printf("   ... and %d more\n", len(types)-5)
					}
				}
			case "Lambda":
				if architectures, ok := info["supportedArchitectures"].([]string); ok {
					fmt.Printf("   Architectures: %s\n", strings.Join(architectures, ", "))
				}
				if memorySizes, ok := info["supportedMemorySizes"].([]int); ok && len(memorySizes) > 0 {
					memoryStrs := make([]string, min(6, len(memorySizes)))
					for i := 0; i < min(6, len(memorySizes)); i++ {
						memoryStrs[i] = fmt.Sprintf("%d MB", memorySizes[i])
					}
					fmt.Printf("   Memory Sizes: %s\n", strings.Join(memoryStrs, ", "))
					if len(memorySizes) > 6 {
						fmt.Printf("   ... and %d more\n", len(memorySizes)-6)
					}
				}
			case "RDS":
				if classes, ok := info["supportedInstanceClasses"].([]string); ok && len(classes) > 0 {
					fmt.Printf("   Instance Classes: %s\n", strings.Join(classes[:min(5, len(classes))], ", "))
					if len(classes) > 5 {
						fmt.Printf("   ... and %d more\n", len(classes)-5)
					}
				}
				if engines, ok := info["supportedEngines"].([]string); ok {
					fmt.Printf("   Database Engines: %s\n", strings.Join(engines, ", "))
				}
			case "S3":
				if classes, ok := info["supportedStorageClasses"].([]string); ok {
					fmt.Printf("   Storage Classes: %s\n", strings.Join(classes, ", "))
				}
			}
		}
	}

	fmt.Println("\nOutput Formats:")
	fmt.Println("===============")
	fmt.Println("• table  - Human-readable table format (default)")
	fmt.Println("• json   - JSON format for programmatic use")
	fmt.Println("• csv    - CSV format for spreadsheet import")

	fmt.Println("\nExample Configuration:")
	fmt.Println("=====================")
	fmt.Println(`{
  "version": "1.0",
  "resources": [
    {
      "type": "EC2",
      "name": "web-server",
      "region": "us-east-1",
      "properties": {
        "instanceType": "t3.micro",
        "count": 1
      }
    }
  ]
}`)

	return nil
}

// runValidate handles the validate command
func runValidate(cmd *cobra.Command, args []string) error {
	configFile := args[0]

	fmt.Printf("🔍 Validating configuration: %s\n", configFile)

	// Validate file exists and has correct extension
	if err := validateConfigFile(configFile); err != nil {
		return err
	}

	// Parse configuration
	parser := config.NewParser()
	cfg, err := parser.ParseConfig(configFile)
	if err != nil {
		fmt.Printf("❌ Configuration parsing failed\n")
		return errors.WrapError(err, errors.ConfigErrorType, "failed to parse configuration file").
			WithContext("configFile", configFile)
	}

	fmt.Printf("✅ Configuration syntax is valid\n")

	// Apply CLI overrides for validation
	if err := applyCliOverrides(cfg); err != nil {
		return err
	}

	// Create factory for validation (doesn't need real AWS client)
	mockClient := &MockAWSClient{}
	factory := estimators.NewFactory(mockClient)

	// Validate configuration
	if err := factory.ValidateConfig(cfg); err != nil {
		fmt.Printf("❌ Configuration validation failed\n")
		return errors.WrapError(err, errors.ValidationErrorType, "configuration validation failed")
	}

	fmt.Printf("✅ Configuration validation passed\n")
	fmt.Printf("📊 Found %d valid resources\n", len(cfg.Resources))

	// Show resource summary
	resourceCounts := make(map[string]int)
	for _, resource := range cfg.Resources {
		resourceCounts[resource.Type]++
	}

	fmt.Println("\nResource Summary:")
	for resourceType, count := range resourceCounts {
		fmt.Printf("  • %s: %d resource(s)\n", resourceType, count)
	}

	return nil
}

// Helper functions

func validateConfigFile(configFile string) error {
	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return errors.FileError("configuration file does not exist").
			WithContext("configFile", configFile).
			WithSuggestion("Check the file path and ensure the file exists").
			WithSuggestion("Use an absolute path or ensure you're in the correct directory")
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(configFile))
	if ext != ".json" {
		return errors.FileError("unsupported file format").
			WithContext("configFile", configFile).
			WithContext("extension", ext).
			WithSuggestion("Use a .json file for configuration").
			WithSuggestion("Convert your configuration to JSON format")
	}

	return nil
}

func applyCliOverrides(cfg *models.EstimationConfig) error {
	// Override region if specified
	if region != "" {
		if verbose {
			fmt.Printf("🌍 Overriding region to: %s\n", region)
		}
		for i := range cfg.Resources {
			cfg.Resources[i].Region = region
		}
	}

	// Override currency if specified
	if currency != "" && currency != "USD" {
		if verbose {
			fmt.Printf("💱 Setting currency to: %s\n", currency)
		}
		cfg.Options.Currency = currency
	}

	// Validate output format
	validFormats := []string{"table", "json", "csv"}
	formatValid := false
	for _, format := range validFormats {
		if outputFormat == format {
			formatValid = true
			break
		}
	}
	if !formatValid {
		return errors.ValidationError("invalid output format").
			WithContext("outputFormat", outputFormat).
			WithContext("validFormats", strings.Join(validFormats, ", ")).
			WithSuggestion("Use one of: table, json, csv")
	}

	return nil
}

func outputResults(result *models.EstimationResult, format string) error {
	switch format {
	case "table":
		return outputTable(result)
	case "json":
		return outputJSON(result)
	case "csv":
		return outputCSV(result)
	default:
		return errors.ValidationError("unsupported output format").
			WithContext("format", format).
			WithSuggestion("Use table, json, or csv")
	}
}

func getHelpTemplate() string {
	return `{{.Long}}

Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// MockAWSClient for listing and validation (doesn't need real AWS connection)
type MockAWSClient struct{}

func (m *MockAWSClient) GetProducts(ctx context.Context, serviceCode string, filters map[string]string) ([]interfaces.PricingProduct, error) {
	return []interfaces.PricingProduct{}, nil
}

func (m *MockAWSClient) DescribeServices(ctx context.Context) ([]interfaces.ServiceInfo, error) {
	return []interfaces.ServiceInfo{}, nil
}

func (m *MockAWSClient) GetRegions(ctx context.Context, serviceCode string) ([]string, error) {
	return []string{"us-east-1", "us-west-2", "eu-west-1"}, nil
}
