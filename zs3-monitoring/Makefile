# Makefile for ZS3 Monitoring Go Binary

# Build configuration
BINARY_NAME=zs3_monitoring
BUILD_DIR=./build
GO_VERSION=1.21
ZS3_MONITORING_APP=./cmd/

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) $(ZS3_MONITORING_APP)
	@echo "Binary built successfully: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for Linux (production)
.PHONY: build-linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)_linux .
	@echo "Linux binary built: $(BUILD_DIR)/$(BINARY_NAME)_linux"

# Install the binary to system
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "Installation complete"

# Run the monitoring script
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

# Test the binary (dry run with mock data)
.PHONY: test
test: build
	@echo "Testing $(BINARY_NAME) with mock data..."
	@mkdir -p /tmp/zs3_test_metadata
	@echo 'BLOBBER_ID_LIST="blobber1,blobber2"' > /tmp/zs3_test_metadata/metadata.env
	@echo 'CLUSTER_ID="test-cluster"' >> /tmp/zs3_test_metadata/metadata.env
	@echo 'USER_ID="test-user"' >> /tmp/zs3_test_metadata/metadata.env
	@echo 'CLIENT_ID="test-client-id"' >> /tmp/zs3_test_metadata/metadata.env
	@METADATA_PATH="/tmp/zs3_test_metadata/metadata.env" LOG_FILE="/tmp/zs3_test.log" $(BUILD_DIR)/$(BINARY_NAME)
	@rm -rf /tmp/zs3_test_metadata

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Format Go code
.PHONY: fmt
fmt:
	@echo "Formatting Go code..."
	@go fmt ./...

# Run Go linter
.PHONY: lint
lint:
	@echo "Running Go linter..."
	@go vet ./...

# Run tests
.PHONY: test-go
test-go:
	@echo "Running Go tests..."
	@go test -v ./...

# Create systemd service file
.PHONY: install-service
install-service: build install
	@echo "Creating systemd service..."
	@echo "[Unit]" > /tmp/zs3-monitoring.service
	@echo "Description=ZS3 Monitoring Service" >> /tmp/zs3-monitoring.service
	@echo "After=network.target" >> /tmp/zs3-monitoring.service
	@echo "" >> /tmp/zs3-monitoring.service
	@echo "[Service]" >> /tmp/zs3-monitoring.service
	@echo "Type=oneshot" >> /tmp/zs3-monitoring.service
	@echo "ExecStart=/usr/local/bin/zs3_monitoring" >> /tmp/zs3-monitoring.service
	@echo "User=ubuntu" >> /tmp/zs3-monitoring.service
	@echo "Group=ubuntu" >> /tmp/zs3-monitoring.service
	@echo "WorkingDirectory=/home/ubuntu" >> /tmp/zs3-monitoring.service
	@echo "StandardOutput=journal" >> /tmp/zs3-monitoring.service
	@echo "StandardError=journal" >> /tmp/zs3-monitoring.service
	@echo "" >> /tmp/zs3-monitoring.service
	@echo "[Install]" >> /tmp/zs3-monitoring.service
	@echo "WantedBy=multi-user.target" >> /tmp/zs3-monitoring.service
	@sudo mv /tmp/zs3-monitoring.service /etc/systemd/system/zs3-monitoring.service
	@echo "Systemd service created: /etc/systemd/system/zs3-monitoring.service"

# Create cron job for weekly execution
.PHONY: install-cron
install-cron: build install
	@echo "Creating cron job for weekly execution..."
	@(crontab -l 2>/dev/null; echo "0 2 * * 0 /usr/local/bin/zs3_monitoring") | crontab -
	@echo "Cron job installed for weekly execution (Sundays at 2 AM)"

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build          - Build the Go binary"
	@echo "  build-linux    - Build for Linux (production)"
	@echo "  install        - Install binary to /usr/local/bin"
	@echo "  run            - Build and run the monitoring script"
	@echo "  test           - Test with mock data"
	@echo "  clean          - Clean build artifacts"
	@echo "  fmt            - Format Go code"
	@echo "  lint           - Run Go linter"
	@echo "  test-go        - Run Go tests"
	@echo "  install-service - Create systemd service"
	@echo "  install-cron   - Create cron job for weekly execution"
	@echo "  help           - Show this help"
