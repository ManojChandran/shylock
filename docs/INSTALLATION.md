# Installation Guide

This guide provides detailed instructions for installing and setting up Shylock on different platforms.

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Installation Methods](#installation-methods)
3. [AWS Configuration](#aws-configuration)
4. [Verification](#verification)
5. [Troubleshooting](#troubleshooting)

## System Requirements

### Minimum Requirements
- **Go**: Version 1.24 or later
- **Operating System**: Linux, macOS, or Windows
- **Memory**: 256 MB RAM minimum, 512 MB recommended
- **Disk Space**: 50 MB for binary and dependencies
- **Network**: Internet connection for AWS API access

### Supported Platforms
- Linux (x86_64, ARM64)
- macOS (Intel, Apple Silicon)
- Windows (x86_64)

## Installation Methods

### Method 1: Build from Source (Recommended)

1. **Install Go**
   
   **Linux (Ubuntu/Debian)**:
   ```bash
   sudo apt update
   sudo apt install golang-go
   ```
   
   **macOS (Homebrew)**:
   ```bash
   brew install go
   ```
   
   **Windows**: Download from [golang.org](https://golang.org/dl/)

2. **Verify Go Installation**
   ```bash
   go version
   # Should show Go 1.24 or later
   ```

3. **Clone Repository**
   ```bash
   git clone <repository-url>
   cd shylock
   ```

4. **Build Binary**
   ```bash
   go build -o shylock
   ```

5. **Make Executable (Linux/macOS)**
   ```bash
   chmod +x shylock
   ```

6. **Optional: Install Globally**
   ```bash
   # Linux/macOS
   sudo mv shylock /usr/local/bin/
   
   # Windows (add to PATH)
   move shylock.exe C:\Windows\System32\
   ```

### Method 2: Cross-Platform Builds

Build for different platforms:

```bash
# Linux x86_64
GOOS=linux GOARCH=amd64 go build -o shylock-linux-amd64

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -o shylock-linux-arm64

# macOS Intel
GOOS=darwin GOARCH=amd64 go build -o shylock-darwin-amd64

# macOS Apple Silicon
GOOS=darwin GOARCH=arm64 go build -o shylock-darwin-arm64

# Windows
GOOS=windows GOARCH=amd64 go build -o shylock-windows-amd64.exe
```

### Method 3: Docker (Alternative)

Create a Dockerfile:

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY . .
RUN go build -o shylock

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/shylock .
ENTRYPOINT ["./shylock"]
```

Build and run:
```bash
docker build -t shylock .
docker run -v $(pwd)/config.json:/config.json shylock estimate /config.json
```

## AWS Configuration

### Prerequisites

You need AWS credentials with the following permissions:

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

### Configuration Methods

#### Method 1: AWS CLI (Recommended)

1. **Install AWS CLI**
   ```bash
   # Linux/macOS
   curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
   unzip awscliv2.zip
   sudo ./aws/install
   
   # macOS (Homebrew)
   brew install awscli
   
   # Windows: Download from AWS website
   ```

2. **Configure Credentials**
   ```bash
   aws configure
   ```
   
   Enter:
   - AWS Access Key ID
   - AWS Secret Access Key
   - Default region (e.g., us-east-1)
   - Output format (json)

3. **Verify Configuration**
   ```bash
   aws sts get-caller-identity
   ```

#### Method 2: Environment Variables

```bash
export AWS_ACCESS_KEY_ID=your-access-key
export AWS_SECRET_ACCESS_KEY=your-secret-key
export AWS_DEFAULT_REGION=us-east-1
```

**Windows (PowerShell)**:
```powershell
$env:AWS_ACCESS_KEY_ID="your-access-key"
$env:AWS_SECRET_ACCESS_KEY="your-secret-key"
$env:AWS_DEFAULT_REGION="us-east-1"
```

#### Method 3: AWS Profiles

Create multiple profiles in `~/.aws/credentials`:

```ini
[default]
aws_access_key_id = your-default-key
aws_secret_access_key = your-default-secret

[production]
aws_access_key_id = your-prod-key
aws_secret_access_key = your-prod-secret

[development]
aws_access_key_id = your-dev-key
aws_secret_access_key = your-dev-secret
```

Use specific profile:
```bash
export AWS_PROFILE=production
./shylock estimate config.json
```

#### Method 4: IAM Roles (EC2/ECS/Lambda)

For applications running on AWS infrastructure, use IAM roles:

1. **Create IAM Role** with pricing permissions
2. **Attach Role** to EC2 instance, ECS task, or Lambda function
3. **No additional configuration** needed in Shylock

## Verification

### Basic Verification

1. **Check Binary**
   ```bash
   ./shylock --version
   ```

2. **Test Help**
   ```bash
   ./shylock --help
   ```

3. **List Services**
   ```bash
   ./shylock list
   ```

### AWS Connectivity Test

1. **Create Test Configuration**
   ```bash
   cat > test-config.json << EOF
   {
     "version": "1.0",
     "resources": [
       {
         "type": "EC2",
         "name": "test-instance",
         "region": "us-east-1",
         "properties": {
           "instanceType": "t3.micro"
         }
       }
     ]
   }
   EOF
   ```

2. **Validate Configuration**
   ```bash
   ./shylock validate test-config.json
   ```

3. **Test Estimation**
   ```bash
   ./shylock estimate test-config.json
   ```

### Expected Output

Successful installation should show:

```
AWS Cost Estimation Results
===========================
Generated: 2024-01-15 10:30:00 UTC
Currency: USD

💰 Cost Summary
---------------
Hourly Cost:  $0.0104
Daily Cost:   $0.2496
Monthly Cost: $7.4880

📊 Resource Breakdown
--------------------
Resource Name    Type    Region      Hourly      Daily      Monthly
test-instance    EC2     us-east-1   $0.0104    $0.2496    $7.4880
```

## Troubleshooting

### Common Installation Issues

#### Go Version Too Old

**Error**: `go: module requires Go 1.24 or later`

**Solution**:
1. Update Go to latest version
2. Verify with `go version`
3. Rebuild: `go build -o shylock`

#### Build Failures

**Error**: `package not found` or `module not found`

**Solution**:
```bash
go mod tidy
go mod download
go build -o shylock
```

#### Permission Denied

**Error**: `permission denied: ./shylock`

**Solution**:
```bash
chmod +x shylock
```

### AWS Configuration Issues

#### No Credentials Found

**Error**: `NoCredentialProviders: no valid providers in chain`

**Solutions**:
1. Run `aws configure`
2. Set environment variables
3. Check IAM role attachment (for EC2)
4. Verify profile configuration

#### Invalid Credentials

**Error**: `InvalidUserID.NotFound` or `SignatureDoesNotMatch`

**Solutions**:
1. Verify access key and secret key
2. Check for typos in credentials
3. Ensure credentials haven't expired
4. Test with `aws sts get-caller-identity`

#### Insufficient Permissions

**Error**: `AccessDenied` or `UnauthorizedOperation`

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

### Network Issues

#### Connection Timeout

**Error**: `context deadline exceeded`

**Solutions**:
1. Check internet connection
2. Verify firewall settings
3. Try different AWS region
4. Check proxy configuration

#### DNS Resolution

**Error**: `no such host`

**Solutions**:
1. Check DNS settings
2. Try different DNS servers (8.8.8.8, 1.1.1.1)
3. Verify network connectivity

### Platform-Specific Issues

#### Windows Antivirus

Some antivirus software may flag the binary:
1. Add exception for shylock.exe
2. Temporarily disable real-time protection during build
3. Use Windows Defender exclusions

#### macOS Gatekeeper

**Error**: `"shylock" cannot be opened because it is from an unidentified developer`

**Solution**:
```bash
xattr -d com.apple.quarantine shylock
```

Or go to System Preferences > Security & Privacy > Allow anyway

#### Linux Dependencies

Missing libraries:
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install build-essential

# CentOS/RHEL
sudo yum groupinstall "Development Tools"
```

## Performance Optimization

### Build Optimizations

**Smaller Binary**:
```bash
go build -ldflags="-s -w" -o shylock
```

**Static Binary** (Linux):
```bash
CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-extldflags "-static"' -o shylock
```

### Runtime Optimizations

**Increase Concurrency**:
```bash
export GOMAXPROCS=8  # Set to number of CPU cores
```

**Memory Optimization**:
```bash
export GOGC=100  # Garbage collection frequency
```

## Next Steps

After successful installation:

1. **Read the User Guide**: `docs/USER_GUIDE.md`
2. **Try Examples**: Files in `examples/` directory
3. **Configure for Your Environment**: Create your own JSON configurations
4. **Integrate into CI/CD**: Use JSON output for automation

## Getting Help

If you encounter issues not covered here:

1. **Check Examples**: Working configurations in `examples/`
2. **Use Validation**: `./shylock validate config.json`
3. **Enable Verbose Mode**: `./shylock estimate config.json --verbose`
4. **Review Troubleshooting Guide**: `docs/TROUBLESHOOTING.md`

---

**Installation complete!** You're ready to start estimating AWS costs with Shylock.