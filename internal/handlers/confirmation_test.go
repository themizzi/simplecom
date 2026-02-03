package handlers

import (
	"errors"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adyen/ecommerce/internal/models"
	"github.com/adyen/ecommerce/internal/services"
)

func TestConfirmationHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name              string
		method            string
		queryParams       string
		mockVerifyResult  *services.PaymentVerificationResult
		mockVerifyError   error
		expectedStatus    int
		expectedLocation  string // for redirects
		checkContent      []string
		skipTemplateCheck bool
	}{
		{
			name:        "successful payment verification",
			method:      http.MethodGet,
			queryParams: "?sessionId=sess-123&sessionResult=result-abc",
			mockVerifyResult: &services.PaymentVerificationResult{
				Order: &models.Order{
					Reference:    "ORDER-12345",
					Amount:       2500,
					Currency:     "USD",
					ProductName:  "Premium Widget",
					Status:       models.OrderStatusAuthorized,
					PSPReference: "PSP-67890",
				},
				ResultCode:   "Authorised",
				PSPReference: "PSP-67890",
				Status:       string(models.OrderStatusAuthorized),
			},
			expectedStatus: http.StatusOK,
			checkContent:   []string{"ORDER-12345", "Authorized"},
		},
		{
			name:        "failed payment redirects to failure page",
			method:      http.MethodGet,
			queryParams: "?sessionId=sess-123&sessionResult=result-abc",
			mockVerifyResult: &services.PaymentVerificationResult{
				Order: &models.Order{
					Reference:   "ORDER-99999",
					Amount:      1000,
					Currency:    "USD",
					ProductName: "Widget",
					Status:      models.OrderStatusFailed,
				},
				ResultCode: "Refused",
				Status:     string(models.OrderStatusFailed),
			},
			expectedStatus:    http.StatusSeeOther,
			expectedLocation:  "/order/failed?reference=ORDER-99999&reason=Refused",
			skipTemplateCheck: true,
		},
		{
			name:        "cancelled payment redirects to failure page",
			method:      http.MethodGet,
			queryParams: "?sessionId=sess-456&sessionResult=result-xyz",
			mockVerifyResult: &services.PaymentVerificationResult{
				Order: &models.Order{
					Reference:   "ORDER-88888",
					Amount:      500,
					Currency:    "EUR",
					ProductName: "Basic Widget",
					Status:      models.OrderStatusCancelled,
				},
				ResultCode: "Cancelled",
				Status:     string(models.OrderStatusCancelled),
			},
			expectedStatus:    http.StatusSeeOther,
			expectedLocation:  "/order/failed?reference=ORDER-88888&reason=Cancelled",
			skipTemplateCheck: true,
		},
		{
			name:              "missing sessionId parameter",
			method:            http.MethodGet,
			queryParams:       "?sessionResult=result-abc",
			expectedStatus:    http.StatusBadRequest,
			skipTemplateCheck: true,
		},
		{
			name:              "empty sessionId parameter",
			method:            http.MethodGet,
			queryParams:       "?sessionId=&sessionResult=result-abc",
			expectedStatus:    http.StatusBadRequest,
			skipTemplateCheck: true,
		},
		{
			name:              "payment service error",
			method:            http.MethodGet,
			queryParams:       "?sessionId=sess-123&sessionResult=result-abc",
			mockVerifyError:   errors.New("payment service unavailable"),
			expectedStatus:    http.StatusInternalServerError,
			skipTemplateCheck: true,
		},
		{
			name:              "method not allowed - POST",
			method:            http.MethodPost,
			queryParams:       "?sessionId=sess-123&sessionResult=result-abc",
			expectedStatus:    http.StatusMethodNotAllowed,
			skipTemplateCheck: true,
		},
		{
			name:              "method not allowed - PUT",
			method:            http.MethodPut,
			queryParams:       "?sessionId=sess-123&sessionResult=result-abc",
			expectedStatus:    http.StatusMethodNotAllowed,
			skipTemplateCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock payment service
			mockService := &MockPaymentService{
				VerifyPaymentFunc: func(sessionID, sessionResult string) (*services.PaymentVerificationResult, error) {
					if tt.mockVerifyError != nil {
						return nil, tt.mockVerifyError
					}
					return tt.mockVerifyResult, nil
				},
			}

			// Create handler
			handler, err := NewConfirmationHandler("../../templates/confirmation.html", mockService)
			if err != nil {
				if tt.skipTemplateCheck {
					t.Skip("Template file not available for this test")
					return
				}
				t.Fatalf("Failed to create handler: %v", err)
			}

			// Create request with query parameters
			req := httptest.NewRequest(tt.method, "/order/confirmation"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Execute
			handler.ServeHTTP(w, req)

			// Assert status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// Check redirect location for 303 responses
			if tt.expectedStatus == http.StatusSeeOther {
				location := w.Header().Get("Location")
				if location != tt.expectedLocation {
					t.Errorf("expected redirect to '%s', got '%s'", tt.expectedLocation, location)
				}
			}

			// For successful requests, check response content
			if tt.expectedStatus == http.StatusOK && len(tt.checkContent) > 0 {
				body := w.Body.String()
				for _, content := range tt.checkContent {
					if !strings.Contains(body, content) {
						t.Errorf("expected response to contain '%s'", content)
					}
				}
			}
		})
	}
}

func TestConfirmationHandler_ServiceInvocation(t *testing.T) {
	// Test that the handler calls verify payment with correct parameters
	var capturedSessionID string
	var capturedSessionResult string

	mockService := &MockPaymentService{
		VerifyPaymentFunc: func(sessionID, sessionResult string) (*services.PaymentVerificationResult, error) {
			capturedSessionID = sessionID
			capturedSessionResult = sessionResult

			return &services.PaymentVerificationResult{
				Order: &models.Order{
					Reference:    "ORDER-123",
					Status:       models.OrderStatusAuthorized,
					Amount:       1000,
					Currency:     "USD",
					ProductName:  "Test",
					PSPReference: "PSP-123",
				},
				Status: string(models.OrderStatusAuthorized),
			}, nil
		},
	}

	handler, err := NewConfirmationHandler("../../templates/confirmation.html", mockService)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/order/confirmation?sessionId=test-session-789&sessionResult=test-result-xyz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Verify service was called with correct parameters
	if capturedSessionID != "test-session-789" {
		t.Errorf("expected sessionID 'test-session-789', got '%s'", capturedSessionID)
	}

	if capturedSessionResult != "test-result-xyz" {
		t.Errorf("expected sessionResult 'test-result-xyz', got '%s'", capturedSessionResult)
	}
}

func TestNewConfirmationHandler(t *testing.T) {
	tests := []struct {
		name         string
		templatePath string
		wantErr      bool
	}{
		{
			name:         "invalid template path",
			templatePath: "/invalid/path/to/confirmation.html",
			wantErr:      true,
		},
		{
			name:         "empty template path",
			templatePath: "",
			wantErr:      true,
		},
	}

	mockService := &MockPaymentService{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewConfirmationHandler(tt.templatePath, mockService)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}

			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tt.wantErr && handler != nil {
				t.Error("expected nil handler when error occurs")
			}
		})
	}
}

func TestConfirmationHandler_TemplateExecutionError(t *testing.T) {
	mockService := &MockPaymentService{
		VerifyPaymentFunc: func(sessionID, sessionResult string) (*services.PaymentVerificationResult, error) {
			return &services.PaymentVerificationResult{
				Order: &models.Order{
					Reference:    "ORDER-123",
					Status:       models.OrderStatusAuthorized,
					Amount:       1000,
					Currency:     "USD",
					ProductName:  "Test",
					PSPReference: "PSP-123",
				},
				Status: string(models.OrderStatusAuthorized),
			}, nil
		},
	}

	// Create a handler with a malformed template
	funcMap := template.FuncMap{
		"divf": func(a int64, b float64) float64 {
			return float64(a) / b
		},
	}
	tmpl, err := template.New("confirmation.html").Funcs(funcMap).Parse("{{.InvalidField.NonExistent}}")
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	handler := &ConfirmationHandler{
		template:       tmpl,
		paymentService: mockService,
	}

	req := httptest.NewRequest(http.MethodGet, "/order/confirmation?sessionId=sess-123&sessionResult=result-abc", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 500 due to template execution error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
