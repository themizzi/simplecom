package services

import (
	"fmt"
	"log"

	"github.com/adyen/ecommerce/internal/config"
	"github.com/adyen/ecommerce/internal/models"
)

// PaymentService handles payment-related business logic
type PaymentService interface {
	CreatePaymentSession(productName string, amount int64, currency string, returnURL string) (*PaymentSessionResult, error)
	VerifyPayment(sessionID, sessionResult string) (*PaymentVerificationResult, error)
}

// PaymentServiceImpl implements PaymentService
type PaymentServiceImpl struct {
	adyenClient  AdyenClient
	orderService OrderService
	config       *config.AdyenConfig
}

// NewPaymentService creates a new payment service
func NewPaymentService(adyenClient AdyenClient, orderService OrderService, cfg *config.AdyenConfig) PaymentService {
	return &PaymentServiceImpl{
		adyenClient:  adyenClient,
		orderService: orderService,
		config:       cfg,
	}
}

// PaymentSessionResult represents the result of creating a payment session
type PaymentSessionResult struct {
	SessionID   string
	SessionData string
	ClientKey   string
	OrderRef    string
}

// PaymentVerificationResult represents the result of verifying a payment
type PaymentVerificationResult struct {
	Order        *models.Order
	ResultCode   string
	PSPReference string
	Status       string
}

// CreatePaymentSession creates a new payment session and order
func (s *PaymentServiceImpl) CreatePaymentSession(productName string, amount int64, currency string, returnURL string) (*PaymentSessionResult, error) {
	// Create order in database
	order, err := s.orderService.CreateOrder(productName, amount, currency)
	if err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	log.Printf("Created order: %s", order.Reference)

	// Create session request for Adyen
	sessionReq := &SessionRequest{
		MerchantAccount: s.config.MerchantAccount,
		Amount: Amount{
			Currency: currency,
			Value:    amount,
		},
		Reference:             order.Reference,
		ReturnUrl:             returnURL,
		CountryCode:           "US",
		ShopperLocale:         "en-US",
		Channel:               "Web",
		AllowedPaymentMethods: []string{"scheme"}, // Only allow credit/debit cards
		LineItems: []LineItem{
			{
				Quantity:           1,
				AmountExcludingTax: calculateAmountExcludingTax(amount, 1000), // 10% tax
				TaxPercentage:      1000,                                      // 10%
				Description:        productName,
				ID:                 "widget-001",
				TaxAmount:          calculateTaxAmount(amount, 1000),
				AmountIncludingTax: amount,
			},
		},
	}

	// Create session with Adyen
	sessionResp, err := s.adyenClient.CreateSession(sessionReq)
	if err != nil {
		return nil, fmt.Errorf("failed to create Adyen session: %w", err)
	}

	return &PaymentSessionResult{
		SessionID:   sessionResp.ID,
		SessionData: sessionResp.SessionData,
		ClientKey:   s.config.ClientKey,
		OrderRef:    order.Reference,
	}, nil
}

// VerifyPayment verifies a payment and updates the order status
func (s *PaymentServiceImpl) VerifyPayment(sessionID, sessionResult string) (*PaymentVerificationResult, error) {
	// Get payment status from Adyen
	sessionStatus, err := s.adyenClient.GetSessionStatus(sessionID, sessionResult)
	if err != nil {
		return nil, fmt.Errorf("failed to get session status: %w", err)
	}

	// Extract payment result
	var resultCode, pspReference string
	if len(sessionStatus.Payments) > 0 {
		resultCode = sessionStatus.Payments[0].ResultCode
		pspReference = sessionStatus.Payments[0].PSPReference
	}

	log.Printf("Payment details retrieved - ResultCode: %s, MerchantReference: %s, PSPReference: %s",
		resultCode, sessionStatus.Reference, pspReference)

	// Map Adyen result code to order status
	orderStatus := mapResultCodeToStatus(resultCode)
	log.Printf("Mapped result code '%s' to order status '%s'", resultCode, orderStatus)

	// Get order from database
	order, err := s.orderService.GetOrderByReference(sessionStatus.Reference)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	// Update order status
	if err := s.orderService.UpdateOrderStatus(order.Reference, string(orderStatus), pspReference); err != nil {
		log.Printf("Warning: failed to update order status: %v", err)
		// Continue anyway - we can still return the result
	}

	// Update order object with new values
	order.Status = orderStatus
	order.PSPReference = pspReference

	return &PaymentVerificationResult{
		Order:        order,
		ResultCode:   resultCode,
		PSPReference: pspReference,
		Status:       string(orderStatus),
	}, nil
}

// mapResultCodeToStatus maps Adyen result code to our order status
func mapResultCodeToStatus(resultCode string) models.OrderStatus {
	switch resultCode {
	case "Authorised":
		return models.OrderStatusAuthorized
	case "Refused", "Error":
		return models.OrderStatusFailed
	case "Cancelled":
		return models.OrderStatusCancelled
	default:
		return models.OrderStatusPending
	}
}

// calculateAmountExcludingTax calculates the amount excluding tax
func calculateAmountExcludingTax(amountIncludingTax int64, taxPercentage int64) int64 {
	// taxPercentage is in basis points (1000 = 10%)
	return (amountIncludingTax * 10000) / (10000 + taxPercentage)
}

// calculateTaxAmount calculates the tax amount
func calculateTaxAmount(amountIncludingTax int64, taxPercentage int64) int64 {
	excludingTax := calculateAmountExcludingTax(amountIncludingTax, taxPercentage)
	return amountIncludingTax - excludingTax
}
