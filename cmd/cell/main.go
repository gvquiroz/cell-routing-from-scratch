package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	cellName := os.Getenv("CELL_NAME")
	if cellName == "" {
		cellName = "unknown-cell"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "9000"
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		headers := extractHeaders(r)

		// Log incoming request
		logEntry := map[string]interface{}{
			"timestamp":   time.Now().UTC().Format(time.RFC3339),
			"cell":        cellName,
			"method":      r.Method,
			"path":        r.URL.Path,
			"query":       r.URL.RawQuery,
			"remote_addr": r.RemoteAddr,
		}

		// Add interesting headers to log
		if requestID := r.Header.Get("X-Request-Id"); requestID != "" {
			logEntry["request_id"] = requestID
		}
		if routingKey := r.Header.Get("X-Routing-Key"); routingKey != "" {
			logEntry["routing_key"] = routingKey
		}
		if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			logEntry["forwarded_for"] = forwardedFor
		}

		logData, _ := json.Marshal(logEntry)
		log.Println(string(logData))

		response := map[string]interface{}{
			"cell":      cellName,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"method":    r.Method,
			"path":      r.URL.Path,
			"query":     r.URL.RawQuery,
			"headers":   headers,
			"message":   fmt.Sprintf("Hello from %s!", cellName),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cell-Name", cellName)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// Log health check request
		logEntry := map[string]interface{}{
			"timestamp": time.Now().UTC().Format(time.RFC3339),
			"cell":      cellName,
			"method":    r.Method,
			"path":      "/health",
		}
		logData, _ := json.Marshal(logEntry)
		log.Println(string(logData))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "healthy",
			"cell":   cellName,
		})
	})

	log.Printf("Cell '%s' starting on port %s", cellName, port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Cell server failed: %v", err)
	}
}

func extractHeaders(r *http.Request) map[string]string {
	headers := make(map[string]string)

	// Include commonly forwarded headers
	interestingHeaders := []string{
		"X-Request-Id",
		"X-Routing-Key",
		"X-Forwarded-For",
		"X-Forwarded-Proto",
		"User-Agent",
		"Content-Type",
	}

	for _, key := range interestingHeaders {
		if value := r.Header.Get(key); value != "" {
			headers[key] = value
		}
	}

	return headers
}
