.PHONY: build test clean miner install help

# Default target
help:
	@echo "CryptoChain Blockchain - Build Targets"
	@echo "====================================="
	@echo "build      - Build the main application"
	@echo "miner      - Build the mining CLI"
	@echo "test       - Run all tests"
	@echo "clean      - Clean build artifacts"
	@echo "install    - Install the miner CLI"
	@echo "help       - Show this help message"

# Build main application
build:
	@echo "Building main application..."
	go build -o bin/blockchain ./main.go

# Build mining CLI
miner:
	@echo "Building mining CLI..."
	go build -o bin/miner ./cmd/miner

# Run tests
test:
	@echo "Running all tests..."
	go test -v ./...

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	go clean

# Install miner CLI to system
install: miner
	@echo "Installing miner CLI..."
	sudo cp bin/miner /usr/local/bin/cryptochain-miner
	@echo "✅ Miner CLI installed as 'cryptochain-miner'"

# Development targets
dev-test:
	@echo "Running tests with coverage..."
	go test -v -cover ./...

dev-lint:
	@echo "Running linter..."
	golangci-lint run

dev-fmt:
	@echo "Formatting code..."
	go fmt ./...

# Quick start for development
quick-start: miner
	@echo "Starting miner with default settings..."
	./bin/miner start -miner dev-user -difficulty 1