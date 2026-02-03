package e2e

import (
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

// TestCheckoutNavigationFromProductPage tests navigation from product to checkout
// Feature: Adyen Drop-in Checkout Display
//
//	Scenario: Navigate to checkout from product page
//	  Given I am viewing the Premium Widget product page
//	  When I click the "Buy Now" button
//	  Then I should be redirected to the checkout page
//	  And I should see the product name "Premium Widget"
//	  And I should see the price "$1.00"
func TestCheckoutNavigationFromProductPage(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	// Given I am viewing the Premium Widget product page
	if _, err = page.Goto("http://localhost:8080/"); err != nil {
		t.Fatalf("Failed to navigate to product page: %v", err)
	}

	// When I click the "Buy Now" button
	buyButton := page.Locator("button:has-text('Buy Now')")
	if err = buyButton.Click(); err != nil {
		t.Fatalf("Failed to click Buy Now button: %v", err)
	}

	// Then I should be redirected to the checkout page
	if err = page.WaitForURL("**/checkout", playwright.PageWaitForURLOptions{
		Timeout: playwright.Float(5000),
	}); err != nil {
		t.Fatalf("Did not redirect to checkout page: %v", err)
	}

	// And I should see the product name "Premium Widget"
	productName, err := page.Locator(".order-item-details h3").TextContent()
	if err != nil {
		t.Fatalf("Failed to find product name: %v", err)
	}
	if productName != "Premium Widget" {
		t.Errorf("Expected product name 'Premium Widget', got '%s'", productName)
	}

	// And I should see the price "$1.00"
	price, err := page.Locator(".order-item-price").TextContent()
	if err != nil {
		t.Fatalf("Failed to find price: %v", err)
	}
	if price != "$1.00" {
		t.Errorf("Expected price '$1.00', got '%s'", price)
	}
}

// TestCheckoutAdyenDropinComponent tests the Adyen Drop-in component initialization
// Feature: Adyen Drop-in Checkout Display
//
//	Scenario: Display Adyen Drop-in component
//	  Given I am on the checkout page
//	  When the page loads
//	  Then a payment session should be created via the backend
//	  And the Adyen Drop-in component should be initialized
//	  And I should see payment method options
//	  And I should see a card payment form
func TestCheckoutAdyenDropinComponent(t *testing.T) {
	page, err := browser.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	// Given I am on the checkout page
	if _, err = page.Goto("http://localhost:8080/checkout"); err != nil {
		t.Fatalf("Failed to navigate to checkout page: %v", err)
	}

	// When the page loads
	// Wait for the loading message to disappear (indicates session was created)
	loadingMessage := page.Locator("#loading-container")
	if err = loadingMessage.WaitFor(playwright.LocatorWaitForOptions{
		State:   playwright.WaitForSelectorStateHidden,
		Timeout: playwright.Float(10000),
	}); err != nil {
		t.Fatalf("Loading message did not disappear: %v", err)
	}

	// Then a payment session should be created via the backend
	// (verified by loading message disappearing)

	// And the Adyen Drop-in component should be initialized
	dropinContainer := page.Locator("#dropin-container")

	// Wait a bit for the drop-in to fully render
	time.Sleep(2 * time.Second)

	// And I should see payment method options
	// The Adyen Drop-in component renders payment methods
	dropinContent, err := dropinContainer.Locator(".adyen-checkout__payment-method").Count()
	if err != nil {
		t.Fatalf("Failed to check for payment methods: %v", err)
	}
	if dropinContent == 0 {
		t.Error("No payment methods found in drop-in component")
	}

	// And I should see a card payment form
	// Check for card payment option
	cardOption := page.Locator("[data-cse='card'], .adyen-checkout__payment-method--card, .adyen-checkout__payment-method:has-text('Card')")
	visible, err := cardOption.First().IsVisible()
	if err != nil {
		t.Logf("Warning: Could not verify card payment form visibility: %v", err)
	} else if !visible {
		t.Error("Card payment option is not visible")
	}
}
