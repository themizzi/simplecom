package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adyen/ecommerce/internal/config"
)

func TestCheckoutHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		checkContent   []string
	}{
		{
			name:           "successful request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			checkContent:   []string{"Test Widget", "test_CLIENT_KEY_123"},
		},
		{
			name:           "POST request also works",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "PUT request works",
			method:         http.MethodPut,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test configuration
			cfg := &config.AdyenConfig{
				ClientKey: "test_CLIENT_KEY_123",
				APIKey:    "test_api_key",
			}

			// Create test product
			product := Product{
				Name:        "Test Widget",
				Description: "A premium test widget",
				Price:       "$25.00",
				ImageURL:    "/static/images/widget.jpg",
			}

			// Create handler
			handler, err := NewCheckoutHandler("../../templates/checkout.html", product, cfg)
			if err != nil {
				t.Fatalf("Failed to create handler: %v", err)
			}

			// Create request
			req := httptest.NewRequest(tt.method, "/checkout", nil)
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
					if !strings.Contains(body, content) {
						t.Errorf("expected response to contain '%s', but it was not found", content)
					}
				}
			}
		})
	}
}

func TestCheckoutHandler_TemplateData(t *testing.T) {
	// Test that CheckoutData is properly populated
	cfg := &config.AdyenConfig{
		ClientKey: "my_client_key_abc123",
		APIKey:    "api_key",
	}

	product := Product{
		Name:        "Super Product",
		Description: "The best product ever",
		Price:       "$99.99",
		ImageURL:    "/images/super.jpg",
	}

	handler, err := NewCheckoutHandler("../../templates/checkout.html", product, cfg)
	if err != nil {
		t.Fatalf("Failed to create handler: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/checkout", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Verify that the product data is rendered in the response
	if !strings.Contains(body, product.Name) {
		t.Errorf("expected response to contain product name '%s'", product.Name)
	}

	// Verify that the client key is rendered
	if !strings.Contains(body, cfg.ClientKey) {
		t.Errorf("expected response to contain client key '%s'", cfg.ClientKey)
	}
}

func TestNewCheckoutHandler(t *testing.T) {
	tests := []struct {
		name         string
		templatePath string
		product      Product
		config       *config.AdyenConfig
		wantErr      bool
	}{
		{
			name:         "invalid template path",
			templatePath: "/invalid/path/to/checkout.html",
			product:      Product{Name: "Test"},
			config:       &config.AdyenConfig{ClientKey: "key"},
			wantErr:      true,
		},
		{
			name:         "empty template path",
			templatePath: "",
			product:      Product{Name: "Test"},
			config:       &config.AdyenConfig{ClientKey: "key"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewCheckoutHandler(tt.templatePath, tt.product, tt.config)

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

func TestCheckoutHandler_TemplateExecutionError(t *testing.T) {
	// Create a handler with a broken template to test error handling
	cfg := &config.AdyenConfig{
		ClientKey: "test_key",
	}

	product := Product{
		Name: "Test Product",
	}

	// Create a handler with a malformed template
	tmpl, err := template.New("checkout.html").Parse("{{.InvalidField.NonExistent}}")
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	handler := &CheckoutHandler{
		template: tmpl,
		product:  product,
		config:   cfg,
	}

	req := httptest.NewRequest(http.MethodGet, "/checkout", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 500 due to template execution error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}
