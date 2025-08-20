# Shylock User Guide

This comprehensive guide will help you get the most out of Shylock, the AWS cost estimation tool.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Configuration File Format](#configuration-file-format)
3. [Service-Specific Guides](#service-specific-guides)
4. [Advanced Usage](#advanced-usage)
5. [Best Practices](#best-practices)
6. [Troubleshooting](#troubleshooting)

## Getting Started

### Installation and Setup

1. **Build Shylock**
   ```bash
   git clone <repository-url>
   cd shylock
   go build -o shylock
   ```

2. **Configure AWS Credentials**
   ```bash
   # Option 1: AWS CLI
   aws configure
   
   # Option 2: Environment Variables
   export AWS_ACCESS_KEY_ID=your-access-key
   export AWS_SECRET_ACCESS_KEY=your-secret-key
   export AWS_REGION=us-east-1
   ```

3. **Verify Installation**
   ```bash
   ./shylock --help
   ./shylock list
   ```

### Your First Estimation

Create a simple configuration file:

```json
{
  "version": "1.0",
  "resources": [
    {
      "type": "EC2",
      "name": "my-server",
      "region": "us-east-1",
      "properties": {
        "instanceType": "t3.micro",
        "count": 1
      }
    }
  ]
}
```

Run the estimation:
```bash
./shylock estimate my-config.json
```

## Configuration File Format

### Basic Structure

```json
{
  "version": "1.0",
  "resources": [
    {
      "type": "SERVICE_TYPE",
      "name": "resource-name",
      "region": "aws-region",
      "properties": {
        // Service-specific properties
      }
    }
  ],
  "options": {
    "defaultRegion": "us-east-1",
    "currency": "USD",
    "timeFrame": "monthly"
  }
}
```

### Required Fields

- `version`: Configuration format version (currently "1.0")
- `resources`: Array of AWS resources to estimate
- `type`: AWS service type (EC2, ALB, RDS, Lambda, S3)
- `name`: Unique identifier for the resource
- `region`: AWS region for the resource
- `properties`: Service-specific configuration

### Optional Fields

- `options`: Global configuration options
- `defaultRegion`: Default region for resources without explicit region
- `currency`: Cost display currency (USD, EUR, GBP, JPY)
- `timeFrame`: Time frame for cost display (hourly, daily, monthly)

## Service-Specific Guides

### EC2 - Elastic Compute Cloud

EC2 instances are virtual servers in the cloud.

#### Required Properties
- `instanceType`: EC2 instance type (e.g., "t3.micro", "m5.large")

#### Optional Properties
- `count`: Number of instances (default: 1)
- `operatingSystem`: OS type (default: "Linux")
  - Options: "Linux", "Windows", "RHEL", "SUSE"
- `tenancy`: Instance tenancy (default: "Shared")
  - Options: "Shared", "Dedicated", "Host"

#### Example Configurations

**Basic Web Server**
```json
{
  "type": "EC2",
  "name": "web-server",
  "region": "us-east-1",
  "properties": {
    "instanceType": "t3.small",
    "count": 2,
    "operatingSystem": "Linux"
  }
}
```

**Windows Server**
```json
{
  "type": "EC2",
  "name": "windows-server",
  "region": "us-east-1",
  "properties": {
    "instanceType": "m5.large",
    "operatingSystem": "Windows",
    "tenancy": "Dedicated"
  }
}
```

#### Common Instance Types
- **Burstable**: t3.nano, t3.micro, t3.small, t3.medium, t3.large
- **General Purpose**: m5.large, m5.xlarge, m5.2xlarge, m5.4xlarge
- **Compute Optimized**: c5.large, c5.xlarge, c5.2xlarge, c5.4xlarge
- **Memory Optimized**: r5.large, r5.xlarge, r5.2xlarge, r5.4xlarge

### ALB - Application Load Balancer

Load balancers distribute incoming traffic across multiple targets.

#### Required Properties
- `type`: Load balancer type
  - "application": Layer 7 (HTTP/HTTPS)
  - "network": Layer 4 (TCP/UDP)

#### Optional Properties
- `dataProcessingGB`: Data processed per month (default: 0)
- `newConnectionsPerSecond`: New connections per second (default: 0)
- `activeConnectionsPerMinute`: Active connections per minute (default: 0)
- `ruleEvaluations`: Rule evaluations per second (default: 0, ALB only)

#### Example Configurations

**Application Load Balancer**
```json
{
  "type": "ALB",
  "name": "web-alb",
  "region": "us-east-1",
  "properties": {
    "type": "application",
    "dataProcessingGB": 100,
    "newConnectionsPerSecond": 50,
    "activeConnectionsPerMinute": 2000,
    "ruleEvaluations": 1000
  }
}
```

**Network Load Balancer**
```json
{
  "type": "ALB",
  "name": "tcp-nlb",
  "region": "us-east-1",
  "properties": {
    "type": "network",
    "dataProcessingGB": 500,
    "newConnectionsPerSecond": 200,
    "activeConnectionsPerMinute": 10000
  }
}
```

#### LCU Pricing Model
ALB uses Load Balancer Capacity Units (LCU) for pricing:
- **Application LB**: 1 LCU = 1 GB data OR 25 new connections/sec OR 3,000 active connections/min OR 1,000 rule evaluations/sec
- **Network LB**: 1 LCU = 1 GB data OR 800 new connections/sec OR 100,000 active connections/min

### RDS - Relational Database Service

Managed database service supporting multiple database engines.

#### Required Properties
- `instanceClass`: RDS instance class (e.g., "db.t3.micro", "db.r5.large")
- `engine`: Database engine

#### Supported Engines
- `mysql`: MySQL Community Edition
- `postgres`: PostgreSQL
- `mariadb`: MariaDB Community Edition
- `oracle-ee`: Oracle Database Enterprise Edition
- `oracle-se2`: Oracle Database Standard Edition 2
- `sqlserver-ex`: SQL Server Express Edition
- `sqlserver-web`: SQL Server Web Edition
- `sqlserver-se`: SQL Server Standard Edition
- `sqlserver-ee`: SQL Server Enterprise Edition
- `aurora-mysql`: Aurora MySQL-Compatible Edition
- `aurora-postgresql`: Aurora PostgreSQL-Compatible Edition

#### Optional Properties
- `storageGB`: Storage size in GB (default: 20)
- `storageType`: Storage type (default: "gp2")
  - Options: "gp2", "gp3", "io1", "io2"
- `multiAZ`: Multi-AZ deployment (default: false)
- `encrypted`: Encryption at rest (default: false)

#### Example Configurations

**Production MySQL Database**
```json
{
  "type": "RDS",
  "name": "prod-mysql",
  "region": "us-east-1",
  "properties": {
    "instanceClass": "db.r5.xlarge",
    "engine": "mysql",
    "storageGB": 500,
    "storageType": "gp2",
    "multiAZ": true,
    "encrypted": true
  }
}
```

**Development PostgreSQL**
```json
{
  "type": "RDS",
  "name": "dev-postgres",
  "region": "us-west-2",
  "properties": {
    "instanceClass": "db.t3.micro",
    "engine": "postgres",
    "storageGB": 20,
    "multiAZ": false
  }
}
```

#### Common Instance Classes
- **Burstable**: db.t3.micro, db.t3.small, db.t3.medium, db.t3.large
- **General Purpose**: db.m5.large, db.m5.xlarge, db.m5.2xlarge
- **Memory Optimized**: db.r5.large, db.r5.xlarge, db.r5.2xlarge
- **Compute Optimized**: db.c5.large, db.c5.xlarge, db.c5.2xlarge

### Lambda - Serverless Functions

Event-driven, serverless compute service.

#### Required Properties
- `memoryMB`: Memory allocation in MB (128-10240)

#### Optional Properties
- `requestsPerMonth`: Number of requests per month (default: 1,000,000)
- `averageDurationMs`: Average execution duration in milliseconds (default: 100)
- `architecture`: Processor architecture (default: "x86_64")
  - Options: "x86_64", "arm64"
- `storageGB`: Additional storage in GB (default: 0)

#### Example Configurations

**API Backend Function**
```json
{
  "type": "Lambda",
  "name": "api-handler",
  "region": "us-east-1",
  "properties": {
    "memoryMB": 512,
    "requestsPerMonth": 5000000,
    "averageDurationMs": 200,
    "architecture": "arm64"
  }
}
```

**Data Processing Function**
```json
{
  "type": "Lambda",
  "name": "data-processor",
  "region": "us-east-1",
  "properties": {
    "memoryMB": 3008,
    "requestsPerMonth": 100000,
    "averageDurationMs": 30000,
    "storageGB": 10
  }
}
```

#### Memory and Performance
- Higher memory allocation provides more CPU power
- ARM64 (Graviton2) offers up to 34% better price performance
- Memory range: 128 MB to 10,240 MB (10 GB)

### S3 - Simple Storage Service

Object storage service with multiple storage classes.

#### Required Properties
- `storageClass`: S3 storage class

#### Supported Storage Classes
- `STANDARD`: Frequently accessed data
- `STANDARD_IA`: Infrequently accessed data
- `ONEZONE_IA`: Infrequently accessed data in single AZ
- `GLACIER`: Long-term archive (minutes to hours retrieval)
- `DEEP_ARCHIVE`: Lowest cost archive (12+ hours retrieval)
- `INTELLIGENT_TIERING`: Automatic cost optimization
- `REDUCED_REDUNDANCY`: Legacy storage class

#### Optional Properties
- `sizeGB`: Storage size in GB (default: 1)
- `requestsPerMonth`: Number of requests per month (default: 0)

#### Example Configurations

**Website Assets**
```json
{
  "type": "S3",
  "name": "website-assets",
  "region": "us-east-1",
  "properties": {
    "storageClass": "STANDARD",
    "sizeGB": 100,
    "requestsPerMonth": 1000000
  }
}
```

**Backup Storage**
```json
{
  "type": "S3",
  "name": "backup-archive",
  "region": "us-west-2",
  "properties": {
    "storageClass": "GLACIER",
    "sizeGB": 10000,
    "requestsPerMonth": 100
  }
}
```

## Advanced Usage

### Multi-Service Architectures

Combine multiple services for complete architecture estimation:

```json
{
  "version": "1.0",
  "resources": [
    {
      "type": "ALB",
      "name": "load-balancer",
      "region": "us-east-1",
      "properties": {
        "type": "application",
        "dataProcessingGB": 200,
        "newConnectionsPerSecond": 100
      }
    },
    {
      "type": "EC2",
      "name": "web-servers",
      "region": "us-east-1",
      "properties": {
        "instanceType": "t3.medium",
        "count": 3
      }
    },
    {
      "type": "RDS",
      "name": "database",
      "region": "us-east-1",
      "properties": {
        "instanceClass": "db.r5.large",
        "engine": "postgres",
        "storageGB": 500,
        "multiAZ": true
      }
    },
    {
      "type": "Lambda",
      "name": "background-jobs",
      "region": "us-east-1",
      "properties": {
        "memoryMB": 1024,
        "requestsPerMonth": 2000000,
        "averageDurationMs": 5000
      }
    },
    {
      "type": "S3",
      "name": "data-storage",
      "region": "us-east-1",
      "properties": {
        "storageClass": "STANDARD",
        "sizeGB": 2000,
        "requestsPerMonth": 500000
      }
    }
  ],
  "options": {
    "currency": "USD",
    "timeFrame": "monthly"
  }
}
```

### Output Formats for Automation

#### JSON for CI/CD Pipelines
```bash
./shylock estimate config.json --output json | jq '.totalMonthlyCost'
```

#### CSV for Spreadsheet Analysis
```bash
./shylock estimate config.json --output csv > monthly-costs.csv
```

#### Verbose Mode for Detailed Analysis
```bash
./shylock estimate config.json --verbose
```

### Region and Currency Overrides

```bash
# Override region for all resources
./shylock estimate config.json --region eu-west-1

# Change currency display
./shylock estimate config.json --currency EUR

# Combine options
./shylock estimate config.json --region ap-southeast-1 --currency JPY --verbose
```

## Best Practices

### Configuration Management

1. **Use Descriptive Names**
   ```json
   {
     "name": "prod-web-server-cluster",  // Good
     "name": "server1"                   // Avoid
   }
   ```

2. **Organize by Environment**
   ```
   configs/
   ├── production.json
   ├── staging.json
   └── development.json
   ```

3. **Version Control Configurations**
   - Store configurations in Git
   - Use meaningful commit messages
   - Tag releases for cost tracking

### Cost Optimization

1. **Right-Size Resources**
   - Start with smaller instances
   - Monitor actual usage
   - Scale up as needed

2. **Use Appropriate Storage Classes**
   - Standard for frequently accessed data
   - IA for infrequent access
   - Glacier for archival

3. **Consider Reserved Instances**
   - Shylock shows on-demand pricing
   - Factor in Reserved Instance discounts manually
   - Use AWS Cost Explorer for RI recommendations

### Validation Workflow

1. **Always Validate First**
   ```bash
   ./shylock validate config.json
   ```

2. **Test with Small Configurations**
   - Start with single resources
   - Gradually add complexity

3. **Use Version Control**
   - Track configuration changes
   - Compare cost estimates over time

## Troubleshooting

### Common Errors and Solutions

#### "No AWS credentials found"
**Problem**: AWS credentials not configured
**Solution**: 
```bash
# Configure AWS CLI
aws configure

# Or set environment variables
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
```

#### "Invalid instance type format"
**Problem**: Incorrect instance type specification
**Solution**: Use proper format like "t3.micro", "m5.large"

#### "Unsupported region"
**Problem**: Invalid or unsupported AWS region
**Solution**: Use standard AWS region codes (us-east-1, eu-west-1, etc.)

#### "Invalid JSON format"
**Problem**: Malformed JSON configuration
**Solution**: 
- Use a JSON validator
- Check for missing commas, brackets, quotes
- Validate with `./shylock validate config.json`

#### "Resource validation failed"
**Problem**: Invalid resource properties
**Solution**: 
- Check required properties for each service type
- Refer to service-specific guides above
- Use `./shylock list` to see supported options

### Performance Issues

#### Slow Estimation
**Causes**: 
- Large number of resources
- Network latency to AWS APIs
- No caching

**Solutions**:
- Use concurrent processing (automatic for 2+ resources)
- Enable caching (enabled by default)
- Process in smaller batches

#### Memory Usage
**Causes**: 
- Very large configurations
- Cache size too large

**Solutions**:
- Use batch processing
- Reduce cache size if needed
- Process configurations in chunks

### Getting Help

1. **Built-in Help**
   ```bash
   ./shylock --help
   ./shylock estimate --help
   ./shylock list
   ```

2. **Validation Tools**
   ```bash
   ./shylock validate config.json
   ```

3. **Example Configurations**
   - Check the `examples/` directory
   - Start with simple examples
   - Build complexity gradually

4. **Verbose Output**
   ```bash
   ./shylock estimate config.json --verbose
   ```

---

This guide covers the essential aspects of using Shylock effectively. For more advanced scenarios or specific use cases, refer to the examples in the repository or the detailed API documentation.