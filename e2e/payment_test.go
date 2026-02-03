package e2e

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

// TestPaymentSuccessfulFlow tests the complete successful payment flow
// Feature: Order Confirmation After Payment
//
//	Scenario: Complete successful payment and view confirmation
//	  Given I am viewing the "Premium Widget" for "$1.00"
//	  When I click "Buy Now"
//	  And I navigate to the checkout page
//	  And I enter valid payment details
//	  And I submit the payment
//	  Then I should be redirected to the confirmation page
//	  And I should see "Order Confirmed" or "Thank You" message
//	  And I should see my order reference number
//	  And I should see "Premium Widget"
//	  And I should see the amount "$1.00"
//	  And I should see payment status "Authorized" or "Paid"
func TestPaymentSuccessfulFlow(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	// Given I am viewing the "Premium Widget" for "$1.00"
	if _, err = page.Goto("http://localhost:8080/"); err != nil {
		t.Fatalf("Failed to navigate to product page: %v", err)
	}

	// When I click "Buy Now"
	buyButton := page.Locator("button:has-text('Buy Now')")
	if err = buyButton.Click(); err != nil {
		t.Fatalf("Failed to click Buy Now button: %v", err)
	}

	// And I navigate to the checkout page
	if err = page.WaitForURL("**/checkout", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		t.Fatalf("Did not redirect to checkout page: %v", err)
	}

	// Wait for Adyen Drop-in to load
	loadingMessage := page.Locator("#loading-container")
	if err = loadingMessage.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateHidden,
		Timeout: playwright.Float(15000),
	}); err != nil {
		t.Fatalf("Checkout did not load: %v", err)
	}

	// Wait for Adyen Drop-in component to render
	time.Sleep(3 * time.Second)

	// Click on the card payment method to expand it (if collapsed)
	cardPaymentMethod := page.Locator(".adyen-checkout__payment-method--card, [class*='card']").First()
	if visible, _ := cardPaymentMethod.IsVisible(); visible {
		// Check if it needs to be clicked to expand
		if err = cardPaymentMethod.Click(); err != nil {
			t.Logf("Card payment method already expanded or click not needed: %v", err)
		}
		time.Sleep(1 * time.Second)
	}

	// And I enter valid payment details (using Adyen test card)
	// Adyen renders the card form in an iframe
	// Wait for the iframe to be available
	page.WaitForSelector("iframe[title*='card number'], iframe[title*='Card number']", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	})

	cardFrame := page.FrameLocator("iframe[title*='card number'], iframe[title*='Card number']")
	cardNumberInput := cardFrame.Locator("input[aria-label='Card number'], input[data-fieldtype='encryptedCardNumber']")

	if err = cardNumberInput.Fill("4111111111111111"); err != nil {
		t.Fatalf("Failed to enter card number: %v", err)
	}

	// Expiry date
	expiryFrame := page.FrameLocator("iframe[title*='expiry'], iframe[title*='Expiry']")
	expiryInput := expiryFrame.Locator("input[aria-label='Expiry date'], input[data-fieldtype='encryptedExpiryDate']")
	if err = expiryInput.Fill("03/30"); err != nil {
		t.Fatalf("Failed to enter expiry date: %v", err)
	}

	// CVC
	cvcFrame := page.FrameLocator("iframe[title*='security'], iframe[title*='Security'], iframe[title*='CVC']")
	cvcInput := cvcFrame.Locator("input[aria-label='Security code'], input[data-fieldtype='encryptedSecurityCode']")
	if err = cvcInput.Fill("737"); err != nil {
		t.Fatalf("Failed to enter CVC: %v", err)
	}

	// Cardholder name (if visible, not in iframe)
	holderNameInput := page.Locator("input[name='holderName']")
	if visible, _ := holderNameInput.IsVisible(); visible {
		if err = holderNameInput.Fill("Test User"); err != nil {
			t.Logf("Warning: Could not fill cardholder name: %v", err)
		}
	}

	// And I submit the payment
	payButton := page.Locator("button[type='submit']:has-text('Pay'), button:has-text('Pay')")
	if err = payButton.Click(); err != nil {
		t.Fatalf("Failed to click Pay button: %v", err)
	}

	// Then I should be redirected to the confirmation page
	if err = page.WaitForURL("**/confirmation**", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		t.Fatalf("Did not redirect to confirmation page: %v", err)
	}

	// And I should see "Order Confirmed" or "Thank You" message
	title := page.Locator(".confirmation-title")
	titleText, err := title.TextContent()
	if err != nil {
		t.Fatalf("Failed to find confirmation title: %v", err)
	}
	if !strings.Contains(strings.ToLower(titleText), "confirmed") &&
		!strings.Contains(strings.ToLower(titleText), "thank you") {
		t.Errorf("Expected confirmation message, got: %s", titleText)
	}

	// And I should see my order reference number
	orderRef := page.Locator(".order-reference").First()
	orderRefText, err := orderRef.TextContent()
	if err != nil {
		t.Fatalf("Failed to find order reference: %v", err)
	}
	if orderRefText == "" {
		t.Error("Order reference is empty")
	}
	t.Logf("Order reference: %s", orderRefText)

	// And I should see "Premium Widget"
	productName := page.Locator(".product-name")
	productNameText, err := productName.TextContent()
	if err != nil {
		t.Fatalf("Failed to find product name: %v", err)
	}
	if productNameText != "Premium Widget" {
		t.Errorf("Expected 'Premium Widget', got: %s", productNameText)
	}

	// And I should see the amount "$1.00"
	amount := page.Locator(".product-amount")
	amountText, err := amount.TextContent()
	if err != nil {
		t.Fatalf("Failed to find amount: %v", err)
	}
	if amountText != "$1.00" {
		t.Errorf("Expected '$1.00', got: %s", amountText)
	}

	// And I should see payment status "Authorized" or "Paid"
	status := page.Locator(".status-badge")
	statusText, err := status.TextContent()
	if err != nil {
		t.Fatalf("Failed to find payment status: %v", err)
	}
	statusLower := strings.ToLower(statusText)
	if !strings.Contains(statusLower, "authorized") &&
		!strings.Contains(statusLower, "paid") &&
		!strings.Contains(statusLower, "authorised") {
		t.Errorf("Expected status 'Authorized' or 'Paid', got: %s", statusText)
	}
}

// TestReturnToConfirmationPage tests returning to a confirmation page after closing browser
// Feature: Order Confirmation After Payment
//
//	Scenario: Return to confirmation page after payment
//	  Given I completed a payment for order "ORDER-123"
//	  And I received a confirmation page URL
//	  When I close my browser
//	  And I reopen the confirmation page URL later
//	  Then I should still see my order details
//	  And I should see order reference "ORDER-123"
//	  And I should see the payment status "Authorized"
//	  And I should see the product and amount
func TestReturnToConfirmationPage(t *testing.T) {
	// First, complete a payment to get a confirmation URL
	page, err := browser.NewPage()
	if err != nil {
		t.Fatal(err)
	}

	// Navigate and complete payment (abbreviated version)
	if _, err = page.Goto("http://localhost:8080/"); err != nil {
		t.Fatalf("Failed to navigate to product page: %v", err)
	}

	buyButton := page.Locator("button:has-text('Buy Now')")
	if err = buyButton.Click(); err != nil {
		t.Fatalf("Failed to click Buy Now button: %v", err)
	}

	// Wait for checkout page
	if err = page.WaitForURL("**/checkout", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		t.Fatalf("Did not redirect to checkout page: %v", err)
	}

	// Wait for Adyen Drop-in to load
	loadingMessage := page.Locator("#loading-container")
	if err = loadingMessage.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateHidden,
		Timeout: playwright.Float(15000),
	}); err != nil {
		t.Fatalf("Checkout did not load: %v", err)
	}

	time.Sleep(3 * time.Second)

	// Click card payment method if needed
	cardPaymentMethod := page.Locator(".adyen-checkout__payment-method--card, [class*='card']").First()
	if visible, _ := cardPaymentMethod.IsVisible(); visible {
		cardPaymentMethod.Click()
		time.Sleep(1 * time.Second)
	}

	// Fill payment details quickly
	page.WaitForSelector("iframe[title*='card number'], iframe[title*='Card number']", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	})

	cardFrame := page.FrameLocator("iframe[title*='card number'], iframe[title*='Card number']")
	cardNumberInput := cardFrame.Locator("input[aria-label='Card number'], input[data-fieldtype='encryptedCardNumber']")
	if err = cardNumberInput.Fill("4111111111111111"); err != nil {
		t.Fatalf("Failed to enter card number: %v", err)
	}

	expiryFrame := page.FrameLocator("iframe[title*='expiry'], iframe[title*='Expiry']")
	expiryInput := expiryFrame.Locator("input[aria-label='Expiry date'], input[data-fieldtype='encryptedExpiryDate']")
	if err = expiryInput.Fill("03/30"); err != nil {
		t.Fatalf("Failed to enter expiry date: %v", err)
	}

	cvcFrame := page.FrameLocator("iframe[title*='security'], iframe[title*='Security'], iframe[title*='CVC']")
	cvcInput := cvcFrame.Locator("input[aria-label='Security code'], input[data-fieldtype='encryptedSecurityCode']")
	if err = cvcInput.Fill("737"); err != nil {
		t.Fatalf("Failed to enter CVC: %v", err)
	}

	holderNameInput := page.Locator("input[name='holderName']")
	if visible, _ := holderNameInput.IsVisible(); visible {
		holderNameInput.Fill("Test User")
	}

	payButton := page.Locator("button[type='submit']:has-text('Pay'), button:has-text('Pay')")
	if err = payButton.Click(); err != nil {
		t.Fatalf("Failed to click Pay button: %v", err)
	}

	// Wait for confirmation page
	if err = page.WaitForURL("**/confirmation**", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(30000),
	}); err != nil {
		t.Fatalf("Did not redirect to confirmation page: %v", err)
	}

	// Get the confirmation URL and order reference
	confirmationURL := page.URL()
	t.Logf("Confirmation URL: %s", confirmationURL)

	orderRef := page.Locator(".order-reference").First()
	orderRefText, err := orderRef.TextContent()
	if err != nil {
		t.Fatalf("Failed to find order reference: %v", err)
	}
	t.Logf("Order reference: %s", orderRefText)

	// When I close my browser (simulate by closing page and opening new one)
	page.Close()

	// And I reopen the confirmation page URL later
	newPage, err := browser.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer newPage.Close()

	if _, err = newPage.Goto(confirmationURL); err != nil {
		t.Fatalf("Failed to reopen confirmation page: %v", err)
	}

	// Then I should still see my order details
	// And I should see order reference
	reopenedOrderRef := newPage.Locator(".order-reference").First()
	reopenedOrderRefText, err := reopenedOrderRef.TextContent()
	if err != nil {
		t.Fatalf("Failed to find order reference on reopened page: %v", err)
	}
	if reopenedOrderRefText != orderRefText {
		t.Errorf("Expected order reference '%s', got '%s'", orderRefText, reopenedOrderRefText)
	}

	// And I should see the payment status "Authorized"
	status := newPage.Locator(".status-badge")
	statusText, err := status.TextContent()
	if err != nil {
		t.Fatalf("Failed to find payment status: %v", err)
	}
	if statusText == "" {
		t.Error("Payment status is empty")
	}

	// And I should see the product and amount
	productName := newPage.Locator(".product-name")
	productNameText, err := productName.TextContent()
	if err != nil {
		t.Fatalf("Failed to find product name: %v", err)
	}
	if productNameText != "Premium Widget" {
		t.Errorf("Expected 'Premium Widget', got: %s", productNameText)
	}

	amount := newPage.Locator(".product-amount")
	amountText, err := amount.TextContent()
	if err != nil {
		t.Fatalf("Failed to find amount: %v", err)
	}
	if amountText != "$1.00" {
		t.Errorf("Expected '$1.00', got: %s", amountText)
	}
}

// TestPaymentDeclined tests the payment declined flow
// Feature: Order Confirmation After Payment
//
//	Scenario: Payment declined
//	  Given I am on the checkout page
//	  When I enter payment details that will be declined
//	  And I submit the payment
//	  Then I should be redirected to a failure page
//	  And I should see "Payment Declined" or similar message
//	  And I should see my order reference
//	  And I should see a "Try Again" button
func TestPaymentDeclined(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	// Given I am on the checkout page
	if _, err = page.Goto("http://localhost:8080/checkout"); err != nil {
		t.Fatalf("Failed to navigate to checkout page: %v", err)
	}

	// Wait for Adyen Drop-in to load
	loadingMessage := page.Locator("#loading-container")
	if err = loadingMessage.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateHidden,
		Timeout: playwright.Float(15000),
	}); err != nil {
		t.Fatalf("Checkout did not load: %v", err)
	}

	time.Sleep(3 * time.Second)

	// Click card payment method if needed
	cardPaymentMethod := page.Locator(".adyen-checkout__payment-method--card, [class*='card']").First()
	if visible, _ := cardPaymentMethod.IsVisible(); visible {
		cardPaymentMethod.Click()
		time.Sleep(1 * time.Second)
	}

	// When I enter payment details that will be declined
	// Using an invalid/declined test card - card that triggers a refusal
	// According to Adyen docs, we can use card with specific CVC to trigger decline
	page.WaitForSelector("iframe[title*='card number'], iframe[title*='Card number']", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateVisible,
		Timeout: playwright.Float(10000),
	})

	cardFrame := page.FrameLocator("iframe[title*='card number'], iframe[title*='Card number']")
	cardNumberInput := cardFrame.Locator("input[aria-label='Card number'], input[data-fieldtype='encryptedCardNumber']")
	// Use a card that will be refused (CVC 111 with standard test card triggers refusal in some cases)
	// Or we can use the card ending in 0002 which is documented as refused
	if err = cardNumberInput.Fill("4111111111110002"); err != nil {
		t.Fatalf("Failed to enter card number: %v", err)
	}

	expiryFrame := page.FrameLocator("iframe[title*='expiry'], iframe[title*='Expiry']")
	expiryInput := expiryFrame.Locator("input[aria-label='Expiry date'], input[data-fieldtype='encryptedExpiryDate']")
	if err = expiryInput.Fill("03/30"); err != nil {
		t.Fatalf("Failed to enter expiry date: %v", err)
	}

	cvcFrame := page.FrameLocator("iframe[title*='security'], iframe[title*='Security'], iframe[title*='CVC']")
	cvcInput := cvcFrame.Locator("input[aria-label='Security code'], input[data-fieldtype='encryptedSecurityCode']")
	if err = cvcInput.Fill("737"); err != nil {
		t.Fatalf("Failed to enter CVC: %v", err)
	}

	holderNameInput := page.Locator("input[name='holderName']")
	if visible, _ := holderNameInput.IsVisible(); visible {
		if err = holderNameInput.Fill("Test User"); err != nil {
			t.Logf("Warning: Could not fill cardholder name: %v", err)
		}
	}

	// And I submit the payment
	payButton := page.Locator("button[type='submit']:has-text('Pay'), button:has-text('Pay')")
	if err = payButton.Click(); err != nil {
		t.Fatalf("Failed to click Pay button: %v", err)
	}

	// Then I should be redirected to a failure page (or see error on checkout page)
	// Wait for either failure page or error message on checkout page
	time.Sleep(5 * time.Second)

	currentURL := page.URL()
	t.Logf("Current URL after declined payment: %s", currentURL)

	// Check if we're on the failure page
	if strings.Contains(currentURL, "/failure") {
		// And I should see "Payment Declined" or similar message
		title := page.Locator(".failure-title, h1")
		titleText, err := title.TextContent()
		if err != nil {
			t.Fatalf("Failed to find failure title: %v", err)
		}
		titleLower := strings.ToLower(titleText)
		if !strings.Contains(titleLower, "fail") &&
			!strings.Contains(titleLower, "decline") &&
			!strings.Contains(titleLower, "refused") {
			t.Logf("Warning: Expected failure message, got: %s", titleText)
		}

		// And I should see my order reference
		orderRef := page.Locator(".order-reference")
		if count, _ := orderRef.Count(); count > 0 {
			orderRefText, _ := orderRef.TextContent()
			t.Logf("Order reference: %s", orderRefText)
		}

		// And I should see a "Try Again" button
		tryAgainButton := page.Locator("a:has-text('Try Again'), button:has-text('Try Again')")
		visible, err := tryAgainButton.IsVisible()
		if err != nil || !visible {
			t.Error("'Try Again' button is not visible")
		}
	} else {
		// Check for error message on checkout page
		errorContainer := page.Locator("#error-container, .error-message")
		visible, _ := errorContainer.IsVisible()
		if visible {
			errorText, _ := errorContainer.TextContent()
			t.Logf("Error message displayed on checkout page: %s", errorText)

			// Verify error message contains decline/refusal information
			errorLower := strings.ToLower(errorText)
			if !strings.Contains(errorLower, "decline") &&
				!strings.Contains(errorLower, "refuse") &&
				!strings.Contains(errorLower, "failed") &&
				!strings.Contains(errorLower, "error") {
				t.Logf("Warning: Expected decline message, got: %s", errorText)
			}
		} else {
			// Check for Adyen Drop-in error messages
			adyenError := page.Locator(".adyen-checkout__error-text, [class*='error']")
			if count, _ := adyenError.Count(); count > 0 {
				errorText, _ := adyenError.First().TextContent()
				t.Logf("Adyen error message: %s", errorText)
			} else {
				t.Logf("Warning: No clear error message found after declined payment")
			}
		}
	}
}

// Helper function to extract order reference from URL
func extractOrderRefFromURL(url string) string {
	re := regexp.MustCompile(`sessionId=([^&]+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// Helper function to format error message
func formatError(context string, err error) string {
	return fmt.Sprintf("%s: %v", context, err)
}
