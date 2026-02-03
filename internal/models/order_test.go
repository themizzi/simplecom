package models

import (
	"testing"
)

func TestNewOrder(t *testing.T) {
	tests := []struct {
		name        string
		productName string
		amount      int64
		currency    string
		wantErr     error
	}{
		{
			name:        "valid order",
			productName: "Test Product",
			amount:      1000,
			currency:    "EUR",
			wantErr:     nil,
		},
		{
			name:        "invalid amount - zero",
			productName: "Test Product",
			amount:      0,
			currency:    "EUR",
			wantErr:     ErrInvalidAmount,
		},
		{
			name:        "invalid amount - negative",
			productName: "Test Product",
			amount:      -100,
			currency:    "EUR",
			wantErr:     ErrInvalidAmount,
		},
		{
			name:        "invalid currency - too short",
			productName: "Test Product",
			amount:      1000,
			currency:    "US",
			wantErr:     ErrInvalidCurrency,
		},
		{
			name:        "invalid currency - too long",
			productName: "Test Product",
			amount:      1000,
			currency:    "EURO",
			wantErr:     ErrInvalidCurrency,
		},
		{
			name:        "empty product name",
			productName: "",
			amount:      1000,
			currency:    "EUR",
			wantErr:     ErrInvalidProductName,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := NewOrder(tt.productName, tt.amount, tt.currency)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("NewOrder() error = %v, wantErr %v", err, tt.wantErr)
				}
				if order != nil {
					t.Error("Expected order to be nil when error occurs")
				}
				return
			}

			if err != nil {
				t.Errorf("NewOrder() unexpected error = %v", err)
				return
			}

			if order.ID == "" {
				t.Error("Order ID should not be empty")
			}
			if order.Reference == "" {
				t.Error("Order reference should not be empty")
			}
			if order.Status != OrderStatusPending {
				t.Errorf("Expected status %s, got %s", OrderStatusPending, order.Status)
			}
			if order.Amount != tt.amount {
				t.Errorf("Expected amount %d, got %d", tt.amount, order.Amount)
			}
			if order.Currency != tt.currency {
				t.Errorf("Expected currency %s, got %s", tt.currency, order.Currency)
			}
		})
	}
}

func TestOrder_Authorize(t *testing.T) {
	tests := []struct {
		name         string
		initialState OrderStatus
		pspReference string
		wantErr      bool
	}{
		{
			name:         "authorize pending order",
			initialState: OrderStatusPending,
			pspReference: "PSP-123",
			wantErr:      false,
		},
		{
			name:         "cannot authorize already authorized order",
			initialState: OrderStatusAuthorized,
			pspReference: "PSP-123",
			wantErr:      true,
		},
		{
			name:         "cannot authorize failed order",
			initialState: OrderStatusFailed,
			pspReference: "PSP-123",
			wantErr:      true,
		},
		{
			name:         "cannot authorize cancelled order",
			initialState: OrderStatusCancelled,
			pspReference: "PSP-123",
			wantErr:      true,
		},
		{
			name:         "empty PSP reference",
			initialState: OrderStatusPending,
			pspReference: "",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{
				ID:       "test-id",
				Status:   tt.initialState,
				Amount:   1000,
				Currency: "EUR",
			}

			err := order.Authorize(tt.pspReference)

			if (err != nil) != tt.wantErr {
				t.Errorf("Authorize() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if order.Status != OrderStatusAuthorized {
					t.Errorf("Expected status %s, got %s", OrderStatusAuthorized, order.Status)
				}
				if order.PSPReference != tt.pspReference {
					t.Errorf("Expected PSPReference %s, got %s", tt.pspReference, order.PSPReference)
				}
			}
		})
	}
}

func TestOrder_Fail(t *testing.T) {
	tests := []struct {
		name         string
		initialState OrderStatus
		wantErr      bool
	}{
		{
			name:         "fail pending order",
			initialState: OrderStatusPending,
			wantErr:      false,
		},
		{
			name:         "cannot fail authorized order",
			initialState: OrderStatusAuthorized,
			wantErr:      true,
		},
		{
			name:         "cannot fail cancelled order",
			initialState: OrderStatusCancelled,
			wantErr:      true,
		},
		{
			name:         "can fail already failed order (idempotent)",
			initialState: OrderStatusFailed,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{
				ID:       "test-id",
				Status:   tt.initialState,
				Amount:   1000,
				Currency: "EUR",
			}

			err := order.Fail()

			if (err != nil) != tt.wantErr {
				t.Errorf("Fail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && order.Status != OrderStatusFailed {
				t.Errorf("Expected status %s, got %s", OrderStatusFailed, order.Status)
			}
		})
	}
}

func TestOrder_Cancel(t *testing.T) {
	tests := []struct {
		name         string
		initialState OrderStatus
		wantErr      bool
	}{
		{
			name:         "cancel pending order",
			initialState: OrderStatusPending,
			wantErr:      false,
		},
		{
			name:         "cannot cancel authorized order",
			initialState: OrderStatusAuthorized,
			wantErr:      true,
		},
		{
			name:         "cancel failed order",
			initialState: OrderStatusFailed,
			wantErr:      false,
		},
		{
			name:         "can cancel already cancelled order (idempotent)",
			initialState: OrderStatusCancelled,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{
				ID:       "test-id",
				Status:   tt.initialState,
				Amount:   1000,
				Currency: "EUR",
			}

			err := order.Cancel()

			if (err != nil) != tt.wantErr {
				t.Errorf("Cancel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && order.Status != OrderStatusCancelled {
				t.Errorf("Expected status %s, got %s", OrderStatusCancelled, order.Status)
			}
		})
	}
}

func TestOrder_StatusChecks(t *testing.T) {
	order := &Order{
		ID:       "test-id",
		Status:   OrderStatusPending,
		Amount:   1000,
		Currency: "EUR",
	}

	if !order.IsPending() {
		t.Error("Expected order to be pending")
	}
	if !order.CanBeModified() {
		t.Error("Expected pending order to be modifiable")
	}

	order.Status = OrderStatusAuthorized
	if !order.IsAuthorized() {
		t.Error("Expected order to be authorized")
	}
	if order.CanBeModified() {
		t.Error("Expected authorized order to not be modifiable")
	}

	order.Status = OrderStatusFailed
	if !order.IsFailed() {
		t.Error("Expected order to be failed")
	}

	order.Status = OrderStatusCancelled
	if !order.IsCancelled() {
		t.Error("Expected order to be cancelled")
	}
}

func TestOrder_GetFormattedAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   int64
		currency string
		expected string
	}{
		{
			name:     "100 EUR cents",
			amount:   100,
			currency: "EUR",
			expected: "1.00 EUR",
		},
		{
			name:     "1234 USD cents",
			amount:   1234,
			currency: "USD",
			expected: "12.34 USD",
		},
		{
			name:     "99999 GBP cents",
			amount:   99999,
			currency: "GBP",
			expected: "999.99 GBP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order := &Order{
				Amount:   tt.amount,
				Currency: tt.currency,
			}

			result := order.GetFormattedAmount()
			if result != tt.expected {
				t.Errorf("GetFormattedAmount() = %s, want %s", result, tt.expected)
			}
		})
	}
}
