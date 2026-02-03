# Simple E-Commerce Application

A Go-based e-commerce application demonstrating payment integration with Adyen.

## Features

- Product display and checkout
- Payment session creation
- Payment verification and order confirmation

## Prerequisites

- Go 1.21 or higher
- PostgreSQL (for integration tests)
- Adyen API credentials

## Getting Started

### Environment Setup

Copy the example environment file and configure your settings:

```bash
cp .env.example .env
```

Edit `.env` and add your Adyen credentials and database connection details.

### Available Make Commands

The project uses a Makefile for common tasks. Run `make help` to see all available commands.

#### Building and Running

```bash
make build    # Build the application
make run      # Run the application
make clean    # Clean build artifacts
```

#### Testing

```bash
make test                    # Run unit tests
make test-unit               # Run unit tests only (no integration/e2e)
make test-integration        # Run integration tests (requires PostgreSQL)
make test-e2e                # Run end-to-end tests
make test-all                # Run all tests (unit + integration + e2e)
make test-race               # Run tests with race detection
make test-integration-race   # Run integration tests with race detection
```

#### Code Coverage

```bash
make coverage                # Generate coverage report for unit tests
make coverage-integration    # Generate coverage report for integration tests
make coverage-all            # Generate combined coverage report (unit + integration)
make coverage-serve          # Serve coverage report in browser (port 8888)
make clean-coverage          # Clean coverage reports
```

To view the coverage report:

1. Generate coverage: `make coverage-all`
2. Start the server: `make coverage-serve`
3. Open in VS Code Simple Browser: `http://localhost:8888/coverage.html`
4. Press `Ctrl+C` to stop the server

#### Code Quality

```bash
make fmt     # Format code
make lint    # Run linter (requires golangci-lint)
make deps    # Install/update dependencies
```

## Project Structure

```
.
├── cmd/
│   └── simplecom/        # Application entry point
├── internal/
│   ├── cli/              # CLI commands
│   ├── config/           # Configuration management
│   ├── database/         # Database utilities
│   ├── handlers/         # HTTP handlers (100% test coverage)
│   ├── models/           # Domain models
│   ├── repository/       # Data access layer
│   └── services/         # Business logic
├── e2e/                  # End-to-end tests
├── static/               # Static assets (CSS, JS, images)
├── templates/            # HTML templates
└── Makefile             # Build and test automation
```
