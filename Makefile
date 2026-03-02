BINARY_NAME=txtv
BUILD_DIR=.
CMD_DIR=./cmd/txtv

# Default target
all: build

# Build the binary
build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)

# Run all tests (unit + integration)
test:
	go test -v ./internal/... ./cmd/...

# Run only unit tests
test-unit:
	go test -v ./internal/...

# Clean build artifacts
clean:
	rm -f $(BUILD_DIR)/$(BINARY_NAME)
	go clean

# Format code
fmt:
	go fmt ./...

# Update dependencies
deps:
	go mod tidy

.PHONY: all build test test-unit clean fmt deps
