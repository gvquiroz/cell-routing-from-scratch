package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gvquiroz/cell-routing-from-scratch/internal/config"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/dataplane"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/debug"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/logging"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/proxy"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/routing"
)

func main() {
	// Initialize logger
	logger := logging.NewLogger()

	// Determine config path based on mode
	cpURL := os.Getenv("CONTROL_PLANE_URL")
	var configPath string
	if cpURL != "" {
		// CP mode: use initial config only (will be replaced by CP)
		configPath = getEnv("CONFIG_PATH", "config/dataplane-initial.json")
	} else {
		// File-only mode: use main config with hot-reload
		configPath = getEnv("CONFIG_PATH", "config/routing.json")
	}

	configLoader := config.NewLoader(configPath, 5*time.Second)

	// Load initial config (fail fast if invalid)
	if err := configLoader.LoadInitial(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Connect to control plane if configured
	if cpURL != "" {
		// CP mode: only accept updates from control plane
		dpClient := dataplane.NewClient(cpURL, configLoader)
		dpClient.Start()
		defer dpClient.Stop()
		log.Printf("Connected to control plane at %s - config updates via CP only", cpURL)
	} else {
		// File-only mode: watch for file changes
		configLoader.StartReloadLoop()
		defer configLoader.Stop()
		log.Println("No control plane configured, using file-based config with hot-reload")
	}

	// Create router with config loader
	router := routing.NewRouter(configLoader)

	// Create proxy handler (pass config for resilience mechanisms)
	handler := proxy.NewHandler(router, configLoader.GetConfig(), logger)
	defer handler.Stop()

	// Create debug handler
	debugHandler := debug.NewHandler(configLoader)

	// Set up routing
	mux := http.NewServeMux()
	mux.Handle("/debug/config", debugHandler)
	mux.Handle("/", handler)

	// Configure HTTP server
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
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
