package config

import "os"

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port string
}

// LoadServerConfig loads server configuration from environment variables
func LoadServerConfig() ServerConfig {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default to port 8080
	}

	return ServerConfig{
		Port: port,
	}
}
