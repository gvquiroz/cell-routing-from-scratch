package logging

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

// Logger provides structured JSON logging
type Logger struct {
	logger *log.Logger
}

// NewLogger creates a new structured logger
func NewLogger() *Logger {
	return &Logger{
		logger: log.New(os.Stdout, "", 0),
	}
}

// RequestLog contains fields for logging HTTP requests
type RequestLog struct {
	Timestamp    string  `json:"timestamp"`
	RequestID    string  `json:"request_id"`
	Method       string  `json:"method"`
	Path         string  `json:"path"`
	RoutingKey   string  `json:"routing_key,omitempty"`
	PlacementKey string  `json:"placement_key"`
	RouteReason  string  `json:"route_reason"`
	UpstreamURL  string  `json:"upstream_url"`
	StatusCode   int     `json:"status_code"`
	DurationMs   float64 `json:"duration_ms"`
}

// LogRequest logs a completed request
func (l *Logger) LogRequest(req RequestLog) {
	req.Timestamp = time.Now().UTC().Format(time.RFC3339)

	data, err := json.Marshal(req)
	if err != nil {
		l.logger.Printf("error marshaling log: %v", err)
		return
	}

	l.logger.Println(string(data))
}

// LogError logs an error message
func (l *Logger) LogError(msg string, err error, fields map[string]interface{}) {
	logData := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     "error",
		"message":   msg,
	}

	if err != nil {
		logData["error"] = err.Error()
	}

	for k, v := range fields {
		logData[k] = v
	}

	data, _ := json.Marshal(logData)
	l.logger.Println(string(data))
}

// LogInfo logs an informational message
func (l *Logger) LogInfo(msg string, fields map[string]interface{}) {
	logData := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"level":     "info",
		"message":   msg,
	}

	for k, v := range fields {
		logData[k] = v
	}

	data, _ := json.Marshal(logData)
	l.logger.Println(string(data))
}
