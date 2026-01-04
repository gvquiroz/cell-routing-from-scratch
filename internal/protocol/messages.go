package protocol

// MessageType identifies the type of WebSocket message
type MessageType string

const (
	// MessageTypeConfigSnapshot is sent from CP to DP with full config
	MessageTypeConfigSnapshot MessageType = "config_snapshot"
	// MessageTypeAck is sent from DP to CP when config is applied
	MessageTypeAck MessageType = "ack"
	// MessageTypeNack is sent from DP to CP when config is rejected
	MessageTypeNack MessageType = "nack"
)

// Message is the base WebSocket message structure
type Message struct {
	Type    MessageType `json:"type"`
	Version string      `json:"version"`
}

// ConfigSnapshotMessage contains a full routing configuration
type ConfigSnapshotMessage struct {
	Type             MessageType       `json:"type"`
	Version          string            `json:"version"`
	RoutingTable     map[string]string `json:"routingTable"`
	CellEndpoints    map[string]string `json:"cellEndpoints"`
	DefaultPlacement string            `json:"defaultPlacement"`
}

// AckMessage acknowledges successful config application
type AckMessage struct {
	Type    MessageType `json:"type"`
	Version string      `json:"version"`
}

// NackMessage reports config rejection
type NackMessage struct {
	Type    MessageType `json:"type"`
	Version string      `json:"version"`
	Error   string      `json:"error"`
}
