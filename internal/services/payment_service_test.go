package services

import (
	"errors"
	"testing"

	"github.com/adyen/ecommerce/internal/config"
	"github.com/adyen/ecommerce/internal/models"
)

// MockAdyenClient is a mock implementation of AdyenClient for testing
type MockAdyenClient struct {
	CreateSessionFunc    func(*SessionRequest) (*SessionResponse, error)
	GetSessionStatusFunc func(string, string) (*SessionStatusResponse, error)
}

func (m *MockAdyenClient) CreateSession(req *SessionRequest) (*SessionResponse, error) {
	if m.CreateSessionFunc != nil {
		return m.CreateSessionFunc(req)
	}
	return &SessionResponse{
		ID:          "session-123",
		SessionData: "test-data",
		ExpiresAt:   "2026-02-03T12:00:00Z",
	}, nil
}

func (m *MockAdyenClient) GetSessionStatus(sessionID, sessionResult string) (*SessionStatusResponse, error) {
	if m.GetSessionStatusFunc != nil {
		return m.GetSessionStatusFunc(sessionID, sessionResult)
	}
	return &SessionStatusResponse{
		ID:        sessionID,
		Status:    "completed",
		Reference: "ORDER-123",
		Payments: []struct {
			ResultCode   string `json:"resultCode"`
			PSPReference string `json:"pspReference"`
		}{
			{
				ResultCode:   "Authorised",
				PSPReference: "PSP-123",
			},
		},
	}, nil
}

// MockOrderService is a mock implementation of OrderService for testing
type MockOrderService struct {
	CreateOrderFunc         func(string, int64, string) (*models.Order, error)
	GetOrderByReferenceFunc func(string) (*models.Order, error)
	UpdateOrderStatusFunc   func(string, string, string) error
}

func (m *MockOrderService) CreateOrder(productName string, amount int64, currency string) (*models.Order, error) {
	if m.CreateOrderFunc != nil {
		return m.CreateOrderFunc(productName, amount, currency)
	}
	return &models.Order{
		Reference:   "ORDER-123",
		Amount:      amount,
		Currency:    currency,
		ProductName: productName,
		Status:      models.OrderStatusPending,
	}, nil
}

func (m *MockOrderService) GetOrderByReference(reference string) (*models.Order, error) {
	if m.GetOrderByReferenceFunc != nil {
		return m.GetOrderByReferenceFunc(reference)
	}
	return &models.Order{
		Reference: reference,
		Status:    models.OrderStatusPending,
	}, nil
}

func (m *MockOrderService) UpdateOrderStatus(reference, status, pspReference string) error {
	if m.UpdateOrderStatusFunc != nil {
		return m.UpdateOrderStatusFunc(reference, status, pspReference)
	}
	return nil
}

func TestPaymentService_CreatePaymentSession(t *testing.T) {
	tests := []struct {
		name         string
		productName  string
		amount       int64
		currency     string
		returnURL    string
		orderError   error
		sessionError error
		wantErr      bool
	}{
		{
			name:         "successful session creation",
			productName:  "Test Product",
			amount:       100,
			currency:     "USD",
			returnURL:    "http://localhost:8080/confirmation",
			orderError:   nil,
			sessionError: nil,
			wantErr:      false,
		},
		{
			name:         "order creation fails",
			productName:  "Test Product",
			amount:       100,
			currency:     "USD",
			returnURL:    "http://localhost:8080/confirmation",
			orderError:   errors.New("database error"),
			sessionError: nil,
			wantErr:      true,
		},
		{
			name:         "Adyen session creation fails",
			productName:  "Test Product",
			amount:       100,
			currency:     "USD",
			returnURL:    "http://localhost:8080/confirmation",
			orderError:   nil,
			sessionError: errors.New("API error"),
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdyen := &MockAdyenClient{
				CreateSessionFunc: func(req *SessionRequest) (*SessionResponse, error) {
					if tt.sessionError != nil {
						return nil, tt.sessionError
					}
					// Verify request fields
					if req.Amount.Value != tt.amount {
						t.Errorf("Expected amount %d, got %d", tt.amount, req.Amount.Value)
					}
					if req.Amount.Currency != tt.currency {
						t.Errorf("Expected currency %s, got %s", tt.currency, req.Amount.Currency)
					}
					if req.ReturnUrl != tt.returnURL {
						t.Errorf("Expected return URL %s, got %s", tt.returnURL, req.ReturnUrl)
					}
					return &SessionResponse{
						ID:          "session-123",
						SessionData: "test-data",
					}, nil
				},
			}

			mockOrder := &MockOrderService{
				CreateOrderFunc: func(productName string, amount int64, currency string) (*models.Order, error) {
					if tt.orderError != nil {
						return nil, tt.orderError
					}
					return &models.Order{
						Reference:   "ORDER-123",
						Amount:      amount,
						Currency:    currency,
						ProductName: productName,
						Status:      models.OrderStatusPending,
					}, nil
				},
			}

			cfg := &config.AdyenConfig{
				MerchantAccount: "TestMerchant",
				ClientKey:       "test-client-key",
			}

			service := NewPaymentService(mockAdyen, mockOrder, cfg)
			result, err := service.CreatePaymentSession(tt.productName, tt.amount, tt.currency, tt.returnURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePaymentSession() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("Expected result to be returned, got nil")
					return
				}
				if result.SessionID != "session-123" {
					t.Errorf("Expected session ID 'session-123', got '%s'", result.SessionID)
				}
				if result.ClientKey != cfg.ClientKey {
					t.Errorf("Expected client key '%s', got '%s'", cfg.ClientKey, result.ClientKey)
				}
			}
		})
	}
}

func TestPaymentService_VerifyPayment(t *testing.T) {
	tests := []struct {
		name           string
		sessionID      string
		sessionResult  string
		sessionStatus  *SessionStatusResponse
		sessionError   error
		orderError     error
		updateError    error
		wantErr        bool
		expectedStatus string
	}{
		{
			name:          "successful authorization",
			sessionID:     "session-123",
			sessionResult: "result-123",
			sessionStatus: &SessionStatusResponse{
				ID:        "session-123",
				Status:    "completed",
				Reference: "ORDER-123",
				Payments: []struct {
					ResultCode   string `json:"resultCode"`
					PSPReference string `json:"pspReference"`
				}{
					{
						ResultCode:   "Authorised",
						PSPReference: "PSP-123",
					},
				},
			},
			sessionError:   nil,
			orderError:     nil,
			updateError:    nil,
			wantErr:        false,
			expectedStatus: string(models.OrderStatusAuthorized),
		},
		{
			name:          "payment refused",
			sessionID:     "session-123",
			sessionResult: "result-123",
			sessionStatus: &SessionStatusResponse{
				ID:        "session-123",
				Status:    "completed",
				Reference: "ORDER-123",
				Payments: []struct {
					ResultCode   string `json:"resultCode"`
					PSPReference string `json:"pspReference"`
				}{
					{
						ResultCode:   "Refused",
						PSPReference: "PSP-456",
					},
				},
			},
			sessionError:   nil,
			orderError:     nil,
			updateError:    nil,
			wantErr:        false,
			expectedStatus: string(models.OrderStatusFailed),
		},
		{
			name:           "session status error",
			sessionID:      "session-123",
			sessionResult:  "result-123",
			sessionStatus:  nil,
			sessionError:   errors.New("API error"),
			orderError:     nil,
			updateError:    nil,
			wantErr:        true,
			expectedStatus: "",
		},
		{
			name:          "order not found",
			sessionID:     "session-123",
			sessionResult: "result-123",
			sessionStatus: &SessionStatusResponse{
				ID:        "session-123",
				Reference: "ORDER-999",
				Payments: []struct {
					ResultCode   string `json:"resultCode"`
					PSPReference string `json:"pspReference"`
				}{
					{ResultCode: "Authorised", PSPReference: "PSP-123"},
				},
			},
			sessionError:   nil,
			orderError:     errors.New("order not found"),
			updateError:    nil,
			wantErr:        true,
			expectedStatus: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockAdyen := &MockAdyenClient{
				GetSessionStatusFunc: func(sessionID, sessionResult string) (*SessionStatusResponse, error) {
					if tt.sessionError != nil {
						return nil, tt.sessionError
					}
					return tt.sessionStatus, nil
				},
			}

			mockOrder := &MockOrderService{
				GetOrderByReferenceFunc: func(reference string) (*models.Order, error) {
					if tt.orderError != nil {
						return nil, tt.orderError
					}
					return &models.Order{
						Reference: reference,
						Status:    models.OrderStatusPending,
					}, nil
				},
				UpdateOrderStatusFunc: func(reference, status, pspReference string) error {
					if tt.updateError != nil {
						return tt.updateError
					}
					return nil
				},
			}

			cfg := &config.AdyenConfig{
				MerchantAccount: "TestMerchant",
			}

			service := NewPaymentService(mockAdyen, mockOrder, cfg)
			result, err := service.VerifyPayment(tt.sessionID, tt.sessionResult)

			if (err != nil) != tt.wantErr {
				t.Errorf("VerifyPayment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("Expected result to be returned, got nil")
					return
				}
				if result.Status != tt.expectedStatus {
					t.Errorf("Expected status '%s', got '%s'", tt.expectedStatus, result.Status)
				}
				if result.Order == nil {
					t.Error("Expected order in result, got nil")
				}
			}
		})
	}
}

func TestMapResultCodeToStatus(t *testing.T) {
	tests := []struct {
		resultCode     string
		expectedStatus models.OrderStatus
	}{
		{"Authorised", models.OrderStatusAuthorized},
		{"Refused", models.OrderStatusFailed},
		{"Error", models.OrderStatusFailed},
		{"Cancelled", models.OrderStatusCancelled},
		{"Pending", models.OrderStatusPending},
		{"Unknown", models.OrderStatusPending},
	}

	for _, tt := range tests {
		t.Run(tt.resultCode, func(t *testing.T) {
			status := mapResultCodeToStatus(tt.resultCode)
			if status != tt.expectedStatus {
				t.Errorf("mapResultCodeToStatus(%s) = %s, want %s", tt.resultCode, status, tt.expectedStatus)
			}
		})
	}
}

func TestCalculateAmountExcludingTax(t *testing.T) {
	tests := []struct {
		name               string
		amountIncludingTax int64
		taxPercentage      int64
		expected           int64
	}{
		{"10% tax on 100", 100, 1000, 90},
		{"20% tax on 120", 120, 2000, 100},
		{"0% tax on 100", 100, 0, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAmountExcludingTax(tt.amountIncludingTax, tt.taxPercentage)
			if result != tt.expected {
				t.Errorf("calculateAmountExcludingTax(%d, %d) = %d, want %d",
					tt.amountIncludingTax, tt.taxPercentage, result, tt.expected)
			}
		})
	}
}

func TestCalculateTaxAmount(t *testing.T) {
	tests := []struct {
		name               string
		amountIncludingTax int64
		taxPercentage      int64
		expected           int64
	}{
		{"10% tax on 100", 100, 1000, 10},
		{"20% tax on 120", 120, 2000, 20},
		{"0% tax on 100", 100, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateTaxAmount(tt.amountIncludingTax, tt.taxPercentage)
			if result != tt.expected {
				t.Errorf("calculateTaxAmount(%d, %d) = %d, want %d",
					tt.amountIncludingTax, tt.taxPercentage, result, tt.expected)
			}
		})
	}
}
