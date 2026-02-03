package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/adyen/ecommerce/internal/database"
	"github.com/adyen/ecommerce/internal/models"
)

// OrderRepository handles database operations for orders
type OrderRepository struct {
	db *sql.DB
}

// NewOrderRepository creates a new order repository
func NewOrderRepository() *OrderRepository {
	return &OrderRepository{
		db: database.DB,
	}
}

// NewOrderRepositoryWithDB creates a new order repository with a specific database connection
func NewOrderRepositoryWithDB(db *sql.DB) *OrderRepository {
	return &OrderRepository{
		db: db,
	}
}

// CreateOrder creates a new order in the database
func (r *OrderRepository) CreateOrder(order *models.Order) error {
	query := `
		INSERT INTO orders (id, reference, amount, currency, status, product_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	_, err := r.db.Exec(query,
		order.ID,
		order.Reference,
		order.Amount,
		order.Currency,
		order.Status,
		order.ProductName,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}

	order.CreatedAt = now
	order.UpdatedAt = now

	return nil
}

// GetOrderByReference retrieves an order by its reference
func (r *OrderRepository) GetOrderByReference(reference string) (*models.Order, error) {
	query := `
		SELECT id, reference, amount, currency, status, product_name, 
		       COALESCE(psp_reference, ''), created_at, updated_at
		FROM orders
		WHERE reference = $1
	`

	order := &models.Order{}
	err := r.db.QueryRow(query, reference).Scan(
		&order.ID,
		&order.Reference,
		&order.Amount,
		&order.Currency,
		&order.Status,
		&order.ProductName,
		&order.PSPReference,
		&order.CreatedAt,
		&order.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("order not found")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	return order, nil
}

// UpdateOrderStatus updates the status and PSP reference of an order
func (r *OrderRepository) UpdateOrderStatus(reference, status, pspReference string) error {
	query := `
		UPDATE orders
		SET status = $1, psp_reference = $2, updated_at = $3
		WHERE reference = $4
	`

	result, err := r.db.Exec(query, status, pspReference, time.Now(), reference)
	if err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("order not found")
	}

	return nil
}
