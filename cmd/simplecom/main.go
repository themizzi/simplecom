package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	internalcli "github.com/adyen/ecommerce/internal/cli"
	"github.com/adyen/ecommerce/internal/config"
	"github.com/adyen/ecommerce/internal/database"
	"github.com/adyen/ecommerce/internal/handlers"
	"github.com/adyen/ecommerce/internal/repository"
	"github.com/adyen/ecommerce/internal/services"
	"github.com/joho/godotenv"
	"github.com/urfave/cli/v2"
)

var version = "0.1.0"

// buildServerDependencies creates all dependencies needed for the server
func buildServerDependencies() (internalcli.ServerDependencies, error) {
	var deps internalcli.ServerDependencies

	// Create order repository
	deps.OrderRepo = repository.NewOrderRepository()

	// Load server configuration
	deps.ServerConfig = config.LoadServerConfig()

	// Load Adyen configuration
	adyenConfig, err := config.LoadAdyenConfig()
	if err != nil {
		return deps, fmt.Errorf("missing required Adyen configuration: %w", err)
	}
	deps.AdyenConfig = adyenConfig

	// Create service layer
	adyenClient := services.NewAdyenClient(adyenConfig)
	orderService := services.NewOrderService(deps.OrderRepo)
	paymentService := services.NewPaymentService(adyenClient, orderService, adyenConfig)

	// Create product
	deps.Product = handlers.Product{
		Name:        "Premium Widget",
		Description: "A high-quality widget perfect for all your widget needs. Durable, reliable, and designed to last.",
		Price:       "$1.00",
		ImageURL:    "/static/images/widget-placeholder.svg",
	}

	// Create product handler with injected product
	productHandler, err := handlers.NewProductHandler("templates/product.html", deps.Product)
	if err != nil {
		return deps, fmt.Errorf("failed to create product handler: %w", err)
	}
	deps.ProductHandler = productHandler

	// Create checkout handler
	checkoutHandler, err := handlers.NewCheckoutHandler("templates/checkout.html", deps.Product, deps.AdyenConfig)
	if err != nil {
		return deps, fmt.Errorf("failed to create checkout handler: %w", err)
	}
	deps.CheckoutHandler = checkoutHandler

	// Create session API handler with payment service
	deps.SessionHandler = handlers.NewSessionHandler(paymentService, deps.Product)

	// Create confirmation handler with payment service
	confirmationHandler, err := handlers.NewConfirmationHandler("templates/confirmation.html", paymentService)
	if err != nil {
		return deps, fmt.Errorf("failed to create confirmation handler: %w", err)
	}
	deps.ConfirmationHandler = confirmationHandler

	// Create failure handler
	failureHandler, err := handlers.NewFailureHandler("templates/failure.html", deps.OrderRepo)
	if err != nil {
		return deps, fmt.Errorf("failed to create failure handler: %w", err)
	}
	deps.FailureHandler = failureHandler

	return deps, nil
}

// ServeCommand returns the serve command
func ServeCommand(db *sql.DB) *cli.Command {
	return &cli.Command{
		Name:  "serve",
		Usage: "Start the e-commerce web server",
		Action: func(c *cli.Context) error {
			// Connect to database
			if err := database.Connect(); err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer database.Close()
			log.Println("Connected to database successfully")

			// Run database migrations
			if err := database.RunMigrations(); err != nil {
				return fmt.Errorf("failed to run database migrations: %w", err)
			}

			// Build all server dependencies
			deps, err := buildServerDependencies()
			if err != nil {
				return err
			}

			return internalcli.RunServe(deps)
		},
	}
}

func main() {
	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found, using environment variables")
	}

	app := &cli.App{
		Name:    "simplecom",
		Usage:   "E-commerce application management tool",
		Version: version,
		Commands: []*cli.Command{
			ServeCommand(nil),
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		log.Fatal(err)
	}
}
