package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/adyen/ecommerce/internal/config"
)

// AdyenClient handles communication with Adyen API
type AdyenClient interface {
	CreateSession(req *SessionRequest) (*SessionResponse, error)
	GetSessionStatus(sessionID, sessionResult string) (*SessionStatusResponse, error)
}

// HTTPAdyenClient implements AdyenClient using HTTP
type HTTPAdyenClient struct {
	config     *config.AdyenConfig
	httpClient *http.Client
}

// NewAdyenClient creates a new Adyen API client
func NewAdyenClient(cfg *config.AdyenConfig) AdyenClient {
	return &HTTPAdyenClient{
		config:     cfg,
		httpClient: &http.Client{},
	}
}

// SessionRequest represents the request to create a payment session
type SessionRequest struct {
	MerchantAccount       string                 `json:"merchantAccount"`
	Amount                Amount                 `json:"amount"`
	Reference             string                 `json:"reference"`
	ReturnUrl             string                 `json:"returnUrl"`
	CountryCode           string                 `json:"countryCode"`
	ShopperLocale         string                 `json:"shopperLocale"`
	Channel               string                 `json:"channel"`
	AllowedPaymentMethods []string               `json:"allowedPaymentMethods,omitempty"`
	LineItems             []LineItem             `json:"lineItems,omitempty"`
	Metadata              map[string]interface{} `json:"metadata,omitempty"`
}

// Amount represents a monetary amount
type Amount struct {
	Currency string `json:"currency"`
	Value    int64  `json:"value"`
}

// LineItem represents a product line item
type LineItem struct {
	Quantity           int64  `json:"quantity"`
	AmountExcludingTax int64  `json:"amountExcludingTax"`
	TaxPercentage      int64  `json:"taxPercentage"`
	Description        string `json:"description"`
	ID                 string `json:"id"`
	TaxAmount          int64  `json:"taxAmount"`
	AmountIncludingTax int64  `json:"amountIncludingTax"`
}

// SessionResponse represents the response from Adyen session creation
type SessionResponse struct {
	SessionData string `json:"sessionData"`
	ID          string `json:"id"`
	ExpiresAt   string `json:"expiresAt"`
}

// SessionStatusResponse represents the response from Adyen session status
type SessionStatusResponse struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Reference string `json:"reference"`
	Payments  []struct {
		ResultCode   string `json:"resultCode"`
		PSPReference string `json:"pspReference"`
	} `json:"payments"`
}

// CreateSession creates a new payment session with Adyen
func (c *HTTPAdyenClient) CreateSession(req *SessionRequest) (*SessionResponse, error) {
	// Set merchant account from config if not provided
	if req.MerchantAccount == "" {
		req.MerchantAccount = c.config.MerchantAccount
	}

	// Marshal request
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Determine API endpoint based on environment
	apiURL := c.getAPIEndpoint("/v71/sessions")

	// Create HTTP request
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Key", c.config.APIKey)

	// Send request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		log.Printf("Adyen API error (status %d): %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Adyen session created successfully (status %d)", resp.StatusCode)

	// Parse response
	var sessionResp SessionResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Printf("Session ID: %s", sessionResp.ID)

	return &sessionResp, nil
}

// GetSessionStatus retrieves the status of a payment session
func (c *HTTPAdyenClient) GetSessionStatus(sessionID, sessionResult string) (*SessionStatusResponse, error) {
	log.Printf("Fetching session status for sessionId: %s with sessionResult: %s", sessionID, sessionResult)

	// Determine API endpoint based on environment
	apiURL := c.getAPIEndpoint(fmt.Sprintf("/v71/sessions/%s?sessionResult=%s", sessionID, sessionResult))

	// Create HTTP request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-API-Key", c.config.APIKey)

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	log.Printf("Session status API response (status %d): %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse session response
	var sessionResp SessionStatusResponse
	if err := json.Unmarshal(body, &sessionResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &sessionResp, nil
}

// getAPIEndpoint returns the full API endpoint URL based on environment
func (c *HTTPAdyenClient) getAPIEndpoint(path string) string {
	if c.config.Environment == "LIVE" {
		return "https://checkout-live.adyen.com" + path
	}
	return "https://checkout-test.adyen.com" + path
}
