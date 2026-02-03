package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/adyen/ecommerce/internal/services"
)

// SessionHandler handles payment session creation
type SessionHandler struct {
	paymentService services.PaymentService
	product        Product
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(paymentService services.PaymentService, product Product) *SessionHandler {
	return &SessionHandler{
		paymentService: paymentService,
		product:        product,
	}
}

// ClientResponse represents the response sent to the client
type ClientResponse struct {
	SessionID   string `json:"sessionId"`
	SessionData string `json:"sessionData"`
	ClientKey   string `json:"clientKey"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ServeHTTP handles the session creation request
func (h *SessionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Create payment session through service
	result, err := h.paymentService.CreatePaymentSession(
		h.product.Name,
		100, // $1.00 in cents
		"USD",
		"http://localhost:8080/order/confirmation",
	)
	if err != nil {
		log.Printf("Error creating payment session: %v", err)
		sendErrorResponse(w, "Failed to create payment session", http.StatusInternalServerError)
		return
	}

	log.Printf("Payment session created successfully - SessionID: %s, OrderRef: %s", result.SessionID, result.OrderRef)

	// Send response to client
	clientResp := ClientResponse{
		SessionID:   result.SessionID,
		SessionData: result.SessionData,
		ClientKey:   result.ClientKey,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(clientResp); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// sendErrorResponse sends a JSON error response
func sendErrorResponse(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
	})
}
