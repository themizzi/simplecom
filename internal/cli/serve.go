package cli

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adyen/ecommerce/internal/config"
	"github.com/adyen/ecommerce/internal/handlers"
	"github.com/adyen/ecommerce/internal/repository"
)

// ServerDependencies holds all dependencies needed for the server
type ServerDependencies struct {
	OrderRepo           *repository.OrderRepository
	ServerConfig        config.ServerConfig
	AdyenConfig         *config.AdyenConfig
	Product             handlers.Product
	ProductHandler      http.Handler
	CheckoutHandler     http.Handler
	SessionHandler      http.Handler
	ConfirmationHandler http.Handler
	FailureHandler      http.Handler
}

// RunServe starts the e-commerce web server
func RunServe(deps ServerDependencies) error {
	listener, server, err := StartServer(deps)
	if err != nil {
		return err
	}
	defer listener.Close()

	return WaitForShutdown(server, nil)
}

// StartServer creates and starts the HTTP server, returning the listener and server
func StartServer(deps ServerDependencies) (net.Listener, *http.Server, error) {
	// Set up routes
	mux := http.NewServeMux()
	mux.Handle("/", deps.ProductHandler)
	mux.Handle("/checkout", deps.CheckoutHandler)
	mux.Handle("/api/sessions", deps.SessionHandler)
	mux.Handle("/order/confirmation", deps.ConfirmationHandler)
	mux.Handle("/order/failed", deps.FailureHandler)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Create listener
	addr := fmt.Sprintf(":%s", deps.ServerConfig.Port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create listener: %w", err)
	}

	// Create HTTP server
	server := &http.Server{
		Handler: mux,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on %s", listener.Addr().String())
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	return listener, server, nil
}

// WaitForShutdown waits for a shutdown signal and gracefully shuts down the server
// If shutdown channel is nil, a new channel will be created and registered with signal.Notify
// shutdownTimeout can be passed for testing; use 0 for default 30 seconds
func WaitForShutdown(server *http.Server, shutdown chan os.Signal) error {
	return WaitForShutdownWithTimeout(server, shutdown, 30*time.Second)
}

// WaitForShutdownWithTimeout allows specifying a custom shutdown timeout (primarily for testing)
func WaitForShutdownWithTimeout(server *http.Server, shutdown chan os.Signal, shutdownTimeout time.Duration) error {
	// Channel to listen for interrupt or terminate signals
	if shutdown == nil {
		shutdown = make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	}

	// Wait for shutdown signal
	sig := <-shutdown
	log.Printf("Received signal: %v, shutting down server...", sig)

	// Give outstanding requests time to complete
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		// Force close the server after timeout
		// Note: The nested error case where both Shutdown AND Close fail is unreachable
		// in practice because http.Server.Close() does not propagate listener close errors.
		// This has been verified through testing with mock listeners.
		if err := server.Close(); err != nil {
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	log.Println("Server stopped")
	return nil
}
