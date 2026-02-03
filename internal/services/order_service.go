package services

import (
	"fmt"

	"github.com/adyen/ecommerce/internal/models"
)

// OrderRepository defines the interface for order persistence
type OrderRepository interface {
	CreateOrder(order *models.Order) error
	GetOrderByReference(reference string) (*models.Order, error)
	UpdateOrderStatus(reference, status, pspReference string) error
}

// OrderService handles order business logic
type OrderService interface {
	CreateOrder(productName string, amount int64, currency string) (*models.Order, error)
	GetOrderByReference(reference string) (*models.Order, error)
	UpdateOrderStatus(reference, status, pspReference string) error
}

// OrderServiceImpl implements OrderService
type OrderServiceImpl struct {
	orderRepo OrderRepository
}

// NewOrderService creates a new order service
func NewOrderService(orderRepo OrderRepository) OrderService {
	return &OrderServiceImpl{
		orderRepo: orderRepo,
	}
}

// CreateOrder creates a new order with generated ID and reference
func (s *OrderServiceImpl) CreateOrder(productName string, amount int64, currency string) (*models.Order, error) {
	// Create order using domain factory method
	order, err := models.NewOrder(productName, amount, currency)
	if err != nil {
		return nil, fmt.Errorf("invalid order: %w", err)
	}

	// Persist to database
	if err := s.orderRepo.CreateOrder(order); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	return order, nil
}

// GetOrderByReference retrieves an order by its reference
func (s *OrderServiceImpl) GetOrderByReference(reference string) (*models.Order, error) {
	order, err := s.orderRepo.GetOrderByReference(reference)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	return order, nil
}

// UpdateOrderStatus updates the status of an order
func (s *OrderServiceImpl) UpdateOrderStatus(reference, status, pspReference string) error {
	// Get the order
	order, err := s.orderRepo.GetOrderByReference(reference)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Use domain methods to transition state
	switch models.OrderStatus(status) {
	case models.OrderStatusAuthorized:
		if err := order.Authorize(pspReference); err != nil {
			return err
		}
	case models.OrderStatusFailed:
		if err := order.Fail(); err != nil {
			return err
		}
	case models.OrderStatusCancelled:
		if err := order.Cancel(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("invalid order status: %s", status)
	}

	// Update in database
	if err := s.orderRepo.UpdateOrderStatus(reference, string(order.Status), order.PSPReference); err != nil {
		return fmt.Errorf("failed to update order status: %w", err)
	}

	return nil
}
