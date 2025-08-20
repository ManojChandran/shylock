# Release Guide

This guide explains how to create releases for Shylock using GoReleaser.

## Overview

Shylock uses [GoReleaser](https://goreleaser.com/) to automate the release process, creating:

- **Cross-platform binaries** for Linux, macOS, and Windows
- **Docker images** for containerized deployments
- **Package manager integrations** (Homebrew, Scoop)
- **GitHub releases** with changelogs and assets
- **Checksums and signatures** for security

## Supported Platforms

### Binaries
- **Linux**: amd64, arm64
- **macOS**: amd64 (Intel), arm64 (Apple Silicon)
- **Windows**: amd64

### Package Managers
- **Homebrew** (macOS/Linux)
- **Scoop** (Windows)
- **Docker** (Multi-arch images)

## Release Process

### Prerequisites

1. **Repository Setup**
   - GitHub repository with proper permissions
   - GitHub Actions enabled
   - Secrets configured (if using Docker Hub)

2. **Local Setup**
   ```bash
   # Install GoReleaser (optional for manual releases)
   brew install goreleaser
   
   # Or download from https://github.com/goreleaser/goreleaser/releases
   ```

### Automated Releases (Recommended)

Releases are automatically triggered when you push a Git tag:

1. **Create and Push Tag**
   ```bash
   # Create a new tag
   git tag -a v1.0.0 -m "Release v1.0.0"
   
   # Push the tag
   git push origin v1.0.0
   ```

2. **GitHub Actions Workflow**
   - The `.github/workflows/release.yml` workflow triggers automatically
   - Runs tests to ensure quality
   - Builds cross-platform binaries
   - Creates Docker images
   - Publishes GitHub release with assets

3. **Monitor Progress**
   - Check the Actions tab in your GitHub repository
   - View build logs and any errors
   - Verify the release appears in the Releases section

### Manual Releases (Alternative)

For local testing or manual releases:

1. **Test Configuration**
   ```bash
   # Check GoReleaser configuration
   goreleaser check
   
   # Build snapshot (no release)
   goreleaser release --snapshot --clean
   ```

2. **Create Release**
   ```bash
   # Set required environment variables
   export GITHUB_TOKEN=your_github_token
   export GITHUB_OWNER=your_username
   export GITHUB_REPO=shylock
   
   # Create release
   goreleaser release --clean
   ```

## Version Management

### Semantic Versioning

Shylock follows [Semantic Versioning](https://semver.org/):

- **MAJOR** version: Incompatible API changes
- **MINOR** version: New functionality (backward compatible)
- **PATCH** version: Bug fixes (backward compatible)

Examples:
- `v1.0.0` - Initial release
- `v1.1.0` - New features added
- `v1.1.1` - Bug fixes
- `v2.0.0` - Breaking changes

### Pre-releases

For beta or release candidate versions:

```bash
# Beta release
git tag -a v1.1.0-beta.1 -m "Beta release v1.1.0-beta.1"

# Release candidate
git tag -a v1.1.0-rc.1 -m "Release candidate v1.1.0-rc.1"
```

Pre-releases are automatically marked as "pre-release" on GitHub.

## Release Assets

Each release includes:

### Binaries
- `shylock_1.0.0_linux_amd64.tar.gz`
- `shylock_1.0.0_linux_arm64.tar.gz`
- `shylock_1.0.0_darwin_amd64.tar.gz`
- `shylock_1.0.0_darwin_arm64.tar.gz`
- `shylock_1.0.0_windows_amd64.zip`

### Documentation
Each archive includes:
- `README.md` - Main documentation
- `LICENSE` - License file
- `docs/` - Complete documentation
- `examples/` - Configuration examples

### Checksums
- `shylock_1.0.0_checksums.txt` - SHA256 checksums for all assets

### Docker Images
- `ghcr.io/owner/shylock:1.0.0`
- `ghcr.io/owner/shylock:latest`
- Multi-architecture support (amd64, arm64)

## Installation Methods

### Direct Download

Users can download binaries directly from GitHub releases:

```bash
# Linux amd64
curl -L https://github.com/owner/shylock/releases/download/v1.0.0/shylock_1.0.0_linux_amd64.tar.gz | tar xz

# macOS (Intel)
curl -L https://github.com/owner/shylock/releases/download/v1.0.0/shylock_1.0.0_darwin_amd64.tar.gz | tar xz

# macOS (Apple Silicon)
curl -L https://github.com/owner/shylock/releases/download/v1.0.0/shylock_1.0.0_darwin_arm64.tar.gz | tar xz
```

### Package Managers

#### Homebrew (macOS/Linux)
```bash
# Add tap (one-time setup)
brew tap owner/tap

# Install
brew install shylock

# Update
brew upgrade shylock
```

#### Scoop (Windows)
```powershell
# Add bucket (one-time setup)
scoop bucket add owner https://github.com/owner/scoop-bucket

# Install
scoop install shylock

# Update
scoop update shylock
```

#### Docker
```bash
# Pull image
docker pull ghcr.io/owner/shylock:latest

# Run
docker run --rm -v $(pwd)/config.json:/config.json ghcr.io/owner/shylock:latest estimate /config.json
```

## Configuration

### GitHub Secrets

For automated releases, configure these secrets in your GitHub repository:

#### Required
- `GITHUB_TOKEN` - Automatically provided by GitHub Actions

#### Optional (for Docker Hub)
- `DOCKER_USERNAME` - Docker Hub username
- `DOCKER_PASSWORD` - Docker Hub password/token

### Environment Variables

For manual releases or custom configurations:

```bash
# Required
export GITHUB_TOKEN=your_personal_access_token
export GITHUB_OWNER=your_github_username
export GITHUB_REPO=shylock

# Optional
export DOCKER_REGISTRY=ghcr.io  # or docker.io
```

## Troubleshooting

### Common Issues

#### Build Failures

**Problem**: Cross-compilation fails
```
Solution: Ensure CGO_ENABLED=0 for static binaries
Check: .goreleaser.yaml build configuration
```

**Problem**: Missing dependencies
```
Solution: Run `go mod tidy` before release
Check: go.mod and go.sum files are up to date
```

#### Release Failures

**Problem**: GitHub token permissions
```
Solution: Ensure token has 'repo' and 'write:packages' scopes
Check: GitHub repository settings > Actions > General
```

**Problem**: Tag already exists
```
Solution: Delete existing tag or use new version
Commands:
  git tag -d v1.0.0          # Delete local tag
  git push origin :v1.0.0    # Delete remote tag
```

#### Docker Issues

**Problem**: Docker login fails
```
Solution: Check DOCKER_USERNAME and DOCKER_PASSWORD secrets
Alternative: Use GitHub Container Registry (ghcr.io)
```

**Problem**: Multi-arch build fails
```
Solution: Ensure Docker Buildx is properly configured
Check: GitHub Actions workflow setup-buildx step
```

### Debug Commands

```bash
# Test GoReleaser configuration
goreleaser check

# Build without releasing
goreleaser build --snapshot --clean

# Release in dry-run mode
goreleaser release --skip-publish --clean

# Check specific build target
goreleaser build --single-target --snapshot
```

### Logs and Monitoring

1. **GitHub Actions**
   - Repository > Actions tab
   - Click on workflow run for detailed logs
   - Download artifacts for debugging

2. **GoReleaser Output**
   - Check build logs for compilation errors
   - Verify asset generation
   - Review changelog generation

3. **Release Verification**
   - Test download links
   - Verify checksums
   - Test installation on target platforms

## Best Practices

### Before Release

1. **Update Documentation**
   - Update README.md with new features
   - Update CHANGELOG.md (if maintained)
   - Review and update examples

2. **Test Thoroughly**
   - Run full test suite: `go test ./...`
   - Test cross-platform builds locally
   - Validate configuration files

3. **Version Bump**
   - Update version references in documentation
   - Ensure semantic versioning compliance
   - Create meaningful tag messages

### After Release

1. **Verify Release**
   - Test download and installation
   - Verify all platforms work correctly
   - Check Docker images

2. **Update Package Managers**
   - Homebrew formula (if manual)
   - Scoop manifest (if manual)
   - Update installation instructions

3. **Announce Release**
   - Update project documentation
   - Notify users of new features
   - Share on relevant platforms

## Security

### Checksums

All releases include SHA256 checksums:

```bash
# Verify download integrity
sha256sum -c shylock_1.0.0_checksums.txt
```

### Signatures

For enhanced security, consider adding GPG signatures:

```yaml
# Add to .goreleaser.yaml
signs:
  - artifacts: checksum
    args: ["--batch", "-u", "{{ .Env.GPG_FINGERPRINT }}", "--output", "${signature}", "--detach-sign", "${artifact}"]
```

### Supply Chain Security

- All builds run in GitHub Actions (auditable)
- Dependencies are locked in go.sum
- Reproducible builds with fixed Go version
- Container images use minimal base (Alpine)

---

This release process ensures reliable, secure, and user-friendly distribution of Shylock across all supported platforms.