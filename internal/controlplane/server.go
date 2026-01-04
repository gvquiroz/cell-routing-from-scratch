package controlplane

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/config"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/protocol"
)

// Server manages WebSocket connections to data plane instances
type Server struct {
	clients      map[*websocket.Conn]bool
	clientsMutex sync.RWMutex
	configLoader *config.Loader
}

// NewServer creates a new control plane server
func NewServer(configLoader *config.Loader) *Server {
	return &Server{
		clients:      make(map[*websocket.Conn]bool),
		configLoader: configLoader,
	}
}

// RegisterClient adds a new data plane connection
func (s *Server) RegisterClient(conn *websocket.Conn) {
	s.clientsMutex.Lock()
	s.clients[conn] = true
	s.clientsMutex.Unlock()
	log.Printf("Data plane connected (total clients: %d)", len(s.clients))
}

// UnregisterClient removes a disconnected data plane
func (s *Server) UnregisterClient(conn *websocket.Conn) {
	s.clientsMutex.Lock()
	delete(s.clients, conn)
	s.clientsMutex.Unlock()
	conn.Close()
	log.Printf("Data plane disconnected (total clients: %d)", len(s.clients))
}

// BroadcastConfig sends current config to all connected data planes
func (s *Server) BroadcastConfig() {
	cfg := s.configLoader.GetConfig()

	msg := protocol.ConfigSnapshotMessage{
		Type:             protocol.MessageTypeConfigSnapshot,
		Version:          cfg.Version,
		RoutingTable:     cfg.RoutingTable,
		CellEndpoints:    cfg.CellEndpoints,
		DefaultPlacement: cfg.DefaultPlacement,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal config: %v", err)
		return
	}

	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	for conn := range s.clients {
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("Failed to send config to client: %v", err)
		} else {
			log.Printf("Pushed config version %s to data plane", cfg.Version)
		}
	}
}

// HandleConnection manages a WebSocket connection from a data plane
func (s *Server) HandleConnection(conn *websocket.Conn) {
	s.RegisterClient(conn)
	defer s.UnregisterClient(conn)

	// Send initial config snapshot
	s.sendConfigToClient(conn)

	// Read acknowledgments from data plane
	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		if messageType == websocket.TextMessage {
			s.handleDataPlaneMessage(data)
		}
	}
}

// sendConfigToClient sends current config to a specific client
func (s *Server) sendConfigToClient(conn *websocket.Conn) {
	cfg := s.configLoader.GetConfig()

	msg := protocol.ConfigSnapshotMessage{
		Type:             protocol.MessageTypeConfigSnapshot,
		Version:          cfg.Version,
		RoutingTable:     cfg.RoutingTable,
		CellEndpoints:    cfg.CellEndpoints,
		DefaultPlacement: cfg.DefaultPlacement,
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("Failed to marshal config: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("Failed to send initial config: %v", err)
	} else {
		log.Printf("Sent initial config version %s to new data plane", cfg.Version)
	}
}

// handleDataPlaneMessage processes ack/nack messages from data plane
func (s *Server) handleDataPlaneMessage(data []byte) {
	var baseMsg protocol.Message
	if err := json.Unmarshal(data, &baseMsg); err != nil {
		log.Printf("Failed to parse data plane message: %v", err)
		return
	}

	switch baseMsg.Type {
	case protocol.MessageTypeAck:
		var ackMsg protocol.AckMessage
		if err := json.Unmarshal(data, &ackMsg); err == nil {
			log.Printf("Data plane acknowledged config version %s", ackMsg.Version)
		}
	case protocol.MessageTypeNack:
		var nackMsg protocol.NackMessage
		if err := json.Unmarshal(data, &nackMsg); err == nil {
			log.Printf("Data plane rejected config version %s: %s", nackMsg.Version, nackMsg.Error)
		}
	}
}

// WatchConfigChanges monitors config file and broadcasts updates
func (s *Server) WatchConfigChanges() {
	lastVersion := s.configLoader.GetConfig().Version
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		currentVersion := s.configLoader.GetConfig().Version
		if currentVersion != lastVersion {
			log.Printf("Config changed from %s to %s, broadcasting to data planes", lastVersion, currentVersion)
			s.BroadcastConfig()
			lastVersion = currentVersion
		}
	}
}
