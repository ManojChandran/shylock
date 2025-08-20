# Implementation Plan

- [x] 1. Set up Go project structure and dependencies
  - Initialize Go module with name "shylock" (matching README)
  - Add AWS SDK for Go v2 dependencies (github.com/aws/aws-sdk-go-v2)
  - Add CLI framework dependency (github.com/spf13/cobra)
  - Create directory structure: cmd/, internal/, pkg/
  - Set up basic main.go entry point
  - _Requirements: 1.1, 4.1_

- [x] 2. Implement core data models and interfaces
  - Create internal/models package with ResourceSpec, EstimationConfig, and CostEstimate structs
  - Implement JSON unmarshaling with proper validation tags
  - Create internal/interfaces package with ResourceEstimator, AWSPricingClient, and ConfigParser interfaces
  - Write unit tests for data model validation and JSON parsing
  - _Requirements: 1.1, 3.1, 3.2, 3.3_

- [x] 3. Implement configuration file parsing and validation
  - Create internal/config package with ConfigParser implementation
  - Add JSON file reading with comprehensive error handling
  - Implement configuration validation logic for required fields and resource types
  - Create sample JSON configuration files in examples/ directory
  - Write unit tests for various JSON input scenarios (valid, invalid, malformed)
  - _Requirements: 1.1, 1.4, 3.1, 3.2, 3.3_

- [x] 4. Implement structured error handling system
  - Create internal/errors package with EstimationError types
  - Implement error categories: ConfigError, AuthError, APIError, NetworkError
  - Add error context and structured error messages
  - Write unit tests for error handling scenarios
  - _Requirements: 1.4, 2.3, 2.4, 4.2, 4.3, 4.4_

- [x] 5. Set up AWS client and authentication
  - Create internal/aws package with AWSPricingClient implementation
  - Implement AWS credential chain integration using AWS SDK v2
  - Add AWS session configuration with region support
  - Implement retry logic with exponential backoff for AWS API calls
  - Write unit tests with mocked AWS clients
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 6. Implement basic AWS Pricing API integration
  - Create functions to call AWS Pricing API GetProducts and DescribeServices
  - Implement pricing data parsing and filtering logic
  - Add comprehensive error handling for AWS API calls
  - Write integration tests with real AWS API calls (optional, with proper credentials)
  - _Requirements: 1.2, 2.1, 2.2, 4.3_

- [x] 7. Implement EC2 cost estimation logic
  - Create internal/estimators/ec2 package with EC2-specific cost calculator
  - Parse EC2 pricing data from AWS Pricing API responses
  - Calculate hourly, daily, and monthly costs for EC2 instances
  - Handle different instance types, operating systems, and tenancy options
  - Write unit tests for EC2 cost calculations with mock pricing data
  - _Requirements: 3.1, 5.1, 5.4_

- [x] 8. Implement S3 cost estimation logic
  - Create internal/estimators/s3 package with S3-specific cost calculator
  - Parse S3 pricing data for different storage classes and regions
  - Calculate storage costs based on GB and access patterns
  - Write unit tests for S3 cost calculations with mock pricing data
  - _Requirements: 3.2, 5.1, 5.4_

- [x] 9. Create resource estimator factory and dispatcher
  - Create internal/estimators package with ResourceEstimator interface implementation
  - Implement factory pattern for different resource types (EC2, S3)
  - Add dispatcher to handle multiple resources from JSON config
  - Implement cost aggregation and totaling logic
  - Write unit tests for resource type routing and estimation
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 10. Implement CLI argument parsing and command structure
  - Create cmd/root.go with cobra CLI setup
  - Add main command with flags: --output, --region, --verbose, --help
  - Implement help text and usage examples
  - Add input file validation and argument parsing
  - Write unit tests for CLI argument parsing and validation
  - _Requirements: 1.1, 4.1, 4.2, 5.2_

- [x] 11. Implement output formatting and display
  - Create internal/output package with formatters for table, JSON, and CSV formats
  - Implement cost result aggregation and summary display
  - Add verbose mode output with detailed API information and assumptions
  - Create table formatter with proper alignment and currency formatting
  - Write unit tests for different output formats
  - _Requirements: 1.3, 5.1, 5.2, 5.3_

- [x] 12. Integrate all components in main application flow
  - Wire together CLI parsing, config loading, AWS client, and cost estimation in main.go
  - Implement complete execution flow from JSON file input to cost output
  - Add graceful error handling, cleanup, and proper exit codes
  - Ensure all error types are properly handled and displayed to user
  - Write end-to-end integration tests with sample JSON files
  - _Requirements: 1.1, 1.2, 1.3, 4.1, 4.2_

- [x] 13. Implement ALB (Application Load Balancer) cost estimation
  - Create internal/estimators/alb package with ALB-specific cost calculator
  - Implement ALB pricing API integration for load balancer hours and LCU (Load Balancer Capacity Units)
  - Calculate costs based on ALB type (Application/Network), data processing, and connection hours
  - Add ALB validation to config parser with required properties (type, dataProcessingGB, newConnectionsPerSecond)
  - Register ALB estimator in factory and update supported resource types
  - Write unit tests for ALB cost calculations with mock pricing data
  - Create example JSON configuration file for ALB resources
  - _Requirements: 3.3, 3.4, 5.3_

- [x] 14. Add support for additional AWS services (optional enhancement)
  - Extend resource estimator to support RDS, Lambda, and other common services
  - Create service-specific cost calculators following established patterns
  - Update sample JSON configuration files for new resource types
  - Write unit tests for new service cost calculations
  - _Requirements: 3.3, 3.4, 5.3_

- [x] 15. Implement performance optimizations (optional enhancement)
  - Add concurrent processing for multiple resource cost calculations using goroutines
  - Implement pricing data caching within single execution to reduce API calls
  - Add connection pooling for AWS SDK clients
  - Write performance tests and benchmarks
  - _Requirements: 4.4, 5.4_

- [x] 16. Add missing unit tests for output package
  - Create internal/output/formatters_test.go with comprehensive test coverage
  - Test all output formatters (table, JSON, CSV, YAML) with various data scenarios
  - Test formatter factory registration and retrieval
  - Test error handling in formatting functions
  - Ensure 100% test coverage for critical output functionality
  - _Requirements: 1.3, 5.1, 5.2, 5.3_

- [x] 17. Fix RDS estimator implementation gap
  - Remove RDS from supported types in config parser OR implement RDS estimator
  - If implementing: create internal/estimators/rds package with RDS cost calculator
  - Add RDS pricing API integration following EC2/S3 patterns
  - Update factory to register RDS estimator
  - Write unit tests for RDS cost calculations
  - _Requirements: 3.3, 3.4, 5.3_

- [x] 18. Create comprehensive documentation and examples
  - Update README.md with installation and usage instructions
  - Create comprehensive example JSON configuration files for different scenarios
  - Add code documentation and comments following Go conventions
  - Create user guide with common use cases and troubleshooting
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 5.2_