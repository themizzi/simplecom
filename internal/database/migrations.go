package database

import (
	"fmt"
	"log"
)

// RunMigrations creates the necessary database tables
func RunMigrations() error {
	if DB == nil {
		return fmt.Errorf("database connection not initialized")
	}

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

	_, err := DB.Exec(createOrdersTable)
	if err != nil {
		return fmt.Errorf("failed to create orders table: %w", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}
