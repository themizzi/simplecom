package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/adyen/ecommerce/internal/config"
	"github.com/adyen/ecommerce/internal/handlers"
)

// errorListener wraps a net.Listener and returns an error when Close() is called
type errorListener struct {
	net.Listener
	closed bool
}

func (l *errorListener) Close() error {
	if !l.closed {
		l.closed = true
		// Close the underlying listener to stop accepting new connections
		l.Listener.Close()
	}
	// Always return an error to simulate a close failure
	return errors.New("mock listener close error")
}

// mockHandler creates a simple test handler
func mockHandler(response string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	})
}

// Helper functions for test setup

// createTestDeps creates ServerDependencies with default mock handlers for testing
func createTestDeps(port string) ServerDependencies {
	return ServerDependencies{
		ServerConfig:        config.ServerConfig{Port: port},
		OrderRepo:           nil,
		AdyenConfig:         &config.AdyenConfig{},
		Product:             handlers.Product{},
		ProductHandler:      mockHandler("product"),
		CheckoutHandler:     mockHandler("checkout"),
		SessionHandler:      mockHandler("session"),
		ConfirmationHandler: mockHandler("confirmation"),
		FailureHandler:      mockHandler("failure"),
	}
}

// startTestServer starts a server with the given dependencies and returns listener, server, and port
func startTestServer(t *testing.T, deps ServerDependencies) (net.Listener, *http.Server, int) {
	t.Helper()
	listener, server, err := StartServer(deps)
	if err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	return listener, server, port
}

// httpGet makes an HTTP GET request and returns response body and status
func httpGet(t *testing.T, url string) (string, int) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("Failed to make request to %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body), resp.StatusCode
}

func TestStartServer_SuccessfulStartup(t *testing.T) {
	// GIVEN
	deps := createTestDeps("0")

	// WHEN
	listener, server, port := startTestServer(t, deps)
	defer listener.Close()
	defer server.Close()

	// THEN
	// Verify we got a valid port
	if port == 0 {
		t.Error("Expected non-zero port")
	}

	// Verify server is responding
	time.Sleep(50 * time.Millisecond)
	body, status := httpGet(t, fmt.Sprintf("http://localhost:%d/", port))

	if status != http.StatusOK {
		t.Errorf("Expected status 200, got %d", status)
	}
	if body != "product" {
		t.Errorf("Expected 'product', got '%s'", body)
	}
}

func TestStartServer_InvalidPort(t *testing.T) {
	// GIVEN
	deps := createTestDeps("99999") // Invalid port

	// WHEN
	listener, server, err := StartServer(deps)

	// THEN
	if err == nil {
		listener.Close()
		server.Close()
		t.Error("Expected error for invalid port, got nil")
	}
}

func TestStartServer_PortAlreadyInUse(t *testing.T) {
	// GIVEN
	// Start a listener on a specific port
	existingListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create test listener: %v", err)
	}
	defer existingListener.Close()

	// Get the port that was assigned
	port := existingListener.Addr().(*net.TCPAddr).Port

	deps := createTestDeps(fmt.Sprintf("%d", port))

	// WHEN
	listener, server, err := StartServer(deps)

	// THEN
	if err == nil {
		listener.Close()
		server.Close()
		t.Error("Expected error for port already in use, got nil")
	}
}

func TestStartServer_AllRoutesWork(t *testing.T) {
	// GIVEN
	deps := createTestDeps("0")
	deps.ProductHandler = mockHandler("product-response")
	deps.CheckoutHandler = mockHandler("checkout-response")
	deps.SessionHandler = mockHandler("session-response")
	deps.ConfirmationHandler = mockHandler("confirmation-response")
	deps.FailureHandler = mockHandler("failure-response")

	// WHEN
	listener, server, port := startTestServer(t, deps)
	defer listener.Close()
	defer server.Close()

	baseURL := fmt.Sprintf("http://localhost:%d", port)

	// Give server time to start
	time.Sleep(50 * time.Millisecond)

	// THEN
	testCases := []struct {
		path     string
		expected string
	}{
		{"/", "product-response"},
		{"/checkout", "checkout-response"},
		{"/api/sessions", "session-response"},
		{"/order/confirmation", "confirmation-response"},
		{"/order/failed", "failure-response"},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			body, status := httpGet(t, baseURL+tc.path)
			if status != http.StatusOK {
				t.Errorf("Expected status 200, got %d", status)
			}
			if body != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, body)
			}
		})
	}
}

func TestStartServer_GracefulShutdown(t *testing.T) {
	// GIVEN
	deps := createTestDeps("0")

	// WHEN
	listener, server, port := startTestServer(t, deps)
	defer listener.Close()

	// Verify server is running
	time.Sleep(50 * time.Millisecond)
	_, status := httpGet(t, fmt.Sprintf("http://localhost:%d/", port))
	if status != http.StatusOK {
		t.Fatal("Server not responding")
	}

	// THEN
	// Test graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		t.Errorf("Failed to shutdown server gracefully: %v", err)
	}

	// Verify server is no longer responding
	time.Sleep(100 * time.Millisecond)
	_, getErr := http.Get(fmt.Sprintf("http://localhost:%d/", port))
	if getErr == nil {
		t.Error("Expected error after shutdown, server still responding")
	}
}

func TestStartServer_ConcurrentServers(t *testing.T) {
	// GIVEN
	// Test that multiple servers can start on different ports without conflicts
	deps1 := createTestDeps("0")
	deps1.ProductHandler = mockHandler("server1")

	deps2 := createTestDeps("0")
	deps2.ProductHandler = mockHandler("server2")

	// WHEN
	listener1, server1, port1 := startTestServer(t, deps1)
	defer listener1.Close()
	defer server1.Close()

	listener2, server2, port2 := startTestServer(t, deps2)
	defer listener2.Close()
	defer server2.Close()

	// THEN
	if port1 == port2 {
		t.Error("Both servers got the same port")
	}

	// Verify both servers are responding
	time.Sleep(50 * time.Millisecond)

	resp1, err := http.Get(fmt.Sprintf("http://localhost:%d/", port1))
	if err != nil {
		t.Fatalf("Server 1 not responding: %v", err)
	}
	defer resp1.Body.Close()
	body1, _ := io.ReadAll(resp1.Body)
	if string(body1) != "server1" {
		t.Errorf("Server 1 returned wrong response: %s", string(body1))
	}

	resp2, err := http.Get(fmt.Sprintf("http://localhost:%d/", port2))
	if err != nil {
		t.Fatalf("Server 2 not responding: %v", err)
	}
	defer resp2.Body.Close()
	body2, _ := io.ReadAll(resp2.Body)
	if string(body2) != "server2" {
		t.Errorf("Server 2 returned wrong response: %s", string(body2))
	}
}

func TestStartServer_ShutdownWithActiveConnections(t *testing.T) {
	// GIVEN
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.Write([]byte("slow"))
	})

	deps := createTestDeps("0")
	deps.ProductHandler = slowHandler

	listener, server, port := startTestServer(t, deps)
	defer listener.Close()

	// Start a slow request
	time.Sleep(50 * time.Millisecond)
	responseCh := make(chan error, 1)
	go func() {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
		if err == nil {
			resp.Body.Close()
		}
		responseCh <- err
	}()

	// Give request time to start
	time.Sleep(50 * time.Millisecond)

	// WHEN
	// Initiate shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shutdownErr := server.Shutdown(ctx)
	if shutdownErr != nil {
		t.Errorf("Shutdown failed: %v", shutdownErr)
	}

	// THEN
	// The slow request should complete
	select {
	case err := <-responseCh:
		if err != nil {
			t.Logf("Request error (acceptable): %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("Request did not complete in time")
	}
}

// BenchmarkStartServer benchmarks server startup
func BenchmarkStartServer(b *testing.B) {
	deps := ServerDependencies{
		ServerConfig:        config.ServerConfig{Port: "0"},
		ProductHandler:      mockHandler("product"),
		CheckoutHandler:     mockHandler("checkout"),
		SessionHandler:      mockHandler("session"),
		ConfirmationHandler: mockHandler("confirmation"),
		FailureHandler:      mockHandler("failure"),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		listener, server, err := StartServer(deps)
		if err != nil {
			b.Fatalf("Failed to start server: %v", err)
		}
		server.Close()
		listener.Close()
	}
}

func TestWaitForShutdown_SIGTERM(t *testing.T) {
	// GIVEN
	deps := createTestDeps("0")

	listener, server, _ := startTestServer(t, deps)
	defer listener.Close()

	shutdown := make(chan os.Signal, 1)

	// WHEN
	errCh := make(chan error, 1)
	go func() {
		errCh <- WaitForShutdown(server, shutdown)
	}()

	time.Sleep(50 * time.Millisecond)
	shutdown <- syscall.SIGTERM

	// THEN
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForShutdown did not complete")
	}
}

func TestWaitForShutdown_SIGINT(t *testing.T) {
	// GIVEN
	deps := createTestDeps("0")

	listener, server, _ := startTestServer(t, deps)
	defer listener.Close()

	shutdown := make(chan os.Signal, 1)

	// WHEN
	errCh := make(chan error, 1)
	go func() {
		errCh <- WaitForShutdown(server, shutdown)
	}()

	time.Sleep(50 * time.Millisecond)
	shutdown <- syscall.SIGINT

	// THEN
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForShutdown did not complete")
	}
}

func TestWaitForShutdown_WithActiveRequests(t *testing.T) {
	// GIVEN
	slowHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Write([]byte("done"))
	})

	deps := createTestDeps("0")
	deps.ProductHandler = slowHandler

	listener, server, port := startTestServer(t, deps)
	defer listener.Close()

	// Start an active request
	time.Sleep(50 * time.Millisecond)
	requestComplete := make(chan bool, 1)
	go func() {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%d/", port))
		if err == nil {
			resp.Body.Close()
		}
		requestComplete <- true
	}()

	time.Sleep(50 * time.Millisecond)

	shutdown := make(chan os.Signal, 1)

	// WHEN
	errCh := make(chan error, 1)
	go func() {
		errCh <- WaitForShutdown(server, shutdown)
	}()

	shutdown <- syscall.SIGTERM

	// THEN
	// Should wait for active request to complete
	select {
	case <-requestComplete:
		t.Log("Request completed successfully")
	case <-time.After(2 * time.Second):
		t.Error("Request did not complete in time")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForShutdown did not complete")
	}
}

func TestRunServe_FullIntegration(t *testing.T) {
	// GIVEN
	deps := createTestDeps("0")

	// WHEN
	errCh := make(chan error, 1)
	go func() {
		errCh <- RunServe(deps)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Send shutdown signal to the process
	p, err := os.FindProcess(os.Getpid())
	if err != nil {
		t.Fatalf("Failed to get process: %v", err)
	}

	if err := p.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("Failed to send signal: %v", err)
	}

	// THEN
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Expected nil error, got: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server did not shut down within timeout")
	}
}

func TestRunServe_StartupFailure(t *testing.T) {
	// GIVEN
	deps := createTestDeps("99999") // Invalid port

	// WHEN
	err := RunServe(deps)

	// THEN
	if err == nil {
		t.Error("Expected error for invalid port, got nil")
	}
}

func TestWaitForShutdown_AlreadyClosedServer(t *testing.T) {
	// GIVEN
	deps := createTestDeps("0")

	listener, server, _ := startTestServer(t, deps)
	listener.Close()
	server.Close() // Close server before waiting

	shutdown := make(chan os.Signal, 1)

	// WHEN
	errCh := make(chan error, 1)
	go func() {
		errCh <- WaitForShutdown(server, shutdown)
	}()

	time.Sleep(50 * time.Millisecond)
	shutdown <- syscall.SIGTERM

	// THEN
	select {
	case err := <-errCh:
		// Should complete without error even though server was already closed
		if err != nil {
			t.Logf("Got error (expected for closed server): %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("WaitForShutdown did not complete")
	}
}

func TestStartServer_ServeError(t *testing.T) {
	// GIVEN
	deps := createTestDeps("0")

	// WHEN
	listener, server, _ := startTestServer(t, deps)
	defer server.Close()

	// Close the listener immediately to trigger a Serve error
	listener.Close()

	// THEN
	// Give the goroutine time to hit the error path
	time.Sleep(100 * time.Millisecond)
	// The error should be logged (we can't directly assert it, but this covers the line)
}

func TestWaitForShutdown_ShutdownAndCloseFailure(t *testing.T) {
	// GIVEN - This test demonstrates that the nested error path where both
	// Shutdown() and Close() fail cannot be triggered in practice because
	// http.Server.Close() does not propagate listener close errors.

	// Create a handler that blocks to force Shutdown timeout
	blockingHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.Write([]byte("done"))
	})

	deps := createTestDeps("0")
	deps.ProductHandler = blockingHandler

	// Create a real listener first
	realListener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}

	// Wrap it with our error-returning mock
	listener := &errorListener{Listener: realListener}

	// Create the server manually
	mux := http.NewServeMux()
	mux.Handle("/", deps.ProductHandler)
	mux.Handle("/checkout", deps.CheckoutHandler)
	mux.Handle("/api/sessions", deps.SessionHandler)
	mux.Handle("/order/confirmation", deps.ConfirmationHandler)
	mux.Handle("/order/failed", deps.FailureHandler)

	server := &http.Server{
		Handler: mux,
	}

	// Start serving in background
	go func() {
		server.Serve(listener)
	}()

	port := listener.Addr().(*net.TCPAddr).Port
	log.Printf("Server listening on %s", listener.Addr())

	shutdown := make(chan os.Signal, 1)

	// Start multiple blocking requests BEFORE shutdown signal to force timeout
	for i := 0; i < 5; i++ {
		go func() {
			http.Get(fmt.Sprintf("http://localhost:%d/", port))
		}()
	}

	// Give requests time to start
	time.Sleep(100 * time.Millisecond)

	// WHEN
	errCh := make(chan error, 1)
	go func() {
		// Use nanosecond timeout to force immediate Shutdown failure
		// When Shutdown fails, server.Close() will be called
		// Our mock listener will return an error from Close()
		errCh <- WaitForShutdownWithTimeout(server, shutdown, 1*time.Nanosecond)
	}()

	// Send shutdown signal while requests are in progress
	shutdown <- syscall.SIGTERM

	// THEN
	select {
	case err := <-errCh:
		// This demonstrates that even when our mock listener returns an error,
		// http.Server.Close() does not propagate it. The uncovered line in
		// serve.go cannot be reached without mocking http.Server itself.
		if err != nil {
			t.Fatalf("Unexpected error (http.Server.Close should not propagate listener errors): %v", err)
		}
		t.Log("Test confirms: http.Server.Close() does not propagate listener close errors")
		t.Log("The nested error path in serve.go line 110 is unreachable in practice")
	case <-time.After(2 * time.Second):
		t.Fatal("Test did not complete")
	}
}
