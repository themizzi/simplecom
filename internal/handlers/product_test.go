package handlers

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProductHandler_ServeHTTP(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		expectedStatus int
		templatePath   string
		templateError  bool
		checkContent   []string
	}{
		{
			name:           "successful GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			templatePath:   "../../templates/product.html",
			checkContent:   []string{"Test Product", "$10.00"},
		},
		{
			name:           "method not allowed - POST",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			templatePath:   "../../templates/product.html",
		},
		{
			name:           "method not allowed - PUT",
			method:         http.MethodPut,
			expectedStatus: http.StatusMethodNotAllowed,
			templatePath:   "../../templates/product.html",
		},
		{
			name:           "method not allowed - DELETE",
			method:         http.MethodDelete,
			expectedStatus: http.StatusMethodNotAllowed,
			templatePath:   "../../templates/product.html",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test product
			product := Product{
				Name:        "Test Product",
				Description: "A wonderful test product",
				Price:       "$10.00",
				ImageURL:    "/static/images/product.jpg",
			}

			// Create handler
			handler, err := NewProductHandler(tt.templatePath, product)
			if err != nil {
				t.Fatalf("Failed to create handler: %v", err)
			}

			// Create request
			req := httptest.NewRequest(tt.method, "/", nil)
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
						t.Errorf("expected response to contain '%s'", content)
					}
				}
			}
		})
	}
}

func TestProductHandler_TemplateExecutionError(t *testing.T) {
	product := Product{
		Name:        "Test Product",
		Description: "Description",
		Price:       "$5.00",
		ImageURL:    "/image.jpg",
	}

	// Create a handler with a malformed template
	tmpl, err := template.New("product.html").Parse("{{.InvalidField.NonExistent}}")
	if err != nil {
		t.Fatalf("Failed to create test template: %v", err)
	}

	handler := &ProductHandler{
		template: tmpl,
		product:  product,
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	// Should return 500 due to template execution error
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestNewProductHandler(t *testing.T) {
	tests := []struct {
		name         string
		templatePath string
		product      Product
		wantErr      bool
	}{
		{
			name:         "invalid template path",
			templatePath: "/invalid/path/to/template.html",
			product:      Product{Name: "Test"},
			wantErr:      true,
		},
		{
			name:         "empty template path",
			templatePath: "",
			product:      Product{Name: "Test"},
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewProductHandler(tt.templatePath, tt.product)

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
