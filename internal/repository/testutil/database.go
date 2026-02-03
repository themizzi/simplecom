package testutil

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/adyen/ecommerce/internal/config"
	_ "github.com/lib/pq"
)

// TestDatabase represents an isolated test database
type TestDatabase struct {
	DB         *sql.DB
	SchemaName string
	connConfig *config.PostgresConfig
	masterDB   *sql.DB
}

// SetupTestDatabase creates an isolated schema for testing
func SetupTestDatabase(t *testing.T) *TestDatabase {
	t.Helper()

	// Load Postgres configuration from environment
	connConfig, err := config.LoadPostgresConfig(func(key string) string {
		switch key {
		case "POSTGRES_USER":
			return getEnvOrDefault("POSTGRES_USER", "postgres")
		case "POSTGRES_PASSWORD":
			return getEnvOrDefault("POSTGRES_PASSWORD", "postgres")
		case "POSTGRES_DB":
			return getEnvOrDefault("POSTGRES_DB", "postgres")
		case "POSTGRES_HOSTNAME":
			return getEnvOrDefault("POSTGRES_HOSTNAME", "localhost")
		default:
			return ""
		}
	})
	if err != nil {
		t.Fatalf("Failed to load postgres config: %v", err)
	}

	// Connect to the master database
	masterConnStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		connConfig.Host, connConfig.User, connConfig.Password, connConfig.Database)

	masterDB, err := sql.Open("postgres", masterConnStr)
	if err != nil {
		t.Fatalf("Failed to connect to master database: %v", err)
	}

	if err := masterDB.Ping(); err != nil {
		masterDB.Close()
		t.Fatalf("Failed to ping master database: %v", err)
	}

	// Generate unique schema name for this test
	schemaName := fmt.Sprintf("test_schema_%d_%d", time.Now().UnixNano(), rand.Intn(10000))

	// Create schema
	_, err = masterDB.Exec(fmt.Sprintf("CREATE SCHEMA %s", schemaName))
	if err != nil {
		masterDB.Close()
		t.Fatalf("Failed to create test schema: %v", err)
	}

	// Connect to the same database but set search_path to the test schema
	testConnStr := fmt.Sprintf("%s search_path=%s", masterConnStr, schemaName)
	testDB, err := sql.Open("postgres", testConnStr)
	if err != nil {
		masterDB.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		masterDB.Close()
		t.Fatalf("Failed to connect to test schema: %v", err)
	}

	if err := testDB.Ping(); err != nil {
		testDB.Close()
		masterDB.Exec(fmt.Sprintf("DROP SCHEMA %s CASCADE", schemaName))
		masterDB.Close()
		t.Fatalf("Failed to ping test database: %v", err)
	}

	// Configure connection pool for test database
	testDB.SetMaxOpenConns(5)
	testDB.SetMaxIdleConns(2)
	testDB.SetConnMaxLifetime(5 * time.Minute)

	testDatabase := &TestDatabase{
		DB:         testDB,
		SchemaName: schemaName,
		connConfig: connConfig,
		masterDB:   masterDB,
	}

	// Run migrations in the test schema
	if err := testDatabase.RunMigrations(); err != nil {
		testDatabase.Teardown(t)
		t.Fatalf("Failed to run migrations: %v", err)
	}

	return testDatabase
}

// RunMigrations creates the necessary database tables in the test schema
func (td *TestDatabase) RunMigrations() error {
	// Create orders table
	createOrdersTable := `
	CREATE TABLE IF NOT EXISTS orders (
		id UUID PRIMARY KEY,
		reference VARCHAR(255) UNIQUE NOT NULL,
		amount INTEGER NOT NULL,
		currency VARCHAR(3) NOT NULL,
		status VARCHAR(50) NOT NULL,
		product_name VARCHAR(255) NOT NULL,
		psp_reference VARCHAR(255),
		created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_orders_reference ON orders(reference);
	CREATE INDEX IF NOT EXISTS idx_orders_status ON orders(status);
	`

	_, err := td.DB.Exec(createOrdersTable)
	if err != nil {
		return fmt.Errorf("failed to create orders table: %w", err)
	}

	return nil
}

// Teardown cleans up the test database schema
func (td *TestDatabase) Teardown(t *testing.T) {
	t.Helper()

	if td.DB != nil {
		td.DB.Close()
	}

	if td.masterDB != nil {
		// Drop the test schema
		_, err := td.masterDB.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", td.SchemaName))
		if err != nil {
			t.Logf("Warning: Failed to drop test schema %s: %v", td.SchemaName, err)
		}
		td.masterDB.Close()
	}
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
