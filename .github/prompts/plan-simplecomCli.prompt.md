## Plan: Build simplecom CLI with serve command

Create a simple CLI tool called "simplecom" using the **urfave/cli** framework for managing the e-commerce application. The initial implementation includes a `serve` command that runs the web server with configuration driven entirely by .env files.

### Steps

1. **Add urfave/cli dependency** - Run `go get github.com/urfave/cli/v2` and update [go.mod](go.mod)

2. **Create CLI package structure** - Create [cmd/simplecom/main.go](cmd/simplecom/main.go) as new entrypoint and [internal/cli/serve.go](internal/cli/serve.go) for the serve command implementation

3. **Implement serve command** - Move server startup logic from [cmd/server/main.go](cmd/server/main.go) to [internal/cli/serve.go](internal/cli/serve.go), keeping the current .env file configuration approach. Replace cmd/server entirely.

4. **Add graceful shutdown** - Implement signal handling (SIGINT, SIGTERM) with context cancellation for clean shutdowns (default 30s timeout)

### Decisions

1. **No backward compatibility** - Replace [cmd/server/main.go](cmd/server/main.go) entirely with simplecom
2. **Use stdlib** - Standard library logging is sufficient for now
3. **Single command only** - Just `serve` command, no additional commands
4. **No CLI flags** - Configuration via .env only (flags can be added later if needed)

---

### Testing Strategy

All tests use Go's built-in `testing` package. Three test types cover the scenarios:

#### 1. Unit Tests (`internal/cli/serve_test.go`)
Test individual serve command logic without running the full server:
- Configuration loading from .env
- Error handling for missing config
- Server initialization logic

```go
func TestServeAction_LoadsEnvFile(t *testing.T)
func TestServeAction_MissingAdyenConfig(t *testing.T)
func TestServeAction_InvalidDatabaseConfig(t *testing.T)
```

#### 2. CLI End-to-End Tests (`internal/cli/serve_e2e_test.go`)
Test the full serve command with a real server (using test database):
- Server starts and responds to requests
- Graceful shutdown with signals (SIGTERM/SIGINT)
- Server lifecycle management

```go
func TestServeCommand_StartsServer(t *testing.T)
func TestServeCommand_GracefulShutdown(t *testing.T)
func TestServeCommand_HandlesMultipleRequests(t *testing.T)
```

Use `t.TempDir()` for temporary .env files, and `syscall.Kill()` to send signals to test processes.

### Gherkin Scenarios for Serve Command

```gherkin
Feature: Serve Command
  As a developer or operator
  I want to start the e-commerce server using the simplecom CLI
  So that I can serve the application with .env file configuration

  Background:
    Given the database is running and accessible
    And valid Adyen credentials are configured in .env file

  Scenario: Start server with default settings from .env
    Given I have a valid .env file in the current directory
    And the .env file contains "PORT=8080"
    When I run "simplecom serve"
    Then the server should start on port 8080
    And database migrations should run automatically
    And I should see "Server listening on :8080" in the output
    And the server should respond to HTTP requests on "http://localhost:8080"

  Scenario: Start server with custom port from .env
    Given I have a .env file with "PORT=3000"
    When I run "simplecom serve"
    Then the server should start on port 3000
    And I should see "Server listening on :3000" in the output
    And the server should respond to HTTP requests on "http://localhost:3000"

  Scenario: Load database configuration from .env
    Given I have a .env file with database settings:
      | POSTGRES_HOSTNAME | db.example.com |
      | POSTGRES_PORT     | 5432           |
      | POSTGRES_USER     | admin          |
      | POSTGRES_PASSWORD | secret123      |
      | POSTGRES_DB       | shopdb         |
    When I run "simplecom serve"
    Then the server should connect to PostgreSQL at "db.example.com:5432"
    And the connection should use username "admin" and database "shopdb"

  Scenario: Load Adyen configuration from .env
    Given I have a .env file with Adyen settings:
      | ADYEN_API_KEY          | test_api_key    |
      | ADYEN_CLIENT_KEY       | test_client_key |
      | ADYEN_MERCHANT_ACCOUNT | MyMerchant      |
      | ADYEN_ENVIRONMENT      | TEST            |
    When I run "simplecom serve"
    Then the Adyen client should be configured with the provided credentials
    And the Adyen environment should be set to "TEST"

  Scenario: Graceful shutdown on SIGTERM
    Given the server is running via "simplecom serve"
    And there are active HTTP connections
    When I send a SIGTERM signal to the process
    Then the server should stop accepting new connections
    And active connections should complete within 30 seconds
    And the server should exit with status code 0
    And I should see "Shutting down server..." in the output

  Scenario: Graceful shutdown on SIGINT (Ctrl+C)
    Given the server is running via "simplecom serve"
    When I press Ctrl+C (send SIGINT)
    Then the server should stop accepting new connections
    And the server should wait for active connections to complete
    And the server should exit gracefully with status code 0

  Scenario: Display help for serve command
    When I run "simplecom serve --help"
    Then I should see usage information for the serve command
    And I should see "Start the e-commerce web server" in the description

  Scenario: Display general help
    When I run "simplecom --help"
    Then I should see a list of available commands
    And I should see "serve" in the command list
    And I should see "version" in the command list

  Scenario: Fail gracefully when database is unreachable
    Given the .env file contains an invalid database host
    When I run "simplecom serve"
    Then the server should fail to start
    And I should see "Failed to connect to database" in the error output
    And the exit code should be non-zero

  Scenario: Fail gracefully when Adyen credentials are missing
    Given the .env file is missing Adyen credentials
    When I run "simplecom serve"
    Then the server should fail to start
    And I should see "Missing required Adyen configuration" in the error output
    And the exit code should be non-zero

  Scenario: Fail gracefully when .env file is missing
    Given no .env file exists in the current directory
    And no environment variables are set
    When I run "simplecom serve"
    Then I should see a warning ".env file not found" in the output
    And the server should attempt to start with environment variables
    And the server should fail if required variables are missing

  Scenario: Server continues running on repeated requests
    Given the server is running via "simplecom serve"
    When I make 100 HTTP requests to "http://localhost:8080"
    Then all requests should complete successfully
    And the server should remain running
```
