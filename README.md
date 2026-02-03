# Simple E-Commerce Application

A Go-based e-commerce application demonstrating payment integration with Adyen.

## Features

- Product display and checkout
- Payment session creation
- Payment verification and order confirmation

## Prerequisites

### Option 1: Using Dev Container (Recommended)

The easiest way to get started is using the provided dev container, which includes all dependencies pre-installed:

- [Visual Studio Code](https://code.visualstudio.com/) with the [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)
- OR [Dev Container CLI](https://github.com/devcontainers/cli)
- Docker Desktop or compatible container runtime

### Option 2: Local Setup

If you prefer to run locally without containers:

- Go 1.21 or higher
- Node.js and npm (for frontend dependencies)
- PostgreSQL (for integration tests)
- Adyen API credentials

## Getting Started

### Using Dev Container (Recommended)

#### With VS Code:

1. Open the project in VS Code
2. When prompted, click "Reopen in Container" (or run `Dev Containers: Reopen in Container` from the Command Palette)
3. Wait for the container to build and start
4. All dependencies are pre-installed and ready to use!

#### With Dev Container CLI:

```bash
# Install the CLI if you haven't already
npm install -g @devcontainers/cli

# Open the project in a dev container
devcontainer up --workspace-folder .

# Execute commands in the container
devcontainer exec --workspace-folder . make test
```

The dev container includes:
- Go 1.21+
- Node.js and npm
- Git
- All required Go tools (gopls, golangci-lint, etc.)
- Pre-configured environment

### Environment Configuration

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
