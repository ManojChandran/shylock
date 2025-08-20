#!/bin/bash

# Shylock Release Helper Script
# This script helps with local testing and release preparation

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if GoReleaser is installed
check_goreleaser() {
    if ! command -v goreleaser &> /dev/null; then
        log_error "GoReleaser is not installed"
        log_info "Install with: brew install goreleaser"
        log_info "Or download from: https://github.com/goreleaser/goreleaser/releases"
        exit 1
    fi
    log_success "GoReleaser is installed: $(goreleaser --version)"
}

# Check GoReleaser configuration
check_config() {
    log_info "Checking GoReleaser configuration..."
    if goreleaser check; then
        log_success "GoReleaser configuration is valid"
    else
        log_error "GoReleaser configuration has errors"
        exit 1
    fi
}

# Run tests
run_tests() {
    log_info "Running tests..."
    if go test ./...; then
        log_success "All tests passed"
    else
        log_error "Tests failed"
        exit 1
    fi
}

# Build snapshot
build_snapshot() {
    log_info "Building snapshot release..."
    if goreleaser release --snapshot --clean; then
        log_success "Snapshot build completed"
        log_info "Binaries available in dist/ directory"
    else
        log_error "Snapshot build failed"
        exit 1
    fi
}

# Test binaries
test_binaries() {
    log_info "Testing built binaries..."
    
    # Test Linux binary (if available)
    if [ -f "dist/shylock_linux_amd64_v1/shylock" ]; then
        log_info "Testing Linux binary..."
        ./dist/shylock_linux_amd64_v1/shylock version
    fi
    
    # Test macOS binary (if available)
    if [ -f "dist/shylock_darwin_amd64_v1/shylock" ]; then
        log_info "Testing macOS binary..."
        ./dist/shylock_darwin_amd64_v1/shylock version
    fi
    
    # Test current platform binary
    if [ -f "./shylock" ]; then
        log_info "Testing current platform binary..."
        ./shylock version
        ./shylock list
        ./shylock validate examples/simple-ec2.json
    fi
    
    log_success "Binary tests completed"
}

# Create release tag
create_tag() {
    local version=$1
    if [ -z "$version" ]; then
        log_error "Version not specified"
        log_info "Usage: $0 tag v1.0.0"
        exit 1
    fi
    
    # Validate version format
    if [[ ! $version =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
        log_error "Invalid version format: $version"
        log_info "Expected format: v1.0.0 or v1.0.0-beta.1"
        exit 1
    fi
    
    # Check if tag already exists
    if git tag -l | grep -q "^$version$"; then
        log_error "Tag $version already exists"
        exit 1
    fi
    
    # Create and push tag
    log_info "Creating tag $version..."
    git tag -a "$version" -m "Release $version"
    
    log_info "Pushing tag to origin..."
    git push origin "$version"
    
    log_success "Tag $version created and pushed"
    log_info "GitHub Actions will automatically create the release"
}

# Clean up
cleanup() {
    log_info "Cleaning up..."
    rm -rf dist/
    rm -f shylock
    log_success "Cleanup completed"
}

# Show help
show_help() {
    echo "Shylock Release Helper"
    echo ""
    echo "Usage: $0 <command> [options]"
    echo ""
    echo "Commands:"
    echo "  check       Check GoReleaser configuration and dependencies"
    echo "  test        Run tests"
    echo "  build       Build snapshot release (no publishing)"
    echo "  test-bins   Test built binaries"
    echo "  tag <ver>   Create and push release tag (triggers GitHub Actions)"
    echo "  clean       Clean up build artifacts"
    echo "  all         Run check, test, and build"
    echo "  help        Show this help message"
    echo ""
    echo "Examples:"
    echo "  $0 check                 # Check configuration"
    echo "  $0 build                 # Build snapshot"
    echo "  $0 tag v1.0.0           # Create release tag"
    echo "  $0 all                   # Full test and build"
    echo ""
}

# Main script logic
case "${1:-help}" in
    "check")
        check_goreleaser
        check_config
        ;;
    "test")
        run_tests
        ;;
    "build")
        check_goreleaser
        check_config
        run_tests
        build_snapshot
        ;;
    "test-bins")
        test_binaries
        ;;
    "tag")
        create_tag "$2"
        ;;
    "clean")
        cleanup
        ;;
    "all")
        check_goreleaser
        check_config
        run_tests
        build_snapshot
        test_binaries
        ;;
    "help"|*)
        show_help
        ;;
esac