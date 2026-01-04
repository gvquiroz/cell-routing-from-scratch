package dataplane

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/config"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/protocol"
)

// Client connects to the control plane and receives config updates.
type Client struct {
	cpURL     string
	loader    *config.Loader
	conn      *websocket.Conn
	mu        sync.Mutex
	stopCh    chan struct{}
	done      chan struct{}
	reconnect bool
}

// NewClient creates a new data plane WebSocket client.
func NewClient(cpURL string, loader *config.Loader) *Client {
	return &Client{
		cpURL:     cpURL,
		loader:    loader,
		stopCh:    make(chan struct{}),
		done:      make(chan struct{}),
		reconnect: true,
	}
}

// Start begins connecting to the control plane.
func (c *Client) Start() {
	go c.connectionLoop()
}

// Stop gracefully stops the client.
func (c *Client) Stop() {
	c.mu.Lock()
	c.reconnect = false
	if c.conn != nil {
		c.conn.Close()
	}
	c.mu.Unlock()

	close(c.stopCh)
	<-c.done
}

// connectionLoop manages connection and reconnection logic with exponential backoff.
func (c *Client) connectionLoop() {
	defer close(c.done)

	backoff := 1 * time.Second
	maxBackoff := 60 * time.Second

	for {
		select {
		case <-c.stopCh:
			return
		default:
		}

		c.mu.Lock()
		shouldReconnect := c.reconnect
		c.mu.Unlock()

		if !shouldReconnect {
			return
		}

		if err := c.connect(); err != nil {
			log.Printf("[DP] Failed to connect to control plane: %v. Retrying in %v", err, backoff)
			time.Sleep(backoff)
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Connected successfully - reset backoff
		backoff = 1 * time.Second
		log.Printf("[DP] Connected to control plane at %s", c.cpURL)

		// Handle messages until connection fails
		c.handleMessages()

		log.Printf("[DP] Connection to control plane lost")
	}
}

// connect establishes WebSocket connection to the control plane.
func (c *Client) connect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, _, err := websocket.DefaultDialer.Dial(c.cpURL, nil)
	if err != nil {
		return err
	}

	c.conn = conn
	return nil
}

// handleMessages reads and processes messages from the control plane.
func (c *Client) handleMessages() {
	for {
		c.mu.Lock()
		conn := c.conn
		c.mu.Unlock()

		if conn == nil {
			return
		}

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var msg protocol.Message
		if err := json.Unmarshal(msgBytes, &msg); err != nil {
			log.Printf("[DP] Failed to unmarshal message: %v", err)
			continue
		}

		switch msg.Type {
		case protocol.MessageTypeConfigSnapshot:
			c.handleConfigSnapshot(msgBytes)
		default:
			log.Printf("[DP] Unknown message type: %s", msg.Type)
		}
	}
}

// handleConfigSnapshot processes a config snapshot from the control plane.
func (c *Client) handleConfigSnapshot(msgBytes []byte) {
	var snapshot protocol.ConfigSnapshotMessage
	if err := json.Unmarshal(msgBytes, &snapshot); err != nil {
		log.Printf("[DP] Failed to unmarshal config snapshot: %v", err)
		c.sendNack(err.Error())
		return
	}

	log.Printf("[DP] Received config snapshot version %s", snapshot.Version)

	// Validate and apply config atomically
	cfg := &config.Config{
		Version:          snapshot.Version,
		RoutingTable:     snapshot.RoutingTable,
		CellEndpoints:    snapshot.CellEndpoints,
		DefaultPlacement: snapshot.DefaultPlacement,
	}

	if err := c.loader.ApplyConfig(cfg); err != nil {
		log.Printf("[DP] Failed to apply config: %v", err)
		c.sendNack(err.Error())
		return
	}

	log.Printf("[DP] Applied config snapshot version %s from control plane", snapshot.Version)
	c.sendAck()
}

// sendAck sends an acknowledgment to the control plane.
func (c *Client) sendAck() {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}

	msg := protocol.AckMessage{
		Type:    protocol.MessageTypeAck,
		Version: "",
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[DP] Failed to marshal ack: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		log.Printf("[DP] Failed to send ack: %v", err)
	}
}

// sendNack sends a negative acknowledgment to the control plane.
func (c *Client) sendNack(reason string) {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return
	}

	msg := protocol.NackMessage{
		Type:    protocol.MessageTypeNack,
		Version: "",
		Error:   reason,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		log.Printf("[DP] Failed to marshal nack: %v", err)
		return
	}

	if err := conn.WriteMessage(websocket.TextMessage, msgBytes); err != nil {
		log.Printf("[DP] Failed to send nack: %v", err)
	}
}
