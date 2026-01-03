package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gvquiroz/cell-routing-from-scratch/internal/logging"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/proxy"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/routing"
)

func main() {
	// Initialize logger
	logger := logging.NewLogger()

	// Define routing mappings (immutable after initialization)
	routingTable := map[string]string{
		"acme":    "tier1",
		"globex":  "tier2",
		"initech": "tier3",
		"visa":    "visa",
	}

	cellEndpoints := map[string]string{
		"tier1": "http://cell-tier1:9001",
		"tier2": "http://cell-tier2:9002",
		"tier3": "http://cell-tier3:9003",
		"visa":  "http://cell-visa:9004",
	}

	defaultPlacement := "tier3"

	// Create router
	router := routing.NewRouter(routingTable, cellEndpoints, defaultPlacement)

	// Create proxy handler
	handler := proxy.NewHandler(router, logger)

	// Configure HTTP server
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting cell router on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
