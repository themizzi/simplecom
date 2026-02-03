package handlers

import (
	"html/template"
	"net/http"

	"github.com/adyen/ecommerce/internal/config"
)

// CheckoutHandler handles the checkout page
type CheckoutHandler struct {
	template *template.Template
	product  Product
	config   *config.AdyenConfig
}

// CheckoutData represents the data passed to the checkout template
type CheckoutData struct {
	Product   Product
	ClientKey string
}

// NewCheckoutHandler creates a new checkout handler
func NewCheckoutHandler(templatePath string, product Product, cfg *config.AdyenConfig) (*CheckoutHandler, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, err
	}

	return &CheckoutHandler{
		template: tmpl,
		product:  product,
		config:   cfg,
	}, nil
}

// ServeHTTP handles the checkout page request
func (h *CheckoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := CheckoutData{
		Product:   h.product,
		ClientKey: h.config.ClientKey,
	}

	if err := h.template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
