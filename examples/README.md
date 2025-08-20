# Shylock Configuration Examples

This directory contains comprehensive examples of Shylock configuration files for various AWS architectures and use cases.

## Quick Start Examples

These simple examples demonstrate individual AWS services:

### Single Service Examples

- **[simple-ec2.json](simple-ec2.json)** - Basic EC2 t3.micro instance
- **[simple-alb.json](simple-alb.json)** - Application Load Balancer with minimal configuration
- **[simple-rds.json](simple-rds.json)** - PostgreSQL database with basic settings
- **[simple-lambda.json](simple-lambda.json)** - Serverless function with ARM64 architecture
- **[s3-storage.json](s3-storage.json)** - S3 buckets with different storage classes

### Usage
```bash
# Validate configuration
./shylock validate examples/simple-ec2.json

# Estimate costs
./shylock estimate examples/simple-ec2.json

# Get detailed output
./shylock estimate examples/simple-ec2.json --verbose
```

## Architecture Examples

These examples demonstrate complete application architectures:

### Web Applications

- **[web-application.json](web-application.json)** - Traditional 3-tier web application
  - Application Load Balancer
  - EC2 web servers
  - RDS database
  - S3 for static assets

- **[serverless-api.json](serverless-api.json)** - Serverless API architecture
  - Lambda functions for API endpoints
  - S3 for data storage
  - Minimal infrastructure footprint

### Development & Analytics

- **[development-environment.json](development-environment.json)** - Development setup
  - Smaller instance sizes
  - Single-AZ RDS
  - Cost-optimized configuration

- **[data-analytics.json](data-analytics.json)** - Data processing pipeline
  - Compute-optimized EC2 instances
  - Large Lambda functions for processing
  - S3 with Intelligent Tiering

### Production Environments

- **[production-setup.json](production-setup.json)** - Production-ready architecture
  - High-availability configuration
  - Multi-AZ RDS
  - Encrypted storage
  - Performance-optimized instances

- **[enterprise-architecture.json](enterprise-architecture.json)** - Large-scale enterprise setup
  - Multiple load balancers
  - Tiered application architecture
  - Multiple databases and read replicas
  - Comprehensive Lambda functions
  - Full S3 storage lifecycle

### Comprehensive Examples

- **[comprehensive-example.json](comprehensive-example.json)** - All services combined
  - Demonstrates every supported AWS service
  - Various configuration options
  - Good for testing all features

## Configuration Patterns

### Load Balancer Configurations

**Application Load Balancer (Layer 7)**:
```json
{
  "type": "ALB",
  "properties": {
    "type": "application",
    "dataProcessingGB": 100,
    "newConnectionsPerSecond": 50,
    "activeConnectionsPerMinute": 2000,
    "ruleEvaluations": 1000
  }
}
```

**Network Load Balancer (Layer 4)**:
```json
{
  "type": "ALB",
  "properties": {
    "type": "network",
    "dataProcessingGB": 500,
    "newConnectionsPerSecond": 200,
    "activeConnectionsPerMinute": 10000
  }
}
```

### EC2 Instance Patterns

**Web Server Tier**:
```json
{
  "type": "EC2",
  "properties": {
    "instanceType": "t3.medium",
    "count": 3,
    "operatingSystem": "Linux"
  }
}
```

**Compute-Intensive Workload**:
```json
{
  "type": "EC2",
  "properties": {
    "instanceType": "c5.2xlarge",
    "count": 2,
    "operatingSystem": "Linux"
  }
}
```

**Memory-Intensive Application**:
```json
{
  "type": "EC2",
  "properties": {
    "instanceType": "r5.xlarge",
    "count": 1,
    "operatingSystem": "Linux"
  }
}
```

### RDS Database Patterns

**Production Database**:
```json
{
  "type": "RDS",
  "properties": {
    "instanceClass": "db.r5.large",
    "engine": "postgres",
    "storageGB": 500,
    "multiAZ": true,
    "encrypted": true
  }
}
```

**Development Database**:
```json
{
  "type": "RDS",
  "properties": {
    "instanceClass": "db.t3.micro",
    "engine": "mysql",
    "storageGB": 20,
    "multiAZ": false
  }
}
```

### Lambda Function Patterns

**API Handler**:
```json
{
  "type": "Lambda",
  "properties": {
    "memoryMB": 512,
    "requestsPerMonth": 5000000,
    "averageDurationMs": 200,
    "architecture": "arm64"
  }
}
```

**Data Processing**:
```json
{
  "type": "Lambda",
  "properties": {
    "memoryMB": 3008,
    "requestsPerMonth": 100000,
    "averageDurationMs": 30000,
    "architecture": "x86_64"
  }
}
```

### S3 Storage Patterns

**Frequently Accessed Data**:
```json
{
  "type": "S3",
  "properties": {
    "storageClass": "STANDARD",
    "sizeGB": 1000,
    "requestsPerMonth": 1000000
  }
}
```

**Infrequently Accessed Data**:
```json
{
  "type": "S3",
  "properties": {
    "storageClass": "STANDARD_IA",
    "sizeGB": 5000,
    "requestsPerMonth": 50000
  }
}
```

**Archive Storage**:
```json
{
  "type": "S3",
  "properties": {
    "storageClass": "GLACIER",
    "sizeGB": 50000,
    "requestsPerMonth": 100
  }
}
```

## Testing Examples

### Validate All Examples
```bash
# Test all examples
for file in examples/*.json; do
  echo "Testing $file..."
  ./shylock validate "$file"
done
```

### Compare Costs
```bash
# Compare different architectures
./shylock estimate examples/development-environment.json --output json > dev-costs.json
./shylock estimate examples/production-setup.json --output json > prod-costs.json

# View side by side
./shylock estimate examples/development-environment.json
./shylock estimate examples/production-setup.json
```

### Generate Reports
```bash
# CSV for spreadsheet analysis
./shylock estimate examples/enterprise-architecture.json --output csv > enterprise-costs.csv

# JSON for automation
./shylock estimate examples/web-application.json --output json | jq '.totalMonthlyCost'
```

## Customization Tips

### Scaling Examples

1. **Horizontal Scaling**: Increase `count` for EC2 instances
2. **Vertical Scaling**: Use larger instance types
3. **Storage Scaling**: Adjust `sizeGB` and `storageGB` values
4. **Traffic Scaling**: Modify Lambda `requestsPerMonth` and ALB traffic metrics

### Regional Variations

```bash
# Test costs in different regions
./shylock estimate examples/web-application.json --region us-west-2
./shylock estimate examples/web-application.json --region eu-west-1
./shylock estimate examples/web-application.json --region ap-southeast-1
```

### Currency Conversion

```bash
# View costs in different currencies
./shylock estimate examples/production-setup.json --currency EUR
./shylock estimate examples/production-setup.json --currency GBP
./shylock estimate examples/production-setup.json --currency JPY
```

## Best Practices

1. **Start Simple**: Begin with single-service examples
2. **Validate First**: Always validate before estimating
3. **Use Realistic Values**: Base configurations on actual requirements
4. **Consider Growth**: Plan for scaling in your estimates
5. **Test Regions**: Compare costs across different AWS regions
6. **Version Control**: Keep your configurations in Git
7. **Document Assumptions**: Add comments about your configuration choices

## Creating Your Own Examples

1. **Copy a Similar Example**: Start with the closest match to your architecture
2. **Modify Gradually**: Change one service at a time
3. **Validate Frequently**: Use `./shylock validate` after each change
4. **Test Estimation**: Verify costs make sense with `./shylock estimate`
5. **Add Documentation**: Comment your configuration choices

---

These examples provide a solid foundation for estimating AWS costs across various architectures and use cases. Start with the simple examples and gradually work up to more complex configurations as you become familiar with Shylock's capabilities.