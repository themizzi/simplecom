package e2e

import (
	"testing"
)

// TestProductDisplay tests the product display feature
// Feature: Product Display
//
//	As a customer
//	I want to view the widget product
//	So that I can decide to purchase it
func TestProductDisplay(t *testing.T) {
	// Scenario: View product page
	//   Given I am on the homepage
	//   Then I should see the widget product
	//   And I should see the price "$1.00"
	//   And I should see a "Buy Now" button

	page, err := browser.NewPage()
	if err != nil {
		t.Fatal(err)
	}
	defer page.Close()

	// Given I am on the homepage
	if _, err = page.Goto("http://localhost:8080/"); err != nil {
		t.Fatalf("Failed to navigate to homepage: %v", err)
	}

	// Then I should see the widget product
	widgetTitle, err := page.Locator(".product-name").TextContent()
	if err != nil {
		t.Fatalf("Failed to find widget title: %v", err)
	}
	if widgetTitle != "Premium Widget" {
		t.Errorf("Expected widget title 'Premium Widget', got '%s'", widgetTitle)
	}

	// And I should see the price "$1.00"
	price, err := page.Locator(".product-price").TextContent()
	if err != nil {
		t.Fatalf("Failed to find price: %v", err)
	}
	if price != "$1.00" {
		t.Errorf("Expected price '$1.00', got '%s'", price)
	}

	// And I should see a "Buy Now" button
	buyButton := page.Locator("button:has-text('Buy Now')")
	visible, err := buyButton.IsVisible()
	if err != nil {
		t.Fatalf("Failed to check if Buy Now button is visible: %v", err)
	}
	if !visible {
		t.Error("Buy Now button is not visible")
	}
}
