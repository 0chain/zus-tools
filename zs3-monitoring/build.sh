#!/bin/bash
# Build script for ZS3 Monitoring Go Binary
# This script sets up the Go environment and builds the binary

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Setup Go environment
setup_go_env() {
    print_info "Setting up Go environment..."
    
    # Add Go to PATH if not already there
    if ! command -v go >/dev/null 2>&1; then
        if [ -d "/usr/local/go/bin" ]; then
            export PATH=$PATH:/usr/local/go/bin
            print_success "Added /usr/local/go/bin to PATH"
        else
            print_error "Go not found. Please install Go or set GOPATH"
            exit 1
        fi
    else
        print_success "Go is already available"
    fi
    
    # Check Go version
    GO_VERSION=$(go version | cut -d' ' -f3)
    print_info "Using Go version: $GO_VERSION"
}

# Build the binary
build_binary() {
    local build_type="${1:-local}"
    
    print_info "Building ZS3 Monitoring binary ($build_type)..."
    
    case "$build_type" in
        "local")
            make build ./cmd/
            ;;
        "linux")
            make build-linux ./cmd/
            ;;
        *)
            print_error "Unknown build type: $build_type"
            print_info "Available build types: local, linux"
            exit 1
            ;;
    esac
    
    print_success "Build completed successfully!"
}

# Run tests
run_tests() {
    print_info "Running tests..."
    make test-go
    print_success "Tests completed!"
}

# Run linter
run_linter() {
    print_info "Running linter..."
    make lint
    print_success "Linting completed!"
}

# Format code
format_code() {
    print_info "Formatting code..."
    make fmt
    print_success "Code formatting completed!"
}

# Install binary
install_binary() {
    print_info "Installing binary..."
    make install
    print_success "Binary installed to /usr/local/bin/zs3_monitoring"
}

# Create systemd service
create_service() {
    print_info "Creating systemd service..."
    make install-service
    print_success "Systemd service created!"
}

# Create cron job
create_cron() {
    print_info "Creating cron job..."
    make install-cron
    print_success "Cron job created for weekly execution!"
}

# Test with mock data
test_mock() {
    print_info "Testing with mock data..."
    make test
    print_success "Mock test completed!"
}

# Show help
show_help() {
    echo "ZS3 Monitoring Build Script"
    echo ""
    echo "Usage: $0 [command] [options]"
    echo ""
    echo "Commands:"
    echo "  deploy|start        - Complete deployment (DEFAULT): build, test, install, service, cron, start"
    echo "  build [local|linux] - Build the binary only"
    echo "  test                - Run Go tests only"
    echo "  lint                - Run Go linter only"
    echo "  fmt                 - Format Go code only"
    echo "  install             - Install binary to system only"
    echo "  service             - Create systemd service only"
    echo "  cron                - Create cron job only"
    echo "  mock                - Test with mock data only"
    echo "  all                 - Build, test, lint, and format (no deployment)"
    echo "  help                - Show this help"
    echo ""
    echo "Examples:"
    echo "  $0                  # Complete deployment (default)"
    echo "  $0 deploy           # Complete deployment"
    echo "  $0 start            # Complete deployment"
    echo "  $0 build            # Build for local system only"
    echo "  $0 build linux      # Build for Linux only"
    echo "  $0 test             # Run tests only"
    echo ""
    echo "Complete deployment includes:"
    echo "  1. Build binary"
    echo "  2. Run unit tests"
    echo "  3. Install binary"
    echo "  4. Create systemd service"
    echo "  5. Create cron job"
    echo "  6. Start systemd service"
    echo "  7. Check service status"
    echo ""
}

# Complete deployment process
deploy_all() {
    print_info "Starting complete deployment process..."
    
    # Step 1: Build the binary
    print_info "Step 1: Building binary..."
    make build
    print_success "Binary built successfully!"
    
    # Step 2: Run unit tests
    print_info "Step 2: Running unit tests..."
    make test-go
    print_success "Unit tests passed!"
    
    # Step 3: Install binary
    print_info "Step 3: Installing binary..."
    make install
    print_success "Binary installed!"
    
    # Step 4: Create systemd service
    print_info "Step 4: Creating systemd service..."
    make install-service
    print_success "Systemd service created!"
    
    # Step 5: Create cron job
    print_info "Step 5: Creating cron job..."
    make install-cron
    print_success "Cron job created!"
    
    # Step 6: Start the service
    print_info "Step 6: Starting systemd service..."
    sudo systemctl start zs3-monitoring.service
    print_success "Service started successfully!"
    
    # Step 7: Check service status
    print_info "Step 7: Checking service status..."
    sudo systemctl status zs3-monitoring.service --no-pager -l
    print_success "Deployment completed successfully!"
}

# Main function
main() {
    local command="${1:-deploy}"
    
    case "$command" in
        "deploy"|"start")
            setup_go_env
            deploy_all
            ;;
        "build")
            setup_go_env
            build_binary "${2:-local}"
            ;;
        "test")
            setup_go_env
            run_tests
            ;;
        "lint")
            setup_go_env
            run_linter
            ;;
        "fmt")
            setup_go_env
            format_code
            ;;
        "install")
            setup_go_env
            install_binary
            ;;
        "service")
            setup_go_env
            create_service
            ;;
        "cron")
            setup_go_env
            create_cron
            ;;
        "mock")
            setup_go_env
            test_mock
            ;;
        "all")
            setup_go_env
            format_code
            run_linter
            run_tests
            build_binary "local"
            print_success "All checks completed successfully!"
            ;;
        "help"|*)
            show_help
            ;;
    esac
}

# Run main function
main "$@"
