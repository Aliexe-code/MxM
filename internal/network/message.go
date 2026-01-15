package network

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	ProtocolVersion = 1
	MaxMessageSize  = 10 * 1024 * 1024 // 10MB
	HeaderSize      = 10 // 1 byte version + 1 byte type + 4 bytes length + 4 bytes checksum
)

// MessageType represents different types of network messages
type MessageType uint8

const (
	MessageTypePing MessageType = iota
	MessageTypePong
	MessageTypeGetBlocks
	MessageTypeBlocks
	MessageTypeNewBlock
	MessageTypeGetPeers
	MessageTypePeers
	MessageTypeTransaction
	MessageTypeGetBlockchain
	MessageTypeBlockchain
	MessageTypeUnknown
)

func (mt MessageType) String() string {
	switch mt {
	case MessageTypePing:
		return "PING"
	case MessageTypePong:
		return "PONG"
	case MessageTypeGetBlocks:
		return "GET_BLOCKS"
	case MessageTypeBlocks:
		return "BLOCKS"
	case MessageTypeNewBlock:
		return "NEW_BLOCK"
	case MessageTypeGetPeers:
		return "GET_PEERS"
	case MessageTypePeers:
		return "PEERS"
	case MessageTypeTransaction:
		return "TRANSACTION"
	case MessageTypeGetBlockchain:
		return "GET_BLOCKCHAIN"
	case MessageTypeBlockchain:
		return "BLOCKCHAIN"
	default:
		return "UNKNOWN"
	}
}

// Message represents a network message
type Message struct {
	Version    uint8      `json:"version"`
	Type       MessageType `json:"type"`
	Length     uint32     `json:"length"`
	Payload    []byte     `json:"payload"`
	Checksum   uint32     `json:"checksum"`
}

// NewMessage creates a new message with the given type and payload
func NewMessage(msgType MessageType, payload []byte) *Message {
	checksum := calculateChecksum(payload)
	return &Message{
		Version:  ProtocolVersion,
		Type:     msgType,
		Length:   uint32(len(payload)),
		Payload:  payload,
		Checksum: checksum,
	}
}

// Serialize converts the message to bytes for transmission
func (m *Message) Serialize() ([]byte, error) {
	if m.Length > MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds maximum %d", m.Length, MaxMessageSize)
	}

	buf := new(bytes.Buffer)

	// Write header
	if err := binary.Write(buf, binary.BigEndian, m.Version); err != nil {
		return nil, fmt.Errorf("failed to write version: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, uint8(m.Type)); err != nil {
		return nil, fmt.Errorf("failed to write message type: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, m.Length); err != nil {
		return nil, fmt.Errorf("failed to write length: %w", err)
	}
	if err := binary.Write(buf, binary.BigEndian, m.Checksum); err != nil {
		return nil, fmt.Errorf("failed to write checksum: %w", err)
	}

	// Write payload
	if _, err := buf.Write(m.Payload); err != nil {
		return nil, fmt.Errorf("failed to write payload: %w", err)
	}

	return buf.Bytes(), nil
}

// Deserialize converts bytes to a message
func Deserialize(data []byte) (*Message, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("data too short: got %d, need at least %d", len(data), HeaderSize)
	}

	buf := bytes.NewReader(data)

	m := &Message{}

	// Read header
	if err := binary.Read(buf, binary.BigEndian, &m.Version); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}
	if m.Version != ProtocolVersion {
		return nil, fmt.Errorf("unsupported protocol version: %d", m.Version)
	}

	var msgType uint8
	if err := binary.Read(buf, binary.BigEndian, &msgType); err != nil {
		return nil, fmt.Errorf("failed to read message type: %w", err)
	}
	m.Type = MessageType(msgType)

	if err := binary.Read(buf, binary.BigEndian, &m.Length); err != nil {
		return nil, fmt.Errorf("failed to read length: %w", err)
	}

	if err := binary.Read(buf, binary.BigEndian, &m.Checksum); err != nil {
		return nil, fmt.Errorf("failed to read checksum: %w", err)
	}

	// Validate length
	if m.Length > MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds maximum %d", m.Length, MaxMessageSize)
	}

	// Read payload
	if uint32(len(data)) < HeaderSize+m.Length {
		return nil, fmt.Errorf("data too short for payload: got %d, need %d", len(data), HeaderSize+m.Length)
	}

	if m.Length > 0 {
		m.Payload = make([]byte, m.Length)
		if _, err := buf.Read(m.Payload); err != nil {
			return nil, fmt.Errorf("failed to read payload: %w", err)
		}
	} else {
		m.Payload = []byte{}
	}

	// Verify checksum
	calculatedChecksum := calculateChecksum(m.Payload)
	if calculatedChecksum != m.Checksum {
		return nil, fmt.Errorf("checksum mismatch: expected %d, got %d", m.Checksum, calculatedChecksum)
	}

	return m, nil
}

// calculateChecksum computes a simple checksum for data integrity
func calculateChecksum(data []byte) uint32 {
	var checksum uint32
	for _, b := range data {
		checksum = checksum*31 + uint32(b)
	}
	return checksum
}

// Validate checks if the message is valid
func (m *Message) Validate() error {
	if m.Version != ProtocolVersion {
		return fmt.Errorf("invalid protocol version: %d", m.Version)
	}
	if m.Type == MessageTypeUnknown {
		return errors.New("unknown message type")
	}
	if m.Length > MaxMessageSize {
		return fmt.Errorf("message size %d exceeds maximum %d", m.Length, MaxMessageSize)
	}
	if uint32(len(m.Payload)) != m.Length {
		return fmt.Errorf("payload length mismatch: expected %d, got %d", m.Length, len(m.Payload))
	}

	calculatedChecksum := calculateChecksum(m.Payload)
	if calculatedChecksum != m.Checksum {
		return fmt.Errorf("checksum mismatch: expected %d, got %d", m.Checksum, calculatedChecksum)
	}

	return nil
}

// ToJSON converts the message to JSON (for debugging/logging)
func (m *Message) ToJSON() (string, error) {
	data, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message to JSON: %w", err)
	}
	return string(data), nil
}

// Helper functions to create specific message types

// NewPingMessage creates a new ping message
func NewPingMessage() *Message {
	return NewMessage(MessageTypePing, nil)
}

// NewPongMessage creates a new pong message
func NewPongMessage() *Message {
	return NewMessage(MessageTypePong, nil)
}

// NewGetBlocksMessage creates a new get blocks message
func NewGetBlocksMessage(startIndex, count uint32) (*Message, error) {
	payload := make([]byte, 8)
	binary.BigEndian.PutUint32(payload[0:4], startIndex)
	binary.BigEndian.PutUint32(payload[4:8], count)
	return NewMessage(MessageTypeGetBlocks, payload), nil
}

// ParseGetBlocksMessage parses a get blocks message
func ParseGetBlocksMessage(msg *Message) (startIndex, count uint32, err error) {
	if msg.Type != MessageTypeGetBlocks {
		return 0, 0, fmt.Errorf("not a get blocks message: %s", msg.Type)
	}
	if len(msg.Payload) != 8 {
		return 0, 0, fmt.Errorf("invalid payload length: %d", len(msg.Payload))
	}
	startIndex = binary.BigEndian.Uint32(msg.Payload[0:4])
	count = binary.BigEndian.Uint32(msg.Payload[4:8])
	return startIndex, count, nil
}

// NewGetPeersMessage creates a new get peers message
func NewGetPeersMessage() *Message {
	return NewMessage(MessageTypeGetPeers, nil)
}

// NewPeersMessage creates a new peers message
func NewPeersMessage(peers []PeerInfo) (*Message, error) {
	payload, err := json.Marshal(peers)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal peers: %w", err)
	}
	return NewMessage(MessageTypePeers, payload), nil
}

// ParsePeersMessage parses a peers message
func ParsePeersMessage(msg *Message) ([]PeerInfo, error) {
	if msg.Type != MessageTypePeers {
		return nil, fmt.Errorf("not a peers message: %s", msg.Type)
	}
	var peers []PeerInfo
	if err := json.Unmarshal(msg.Payload, &peers); err != nil {
		return nil, fmt.Errorf("failed to unmarshal peers: %w", err)
	}
	return peers, nil
}