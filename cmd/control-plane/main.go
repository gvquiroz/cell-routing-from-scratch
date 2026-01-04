package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/config"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/controlplane"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for demo
	},
}

func main() {
	// Load configuration
	configPath := getEnv("CONFIG_PATH", "config/routing.json")
	configLoader := config.NewLoader(configPath, 5*time.Second)

	if err := configLoader.LoadInitial(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Start config reload loop
	configLoader.StartReloadLoop()
	defer configLoader.Stop()

	// Create control plane server
	cpServer := controlplane.NewServer(configLoader)

	// Watch for config changes and broadcast
	go cpServer.WatchConfigChanges()

	// WebSocket endpoint for data planes to connect
	http.HandleFunc("/connect", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("Failed to upgrade connection: %v", err)
			return
		}
		cpServer.HandleConnection(conn)
	})

	// Health endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	})

	port := getEnv("PORT", "8081")
	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Control plane starting on port %s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down control plane...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Control plane forced to shutdown: %v", err)
	}

	log.Println("Control plane stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
