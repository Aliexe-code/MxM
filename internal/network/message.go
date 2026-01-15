package network

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"sync"
	"time"
)

const (
	ProtocolVersion = 1
	MaxMessageSize  = 10 * 1024 * 1024 // 10MB
	HeaderSize      = 10               // 1 byte version + 1 byte type + 4 bytes length + 4 bytes checksum

	// Message authentication constants
	MaxNodeIDLength    = 256
	MaxSignatureLength = 512

	// Fragmentation constants
	MaxFragmentSize    = 64 * 1024 // 64KB per fragment
	FragmentHeaderSize = 10        // 4 bytes fragment ID + 4 bytes total fragments + 2 bytes fragment index
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
	Version   uint8       `json:"version"`
	Type      MessageType `json:"type"`
	Length    uint32      `json:"length"`
	Payload   []byte      `json:"payload"`
	Checksum  uint32      `json:"checksum"`
	Signature []byte      `json:"signature"` // Message signature for authentication
	NodeID    string      `json:"node_id"`   // Node identifier for tracking

	// Fragmentation fields
	FragmentID     uint32 `json:"fragment_id"`     // Unique ID for fragmented message
	TotalFragments uint32 `json:"total_fragments"` // Total number of fragments
	FragmentIndex  uint16 `json:"fragment_index"`  // Current fragment index
	IsFragment     bool   `json:"is_fragment"`     // Whether this is a fragment
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

// NewAuthenticatedMessage creates a new message with authentication
func NewAuthenticatedMessage(msgType MessageType, payload []byte, signature []byte, nodeID string) *Message {
	msg := NewMessage(msgType, payload)

	// Validate node ID length
	if len(nodeID) > MaxNodeIDLength {
		nodeID = nodeID[:MaxNodeIDLength]
	}

	// Validate signature length
	if len(signature) > MaxSignatureLength {
		signature = signature[:MaxSignatureLength]
	}

	msg.Signature = signature
	msg.NodeID = nodeID
	return msg
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

	// Write signature length and signature
	sigLen := uint16(len(m.Signature))
	if err := binary.Write(buf, binary.BigEndian, sigLen); err != nil {
		return nil, fmt.Errorf("failed to write signature length: %w", err)
	}
	if _, err := buf.Write(m.Signature); err != nil {
		return nil, fmt.Errorf("failed to write signature: %w", err)
	}

	// Write node ID length and node ID
	nodeIDLen := uint16(len(m.NodeID))
	if err := binary.Write(buf, binary.BigEndian, nodeIDLen); err != nil {
		return nil, fmt.Errorf("failed to write node ID length: %w", err)
	}
	if _, err := buf.WriteString(m.NodeID); err != nil {
		return nil, fmt.Errorf("failed to write node ID: %w", err)
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

	// Read signature length and signature
	var sigLen uint16
	if err := binary.Read(buf, binary.BigEndian, &sigLen); err != nil {
		return nil, fmt.Errorf("failed to read signature length: %w", err)
	}
	if sigLen > MaxSignatureLength {
		return nil, fmt.Errorf("signature length %d exceeds maximum %d", sigLen, MaxSignatureLength)
	}
	if sigLen > 0 {
		m.Signature = make([]byte, sigLen)
		if _, err := buf.Read(m.Signature); err != nil {
			return nil, fmt.Errorf("failed to read signature: %w", err)
		}
	}

	// Read node ID length and node ID
	var nodeIDLen uint16
	if err := binary.Read(buf, binary.BigEndian, &nodeIDLen); err != nil {
		return nil, fmt.Errorf("failed to read node ID length: %w", err)
	}
	if nodeIDLen > MaxNodeIDLength {
		return nil, fmt.Errorf("node ID length %d exceeds maximum %d", nodeIDLen, MaxNodeIDLength)
	}
	if nodeIDLen > 0 {
		nodeIDBytes := make([]byte, nodeIDLen)
		if _, err := buf.Read(nodeIDBytes); err != nil {
			return nil, fmt.Errorf("failed to read node ID: %w", err)
		}
		m.NodeID = string(nodeIDBytes)
	}

	// Read payload
	if uint32(len(data)) < HeaderSize+2+uint32(sigLen)+2+uint32(nodeIDLen)+m.Length {
		return nil, fmt.Errorf("data too short for payload: got %d, need %d", len(data), HeaderSize+2+uint32(sigLen)+2+uint32(nodeIDLen)+m.Length)
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

// calculateChecksum computes a CRC32 checksum for data integrity
func calculateChecksum(data []byte) uint32 {
	return crc32.ChecksumIEEE(data)
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

// VerifySignature verifies the message signature using a public key
// This is a placeholder - actual implementation would depend on the crypto library used
func (m *Message) VerifySignature(publicKey []byte) bool {
	// Placeholder for signature verification
	// In a real implementation, this would:
	// 1. Create a message to sign (version + type + payload)
	// 2. Verify the signature using the public key
	// 3. Return true if signature is valid, false otherwise

	// For now, we'll just check if a signature exists
	return len(m.Signature) > 0
}

// Sign signs the message using a private key
// This is a placeholder - actual implementation would depend on the crypto library used
func (m *Message) Sign(privateKey []byte) error {
	// Placeholder for message signing
	// In a real implementation, this would:
	// 1. Create a message to sign (version + type + payload)
	// 2. Sign the message using the private key
	// 3. Store the signature in m.Signature

	// For now, we'll just create a dummy signature
	m.Signature = []byte("signed")
	return nil
}

// IsAuthenticated checks if the message has authentication
func (m *Message) IsAuthenticated() bool {
	return len(m.Signature) > 0 && m.NodeID != ""
}

// Fragment splits a large message into smaller fragments
func (m *Message) Fragment() ([]*Message, error) {
	if len(m.Payload) <= MaxFragmentSize {
		return []*Message{m}, nil
	}

	totalFragments := uint32((len(m.Payload) + MaxFragmentSize - 1) / MaxFragmentSize)
	fragments := make([]*Message, totalFragments)

	for i := uint32(0); i < totalFragments; i++ {
		start := int(i * MaxFragmentSize)
		end := start + MaxFragmentSize
		if end > len(m.Payload) {
			end = len(m.Payload)
		}

		fragment := &Message{
			Version:        m.Version,
			Type:           m.Type,
			Length:         uint32(end - start),
			Payload:        m.Payload[start:end],
			Checksum:       calculateChecksum(m.Payload[start:end]),
			Signature:      m.Signature,
			NodeID:         m.NodeID,
			FragmentID:     i,
			TotalFragments: totalFragments,
			FragmentIndex:  uint16(i),
			IsFragment:     true,
		}
		fragments[i] = fragment
	}

	return fragments, nil
}

// DefragmentMap manages fragmented message reassembly
type DefragmentMap struct {
	fragments map[uint32]map[uint16]*Message // fragmentID -> fragmentIndex -> message
	mu        sync.RWMutex
}

// NewDefragmentMap creates a new defragmentation map
func NewDefragmentMap() *DefragmentMap {
	return &DefragmentMap{
		fragments: make(map[uint32]map[uint16]*Message),
	}
}

// AddFragment adds a fragment to the map
func (dm *DefragmentMap) AddFragment(msg *Message) bool {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if !msg.IsFragment {
		return false
	}

	if _, exists := dm.fragments[msg.FragmentID]; !exists {
		dm.fragments[msg.FragmentID] = make(map[uint16]*Message)
	}

	dm.fragments[msg.FragmentID][msg.FragmentIndex] = msg

	// Check if all fragments are received
	return len(dm.fragments[msg.FragmentID]) == int(msg.TotalFragments)
}

// Reassemble reassembles a fragmented message
func (dm *DefragmentMap) Reassemble(fragmentID uint32) (*Message, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	fragments, exists := dm.fragments[fragmentID]
	if !exists {
		return nil, fmt.Errorf("fragment ID %d not found", fragmentID)
	}

	if len(fragments) == 0 {
		return nil, fmt.Errorf("no fragments for ID %d", fragmentID)
	}

	// Get first fragment to get message metadata
	firstFragment := fragments[0]
	if len(fragments) != int(firstFragment.TotalFragments) {
		return nil, fmt.Errorf("incomplete fragments: got %d, need %d", len(fragments), firstFragment.TotalFragments)
	}

	// Reassemble payload
	totalSize := 0
	for _, frag := range fragments {
		totalSize += len(frag.Payload)
	}

	payload := make([]byte, totalSize)
	offset := 0
	for i := uint32(0); i < firstFragment.TotalFragments; i++ {
		frag, exists := fragments[uint16(i)]
		if !exists {
			return nil, fmt.Errorf("missing fragment %d", i)
		}
		copy(payload[offset:], frag.Payload)
		offset += len(frag.Payload)
	}

	// Create reassembled message
	reassembled := &Message{
		Version:    firstFragment.Version,
		Type:       firstFragment.Type,
		Length:     uint32(len(payload)),
		Payload:    payload,
		Checksum:   calculateChecksum(payload),
		Signature:  firstFragment.Signature,
		NodeID:     firstFragment.NodeID,
		IsFragment: false,
	}

	// Clean up fragments
	delete(dm.fragments, fragmentID)

	return reassembled, nil
}

// CleanupOldFragments removes fragments older than the specified duration
func (dm *DefragmentMap) CleanupOldFragments(maxAge time.Duration) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	// In a real implementation, you'd add timestamp tracking to fragments
	// For now, this is a placeholder
	return
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
