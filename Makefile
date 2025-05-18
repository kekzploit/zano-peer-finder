# Variables
BINARY_NAME=peer-finder
BINARY_DIR=bin
CMD_DIR=cmd/peer-finder
MAIN_FILE=$(CMD_DIR)/main.go

# Go related variables
GO=go
GOFMT=gofmt
GOLINT=golangci-lint

# Default target
all: clean build

# Create necessary directories
$(BINARY_DIR):
	mkdir -p $(BINARY_DIR)

# Install dependencies
deps:
	$(GO) mod download
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Build the application
build: $(BINARY_DIR)
	$(GO) build -o $(BINARY_DIR)/$(BINARY_NAME) $(MAIN_FILE)

# Run the application
run: build
	./$(BINARY_DIR)/$(BINARY_NAME)

# Clean build artifacts
clean:
	rm -rf $(BINARY_DIR)
	go clean

# Run tests
test:
	$(GO) test -v ./...

# Run linters
lint:
	$(GOLINT) run

# Format code
fmt:
	$(GOFMT) -w .

# Check for common errors
vet:
	$(GO) vet ./...

# Install development tools
tools:
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Help command
help:
	@echo "Available commands:"
	@echo "  make deps    - Install dependencies"
	@echo "  make build   - Build the application"
	@echo "  make run     - Run the application"
	@echo "  make clean   - Clean build artifacts"
	@echo "  make test    - Run tests"
	@echo "  make lint    - Run linters"
	@echo "  make fmt     - Format code"
	@echo "  make vet     - Check for common errors"
	@echo "  make tools   - Install development tools"

.PHONY: all deps build run clean test lint fmt vet tools help 