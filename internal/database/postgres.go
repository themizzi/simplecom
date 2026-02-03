package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/adyen/ecommerce/internal/config"
	_ "github.com/lib/pq"
)

var DB *sql.DB

// Connect establishes a connection to the PostgreSQL database
func Connect() error {
	pgConfig, err := config.LoadPostgresConfig(os.Getenv)
	if err != nil {
		return fmt.Errorf("failed to load postgres config: %w", err)
	}

	// Connection string
	connStr := pgConfig.ConnectionString()

	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
