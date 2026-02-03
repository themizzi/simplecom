package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adyen/ecommerce/internal/repository"
)

func TestFailureHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		queryParams    string
		expectedStatus int
		checkContent   []string
	}{
		{
			name:           "refused payment",
			method:         http.MethodGet,
			queryParams:    "?reference=ORDER-123&reason=Refused",
			expectedStatus: http.StatusOK,
			checkContent:   []string{"ORDER-123", "Refused"},
		},
		{
			name:           "cancelled payment",
			method:         http.MethodGet,
			queryParams:    "?reference=ORDER-456&reason=Cancelled",
			expectedStatus: http.StatusOK,
			checkContent:   []string{"ORDER-456", "Cancelled"},
		},
		{
			name:           "error payment",
			method:         http.MethodGet,
			queryParams:    "?reference=ORDER-789&reason=Error",
			expectedStatus: http.StatusOK,
			checkContent:   []string{"ORDER-789", "Error"},
		},
		{
			name:           "unknown reason",
			method:         http.MethodGet,
			queryParams:    "?reference=ORDER-999&reason=Unknown",
			expectedStatus: http.StatusOK,
			checkContent:   []string{"ORDER-999"},
		},
		{
			name:           "missing reference parameter",
			method:         http.MethodGet,
			queryParams:    "?reason=Refused",
			expectedStatus: http.StatusOK,
			checkContent:   []string{"declined"},
		},
		{
			name:           "missing reason parameter",
			method:         http.MethodGet,
			queryParams:    "?reference=ORDER-111",
			expectedStatus: http.StatusOK,
			checkContent:   []string{"ORDER-111"},
		},
		{
			name:           "no query parameters",
			method:         http.MethodGet,
			queryParams:    "",
			expectedStatus: http.StatusOK,
			checkContent:   []string{"Payment Failed"},
		},
		{
			name:           "method not allowed - POST",
			method:         http.MethodPost,
			queryParams:    "?reference=ORDER-123&reason=Refused",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method not allowed - PUT",
			method:         http.MethodPut,
			queryParams:    "?reference=ORDER-123&reason=Refused",
			expectedStatus: http.StatusMethodNotAllowed,
		},
		{
			name:           "method not allowed - DELETE",
			method:         http.MethodDelete,
			queryParams:    "?reference=ORDER-123&reason=Refused",
			expectedStatus: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock order repository (not used in current implementation, but passed to handler)
			orderRepo := repository.NewOrderRepositoryWithDB(nil)

			// Create handler
			handler, err := NewFailureHandler("../../templates/failure.html", orderRepo)
			if err != nil {
				t.Fatalf("Failed to create handler: %v", err)
			}

			// Create request with query parameters
			req := httptest.NewRequest(tt.method, "/order/failed"+tt.queryParams, nil)
			w := httptest.NewRecorder()

			// Execute
			handler.ServeHTTP(w, req)

			// Assert status code
			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			// For successful requests, check response content
			if tt.expectedStatus == http.StatusOK && len(tt.checkContent) > 0 {
				body := w.Body.String()
				for _, content := range tt.checkContent {
					if !strings.Contains(strings.ToLower(body), strings.ToLower(content)) {
						t.Errorf("expected response to contain '%s' (case-insensitive)", content)
					}
				}
			}
		})
	}
}

func TestGetFailureMessage(t *testing.T) {
	tests := []struct {
		name           string
		reason         string
		expectedPhrase string // A phrase that should be in the message
	}{
		{
			name:           "refused payment",
			reason:         "Refused",
			expectedPhrase: "declined",
		},
		{
			name:           "cancelled payment",
			reason:         "Cancelled",
			expectedPhrase: "cancelled",
		},
		{
			name:           "error payment",
			reason:         "Error",
			expectedPhrase: "error occurred",
		},
		{
			name:           "unknown reason",
			reason:         "SomeUnknownReason",
			expectedPhrase: "couldn't process",
		},
		{
			name:           "empty reason",
			reason:         "",
			expectedPhrase: "couldn't process",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := getFailureMessage(tt.reason)

			if message == "" {
				t.Error("expected non-empty message")
			}

			if !strings.Contains(strings.ToLower(message), strings.ToLower(tt.expectedPhrase)) {
				t.Errorf("expected message to contain '%s', got: %s", tt.expectedPhrase, message)
			}
		})
	}
}

func TestNewFailureHandler(t *testing.T) {
	tests := []struct {
		name         string
		templatePath string
		wantErr      bool
	}{
		{
			name:         "invalid template path",
			templatePath: "/invalid/path/to/failure.html",
			wantErr:      true,
		},
		{
			name:         "empty template path",
			templatePath: "",
			wantErr:      true,
		},
	}

	orderRepo := repository.NewOrderRepositoryWithDB(nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewFailureHandler(tt.templatePath, orderRepo)

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

func TestFailureHandler_QueryParameterExtraction(t *testing.T) {
	// Test that query parameters are correctly extracted and passed to template
	orderRepo := repository.NewOrderRepositoryWithDB(nil)

	handler, err := NewFailureHandler("../../templates/failure.html", orderRepo)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	testCases := []struct {
		reference string
		reason    string
	}{
		{"ORDER-ABC-123", "Refused"},
		{"ORDER-XYZ-999", "Cancelled"},
		{"ORDER-TEST-456", "Error"},
	}

	for _, tc := range testCases {
		req := httptest.NewRequest(http.MethodGet, "/order/failed?reference="+tc.reference+"&reason="+tc.reason, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
			continue
		}

		body := w.Body.String()

		// Check that reference appears in response
		if tc.reference != "" && !strings.Contains(body, tc.reference) {
			t.Errorf("expected response to contain reference '%s'", tc.reference)
		}
	}
}

func TestFailureHandler_TemplateExecutionError(t *testing.T) {
	orderRepo := repository.NewOrderRepositoryWithDB(nil)

	// Create a handler with a malformed template
	tmpl, err := template.New("failure.html").Parse("{{.InvalidField.NonExistent}}")
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	handler := &FailureHandler{
		template:  tmpl,
		orderRepo: orderRepo,
	}

	req := httptest.NewRequest(http.MethodGet, "/order/failed?reference=ORDER-123&reason=Refused", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 500 due to template execution error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
