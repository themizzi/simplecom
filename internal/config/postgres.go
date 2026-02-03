package config

import (
	"fmt"
)

// PostgresConfig holds configuration for PostgreSQL database connection
type PostgresConfig struct {
	User     string
	Password string
	Database string
	Host     string
}

// LoadPostgresConfig loads PostgreSQL configuration from environment variables
func LoadPostgresConfig(getenv func(string) string) (*PostgresConfig, error) {
	config := &PostgresConfig{
		User:     getenv("POSTGRES_USER"),
		Password: getenv("POSTGRES_PASSWORD"),
		Database: getenv("POSTGRES_DB"),
		Host:     getenv("POSTGRES_HOSTNAME"),
	}

	// Validate required fields
	if config.User == "" {
		return nil, fmt.Errorf("POSTGRES_USER is required")
	}
	if config.Password == "" {
		return nil, fmt.Errorf("POSTGRES_PASSWORD is required")
	}
	if config.Database == "" {
		return nil, fmt.Errorf("POSTGRES_DB is required")
	}
	if config.Host == "" {
		return nil, fmt.Errorf("POSTGRES_HOSTNAME is required")
	}

	return config, nil
}

// ConnectionString returns a PostgreSQL connection string
func (c *PostgresConfig) ConnectionString() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
		c.Host, c.User, c.Password, c.Database)
}
