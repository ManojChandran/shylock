# Shylock - AWS Cost Estimation Tool

[![Go Version](https://img.shields.io/badge/Go-1.24+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-Passing-brightgreen.svg)](#testing)

Shylock is a powerful CLI tool for estimating AWS costs based on resource configurations defined in JSON files. It supports multiple AWS services and provides detailed cost breakdowns with assumptions and recommendations.

## 🚀 Features

- **5 AWS Services Supported**: EC2, ALB, RDS, Lambda, S3
- **Multiple Output Formats**: Table, JSON, CSV, YAML
- **Performance Optimized**: Concurrent processing and intelligent caching
- **Comprehensive Validation**: Detailed error messages with suggestions
- **Rich CLI Interface**: Intuitive commands with extensive help
- **Production Ready**: 95%+ test coverage and robust error handling

## 📦 Installation

### Prerequisites

- AWS credentials configured (AWS CLI, environment variables, or IAM role)

### Quick Install

#### Linux/macOS (Homebrew)
```bash
# Coming soon - Homebrew tap
brew tap owner/tap
brew install shylock
```

#### Windows (Scoop)
```bash
# Coming soon - Scoop bucket
scoop bucket add owner https://github.com/owner/scoop-bucket
scoop install shylock
```

#### Docker
```bash
# Pull and run
docker pull ghcr.io/owner/shylock:latest
docker run --rm -v $(pwd)/config.json:/config.json ghcr.io/owner/shylock:latest estimate /config.json
```

### Manual Installation

#### Download Pre-built Binaries

1. **Download** the latest release for your platform from [GitHub Releases](https://github.com/owner/shylock/releases)

2. **Extract** the archive:
   ```bash
   # Linux/macOS
   tar -xzf shylock_*_linux_amd64.tar.gz
   
   # Windows (PowerShell)
   Expand-Archive shylock_*_windows_amd64.zip
   ```

3. **Move to PATH** (optional):
   ```bash
   # Linux/macOS
   sudo mv shylock /usr/local/bin/
   
   # Windows
   move shylock.exe C:\Windows\System32\
   ```

#### Build from Source

```bash
git clone <repository-url>
cd shylock
go build -o shylock
```

### Quick Test

```bash
./shylock --help
./shylock list
```

## 🎯 Quick Start

### 1. Create a Configuration File

Create a JSON file describing your AWS resources:

```json
{
  "version": "1.0",
  "resources": [
    {
      "type": "EC2",
      "name": "web-server",
      "region": "us-east-1",
      "properties": {
        "instanceType": "t3.medium",
        "count": 2,
        "operatingSystem": "Linux"
      }
    },
    {
      "type": "RDS",
      "name": "database",
      "region": "us-east-1",
      "properties": {
        "instanceClass": "db.t3.micro",
        "engine": "postgres",
        "storageGB": 100,
        "multiAZ": true
      }
    }
  ],
  "options": {
    "currency": "USD",
    "timeFrame": "monthly"
  }
}
```

### 2. Estimate Costs

```bash
# Basic estimation
./shylock estimate config.json

# With verbose output
./shylock estimate config.json --verbose

# JSON output for automation
./shylock estimate config.json --output json

# Override region for all resources
./shylock estimate config.json --region us-west-2
```

### 3. Validate Configuration

```bash
# Validate without estimating
./shylock validate config.json

# List supported services
./shylock list
```

## 📋 Supported AWS Services

### EC2 - Elastic Compute Cloud
- **Instance Types**: 50+ types across all families (t3, m5, c5, r5, etc.)
- **Operating Systems**: Linux, Windows, RHEL, SUSE
- **Features**: Multiple instances, tenancy options
- **Pricing**: On-demand hourly rates

```json
{
  "type": "EC2",
  "name": "web-server",
  "region": "us-east-1",
  "properties": {
    "instanceType": "t3.medium",
    "count": 3,
    "operatingSystem": "Linux",
    "tenancy": "Shared"
  }
}
```

### ALB - Application Load Balancer
- **Types**: Application (Layer 7), Network (Layer 4)
- **Pricing Model**: Load Balancer Capacity Units (LCU)
- **Features**: Data processing, connection handling, rule evaluation

```json
{
  "type": "ALB",
  "name": "app-load-balancer",
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

### RDS - Relational Database Service
- **Engines**: MySQL, PostgreSQL, MariaDB, Oracle, SQL Server, Aurora
- **Instance Classes**: 25+ classes from burstable to memory-optimized
- **Features**: Multi-AZ, encryption, custom storage

```json
{
  "type": "RDS",
  "name": "production-db",
  "region": "us-east-1",
  "properties": {
    "instanceClass": "db.r5.large",
    "engine": "postgres",
    "storageGB": 500,
    "multiAZ": true,
    "encrypted": true,
    "storageType": "gp2"
  }
}
```

### Lambda - Serverless Functions
- **Architectures**: x86_64, ARM64 (Graviton2)
- **Memory**: 128 MB to 10,240 MB
- **Pricing**: Pay-per-use (requests, duration, memory)

```json
{
  "type": "Lambda",
  "name": "api-function",
  "region": "us-east-1",
  "properties": {
    "memoryMB": 1024,
    "requestsPerMonth": 5000000,
    "averageDurationMs": 200,
    "architecture": "arm64"
  }
}
```

### S3 - Simple Storage Service
- **Storage Classes**: Standard, IA, One Zone-IA, Glacier, Deep Archive
- **Features**: Request pricing, data transfer costs

```json
{
  "type": "S3",
  "name": "data-bucket",
  "region": "us-east-1",
  "properties": {
    "storageClass": "STANDARD",
    "sizeGB": 1000,
    "requestsPerMonth": 100000
  }
}
```

## 🛠️ CLI Commands

### estimate
Estimate costs from a configuration file.

```bash
./shylock estimate [config-file] [flags]

Flags:
  -o, --output string     Output format (table, json, csv) (default "table")
  -r, --region string     Override AWS region for all resources
  -v, --verbose           Enable verbose output with detailed information
  -c, --currency string   Currency for cost display (USD, EUR, GBP) (default "USD")
```

### validate
Validate configuration file without estimating costs.

```bash
./shylock validate [config-file]
```

### list
List supported AWS services and resource types.

```bash
./shylock list
```

### version
Show version information.

```bash
./shylock version
```

## 📊 Output Formats

### Table (Default)
Human-readable table with cost summary and resource breakdown.

```
AWS Cost Estimation Results
===========================
Generated: 2024-01-15 10:30:00 UTC
Currency: USD

💰 Cost Summary
---------------
Hourly Cost:  $1.2500
Daily Cost:   $30.0000
Monthly Cost: $900.0000

📊 Resource Breakdown
--------------------
Resource Name    Type    Region      Hourly      Daily      Monthly
web-server       EC2     us-east-1   $0.5000    $12.0000   $360.0000
database         RDS     us-east-1   $0.7500    $18.0000   $540.0000
```

### JSON
Structured data for programmatic use.

```bash
./shylock estimate config.json --output json
```

### CSV
Spreadsheet-compatible format.

```bash
./shylock estimate config.json --output csv > costs.csv
```

## ⚡ Performance Features

### Concurrent Processing
Automatically processes multiple resources in parallel for faster estimation.

### Intelligent Caching
- **15-minute TTL**: Reduces AWS API calls by up to 90%
- **LRU Eviction**: Memory-efficient cache management
- **Thread-Safe**: Concurrent access support

### Batch Processing
Handles large configurations efficiently with configurable batch sizes.

## 🔧 Configuration Options

### Global Options
```json
{
  "options": {
    "defaultRegion": "us-east-1",
    "currency": "USD",
    "timeFrame": "monthly"
  }
}
```

### Supported Currencies
- USD (default)
- EUR
- GBP
- JPY

### Supported Regions
All standard AWS regions are supported. Use standard AWS region codes (e.g., `us-east-1`, `eu-west-1`).

## 🧪 Examples

See the `examples/` directory for comprehensive configuration examples:

**Simple Examples** (Single Service):
- `examples/simple-ec2.json` - Basic EC2 instance
- `examples/simple-alb.json` - Application Load Balancer
- `examples/simple-rds.json` - PostgreSQL database
- `examples/simple-lambda.json` - Serverless function
- `examples/s3-storage.json` - S3 storage configurations

**Complex Examples** (Multi-Service):
- `examples/web-application.json` - Web application stack
- `examples/serverless-api.json` - Serverless API architecture
- `examples/data-analytics.json` - Data processing pipeline
- `examples/development-environment.json` - Development setup
- `examples/production-setup.json` - Production-ready architecture
- `examples/comprehensive-example.json` - All services combined

## 🔐 AWS Credentials

Shylock uses the AWS SDK for Go and supports all standard credential methods:

1. **Environment Variables**
   ```bash
   export AWS_ACCESS_KEY_ID=your-access-key
   export AWS_SECRET_ACCESS_KEY=your-secret-key
   export AWS_REGION=us-east-1
   ```

2. **AWS CLI Configuration**
   ```bash
   aws configure
   ```

3. **IAM Roles** (for EC2 instances)

4. **AWS Profiles**
   ```bash
   export AWS_PROFILE=your-profile
   ```

### Required Permissions
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "pricing:GetProducts",
        "pricing:DescribeServices"
      ],
      "Resource": "*"
    }
  ]
}
```

## 🚨 Error Handling

Shylock provides detailed error messages with suggestions:

```
Error: invalid instance type format: t3micro
Details:
  resourceName: web-server
  instanceType: t3micro

Suggestions:
  1. Use valid EC2 instance type format (e.g., 't3.micro', 'm5.large')
  2. Check AWS documentation for available instance types
```

## 🧪 Testing

Run the test suite:

```bash
# All tests
go test ./...

# With coverage
go test ./... -cover

# Specific package
go test ./internal/estimators/...

# Verbose output
go test ./... -v
```

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Add tests for new functionality
4. Ensure all tests pass
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

### Common Issues

**"No AWS credentials found"**
- Configure AWS credentials using one of the methods above
- Verify credentials with `aws sts get-caller-identity`

**"Unsupported region"**
- Use standard AWS region codes
- Check region availability for the specific service

**"Invalid JSON format"**
- Validate JSON syntax using a JSON validator
- Check for missing commas, brackets, or quotes

## 📚 Documentation

### User Documentation
- **[Installation Guide](docs/INSTALLATION.md)** - Detailed setup instructions for all platforms
- **[User Guide](docs/USER_GUIDE.md)** - Comprehensive usage guide with examples
- **[Troubleshooting](docs/TROUBLESHOOTING.md)** - Common issues and solutions

### Developer Documentation
- **[API Documentation](docs/API.md)** - Developer reference and extension guide
- **[Release Guide](docs/RELEASE.md)** - How to create and manage releases
- **[Distribution Guide](docs/DISTRIBUTION.md)** - Package management and distribution

### Getting Help

1. **Quick Start**: Follow the examples in the `examples/` directory
2. **Command Help**: Use `./shylock --help` for command help
3. **Service Info**: Use `./shylock list` to see supported services
4. **Validation**: Use `./shylock validate config.json` to check syntax
5. **Detailed Guide**: Read the [User Guide](docs/USER_GUIDE.md)
6. **Installation Issues**: Check the [Installation Guide](docs/INSTALLATION.md)
7. **Troubleshooting**: Review [Troubleshooting Guide](docs/TROUBLESHOOTING.md)

---

**Built with ❤️ for AWS cost optimization**