# Troubleshooting Guide

This guide helps you resolve common issues when using Shylock.

## Table of Contents

1. [Installation Issues](#installation-issues)
2. [AWS Credential Problems](#aws-credential-problems)
3. [Configuration Errors](#configuration-errors)
4. [Validation Failures](#validation-failures)
5. [Performance Issues](#performance-issues)
6. [Output Problems](#output-problems)
7. [Network and API Issues](#network-and-api-issues)

## Installation Issues

### Go Version Compatibility

**Problem**: Build fails with Go version errors
```
go: module requires Go 1.24 or later
```

**Solution**: 
1. Check your Go version: `go version`
2. Update Go to 1.24 or later
3. Rebuild: `go build -o shylock`

### Missing Dependencies

**Problem**: Build fails with missing dependencies
```
go: module not found
```

**Solution**:
```bash
go mod tidy
go build -o shylock
```

### Permission Issues

**Problem**: Cannot execute the binary
```
permission denied: ./shylock
```

**Solution**:
```bash
chmod +x shylock
./shylock --help
```

## AWS Credential Problems

### No Credentials Found

**Problem**: 
```
Error: failed to create AWS client
Details:
  error: NoCredentialProviders: no valid providers in chain
```

**Solutions**:

1. **Configure AWS CLI**:
   ```bash
   aws configure
   ```

2. **Set Environment Variables**:
   ```bash
   export AWS_ACCESS_KEY_ID=your-access-key
   export AWS_SECRET_ACCESS_KEY=your-secret-key
   export AWS_REGION=us-east-1
   ```

3. **Use AWS Profile**:
   ```bash
   export AWS_PROFILE=your-profile
   ./shylock estimate config.json
   ```

4. **Verify Credentials**:
   ```bash
   aws sts get-caller-identity
   ```

### Invalid Credentials

**Problem**:
```
Error: failed to connect to AWS Pricing API
Details:
  error: InvalidUserID.NotFound
```

**Solutions**:
1. Verify credentials are correct
2. Check if credentials have expired
3. Ensure credentials have required permissions

### Insufficient Permissions

**Problem**:
```
Error: failed to retrieve pricing data
Details:
  error: AccessDenied
```

**Solution**: Add required IAM permissions:
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

## Configuration Errors

### Invalid JSON Format

**Problem**:
```
Error: failed to parse configuration file
Details:
  error: invalid character '}' looking for beginning of object key string
```

**Solutions**:
1. **Validate JSON syntax**:
   ```bash
   # Use online JSON validator or
   python -m json.tool config.json
   ```

2. **Common JSON errors**:
   - Missing commas between objects
   - Trailing commas (not allowed in JSON)
   - Unmatched brackets or braces
   - Missing quotes around strings

3. **Use validation**:
   ```bash
   ./shylock validate config.json
   ```

### Missing Required Fields

**Problem**:
```
Error: configuration validation failed
Details:
  resourceName: web-server
  error: missing required property 'instanceType'
```

**Solution**: Add the missing required property:
```json
{
  "type": "EC2",
  "name": "web-server",
  "region": "us-east-1",
  "properties": {
    "instanceType": "t3.micro"  // Add this
  }
}
```

### Invalid Resource Types

**Problem**:
```
Error: unsupported resource type 'ECS'
Details:
  resourceType: ECS
  supportedTypes: [ALB, EC2, Lambda, RDS, S3]
```

**Solution**: Use supported resource types:
- ALB (Application Load Balancer)
- EC2 (Elastic Compute Cloud)
- Lambda (Serverless Functions)
- RDS (Relational Database Service)
- S3 (Simple Storage Service)

## Validation Failures

### Invalid Instance Types

**Problem**:
```
Error: invalid instance type format: t3micro
Details:
  resourceName: web-server
  instanceType: t3micro
```

**Solution**: Use correct format with dot separator:
```json
{
  "properties": {
    "instanceType": "t3.micro"  // Correct
  }
}
```

### Invalid Regions

**Problem**:
```
Error: unsupported region: us-east
Details:
  region: us-east
```

**Solution**: Use complete AWS region codes:
- `us-east-1` (not `us-east`)
- `us-west-2` (not `us-west`)
- `eu-west-1` (not `europe`)

### Invalid Property Values

**Problem**:
```
Error: count must be greater than 0, got -1
Details:
  resourceName: web-server
  count: -1
```

**Solution**: Use valid positive values:
```json
{
  "properties": {
    "count": 2  // Must be positive
  }
}
```

### Invalid Memory Sizes (Lambda)

**Problem**:
```
Error: invalid Lambda memory size
Details:
  memoryMB: 100
```

**Solution**: Use valid memory range (128-10240 MB):
```json
{
  "properties": {
    "memoryMB": 512  // Valid range: 128-10240
  }
}
```

## Performance Issues

### Slow Estimation

**Problem**: Cost estimation takes a long time

**Causes and Solutions**:

1. **Large number of resources**:
   - Break into smaller configurations
   - Use batch processing (automatic)

2. **Network latency**:
   - Check internet connection
   - Try different AWS region

3. **API rate limiting**:
   - Wait and retry
   - Reduce concurrent requests

### High Memory Usage

**Problem**: Shylock uses too much memory

**Solutions**:
1. **Process smaller batches**:
   - Split large configurations
   - Process incrementally

2. **Reduce cache size** (if using performance optimizations):
   - Lower cache limits
   - Clear cache periodically

### Timeout Errors

**Problem**:
```
Error: context deadline exceeded
```

**Solutions**:
1. **Check network connectivity**
2. **Retry the operation**
3. **Use smaller configurations**
4. **Check AWS service status**

## Output Problems

### Garbled Table Output

**Problem**: Table formatting looks wrong

**Solutions**:
1. **Use wider terminal**:
   - Resize terminal window
   - Use horizontal scrolling

2. **Use different output format**:
   ```bash
   ./shylock estimate config.json --output json
   ```

3. **Reduce verbose output**:
   ```bash
   ./shylock estimate config.json  # Remove --verbose
   ```

### Empty Results

**Problem**: No cost estimates returned

**Causes and Solutions**:

1. **All resources failed validation**:
   ```bash
   ./shylock validate config.json  # Check for errors
   ```

2. **No matching pricing data**:
   - Check if instance types are available in region
   - Verify service availability in region

3. **Configuration errors**:
   - Review configuration format
   - Check required properties

### Incorrect Cost Calculations

**Problem**: Costs seem wrong or unexpected

**Verification Steps**:

1. **Check assumptions** (use `--verbose`):
   ```bash
   ./shylock estimate config.json --verbose
   ```

2. **Verify configuration**:
   - Instance types and sizes
   - Operating system
   - Storage amounts

3. **Compare with AWS Calculator**:
   - Use AWS Simple Monthly Calculator
   - Verify pricing assumptions

4. **Check currency and region**:
   ```bash
   ./shylock estimate config.json --currency USD --region us-east-1
   ```

## Network and API Issues

### Connection Refused

**Problem**:
```
Error: connection refused
```

**Solutions**:
1. **Check internet connection**
2. **Verify firewall settings**
3. **Check proxy configuration**
4. **Try different network**

### DNS Resolution Issues

**Problem**:
```
Error: no such host
```

**Solutions**:
1. **Check DNS settings**
2. **Try different DNS servers** (8.8.8.8, 1.1.1.1)
3. **Verify network connectivity**

### SSL/TLS Errors

**Problem**:
```
Error: certificate verify failed
```

**Solutions**:
1. **Update system certificates**
2. **Check system time/date**
3. **Verify network security settings**

### AWS Service Outages

**Problem**: Intermittent API failures

**Solutions**:
1. **Check AWS Service Health Dashboard**
2. **Try different AWS region**
3. **Wait and retry later**
4. **Use cached results if available**

## Getting Additional Help

### Enable Verbose Output

Always use verbose mode for troubleshooting:
```bash
./shylock estimate config.json --verbose
```

### Validate Configuration

Always validate before estimating:
```bash
./shylock validate config.json
```

### Check Service Support

List supported services and options:
```bash
./shylock list
```

### Test with Simple Configuration

Start with minimal configuration:
```json
{
  "version": "1.0",
  "resources": [
    {
      "type": "EC2",
      "name": "test",
      "region": "us-east-1",
      "properties": {
        "instanceType": "t3.micro"
      }
    }
  ]
}
```

### Check Examples

Review working examples in the `examples/` directory:
- `examples/simple-ec2.json`
- `examples/development-environment.json`

### Common Command Patterns

```bash
# Full troubleshooting workflow
./shylock validate config.json
./shylock estimate config.json --verbose
./shylock estimate config.json --output json

# Test connectivity
./shylock list

# Test with minimal config
./shylock estimate examples/simple-ec2.json
```

---

If you continue to experience issues after following this guide, please check the examples directory for working configurations or review the user guide for detailed service-specific information.