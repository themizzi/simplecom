package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adyen/ecommerce/internal/services"
)

// MockPaymentService is a mock implementation of PaymentService for testing
type MockPaymentService struct {
	CreatePaymentSessionFunc func(string, int64, string, string) (*services.PaymentSessionResult, error)
	VerifyPaymentFunc        func(string, string) (*services.PaymentVerificationResult, error)
}

func (m *MockPaymentService) CreatePaymentSession(productName string, amount int64, currency string, returnURL string) (*services.PaymentSessionResult, error) {
	if m.CreatePaymentSessionFunc != nil {
		return m.CreatePaymentSessionFunc(productName, amount, currency, returnURL)
	}
	return &services.PaymentSessionResult{
		SessionID:   "test-session-123",
		SessionData: "test-session-data",
		ClientKey:   "test-client-key",
		OrderRef:    "ORDER-123",
	}, nil
}

func (m *MockPaymentService) VerifyPayment(sessionID, sessionResult string) (*services.PaymentVerificationResult, error) {
	if m.VerifyPaymentFunc != nil {
		return m.VerifyPaymentFunc(sessionID, sessionResult)
	}
	return nil, nil
}

func TestSessionHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name               string
		method             string
		mockSessionResult  *services.PaymentSessionResult
		mockSessionError   error
		expectedStatus     int
		expectedSessionID  string
		expectedClientKey  string
		checkErrorResponse bool
	}{
		{
			name:   "successful session creation",
			method: http.MethodPost,
			mockSessionResult: &services.PaymentSessionResult{
				SessionID:   "session-abc-123",
				SessionData: "encrypted-session-data",
				ClientKey:   "test_CLIENTKEY123",
				OrderRef:    "ORDER-001",
			},
			expectedStatus:    http.StatusOK,
			expectedSessionID: "session-abc-123",
			expectedClientKey: "test_CLIENTKEY123",
		},
		{
			name:               "payment service error",
			method:             http.MethodPost,
			mockSessionError:   errors.New("payment service unavailable"),
			expectedStatus:     http.StatusInternalServerError,
			checkErrorResponse: true,
		},
		{
			name:           "method not allowed - GET",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method not allowed - PUT",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method not allowed - DELETE",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock payment service
			mockService := &MockPaymentService{
				CreatePaymentSessionFunc: func(productName string, amount int64, currency string, returnURL string) (*services.PaymentSessionResult, error) {
					if tt.mockSessionError != nil {
						return nil, tt.mockSessionError
					}
					return tt.mockSessionResult, nil
				},
			}

			// Create handler
			product := Product{
				Name:        "Test Product",
				Description: "A test product",
				Price:       "$1.00",
			}
			handler := NewSessionHandler(mockService, product)

			// Create request
			req := httptest.NewRequest(tt.method, "/api/sessions", nil)
			w := httptest.NewRecorder()

			// Execute
			handler.ServeHTTP(w, req)

			// Assert status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// For successful requests, verify JSON response
			if tt.expectedStatus == http.StatusOK {
				var response ClientResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				if err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if response.SessionID != tt.expectedSessionID {
					t.Errorf("expected sessionID %s, got %s", tt.expectedSessionID, response.SessionID)
				}

				if response.ClientKey != tt.expectedClientKey {
					t.Errorf("expected clientKey %s, got %s", tt.expectedClientKey, response.ClientKey)
				}

				if response.SessionData != tt.mockSessionResult.SessionData {
					t.Errorf("expected sessionData %s, got %s", tt.mockSessionResult.SessionData, response.SessionData)
				}

				// Check Content-Type header
				contentType := w.Header().Get("Content-Type")
				if contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %s", contentType)
				}
			}

			// For error responses, verify error JSON structure
			if tt.checkErrorResponse && tt.expectedStatus >= 400 {
				var errorResp ErrorResponse
				err := json.NewDecoder(w.Body).Decode(&errorResp)
				if err != nil {
					t.Fatalf("failed to decode error response: %v", err)
				}

				if errorResp.Error == "" {
					t.Error("expected error field to be non-empty")
				}

				if errorResp.Message == "" {
					t.Error("expected message field to be non-empty")
				}
			}
		})
	}
}

func TestSessionHandler_ServiceInvocation(t *testing.T) {
	// Test that the handler calls the payment service with correct parameters
	var capturedProductName string
	var capturedAmount int64
	var capturedCurrency string
	var capturedReturnURL string

	mockService := &MockPaymentService{
		CreatePaymentSessionFunc: func(productName string, amount int64, currency string, returnURL string) (*services.PaymentSessionResult, error) {
			capturedProductName = productName
			capturedAmount = amount
			capturedCurrency = currency
			capturedReturnURL = returnURL

			return &services.PaymentSessionResult{
				SessionID:   "session-123",
				SessionData: "data",
				ClientKey:   "key",
				OrderRef:    "ORDER-123",
			}, nil
		},
	}

	product := Product{
		Name: "Premium Widget",
	}
	handler := NewSessionHandler(mockService, product)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify service was called with correct parameters
	if capturedProductName != "Premium Widget" {
		t.Errorf("expected product name 'Premium Widget', got '%s'", capturedProductName)
	}

	if capturedAmount != 100 {
		t.Errorf("expected amount 100, got %d", capturedAmount)
	}

	if capturedCurrency != "USD" {
		t.Errorf("expected currency 'USD', got '%s'", capturedCurrency)
	}

	if capturedReturnURL != "http://localhost:8080/order/confirmation" {
		t.Errorf("expected returnURL 'http://localhost:8080/order/confirmation', got '%s'", capturedReturnURL)
	}
}

func TestSessionHandler_JSONEncodingError(t *testing.T) {
	// Test the error path where JSON encoding fails
	// We'll use a response recorder and close it to simulate encoding failure
	mockService := &MockPaymentService{
		CreatePaymentSessionFunc: func(productName string, amount int64, currency string, returnURL string) (*services.PaymentSessionResult, error) {
			return &services.PaymentSessionResult{
				SessionID:   "session-123",
				SessionData: "data",
				ClientKey:   "key",
				OrderRef:    "ORDER-123",
			}, nil
		},
	}

	product := Product{
		Name: "Test Product",
	}
	handler := NewSessionHandler(mockService, product)

	req := httptest.NewRequest(http.MethodPost, "/api/sessions", nil)

	// Create a custom response writer that will fail on write
	w := &failingWriter{
		header: make(http.Header),
	}

	handler.ServeHTTP(w, req)

	// The handler logs the error but doesn't return an error status
	// This test ensures the error path is covered
}

// failingWriter is a ResponseWriter that fails on Write
type failingWriter struct {
	header     http.Header
	statusCode int
}

func (f *failingWriter) Header() http.Header {
	return f.header
}

func (f *failingWriter) Write([]byte) (int, error) {
	return 0, &customError{msg: "write failed"}
}

func (f *failingWriter) WriteHeader(statusCode int) {
	f.statusCode = statusCode
}

type customError struct {
	msg string
}

func (e *customError) Error() string {
	return e.msg
}
