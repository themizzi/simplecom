package handlers

import (
	"html/template"
	"net/http"
)

// Product represents a product item
type Product struct {
	Name        string
	Description string
	Price       string
	ImageURL    string
}

// ProductHandler handles the product page requests
type ProductHandler struct {
	template *template.Template
	product  Product
}

// NewProductHandler creates a new ProductHandler
func NewProductHandler(templatePath string, product Product) (*ProductHandler, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, err
	}

	return &ProductHandler{
		template: tmpl,
		product:  product,
	}, nil
}

// ServeHTTP handles the GET / request
func (h *ProductHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Render the template with the injected product
	if err := h.template.Execute(w, h.product); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
}
