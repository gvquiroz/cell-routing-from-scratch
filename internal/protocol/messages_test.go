package protocol

import (
	"encoding/json"
	"testing"
)

func TestConfigSnapshotSerialization(t *testing.T) {
	msg := ConfigSnapshotMessage{
		Type:    MessageTypeConfigSnapshot,
		Version: "1.0.0",
		RoutingTable: map[string]string{
			"acme": "tier1",
			"visa": "visa",
		},
		CellEndpoints: map[string]string{
			"tier1": "http://localhost:9001",
			"visa":  "http://localhost:9004",
		},
		DefaultPlacement: "tier3",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded ConfigSnapshotMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != MessageTypeConfigSnapshot {
		t.Errorf("Type = %v, want %v", decoded.Type, MessageTypeConfigSnapshot)
	}
	if decoded.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", decoded.Version)
	}
	if len(decoded.RoutingTable) != 2 {
		t.Errorf("RoutingTable length = %d, want 2", len(decoded.RoutingTable))
	}
}

func TestAckMessageSerialization(t *testing.T) {
	msg := AckMessage{
		Type:    MessageTypeAck,
		Version: "1.0.0",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded AckMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != MessageTypeAck {
		t.Errorf("Type = %v, want %v", decoded.Type, MessageTypeAck)
	}
}

func TestNackMessageSerialization(t *testing.T) {
	msg := NackMessage{
		Type:    MessageTypeNack,
		Version: "1.0.0",
		Error:   "validation failed",
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded NackMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.Type != MessageTypeNack {
		t.Errorf("Type = %v, want %v", decoded.Type, MessageTypeNack)
	}
	if decoded.Error != "validation failed" {
		t.Errorf("Error = %v, want 'validation failed'", decoded.Error)
	}
}

func TestMessageTypeDeserialization(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		msgType MessageType
	}{
		{
			name:    "config_snapshot",
			input:   `{"type":"config_snapshot","version":"1.0.0"}`,
			msgType: MessageTypeConfigSnapshot,
		},
		{
			name:    "ack",
			input:   `{"type":"ack","version":"1.0.0"}`,
			msgType: MessageTypeAck,
		},
		{
			name:    "nack",
			input:   `{"type":"nack","version":"1.0.0","error":"error"}`,
			msgType: MessageTypeNack,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg Message
			if err := json.Unmarshal([]byte(tt.input), &msg); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			if msg.Type != tt.msgType {
				t.Errorf("Type = %v, want %v", msg.Type, tt.msgType)
			}
		})
	}
}
