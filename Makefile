.PHONY: build run clean install test

# Binary name
BINARY_NAME=focussessions
MAIN_PATH=cmd/focussessions/main.go

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete!"

# Run the application
run: build
	@./$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@echo "Clean complete!"

# Install to $GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(MAIN_PATH)
	@echo "Installation complete!"

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...
	@echo "Format complete!"

# Lint code
lint:
	@echo "Linting code..."
	@golangci-lint run
	@echo "Lint complete!"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated!"

# Development mode with live reload (requires entr)
dev:
	@find . -name "*.go" | entr -r go run $(MAIN_PATH)

help:
	@echo "Available targets:"
	@echo "  build    - Build the binary"
	@echo "  run      - Build and run the application"
	@echo "  clean    - Remove build artifacts"
	@echo "  install  - Install to GOPATH/bin"
	@echo "  test     - Run tests"
	@echo "  fmt      - Format code"
	@echo "  lint     - Lint code (requires golangci-lint)"
	@echo "  deps     - Download and tidy dependencies"
	@echo "  dev      - Run in development mode with live reload (requires entr)"
	@echo "  help     - Show this help message"