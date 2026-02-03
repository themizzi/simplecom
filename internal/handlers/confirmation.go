package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/adyen/ecommerce/internal/models"
	"github.com/adyen/ecommerce/internal/services"
)

// ConfirmationHandler handles order confirmation page
type ConfirmationHandler struct {
	template       *template.Template
	paymentService services.PaymentService
}

// NewConfirmationHandler creates a new confirmation handler
func NewConfirmationHandler(templatePath string, paymentService services.PaymentService) (*ConfirmationHandler, error) {
	// Create template with custom functions
	funcMap := template.FuncMap{
		"divf": func(a int64, b float64) float64 {
			return float64(a) / b
		},
	}

	tmpl, err := template.New("confirmation.html").Funcs(funcMap).ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &ConfirmationHandler{
		template:       tmpl,
		paymentService: paymentService,
	}, nil
}

// ConfirmationData represents the data for the confirmation template
type ConfirmationData struct {
	Order  *models.Order
	Status string
}

// ServeHTTP handles the confirmation page request
func (h *ConfirmationHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Confirmation page accessed with query params: %v", r.URL.Query())

	// Get sessionId and sessionResult from query parameters
	sessionID := r.URL.Query().Get("sessionId")
	sessionResult := r.URL.Query().Get("sessionResult")

	if sessionID == "" {
		log.Printf("Missing sessionId parameter")
		http.Error(w, "Missing session ID", http.StatusBadRequest)
		return
	}

	log.Printf("Processing payment confirmation - sessionId: %s, sessionResult: %s", sessionID, sessionResult)

	// Verify payment through service
	result, err := h.paymentService.VerifyPayment(sessionID, sessionResult)
	if err != nil {
		log.Printf("Error verifying payment: %v", err)
		http.Error(w, "Failed to verify payment", http.StatusInternalServerError)
		return
	}

	// Check if payment was successful
	if result.Status != string(models.OrderStatusAuthorized) {
		// Redirect to failure page
		failureURL := fmt.Sprintf("/order/failed?reference=%s&reason=%s", result.Order.Reference, result.ResultCode)
		http.Redirect(w, r, failureURL, http.StatusSeeOther)
		return
	}

	// Render confirmation page
	data := ConfirmationData{
		Order:  result.Order,
		Status: "Authorized",
	}

	if err := h.template.Execute(w, data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}
