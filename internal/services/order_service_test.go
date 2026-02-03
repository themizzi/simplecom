package services

import (
	"errors"
	"testing"

	"github.com/adyen/ecommerce/internal/models"
)

// MockOrderRepository is a mock implementation of OrderRepository for testing
type MockOrderRepository struct {
	CreateOrderFunc         func(*models.Order) error
	GetOrderByReferenceFunc func(string) (*models.Order, error)
	UpdateOrderStatusFunc   func(string, string, string) error
}

func (m *MockOrderRepository) CreateOrder(order *models.Order) error {
	if m.CreateOrderFunc != nil {
		return m.CreateOrderFunc(order)
	}
	return nil
}

func (m *MockOrderRepository) GetOrderByReference(reference string) (*models.Order, error) {
	if m.GetOrderByReferenceFunc != nil {
		return m.GetOrderByReferenceFunc(reference)
	}
	return &models.Order{Reference: reference}, nil
}

func (m *MockOrderRepository) UpdateOrderStatus(reference, status, pspReference string) error {
	if m.UpdateOrderStatusFunc != nil {
		return m.UpdateOrderStatusFunc(reference, status, pspReference)
	}
	return nil
}

func TestOrderService_CreateOrder(t *testing.T) {
	tests := []struct {
		name        string
		productName string
		amount      int64
		currency    string
		mockError   error
		wantErr     bool
	}{
		{
			name:        "successful order creation",
			productName: "Test Product",
			amount:      100,
			currency:    "USD",
			mockError:   nil,
			wantErr:     false,
		},
		{
			name:        "repository error",
			productName: "Test Product",
			amount:      100,
			currency:    "USD",
			mockError:   errors.New("database error"),
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockOrderRepository{
				CreateOrderFunc: func(order *models.Order) error {
					if tt.mockError != nil {
						return tt.mockError
					}
					// Verify order fields are set correctly
					if order.ID == "" {
						t.Error("Order ID should not be empty")
					}
					if order.Reference == "" {
						t.Error("Order reference should not be empty")
					}
					if order.Amount != tt.amount {
						t.Errorf("Expected amount %d, got %d", tt.amount, order.Amount)
					}
					if order.Currency != tt.currency {
						t.Errorf("Expected currency %s, got %s", tt.currency, order.Currency)
					}
					if order.Status != models.OrderStatusPending {
						t.Errorf("Expected status %s, got %s", models.OrderStatusPending, order.Status)
					}
					if order.ProductName != tt.productName {
						t.Errorf("Expected product name %s, got %s", tt.productName, order.ProductName)
					}
					return nil
				},
			}

			service := NewOrderService(mockRepo)
			order, err := service.CreateOrder(tt.productName, tt.amount, tt.currency)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && order == nil {
				t.Error("Expected order to be returned, got nil")
			}

			if !tt.wantErr {
				if order.ID == "" {
					t.Error("Order ID should not be empty")
				}
				if order.Reference == "" {
					t.Error("Order reference should not be empty")
				}
			}
		})
	}
}

func TestOrderService_GetOrderByReference(t *testing.T) {
	tests := []struct {
		name      string
		reference string
		mockOrder *models.Order
		mockError error
		wantErr   bool
	}{
		{
			name:      "successful retrieval",
			reference: "ORDER-123",
			mockOrder: &models.Order{
				Reference: "ORDER-123",
				Amount:    100,
			},
			mockError: nil,
			wantErr:   false,
		},
		{
			name:      "order not found",
			reference: "ORDER-999",
			mockOrder: nil,
			mockError: errors.New("order not found"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockOrderRepository{
				GetOrderByReferenceFunc: func(reference string) (*models.Order, error) {
					if tt.mockError != nil {
						return nil, tt.mockError
					}
					return tt.mockOrder, nil
				},
			}

			service := NewOrderService(mockRepo)
			order, err := service.GetOrderByReference(tt.reference)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrderByReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && order == nil {
				t.Error("Expected order to be returned, got nil")
			}
		})
	}
}

func TestOrderService_UpdateOrderStatus(t *testing.T) {
	tests := []struct {
		name         string
		reference    string
		status       string
		pspReference string
		mockError    error
		wantErr      bool
	}{
		{
			name:         "successful update - authorized",
			reference:    "ORDER-123",
			status:       string(models.OrderStatusAuthorized),
			pspReference: "PSP-123",
			mockError:    nil,
			wantErr:      false,
		},
		{
			name:         "successful update - failed",
			reference:    "ORDER-123",
			status:       string(models.OrderStatusFailed),
			pspReference: "PSP-456",
			mockError:    nil,
			wantErr:      false,
		},
		{
			name:         "invalid status",
			reference:    "ORDER-123",
			status:       "invalid_status",
			pspReference: "PSP-123",
			mockError:    nil,
			wantErr:      true,
		},
		{
			name:         "repository error",
			reference:    "ORDER-123",
			status:       string(models.OrderStatusAuthorized),
			pspReference: "PSP-123",
			mockError:    errors.New("database error"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := &MockOrderRepository{
				GetOrderByReferenceFunc: func(reference string) (*models.Order, error) {
					return &models.Order{
						Reference: reference,
						Status:    models.OrderStatusPending,
					}, nil
				},
				UpdateOrderStatusFunc: func(reference, status, pspReference string) error {
					if tt.mockError != nil {
						return tt.mockError
					}
					return nil
				},
			}

			service := NewOrderService(mockRepo)
			err := service.UpdateOrderStatus(tt.reference, tt.status, tt.pspReference)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateOrderStatus() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
