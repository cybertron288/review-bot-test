.PHONY: all build install clean test test-unit test-integration test-e2e test-race lint fmt vet

# Build variables
BINARY_NAME=agentflow
BUILD_DIR=./build
CMD_DIR=./cmd/agentflow

# Go variables
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=gofmt

# Build flags
LDFLAGS=-ldflags "-s -w"

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

install:
	@echo "Installing $(BINARY_NAME)..."
	$(GOCMD) install $(CMD_DIR)

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	$(GOCLEAN)

# Testing
test:
	@echo "Running all tests..."
	$(GOTEST) -v ./...

test-unit:
	@echo "Running unit tests..."
	$(GOTEST) -v -short ./...

test-integration:
	@echo "Running integration tests..."
	$(GOTEST) -v -run Integration ./...

test-e2e:
	@echo "Running E2E tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

test-race:
	@echo "Running tests with race detector..."
	$(GOTEST) -v -race ./...

# Code quality
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run

fmt:
	@echo "Formatting code..."
	$(GOFMT) -s -w .

vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Development
dev: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# Help
help:
	@echo "Available targets:"
	@echo "  all           - Build the binary (default)"
	@echo "  build         - Build the binary"
	@echo "  install       - Install the binary"
	@echo "  clean         - Clean build artifacts"
	@echo "  test          - Run all tests"
	@echo "  test-unit     - Run unit tests only"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e      - Run E2E tests with coverage"
	@echo "  test-race     - Run tests with race detector"
	@echo "  lint          - Run linter"
	@echo "  fmt           - Format code"
	@echo "  vet           - Run go vet"
	@echo "  deps          - Download dependencies"
	@echo "  tidy          - Tidy dependencies"
	@echo "  dev           - Build and run"
	@echo "  help          - Show this help"

