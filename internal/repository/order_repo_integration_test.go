//go:build integration
// +build integration

package repository

import (
	"testing"
	"time"

	"github.com/adyen/ecommerce/internal/models"
	"github.com/adyen/ecommerce/internal/repository/testutil"
	"github.com/google/uuid"
)

func TestOrderRepository_CreateOrder_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Teardown(t)

	repo := NewOrderRepositoryWithDB(testDB.DB)

	tests := []struct {
		name    string
		order   *models.Order
		wantErr bool
	}{
		{
			name: "create valid order",
			order: &models.Order{
				ID:          uuid.New().String(),
				Reference:   "ORDER-TEST-001",
				Amount:      1000,
				Currency:    "USD",
				Status:      models.OrderStatusPending,
				ProductName: "Test Product",
			},
			wantErr: false,
		},
		{
			name: "create order with all fields",
			order: &models.Order{
				ID:           uuid.New().String(),
				Reference:    "ORDER-TEST-002",
				Amount:       2500,
				Currency:     "EUR",
				Status:       models.OrderStatusAuthorized,
				ProductName:  "Premium Widget",
				PSPReference: "PSP-TEST-123",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.CreateOrder(tt.order)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify timestamps were set
				if tt.order.CreatedAt.IsZero() {
					t.Error("CreatedAt should be set")
				}
				if tt.order.UpdatedAt.IsZero() {
					t.Error("UpdatedAt should be set")
				}

				// Verify order can be retrieved
				retrieved, err := repo.GetOrderByReference(tt.order.Reference)
				if err != nil {
					t.Fatalf("Failed to retrieve created order: %v", err)
				}

				if retrieved.ID != tt.order.ID {
					t.Errorf("ID mismatch: got %v, want %v", retrieved.ID, tt.order.ID)
				}
				if retrieved.Amount != tt.order.Amount {
					t.Errorf("Amount mismatch: got %v, want %v", retrieved.Amount, tt.order.Amount)
				}
				if retrieved.Currency != tt.order.Currency {
					t.Errorf("Currency mismatch: got %v, want %v", retrieved.Currency, tt.order.Currency)
				}
				if retrieved.Status != tt.order.Status {
					t.Errorf("Status mismatch: got %v, want %v", retrieved.Status, tt.order.Status)
				}
			}
		})
	}
}

func TestOrderRepository_CreateOrder_DuplicateReference_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Teardown(t)

	repo := NewOrderRepositoryWithDB(testDB.DB)

	order1 := &models.Order{
		ID:          uuid.New().String(),
		Reference:   "ORDER-DUP-001",
		Amount:      1000,
		Currency:    "USD",
		Status:      models.OrderStatusPending,
		ProductName: "Test Product",
	}

	// Create first order
	err := repo.CreateOrder(order1)
	if err != nil {
		t.Fatalf("Failed to create first order: %v", err)
	}

	// Try to create order with same reference
	order2 := &models.Order{
		ID:          uuid.New().String(),
		Reference:   "ORDER-DUP-001", // Same reference
		Amount:      2000,
		Currency:    "EUR",
		Status:      models.OrderStatusPending,
		ProductName: "Different Product",
	}

	err = repo.CreateOrder(order2)
	if err == nil {
		t.Error("Expected error when creating order with duplicate reference, got nil")
	}
}

func TestOrderRepository_GetOrderByReference_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Teardown(t)

	repo := NewOrderRepositoryWithDB(testDB.DB)

	// Create test order
	order := &models.Order{
		ID:          uuid.New().String(),
		Reference:   "ORDER-GET-001",
		Amount:      1500,
		Currency:    "USD",
		Status:      models.OrderStatusPending,
		ProductName: "Test Product",
	}

	err := repo.CreateOrder(order)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	tests := []struct {
		name      string
		reference string
		wantErr   bool
	}{
		{
			name:      "get existing order",
			reference: "ORDER-GET-001",
			wantErr:   false,
		},
		{
			name:      "get non-existent order",
			reference: "ORDER-NONEXISTENT",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			retrieved, err := repo.GetOrderByReference(tt.reference)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrderByReference() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if retrieved == nil {
					t.Error("Expected order to be returned, got nil")
					return
				}

				if retrieved.Reference != tt.reference {
					t.Errorf("Reference mismatch: got %v, want %v", retrieved.Reference, tt.reference)
				}
				if retrieved.ID != order.ID {
					t.Errorf("ID mismatch: got %v, want %v", retrieved.ID, order.ID)
				}
			}
		})
	}
}

func TestOrderRepository_UpdateOrderStatus_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Teardown(t)

	repo := NewOrderRepositoryWithDB(testDB.DB)

	// Create test order
	order := &models.Order{
		ID:          uuid.New().String(),
		Reference:   "ORDER-UPDATE-001",
		Amount:      1500,
		Currency:    "USD",
		Status:      models.OrderStatusPending,
		ProductName: "Test Product",
	}

	err := repo.CreateOrder(order)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	tests := []struct {
		name         string
		reference    string
		status       models.OrderStatus
		pspReference string
		wantErr      bool
	}{
		{
			name:         "update to authorized",
			reference:    "ORDER-UPDATE-001",
			status:       models.OrderStatusAuthorized,
			pspReference: "PSP-AUTH-123",
			wantErr:      false,
		},
		{
			name:         "update non-existent order",
			reference:    "ORDER-NONEXISTENT",
			status:       models.OrderStatusAuthorized,
			pspReference: "PSP-NONE",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := repo.UpdateOrderStatus(tt.reference, string(tt.status), tt.pspReference)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateOrderStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify the update
				retrieved, err := repo.GetOrderByReference(tt.reference)
				if err != nil {
					t.Fatalf("Failed to retrieve updated order: %v", err)
				}

				if retrieved.Status != tt.status {
					t.Errorf("Status mismatch: got %v, want %v", retrieved.Status, tt.status)
				}
				if retrieved.PSPReference != tt.pspReference {
					t.Errorf("PSPReference mismatch: got %v, want %v", retrieved.PSPReference, tt.pspReference)
				}

				// Verify UpdatedAt changed
				if !retrieved.UpdatedAt.After(retrieved.CreatedAt) {
					t.Error("UpdatedAt should be after CreatedAt")
				}
			}
		})
	}
}

func TestOrderRepository_UpdateOrderStatus_Multiple_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Teardown(t)

	repo := NewOrderRepositoryWithDB(testDB.DB)

	// Create test order
	order := &models.Order{
		ID:          uuid.New().String(),
		Reference:   "ORDER-MULTI-001",
		Amount:      1500,
		Currency:    "USD",
		Status:      models.OrderStatusPending,
		ProductName: "Test Product",
	}

	err := repo.CreateOrder(order)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	// First update
	err = repo.UpdateOrderStatus(order.Reference, string(models.OrderStatusAuthorized), "PSP-001")
	if err != nil {
		t.Fatalf("First update failed: %v", err)
	}

	// Small delay to ensure timestamp changes
	time.Sleep(10 * time.Millisecond)

	// Second update
	err = repo.UpdateOrderStatus(order.Reference, string(models.OrderStatusFailed), "PSP-002")
	if err != nil {
		t.Fatalf("Second update failed: %v", err)
	}

	// Verify final state
	retrieved, err := repo.GetOrderByReference(order.Reference)
	if err != nil {
		t.Fatalf("Failed to retrieve order: %v", err)
	}

	if retrieved.Status != models.OrderStatusFailed {
		t.Errorf("Expected status %v, got %v", models.OrderStatusFailed, retrieved.Status)
	}
	if retrieved.PSPReference != "PSP-002" {
		t.Errorf("Expected PSPReference 'PSP-002', got %v", retrieved.PSPReference)
	}
}

func TestOrderRepository_ConcurrentCreates_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Teardown(t)

	repo := NewOrderRepositoryWithDB(testDB.DB)

	const numOrders = 10
	errChan := make(chan error, numOrders)

	// Create multiple orders concurrently
	for i := 0; i < numOrders; i++ {
		go func(idx int) {
			order := &models.Order{
				ID:          uuid.New().String(),
				Reference:   uuid.New().String(), // Unique reference
				Amount:      int64(1000 + idx),
				Currency:    "USD",
				Status:      models.OrderStatusPending,
				ProductName: "Test Product",
			}
			errChan <- repo.CreateOrder(order)
		}(i)
	}

	// Collect results
	for i := 0; i < numOrders; i++ {
		if err := <-errChan; err != nil {
			t.Errorf("Concurrent create failed: %v", err)
		}
	}
}

func TestOrderRepository_TransactionIsolation_Integration(t *testing.T) {
	// Create two separate test databases to simulate different connections
	testDB1 := testutil.SetupTestDatabase(t)
	defer testDB1.Teardown(t)

	testDB2 := testutil.SetupTestDatabase(t)
	defer testDB2.Teardown(t)

	repo1 := NewOrderRepositoryWithDB(testDB1.DB)
	repo2 := NewOrderRepositoryWithDB(testDB2.DB)

	// Create order in first database
	order := &models.Order{
		ID:          uuid.New().String(),
		Reference:   "ORDER-ISO-001",
		Amount:      1000,
		Currency:    "USD",
		Status:      models.OrderStatusPending,
		ProductName: "Test Product",
	}

	err := repo1.CreateOrder(order)
	if err != nil {
		t.Fatalf("Failed to create order in first database: %v", err)
	}

	// Verify it exists in first database
	_, err = repo1.GetOrderByReference(order.Reference)
	if err != nil {
		t.Errorf("Order should exist in first database: %v", err)
	}

	// Verify it doesn't exist in second database (different schema)
	_, err = repo2.GetOrderByReference(order.Reference)
	if err == nil {
		t.Error("Order should not exist in second database (different schema)")
	}
}

func TestOrderRepository_PSPReferenceNullHandling_Integration(t *testing.T) {
	testDB := testutil.SetupTestDatabase(t)
	defer testDB.Teardown(t)

	repo := NewOrderRepositoryWithDB(testDB.DB)

	// Create order without PSP reference
	order := &models.Order{
		ID:          uuid.New().String(),
		Reference:   "ORDER-NULL-PSP-001",
		Amount:      1000,
		Currency:    "USD",
		Status:      models.OrderStatusPending,
		ProductName: "Test Product",
	}

	err := repo.CreateOrder(order)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	// Retrieve and verify PSP reference is empty
	retrieved, err := repo.GetOrderByReference(order.Reference)
	if err != nil {
		t.Fatalf("Failed to retrieve order: %v", err)
	}

	if retrieved.PSPReference != "" {
		t.Errorf("Expected empty PSPReference, got %v", retrieved.PSPReference)
	}

	// Update with PSP reference
	err = repo.UpdateOrderStatus(order.Reference, string(models.OrderStatusAuthorized), "PSP-123")
	if err != nil {
		t.Fatalf("Failed to update order: %v", err)
	}

	// Retrieve and verify PSP reference is set
	retrieved, err = repo.GetOrderByReference(order.Reference)
	if err != nil {
		t.Fatalf("Failed to retrieve updated order: %v", err)
	}

	if retrieved.PSPReference != "PSP-123" {
		t.Errorf("Expected PSPReference 'PSP-123', got %v", retrieved.PSPReference)
	}
}
