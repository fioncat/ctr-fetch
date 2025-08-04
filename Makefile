# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Binary info
BINARY_NAME=ctr-fetch
BINARY_DIR=./bin
BINARY_PATH=$(BINARY_DIR)/$(BINARY_NAME)

# Build flags
BUILD_FLAGS=-tags "containers_image_ostree_stub exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp"
LDFLAGS=-ldflags "-s -w"

# Default target
.PHONY: all
all: build

# Build the binary
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_PATH) .
	@echo "Build complete: $(BINARY_PATH)"

# Build for multiple platforms
.PHONY: build-linux
build-linux:
	@echo "Building $(BINARY_NAME) for Linux..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 .
	@echo "Linux build complete: $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64"

.PHONY: build-darwin
build-darwin:
	@echo "Building $(BINARY_NAME) for macOS..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 .
	@echo "macOS builds complete"

.PHONY: build-windows
build-windows:
	@echo "Building $(BINARY_NAME) for Windows..."
	@mkdir -p $(BINARY_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe .
	@echo "Windows build complete: $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe"

.PHONY: build-all
build-all: build-linux build-darwin build-windows
	@echo "All platform builds complete"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BINARY_DIR)
	@echo "Clean complete"

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies updated"

# Install the binary to $GOPATH/bin
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BINARY_PATH) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installation complete"

# Run the application
.PHONY: run
run: build
	$(BINARY_PATH)

# Display help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build       - Build the binary for current platform"
	@echo "  build-linux - Build for Linux (amd64)"
	@echo "  build-darwin- Build for macOS (amd64 and arm64)"
	@echo "  build-windows- Build for Windows (amd64)"
	@echo "  build-all   - Build for all platforms"
	@echo "  clean       - Remove build artifacts"
	@echo "  test        - Run tests"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  install     - Install binary to GOPATH/bin"
	@echo "  run         - Build and run the application"
	@echo "  help        - Show this help message" 