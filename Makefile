# Variables
BIN_NAME = watchman
MODULE_NAME = github.com/adrianpk/watchman

# Default target
all: test

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Building:"
	@echo "  build                 - Build the binary"
	@echo "  install               - Install the binary"
	@echo ""
	@echo "Testing:"
	@echo "  test                  - Run all tests"
	@echo "  test-v                - Run tests with verbose output"
	@echo "  test-short            - Run tests in short mode"
	@echo "  test-coverage         - Run tests with coverage report"
	@echo "  test-coverage-profile - Generate coverage profile"
	@echo "  test-coverage-html    - Generate HTML coverage report"
	@echo "  test-coverage-func    - Show function-level coverage"
	@echo "  test-coverage-check   - Check coverage meets 50% threshold"
	@echo "  test-coverage-summary - Display coverage table by package"
	@echo ""
	@echo "Quality Checks:"
	@echo "  lint                  - Run golangci-lint"
	@echo "  format                - Format code"
	@echo "  vet                   - Run go vet"
	@echo "  check                 - Run all quality checks (fmt, vet, test, test-coverage-check)"
	@echo ""
	@echo "Utilities:"
	@echo "  clean                 - Clean coverage files and test cache"
	@echo "  tidy                  - Run go mod tidy"
	@echo "  download              - Download dependencies"

# Build binary
build:
	@echo "Building $(BIN_NAME)..."
	@go build -o $(BIN_NAME) ./cmd/watchman

# Install binary
install:
	@echo "Installing $(BIN_NAME)..."
	@go install ./cmd/watchman

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run --fix

# Format code
format:
	@echo "Formatting code..."
	@gofmt -w .

# Run tests
test:
	@go test ./...

# Run tests with verbose output
test-v:
	@go test -v ./...

# Run tests in short mode
test-short:
	@go test -short ./...

# Run tests with coverage
test-coverage:
	@go test -cover ./...

# Generate coverage profile and show percentage
test-coverage-profile:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report
test-coverage-html: test-coverage-profile
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Show function-level coverage
test-coverage-func: test-coverage-profile
	@go tool cover -func=coverage.out

# Check coverage percentage and fail if below threshold (50%)
test-coverage-check: test-coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $$COVERAGE%"; \
	if [ $$(awk -v cov="$$COVERAGE" 'BEGIN {print (cov < 50)}') -eq 1 ]; then \
		echo "Coverage $$COVERAGE% is below 50% threshold"; \
		exit 1; \
	else \
		echo "Coverage $$COVERAGE% meets the 50% threshold"; \
	fi

# Display coverage summary table by package
test-coverage-summary:
	@echo "Running coverage tests by package..."
	@echo ""
	@echo "Coverage by package:"
	@echo "-----------------------------------------------------------"
	@echo "| Package                                        | Coverage |"
	@echo "-----------------------------------------------------------"
	@for pkg in $$(go list ./... | grep -v -e "/tmp/" -e "/build/"); do \
		pkgname=$$(echo $$pkg | sed 's|$(MODULE_NAME)||' | sed 's|^/||'); \
		if [ -z "$$pkgname" ]; then pkgname="."; fi; \
		result=$$(go test -cover $$pkg 2>&1); \
		cov=$$(echo "$$result" | grep -oE '[0-9]+\.[0-9]+% of statements' | grep -v '^0\.0%' | tail -1 | grep -oE '[0-9]+\.[0-9]+%'); \
		if [ -z "$$cov" ]; then \
			if echo "$$result" | grep -qE '\[no test files\]|no test files'; then \
				cov="no tests"; \
			elif echo "$$result" | grep -q "FAIL"; then \
				cov="FAIL"; \
			else \
				cov="0.0%"; \
			fi; \
		fi; \
		printf "| %-46s | %8s |\n" "$$pkgname" "$$cov"; \
	done
	@echo "-----------------------------------------------------------"

# Run go vet
vet:
	@go vet ./...

# Run all quality checks
check: format vet test test-coverage-check
	@echo "All quality checks passed"

# Clean coverage files and test cache
clean:
	@echo "Cleaning up..."
	@go clean -testcache
	@rm -f coverage.out coverage.html $(BIN_NAME)
	@echo "Clean complete."

# Run go mod tidy
tidy:
	@echo "Running go mod tidy..."
	@go mod tidy

# Download dependencies
download:
	@echo "Downloading dependencies..."
	@go mod download

# Phony targets
.PHONY: all build install test test-v test-short test-coverage test-coverage-profile test-coverage-html test-coverage-func test-coverage-check test-coverage-summary vet check lint format help clean tidy download
