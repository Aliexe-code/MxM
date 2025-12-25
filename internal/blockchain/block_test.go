package blockchain

import (
	"bytes"
	"testing"
)

func TestBlockHashCalculation(t *testing.T) {
	block := &Block{
		Timestamp: 211512124123,
		Data:      []byte("test data"),
		PrevHash:  []byte{},
	}
	hash1 := block.CalculateHash()
	hash2 := block.CalculateHash()

	if !bytes.Equal(hash1, hash2) {
		t.Errorf("Expected same hash , got %s and %s", hash1, hash2)
	}
	if len(hash1) == 0 {
		t.Errorf("Expected non-empty hash, got empty hash")
	}
}

func TestBlockImmutability(t *testing.T) {
	block := NewBlock([]byte("test data"), []byte("prevhash"))
	originalHash := block.Hash

	block.Data = []byte("modified data")
	block.Hash = block.CalculateHash()
	if bytes.Equal(originalHash, block.Hash) {
		t.Errorf("Expected different hash after modification, got same hash")
	}
}

func TestGenesisBlock(t *testing.T) {
	genesis := NewGenesisBlock()

	if len(genesis.PrevHash) != 0 {
		t.Errorf("Expected empty PrevHash for genesis block, got %s", genesis.PrevHash)
	}

	if string(genesis.Data) != "Genesis Block" {
		t.Errorf("Genesis block data should be 'Genesis Block', got %s", genesis.Data)
	}

	if len(genesis.Hash) == 0 {
		t.Error("Genesis block should have calculated hash")
	}
}

func TestNewBlock(t *testing.T) {
	block := NewBlock([]byte("New transaction"), []byte("previoushash123"))

	if string(block.Data) != "New transaction" {
		t.Errorf("Expected 'New transaction' , got %s", block.Data)
	}
	if string(block.PrevHash) != "previoushash123" {
		t.Errorf("Expected 'New transaction' , got %s", block.PrevHash)
	}

	if len(block.Hash) == 0 {
		t.Error("block hash should be calculated")
	}

	expectedHash := block.CalculateHash()
	if !bytes.Equal(expectedHash, block.Hash) {
		t.Errorf("Block hash %s doesn't match calculated hash %s", block.Hash, expectedHash)
	}

}
