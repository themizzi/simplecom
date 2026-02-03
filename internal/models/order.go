package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OrderStatus represents valid order states
type OrderStatus string

// Order statuses
const (
	OrderStatusPending    OrderStatus = "pending"
	OrderStatusAuthorized OrderStatus = "authorized"
	OrderStatusFailed     OrderStatus = "failed"
	OrderStatusCancelled  OrderStatus = "cancelled"
)

// Order represents a customer order with business logic
type Order struct {
	ID           string
	Reference    string
	Amount       int64
	Currency     string
	Status       OrderStatus
	ProductName  string
	PSPReference string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Domain errors
var (
	ErrInvalidAmount           = errors.New("order amount must be positive")
	ErrInvalidCurrency         = errors.New("currency code must be 3 characters")
	ErrInvalidProductName      = errors.New("product name cannot be empty")
	ErrInvalidStatusTransition = errors.New("invalid order status transition")
	ErrOrderAlreadyAuthorized  = errors.New("order is already authorized")
	ErrOrderAlreadyFailed      = errors.New("order is already failed")
	ErrOrderAlreadyCancelled   = errors.New("order is already cancelled")
)

// NewOrder creates a new order with validation
func NewOrder(productName string, amount int64, currency string) (*Order, error) {
	if err := validateOrderInput(productName, amount, currency); err != nil {
		return nil, err
	}

	orderRef := fmt.Sprintf("ORDER-%d", time.Now().Unix())
	now := time.Now()

	return &Order{
		ID:          uuid.New().String(),
		Reference:   orderRef,
		Amount:      amount,
		Currency:    currency,
		Status:      OrderStatusPending,
		ProductName: productName,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// validateOrderInput validates order creation parameters
func validateOrderInput(productName string, amount int64, currency string) error {
	if amount <= 0 {
		return ErrInvalidAmount
	}
	if len(currency) != 3 {
		return ErrInvalidCurrency
	}
	if productName == "" {
		return ErrInvalidProductName
	}
	return nil
}

// Authorize marks the order as authorized with a PSP reference
func (o *Order) Authorize(pspReference string) error {
	if o.Status != OrderStatusPending {
		return fmt.Errorf("%w: cannot authorize order with status %s", ErrInvalidStatusTransition, o.Status)
	}
	if pspReference == "" {
		return errors.New("PSP reference cannot be empty")
	}

	o.Status = OrderStatusAuthorized
	o.PSPReference = pspReference
	o.UpdatedAt = time.Now()
	return nil
}

// Fail marks the order as failed
func (o *Order) Fail() error {
	if o.Status == OrderStatusAuthorized {
		return fmt.Errorf("%w: cannot fail an authorized order", ErrInvalidStatusTransition)
	}
	if o.Status == OrderStatusCancelled {
		return fmt.Errorf("%w: cannot fail a cancelled order", ErrInvalidStatusTransition)
	}

	o.Status = OrderStatusFailed
	o.UpdatedAt = time.Now()
	return nil
}

// Cancel marks the order as cancelled
func (o *Order) Cancel() error {
	if o.Status == OrderStatusAuthorized {
		return fmt.Errorf("%w: cannot cancel an authorized order", ErrInvalidStatusTransition)
	}

	o.Status = OrderStatusCancelled
	o.UpdatedAt = time.Now()
	return nil
}

// IsPending returns true if the order is in pending status
func (o *Order) IsPending() bool {
	return o.Status == OrderStatusPending
}

// IsAuthorized returns true if the order is authorized
func (o *Order) IsAuthorized() bool {
	return o.Status == OrderStatusAuthorized
}

// IsFailed returns true if the order has failed
func (o *Order) IsFailed() bool {
	return o.Status == OrderStatusFailed
}

// IsCancelled returns true if the order is cancelled
func (o *Order) IsCancelled() bool {
	return o.Status == OrderStatusCancelled
}

// CanBeModified returns true if the order can still be modified
func (o *Order) CanBeModified() bool {
	return o.Status == OrderStatusPending
}

// GetFormattedAmount returns the amount formatted with currency
func (o *Order) GetFormattedAmount() string {
	amountInMajorUnits := float64(o.Amount) / 100.0
	return fmt.Sprintf("%.2f %s", amountInMajorUnits, o.Currency)
}
