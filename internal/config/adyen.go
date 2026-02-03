package config

import (
	"fmt"
	"os"
)

// AdyenConfig holds configuration for Adyen integration
type AdyenConfig struct {
	APIKey          string
	ClientKey       string
	MerchantAccount string
	Environment     string
}

// LoadAdyenConfig loads Adyen configuration from environment variables
func LoadAdyenConfig() (*AdyenConfig, error) {
	config := AdyenConfig{
		APIKey:          os.Getenv("ADYEN_API_KEY"),
		ClientKey:       os.Getenv("ADYEN_CLIENT_KEY"),
		MerchantAccount: os.Getenv("ADYEN_MERCHANT_ACCOUNT"),
		Environment:     os.Getenv("ADYEN_ENVIRONMENT"),
	}

	// Validate required fields
	if config.APIKey == "" {
		return nil, fmt.Errorf("ADYEN_API_KEY is required")
	}
	if config.ClientKey == "" {
		return nil, fmt.Errorf("ADYEN_CLIENT_KEY is required")
	}
	if config.MerchantAccount == "" {
		return nil, fmt.Errorf("ADYEN_MERCHANT_ACCOUNT is required")
	}
	if config.Environment == "" {
		config.Environment = "TEST" // Default to TEST environment
	}

	return &config, nil
}
