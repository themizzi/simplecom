package e2e

import (
	"testing"

	"github.com/playwright-community/playwright-go"
)

var (
	pw      *playwright.Playwright
	browser playwright.Browser
)

// TestMain sets up and tears down the Playwright browser for all tests
func TestMain(m *testing.M) {
	var err error

	// Start Playwright (browsers already installed via: go run github.com/playwright-community/playwright-go/cmd/playwright@latest install chromium)
	pw, err = playwright.Run()
	if err != nil {
		panic(err)
	}
	defer pw.Stop()

	// Launch browser in headless mode
	browser, err = pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		panic(err)
	}
	defer browser.Close()

	// Run tests
	m.Run()
}
