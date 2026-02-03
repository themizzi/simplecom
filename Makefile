.PHONY: test test-unit test-integration test-e2e test-all coverage coverage-integration coverage-all coverage-serve clean-coverage help

# Default target
help:
	@echo "Available targets:"
	@echo "  make test              - Run all unit tests"
	@echo "  make test-unit         - Run unit tests only (no integration/e2e)"
	@echo "  make test-integration  - Run integration tests only"
	@echo "  make test-e2e          - Run end-to-end tests only"
	@echo "  make test-all          - Run all tests (unit + integration + e2e)"
	@echo "  make coverage          - Generate coverage report for unit tests"
	@echo "  make coverage-integration - Generate coverage report for integration tests"
	@echo "  make coverage-all      - Generate combined coverage report (unit + integration)"
	@echo "  make coverage-serve    - Serve coverage report in browser (port 8888)"
	@echo "  make build             - Build the application"
	@echo "  make run               - Run the application"
	@echo "  make clean             - Clean build artifacts"
	@echo "  make clean-coverage    - Clean coverage reports"

# Run unit tests (excludes integration and e2e tests)
test: test-unit

test-unit:
	@echo "Running unit tests..."
	go test $$(go list ./... | grep -v /e2e) -v

# Run integration tests only
test-integration:
	@echo "Running integration tests..."
	@echo "Note: Requires PostgreSQL to be running with proper env vars set"
	go test -tags=integration ./internal/repository/... -v

# Run e2e tests only
test-e2e:
	@echo "Running end-to-end tests..."
	go test ./e2e/... -v

# Run all tests
test-all: test-unit test-integration test-e2e

# Generate coverage report for unit tests
coverage:
	@echo "Generating coverage report for unit tests..."
	@mkdir -p coverage
	go test $$(go list ./... | grep -v /e2e) -coverprofile=coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@echo "Coverage report generated: coverage/coverage.html"

# Generate coverage report for integration tests
coverage-integration:
	@echo "Generating coverage report for integration tests..."
	@mkdir -p coverage
	go test -tags=integration ./internal/repository/... -coverprofile=coverage/integration.out
	go tool cover -html=coverage/integration.out -o coverage/integration.html
	@echo "Integration coverage report generated: coverage/integration.html"

# Generate combined coverage report for unit and integration tests
coverage-all:
	@echo "Generating combined coverage report..."
	@mkdir -p coverage
	@echo "Running unit tests..."
	go test $$(go list ./... | grep -v /e2e) -coverprofile=coverage/unit.out
	@echo "Running integration tests..."
	go test -tags=integration ./internal/repository/... -coverprofile=coverage/integration.out
	@echo "Merging coverage reports..."
	@echo "mode: set" > coverage/coverage.out
	@grep -h -v "^mode:" coverage/unit.out coverage/integration.out >> coverage/coverage.out
	go tool cover -html=coverage/coverage.out -o coverage/coverage.html
	@rm -f coverage/unit.out coverage/integration.out
	@echo "Combined coverage report generated: coverage/coverage.html"

# Build the application
build:
	@echo "Building application..."
	go build -o bin/simplecom ./cmd/simplecom

# Run the application
run:
	@echo "Running application..."
	go run ./cmd/simplecom serve

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f bin/simplecom
	rm -f coverage.out coverage.html
	rm -f coverage_integration.out coverage_integration.html
	rm -f coverage_*.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run linter (if golangci-lint is installed)
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	goimports -w .

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	go test -race $$(go list ./... | grep -v /e2e) -v

# Run integration tests with race detection
test-integration-race:
	@echo "Running integration tests with race detection..."
	go test -tags=integration -race ./internal/repository/... -v

# Clean coverage reports
clean-coverage:
	@echo "Cleaning coverage reports..."
	rm -rf coverage/
	@echo "Coverage reports cleaned"

# Serve coverage report in browser
coverage-serve:
	@if [ ! -f coverage/coverage.html ]; then \
		echo "Coverage report not found. Run 'make coverage-all' first."; \
		exit 1; \
	fi
	@echo "Starting HTTP server for coverage report..."
	@echo "Open in VS Code Simple Browser: http://localhost:8888/coverage.html"
	@echo "Or run: open http://localhost:8888/coverage.html"
	@echo "Press Ctrl+C to stop the server"
	cd coverage && python3 -m http.server 8888
