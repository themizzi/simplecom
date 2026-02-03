package handlers

import (
	"fmt"
	"html/template"
	"log"
	"net/http"

	"github.com/adyen/ecommerce/internal/repository"
)

// FailureHandler handles payment failure page
type FailureHandler struct {
	template  *template.Template
	orderRepo *repository.OrderRepository
}

// NewFailureHandler creates a new failure handler
func NewFailureHandler(templatePath string, orderRepo *repository.OrderRepository) (*FailureHandler, error) {
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return &FailureHandler{
		template:  tmpl,
		orderRepo: orderRepo,
	}, nil
}

// FailureData represents the data for the failure template
type FailureData struct {
	OrderReference string
	Reason         string
	Message        string
}

// ServeHTTP handles the failure page request
func (h *FailureHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get parameters from query
	reference := r.URL.Query().Get("reference")
	reason := r.URL.Query().Get("reason")

	// Generate user-friendly message based on reason
	message := getFailureMessage(reason)

	data := FailureData{
		OrderReference: reference,
		Reason:         reason,
		Message:        message,
	}

	if err := h.template.Execute(w, data); err != nil {
		log.Printf("Error rendering template: %v", err)
		http.Error(w, "Failed to render page", http.StatusInternalServerError)
	}
}

// getFailureMessage returns a user-friendly message based on the failure reason
func getFailureMessage(reason string) string {
	switch reason {
	case "Refused":
		return "Your payment was declined. Please check your payment details and try again."
	case "Cancelled":
		return "The payment was cancelled. You can try again when you're ready."
	case "Error":
		return "An error occurred while processing your payment. Please try again."
	default:
		return "We couldn't process your payment. Please try again or contact support."
	}
}
