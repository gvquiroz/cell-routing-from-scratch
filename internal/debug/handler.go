package debug

import (
	"encoding/json"
	"net/http"
	"time"
)

// ConfigProvider provides access to config metadata
type ConfigProvider interface {
	GetConfigVersion() string
	GetConfigSource() interface{}
	LastReloadTime() time.Time
}

// Handler provides debug endpoints
type Handler struct {
	configProvider ConfigProvider
}

// NewHandler creates a new debug handler
func NewHandler(configProvider ConfigProvider) *Handler {
	return &Handler{
		configProvider: configProvider,
	}
}

// ServeHTTP handles /debug/config requests
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	version := h.configProvider.GetConfigVersion()
	source := h.configProvider.GetConfigSource()
	lastReload := h.configProvider.LastReloadTime()

	response := map[string]interface{}{
		"version":        version,
		"source":         source,
		"last_reload_at": lastReload.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
