package dataplane

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/config"
	"github.com/gvquiroz/cell-routing-from-scratch/internal/protocol"
)

func TestClientConnectsToControlPlane(t *testing.T) {
	upgrader := websocket.Upgrader{}
	connected := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("Failed to upgrade: %v", err)
			return
		}
		defer conn.Close()
		connected <- true
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	loader := config.NewLoader("test-config.json", 5*time.Second)
	client := NewClient(wsURL, loader)
	client.Start()
	defer client.Stop()

	select {
	case <-connected:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Client did not connect within timeout")
	}
}

func TestClientReceivesConfigSnapshot(t *testing.T) {
	upgrader := websocket.Upgrader{}
	receivedAck := make(chan bool, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		snapshot := protocol.ConfigSnapshotMessage{
			Type:    protocol.MessageTypeConfigSnapshot,
			Version: "1.0.0",
			RoutingTable: map[string]string{
				"acme": "tier1",
			},
			CellEndpoints: map[string]string{
				"tier1": "http://localhost:9001",
			},
			DefaultPlacement: "tier3",
		}

		data, _ := json.Marshal(snapshot)
		if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
			return
		}

		_, msgBytes, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var ack protocol.AckMessage
		if err := json.Unmarshal(msgBytes, &ack); err == nil && ack.Type == protocol.MessageTypeAck {
			receivedAck <- true
		}
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	loader := config.NewLoader("test-config.json", 5*time.Second)
	loader.LoadInitial()
	client := NewClient(wsURL, loader)
	client.Start()
	defer client.Stop()

	select {
	case <-receivedAck:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Did not receive ACK from client")
	}
}

func TestClientReconnectsAfterDisconnection(t *testing.T) {
	upgrader := websocket.Upgrader{}
	connectionCount := 0
	connectionChan := make(chan int, 2)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		connectionCount++
		connectionChan <- connectionCount

		if connectionCount == 1 {
			conn.Close()
			return
		}

		defer conn.Close()
		time.Sleep(200 * time.Millisecond)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	loader := config.NewLoader("test-config.json", 5*time.Second)
	client := NewClient(wsURL, loader)
	client.Start()
	defer client.Stop()

	select {
	case count := <-connectionChan:
		if count != 1 {
			t.Fatalf("Expected first connection, got count %d", count)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("First connection timeout")
	}

	select {
	case count := <-connectionChan:
		if count != 2 {
			t.Fatalf("Expected reconnection, got count %d", count)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Reconnection timeout")
	}
}
