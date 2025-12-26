package blockchain

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestBlockMarshalJSON(t *testing.T) {
	block := &Block{
		Timestamp: 1234567890,
		Data:      []byte("test data"),
		PrevHash:  []byte("prevhash"),
		Hash:      []byte("hash"),
	}

	data, err := block.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// Unmarshal to check structure
	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshaling JSON failed: %v", err)
	}

	if unmarshaled["timestamp"].(float64) != 1234567890 {
		t.Errorf("Timestamp mismatch: got %v", unmarshaled["timestamp"])
	}
	// Data is base64 encoded
	if unmarshaled["data"].(string) != "dGVzdCBkYXRh" {
		t.Errorf("Data mismatch: got %v", unmarshaled["data"])
	}
	if unmarshaled["prev_hash"].(string) != "cHJldmhhc2g=" {
		t.Errorf("PrevHash mismatch: got %v", unmarshaled["prev_hash"])
	}
	if unmarshaled["hash"].(string) != "aGFzaA==" {
		t.Errorf("Hash mismatch: got %v", unmarshaled["hash"])
	}
}

func TestBlockUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"timestamp": 1234567890,
		"data": "dGVzdCBkYXRh",
		"prev_hash": "cHJldmhhc2g=",
		"hash": "aGFzaA=="
	}`

	var block Block
	err := json.Unmarshal([]byte(jsonData), &block)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	if block.Timestamp != 1234567890 {
		t.Errorf("Timestamp mismatch: got %d", block.Timestamp)
	}
	if string(block.Data) != "test data" {
		t.Errorf("Data mismatch: got %s", block.Data)
	}
	if string(block.PrevHash) != "prevhash" {
		t.Errorf("PrevHash mismatch: got %s", block.PrevHash)
	}
	if string(block.Hash) != "hash" {
		t.Errorf("Hash mismatch: got %s", block.Hash)
	}
}

func TestBlockUnmarshalJSONInvalid(t *testing.T) {
	invalidJSON := `{"timestamp": "invalid", "data": "test"}`

	var block Block
	err := json.Unmarshal([]byte(invalidJSON), &block)
	if err == nil {
		t.Error("Expected error for invalid JSON, but got none")
	}
}

func TestBlockRoundTripJSON(t *testing.T) {
	original := &Block{
		Timestamp: 987654321,
		Data:      []byte("round trip data"),
		PrevHash:  []byte("prev"),
		Hash:      []byte("curr"),
	}

	// Marshal
	data, err := original.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON failed: %v", err)
	}

	// Unmarshal
	var unmarshaled Block
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("UnmarshalJSON failed: %v", err)
	}

	// Compare
	if original.Timestamp != unmarshaled.Timestamp {
		t.Errorf("Timestamp mismatch: %d vs %d", original.Timestamp, unmarshaled.Timestamp)
	}
	if string(original.Data) != string(unmarshaled.Data) {
		t.Errorf("Data mismatch: %s vs %s", original.Data, unmarshaled.Data)
	}
	if string(original.PrevHash) != string(unmarshaled.PrevHash) {
		t.Errorf("PrevHash mismatch: %s vs %s", original.PrevHash, unmarshaled.PrevHash)
	}
	if string(original.Hash) != string(unmarshaled.Hash) {
		t.Errorf("Hash mismatch: %s vs %s", original.Hash, unmarshaled.Hash)
	}
}

func TestBlockchainToJSON(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock("Test data")

	jsonData, err := bc.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	if len(jsonData) == 0 {
		t.Error("ToJSON returned empty data")
	}

	// Check if it's valid JSON
	var temp map[string]interface{}
	if err := json.Unmarshal(jsonData, &temp); err != nil {
		t.Errorf("ToJSON did not produce valid JSON: %v", err)
	}
}

func TestBlockchainFromJSON(t *testing.T) {
	original := NewBlockchain()
	original.AddBlock("Test data")

	jsonData, err := original.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	var loaded Blockchain
	err = loaded.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	if loaded.GetChainLength() != original.GetChainLength() {
		t.Errorf("Chain length mismatch: %d vs %d", loaded.GetChainLength(), original.GetChainLength())
	}

	// Check if loaded is valid
	if !loaded.IsValid() {
		t.Error("Loaded blockchain is not valid")
	}
}

func TestBlockchainFromJSONInvalid(t *testing.T) {
	invalidJSON := `{"invalid": "json"}`

	var bc Blockchain
	err := bc.FromJSON([]byte(invalidJSON))
	if err == nil {
		t.Error("FromJSON should fail with invalid JSON")
	}
}

func TestBlockchainFromJSONInvalidBlockchain(t *testing.T) {
	// Create a blockchain, tamper with JSON
	bc := NewBlockchain()
	jsonData, _ := bc.ToJSON()

	// Tamper with the JSON to make it invalid (e.g., change hash)
	tampered := strings.Replace(string(jsonData), `"hash":`, `"hash":"tampered"`, 1)

	var loaded Blockchain
	err := loaded.FromJSON([]byte(tampered))
	if err == nil {
		t.Error("FromJSON should fail with invalid blockchain")
	}
}

func TestBlockchainSaveToFile(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock("Test data")

	filename := "test_blockchain.json"
	defer os.Remove(filename) // Clean up

	err := bc.SaveToFile(filename)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		t.Error("File was not created")
	}
}

func TestBlockchainSaveToFileError(t *testing.T) {
	bc := NewBlockchain()

	// Try to save to an invalid path
	err := bc.SaveToFile("/invalid/path/blockchain.json")
	if err == nil {
		t.Error("SaveToFile should fail with invalid path")
	}
}

func TestBlockchainLoadFromFile(t *testing.T) {
	original := NewBlockchain()
	original.AddBlock("Test data")

	filename := "test_load_blockchain.json"
	defer os.Remove(filename)

	// Save first
	err := original.SaveToFile(filename)
	if err != nil {
		t.Fatalf("SaveToFile failed: %v", err)
	}

	// Load
	var loaded Blockchain
	err = loaded.LoadFromFile(filename)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if loaded.GetChainLength() != original.GetChainLength() {
		t.Errorf("Chain length mismatch after load: %d vs %d", loaded.GetChainLength(), original.GetChainLength())
	}
}

func TestBlockchainLoadFromFileNotExist(t *testing.T) {
	var bc Blockchain
	err := bc.LoadFromFile("nonexistent.json")
	if err == nil {
		t.Error("LoadFromFile should fail for non-existent file")
	}
}

func TestBlockchainLoadFromFileReadError(t *testing.T) {
	// Create a directory with the name
	dirname := "test_dir"
	os.Mkdir(dirname, 0755)
	defer os.RemoveAll(dirname)

	var bc Blockchain
	err := bc.LoadFromFile(dirname) // Try to read directory as file
	if err == nil {
		t.Error("LoadFromFile should fail when trying to read a directory")
	}
}

func TestBlockchainExportPrettyJSON(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock("Test data")

	prettyJSON, err := bc.ExportPrettyJSON()
	if err != nil {
		t.Fatalf("ExportPrettyJSON failed: %v", err)
	}

	if len(prettyJSON) == 0 {
		t.Error("ExportPrettyJSON returned empty string")
	}

	// Check if it contains indentation (spaces)
	if !strings.Contains(prettyJSON, "  ") {
		t.Error("ExportPrettyJSON should produce indented JSON")
	}
}
