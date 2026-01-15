package network

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestNewMessage(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(MessageTypePing, payload)

	if msg.Version != ProtocolVersion {
		t.Errorf("Expected version %d, got %d", ProtocolVersion, msg.Version)
	}

	if msg.Type != MessageTypePing {
		t.Errorf("Expected type %d, got %d", MessageTypePing, msg.Type)
	}

	if msg.Length != uint32(len(payload)) {
		t.Errorf("Expected length %d, got %d", len(payload), msg.Length)
	}

	if !bytes.Equal(msg.Payload, payload) {
		t.Errorf("Payload mismatch")
	}

	if msg.Checksum == 0 {
		t.Error("Checksum should not be zero")
	}
}

func TestMessageSerialize(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(MessageTypePing, payload)

	data, err := msg.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize message: %v", err)
	}

	// Calculate expected length: HeaderSize (10) + signature length (2) + node ID length (2) + payload
	expectedLen := HeaderSize + 2 + 2 + len(payload)
	if len(data) != expectedLen {
		t.Errorf("Expected data length %d, got %d", expectedLen, len(data))
	}

	// Verify header
	buf := bytes.NewReader(data[:HeaderSize])
	var version uint8
	var msgType uint8
	var length uint32
	var checksum uint32

	if err := binary.Read(buf, binary.BigEndian, &version); err != nil {
		t.Fatalf("Failed to read version: %v", err)
	}

	if err := binary.Read(buf, binary.BigEndian, &msgType); err != nil {
		t.Fatalf("Failed to read message type: %v", err)
	}

	if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
		t.Fatalf("Failed to read length: %v", err)
	}

	if err := binary.Read(buf, binary.BigEndian, &checksum); err != nil {
		t.Fatalf("Failed to read checksum: %v", err)
	}

	if version != ProtocolVersion {
		t.Errorf("Expected version %d, got %d", ProtocolVersion, version)
	}

	if msgType != uint8(MessageTypePing) {
		t.Errorf("Expected type %d, got %d", MessageTypePing, msgType)
	}

	if length != uint32(len(payload)) {
		t.Errorf("Expected length %d, got %d", len(payload), length)
	}

	if checksum != msg.Checksum {
		t.Errorf("Expected checksum %d, got %d", msg.Checksum, checksum)
	}
}

func TestMessageDeserialize(t *testing.T) {
	payload := []byte("test payload")
	originalMsg := NewMessage(MessageTypePing, payload)

	data, err := originalMsg.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize message: %v", err)
	}

	msg, err := Deserialize(data)
	if err != nil {
		t.Fatalf("Failed to deserialize message: %v", err)
	}

	if msg.Version != originalMsg.Version {
		t.Errorf("Version mismatch")
	}

	if msg.Type != originalMsg.Type {
		t.Errorf("Type mismatch")
	}

	if msg.Length != originalMsg.Length {
		t.Errorf("Length mismatch")
	}

	if msg.Checksum != originalMsg.Checksum {
		t.Errorf("Checksum mismatch")
	}

	if !bytes.Equal(msg.Payload, originalMsg.Payload) {
		t.Errorf("Payload mismatch")
	}
}

func TestMessageDeserializeTooShort(t *testing.T) {
	data := []byte("too short")
	_, err := Deserialize(data)
	if err == nil {
		t.Error("Expected error for too short data")
	}
}

func TestMessageDeserializeInvalidVersion(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(MessageTypePing, payload)

	data, err := msg.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize message: %v", err)
	}

	// Corrupt version
	data[0] = 99

	_, err = Deserialize(data)
	if err == nil {
		t.Error("Expected error for invalid version")
	}
}

func TestMessageDeserializeInvalidChecksum(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(MessageTypePing, payload)

	data, err := msg.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize message: %v", err)
	}

	// Corrupt checksum
	data[8] = 99

	_, err = Deserialize(data)
	if err == nil {
		t.Error("Expected error for invalid checksum")
	}
}

func TestMessageValidate(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(MessageTypePing, payload)

	if err := msg.Validate(); err != nil {
		t.Errorf("Message validation failed: %v", err)
	}
}

func TestMessageValidateInvalidVersion(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(MessageTypePing, payload)
	msg.Version = 99

	if err := msg.Validate(); err == nil {
		t.Error("Expected error for invalid version")
	}
}

func TestMessageValidateUnknownType(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(MessageTypeUnknown, payload)

	if err := msg.Validate(); err == nil {
		t.Error("Expected error for unknown type")
	}
}

func TestMessageValidateTooLarge(t *testing.T) {
	payload := make([]byte, MaxMessageSize+1)
	msg := NewMessage(MessageTypePing, payload)

	if err := msg.Validate(); err == nil {
		t.Error("Expected error for too large message")
	}
}

func TestMessageValidateChecksumMismatch(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(MessageTypePing, payload)
	msg.Checksum = 999

	if err := msg.Validate(); err == nil {
		t.Error("Expected error for checksum mismatch")
	}
}

func TestMessageTypeString(t *testing.T) {
	tests := []struct {
		msgType  MessageType
		expected string
	}{
		{MessageTypePing, "PING"},
		{MessageTypePong, "PONG"},
		{MessageTypeGetBlocks, "GET_BLOCKS"},
		{MessageTypeBlocks, "BLOCKS"},
		{MessageTypeNewBlock, "NEW_BLOCK"},
		{MessageTypeGetPeers, "GET_PEERS"},
		{MessageTypePeers, "PEERS"},
		{MessageTypeTransaction, "TRANSACTION"},
		{MessageTypeGetBlockchain, "GET_BLOCKCHAIN"},
		{MessageTypeBlockchain, "BLOCKCHAIN"},
		{MessageTypeUnknown, "UNKNOWN"},
	}

	for _, tt := range tests {
		if tt.msgType.String() != tt.expected {
			t.Errorf("Expected %s, got %s", tt.expected, tt.msgType.String())
		}
	}
}

func TestNewPingMessage(t *testing.T) {
	msg := NewPingMessage()

	if msg.Type != MessageTypePing {
		t.Errorf("Expected type %d, got %d", MessageTypePing, msg.Type)
	}

	if len(msg.Payload) != 0 {
		t.Error("Ping message should have no payload")
	}
}

func TestNewPongMessage(t *testing.T) {
	msg := NewPongMessage()

	if msg.Type != MessageTypePong {
		t.Errorf("Expected type %d, got %d", MessageTypePong, msg.Type)
	}

	if len(msg.Payload) != 0 {
		t.Error("Pong message should have no payload")
	}
}

func TestNewGetBlocksMessage(t *testing.T) {
	startIndex := uint32(10)
	count := uint32(5)

	msg, err := NewGetBlocksMessage(startIndex, count)
	if err != nil {
		t.Fatalf("Failed to create get blocks message: %v", err)
	}

	if msg.Type != MessageTypeGetBlocks {
		t.Errorf("Expected type %d, got %d", MessageTypeGetBlocks, msg.Type)
	}

	if len(msg.Payload) != 8 {
		t.Errorf("Expected payload length 8, got %d", len(msg.Payload))
	}
}

func TestParseGetBlocksMessage(t *testing.T) {
	startIndex := uint32(10)
	count := uint32(5)

	msg, err := NewGetBlocksMessage(startIndex, count)
	if err != nil {
		t.Fatalf("Failed to create get blocks message: %v", err)
	}

	parsedStartIndex, parsedCount, err := ParseGetBlocksMessage(msg)
	if err != nil {
		t.Fatalf("Failed to parse get blocks message: %v", err)
	}

	if parsedStartIndex != startIndex {
		t.Errorf("Expected start index %d, got %d", startIndex, parsedStartIndex)
	}

	if parsedCount != count {
		t.Errorf("Expected count %d, got %d", count, parsedCount)
	}
}

func TestParseGetBlocksMessageInvalidType(t *testing.T) {
	msg := NewPingMessage()

	_, _, err := ParseGetBlocksMessage(msg)
	if err == nil {
		t.Error("Expected error for invalid message type")
	}
}

func TestNewGetPeersMessage(t *testing.T) {
	msg := NewGetPeersMessage()

	if msg.Type != MessageTypeGetPeers {
		t.Errorf("Expected type %d, got %d", MessageTypeGetPeers, msg.Type)
	}

	if len(msg.Payload) != 0 {
		t.Error("Get peers message should have no payload")
	}
}

func TestNewPeersMessage(t *testing.T) {
	peers := []PeerInfo{
		{ID: "peer1", Address: "127.0.0.1", Port: 8000},
		{ID: "peer2", Address: "127.0.0.1", Port: 8001},
	}

	msg, err := NewPeersMessage(peers)
	if err != nil {
		t.Fatalf("Failed to create peers message: %v", err)
	}

	if msg.Type != MessageTypePeers {
		t.Errorf("Expected type %d, got %d", MessageTypePeers, msg.Type)
	}

	if len(msg.Payload) == 0 {
		t.Error("Peers message should have payload")
	}
}

func TestParsePeersMessage(t *testing.T) {
	peers := []PeerInfo{
		{ID: "peer1", Address: "127.0.0.1", Port: 8000},
		{ID: "peer2", Address: "127.0.0.1", Port: 8001},
	}

	msg, err := NewPeersMessage(peers)
	if err != nil {
		t.Fatalf("Failed to create peers message: %v", err)
	}

	parsedPeers, err := ParsePeersMessage(msg)
	if err != nil {
		t.Fatalf("Failed to parse peers message: %v", err)
	}

	if len(parsedPeers) != len(peers) {
		t.Errorf("Expected %d peers, got %d", len(peers), len(parsedPeers))
	}

	for i, peer := range parsedPeers {
		if peer.ID != peers[i].ID {
			t.Errorf("Expected peer ID %s, got %s", peers[i].ID, peer.ID)
		}
		if peer.Address != peers[i].Address {
			t.Errorf("Expected address %s, got %s", peers[i].Address, peer.Address)
		}
		if peer.Port != peers[i].Port {
			t.Errorf("Expected port %d, got %d", peers[i].Port, peer.Port)
		}
	}
}

func TestParsePeersMessageInvalidType(t *testing.T) {
	msg := NewPingMessage()

	_, err := ParsePeersMessage(msg)
	if err == nil {
		t.Error("Expected error for invalid message type")
	}
}

func TestCalculateChecksum(t *testing.T) {
	data1 := []byte("test data")
	data2 := []byte("test data")
	data3 := []byte("different data")

	checksum1 := calculateChecksum(data1)
	checksum2 := calculateChecksum(data2)
	checksum3 := calculateChecksum(data3)

	if checksum1 != checksum2 {
		t.Error("Same data should produce same checksum")
	}

	if checksum1 == checksum3 {
		t.Error("Different data should produce different checksums")
	}
}

func TestMessageSerializeDeserializeRoundTrip(t *testing.T) {
	tests := []struct {
		name    string
		msgType MessageType
		payload []byte
	}{
		{"Ping", MessageTypePing, nil},
		{"Pong", MessageTypePong, nil},
		{"With Payload", MessageTypeTransaction, []byte("transaction data")},
		{"Large Payload", MessageTypeBlocks, make([]byte, 1024)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalMsg := NewMessage(tt.msgType, tt.payload)

			data, err := originalMsg.Serialize()
			if err != nil {
				t.Fatalf("Failed to serialize: %v", err)
			}

			deserializedMsg, err := Deserialize(data)
			if err != nil {
				t.Fatalf("Failed to deserialize: %v", err)
			}

			if deserializedMsg.Version != originalMsg.Version {
				t.Error("Version mismatch after round trip")
			}

			if deserializedMsg.Type != originalMsg.Type {
				t.Error("Type mismatch after round trip")
			}

			if deserializedMsg.Length != originalMsg.Length {
				t.Error("Length mismatch after round trip")
			}

			if deserializedMsg.Checksum != originalMsg.Checksum {
				t.Error("Checksum mismatch after round trip")
			}

			if !bytes.Equal(deserializedMsg.Payload, originalMsg.Payload) {
				t.Error("Payload mismatch after round trip")
			}
		})
	}
}

func TestMessageToJSON(t *testing.T) {
	payload := []byte("test payload")
	msg := NewMessage(MessageTypePing, payload)

	jsonStr, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if len(jsonStr) == 0 {
		t.Error("JSON string should not be empty")
	}

	if jsonStr[0] != '{' {
		t.Error("JSON should start with '{'")
	}
}
