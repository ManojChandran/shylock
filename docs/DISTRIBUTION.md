# Distribution Guide

This document explains how Shylock is packaged and distributed across different platforms and package managers.

## Overview

Shylock uses a modern distribution strategy with multiple installation methods:

- **Direct Downloads**: Pre-built binaries for all platforms
- **Package Managers**: Homebrew (macOS/Linux), Scoop (Windows)
- **Container Images**: Docker images for containerized environments
- **Source Builds**: Traditional Go build process

## Distribution Channels

### 1. GitHub Releases

**Primary distribution method** with automated releases via GitHub Actions.

#### Assets per Release
- Cross-platform binaries (Linux, macOS, Windows)
- Complete documentation and examples
- SHA256 checksums for verification
- Automated changelog generation

#### Download URLs
```
https://github.com/owner/shylock/releases/download/v1.0.0/shylock_1.0.0_linux_amd64.tar.gz
https://github.com/owner/shylock/releases/download/v1.0.0/shylock_1.0.0_darwin_amd64.tar.gz
https://github.com/owner/shylock/releases/download/v1.0.0/shylock_1.0.0_windows_amd64.zip
```

### 2. Package Managers

#### Homebrew (macOS/Linux)

**Repository**: `owner/homebrew-tap`

```bash
# Installation
brew tap owner/tap
brew install shylock

# Updates
brew upgrade shylock
```

**Formula Features**:
- Automatic dependency management
- Version pinning and rollback
- Integration with macOS/Linux package ecosystem
- Includes documentation and examples

#### Scoop (Windows)

**Repository**: `owner/scoop-bucket`

```powershell
# Installation
scoop bucket add owner https://github.com/owner/scoop-bucket
scoop install shylock

# Updates
scoop update shylock
```

**Manifest Features**:
- Windows-native installation
- PATH management
- Automatic updates
- Uninstall support

### 3. Container Registry

#### GitHub Container Registry

**Images**:
- `ghcr.io/owner/shylock:latest`
- `ghcr.io/owner/shylock:v1.0.0`
- `ghcr.io/owner/shylock:v1.0.0-amd64`
- `ghcr.io/owner/shylock:v1.0.0-arm64`

**Multi-architecture Support**:
- Linux AMD64 (x86_64)
- Linux ARM64 (AArch64)

```bash
# Pull and run
docker pull ghcr.io/owner/shylock:latest
docker run --rm -v $(pwd)/config.json:/config.json ghcr.io/owner/shylock:latest estimate /config.json
```

#### Docker Hub (Optional)

If configured with Docker Hub credentials:
- `docker.io/owner/shylock:latest`
- `docker.io/owner/shylock:v1.0.0`

### 4. Source Distribution

#### Go Modules

```bash
# Install latest
go install github.com/owner/shylock@latest

# Install specific version
go install github.com/owner/shylock@v1.0.0
```

#### Git Repository

```bash
# Clone and build
git clone https://github.com/owner/shylock.git
cd shylock
go build -o shylock
```

## Platform Support

### Operating Systems

| OS | Architecture | Binary | Package Manager | Container |
|----|--------------|--------|-----------------|-----------|
| Linux | AMD64 | ✅ | Homebrew | ✅ |
| Linux | ARM64 | ✅ | Homebrew | ✅ |
| macOS | AMD64 (Intel) | ✅ | Homebrew | ❌ |
| macOS | ARM64 (Apple Silicon) | ✅ | Homebrew | ❌ |
| Windows | AMD64 | ✅ | Scoop | ❌ |

### Minimum Requirements

- **Memory**: 256 MB RAM
- **Disk**: 50 MB free space
- **Network**: Internet access for AWS API calls
- **Dependencies**: None (static binaries)

## Installation Methods Comparison

| Method | Pros | Cons | Best For |
|--------|------|------|----------|
| **Package Manager** | Auto-updates, dependency management | Platform-specific setup | Regular users |
| **Direct Download** | Simple, no dependencies | Manual updates | CI/CD, servers |
| **Docker** | Isolated, reproducible | Requires Docker | Containerized environments |
| **Go Install** | Latest version, simple | Requires Go toolchain | Go developers |
| **Source Build** | Customizable, latest code | Requires build tools | Contributors, customization |

## Security and Verification

### Checksums

All releases include SHA256 checksums:

```bash
# Download checksum file
curl -L https://github.com/owner/shylock/releases/download/v1.0.0/shylock_1.0.0_checksums.txt

# Verify binary
sha256sum -c shylock_1.0.0_checksums.txt
```

### Supply Chain Security

- **Reproducible Builds**: Fixed Go version, locked dependencies
- **Automated Releases**: No manual intervention in build process
- **Minimal Dependencies**: Static binaries with no runtime dependencies
- **Container Security**: Minimal Alpine base, non-root user

### Verification Commands

```bash
# Verify installation
shylock version
shylock list

# Test functionality
shylock validate examples/simple-ec2.json
```

## Update Mechanisms

### Automatic Updates

- **Homebrew**: `brew upgrade shylock`
- **Scoop**: `scoop update shylock`
- **Docker**: `docker pull ghcr.io/owner/shylock:latest`

### Manual Updates

- **Direct Download**: Download new version, replace binary
- **Go Install**: `go install github.com/owner/shylock@latest`

### Version Checking

```bash
# Check current version
shylock version

# Check for updates (manual process)
# Visit: https://github.com/owner/shylock/releases
```

## Distribution Metrics

### Download Statistics

GitHub provides download statistics for releases:
- Total downloads per release
- Downloads by platform/architecture
- Geographic distribution

### Package Manager Analytics

- **Homebrew**: Formula analytics via Homebrew API
- **Scoop**: Download statistics via Scoop telemetry
- **Docker**: Pull statistics via registry APIs

## Troubleshooting Distribution Issues

### Common Installation Problems

#### Binary Not Found
```bash
# Check PATH
echo $PATH

# Add to PATH (Linux/macOS)
export PATH=$PATH:/usr/local/bin

# Add to PATH (Windows)
set PATH=%PATH%;C:\Program Files\shylock
```

#### Permission Denied
```bash
# Make executable (Linux/macOS)
chmod +x shylock

# Run as administrator (Windows)
# Right-click > Run as administrator
```

#### Architecture Mismatch
```bash
# Check system architecture
uname -m          # Linux/macOS
echo $PROCESSOR_ARCHITECTURE  # Windows

# Download correct binary for your architecture
```

### Package Manager Issues

#### Homebrew
```bash
# Update Homebrew
brew update

# Reinstall formula
brew uninstall shylock
brew install shylock

# Check formula
brew info shylock
```

#### Scoop
```powershell
# Update Scoop
scoop update

# Reinstall app
scoop uninstall shylock
scoop install shylock

# Check app info
scoop info shylock
```

#### Docker
```bash
# Clear Docker cache
docker system prune

# Pull specific tag
docker pull ghcr.io/owner/shylock:v1.0.0

# Check image
docker images | grep shylock
```

## Future Distribution Plans

### Planned Additions

1. **Linux Package Managers**
   - APT repository (Ubuntu/Debian)
   - YUM repository (CentOS/RHEL)
   - Snap packages (Universal Linux)

2. **Windows Package Managers**
   - Chocolatey packages
   - Windows Package Manager (winget)

3. **Cloud Marketplaces**
   - AWS Marketplace
   - Azure Marketplace
   - Google Cloud Marketplace

4. **Additional Registries**
   - Quay.io container registry
   - Amazon ECR Public Gallery

### Integration Opportunities

- **CI/CD Platforms**: Pre-built actions/plugins
- **Infrastructure Tools**: Terraform providers, Ansible modules
- **Cloud Shells**: Pre-installed in cloud environments
- **IDE Extensions**: VS Code, IntelliJ plugins

## Contributing to Distribution

### Adding New Package Managers

1. **Create Package Repository**
   - Fork appropriate template repository
   - Configure package metadata
   - Set up automated updates

2. **Update GoReleaser Configuration**
   - Add new package manager section
   - Configure repository settings
   - Test package generation

3. **Update Documentation**
   - Add installation instructions
   - Update platform support matrix
   - Include troubleshooting guides

### Testing Distribution

```bash
# Test all distribution methods
scripts/release.sh all

# Test specific package manager
# (Manual testing required for each platform)
```

---

This distribution strategy ensures Shylock is easily accessible across all major platforms while maintaining security and reliability standards.