package blockchain

import (
	"bytes"
	"testing"
)

func TestNewBlockchain(t *testing.T) {
	bc := NewBlockchain()

	if bc == nil {
		t.Fatal("NewBlockchain returned nil")
	}

	if len(bc.Blocks) != 1 {
		t.Errorf("Expected 1 block (genesis), got %d", len(bc.Blocks))
	}

	genesis := bc.Blocks[0]
	if string(genesis.Data) != "Genesis Block" {
		t.Errorf("Genesis data should be 'Genesis Block', got %s", genesis.Data)
	}

	if len(genesis.PrevHash) != 0 {
		t.Errorf("Genesis PrevHash should be empty, got %s", genesis.PrevHash)
	}

	if len(genesis.Hash) == 0 {
		t.Error("Genesis should have calculated hash")
	}

	if !bytes.Equal(genesis.Hash, genesis.CalculateHash()) {
		t.Error("Genesis hash should match calculated hash")
	}
}

func TestAddBlock(t *testing.T) {
	bc := NewBlockchain()

	// Add first block
	err := bc.AddBlock("First transaction")
	if err != nil {
		t.Errorf("Error adding block: %v", err)
	}

	if len(bc.Blocks) != 2 {
		t.Errorf("Expected 2 blocks, got %d", len(bc.Blocks))
	}

	newBlock := bc.Blocks[1]
	if string(newBlock.Data) != "First transaction" {
		t.Errorf("Expected 'First transaction', got %s", string(newBlock.Data))
	}

	// Check hash linking
	if !bytes.Equal(newBlock.PrevHash, bc.Blocks[0].Hash) {
		t.Error("New block should point to previous block")
	}

	// Check hash is calculated
	if !bytes.Equal(newBlock.Hash, newBlock.CalculateHash()) {
		t.Error("New block hash should match calculated hash")
	}

	// Add another block
	err = bc.AddBlock("Second transaction")
	if err != nil {
		t.Errorf("Error adding second block: %v", err)
	}

	if len(bc.Blocks) != 3 {
		t.Errorf("Expected 3 blocks, got %d", len(bc.Blocks))
	}

	secondBlock := bc.Blocks[2]
	if string(secondBlock.Data) != "Second transaction" {
		t.Errorf("Expected 'Second transaction', got %s", string(secondBlock.Data))
	}

	if !bytes.Equal(secondBlock.PrevHash, newBlock.Hash) {
		t.Error("Second block should point to first added block")
	}
}

func TestGetLatestBlock(t *testing.T) {
	bc := NewBlockchain()

	// Initially, latest is genesis
	latest := bc.GetLatestBlock()
	if latest == nil {
		t.Fatal("GetLatestBlock returned nil for non-empty chain")
	}

	if string(latest.Data) != "Genesis Block" {
		t.Errorf("Latest should be genesis, got %s", string(latest.Data))
	}

	// Add a block
	bc.AddBlock("Test block")
	latest = bc.GetLatestBlock()
	if string(latest.Data) != "Test block" {
		t.Errorf("Latest should be added block, got %s", string(latest.Data))
	}
}

func TestIsValid(t *testing.T) {
	bc := NewBlockchain()

	// Empty chain should be invalid
	emptyBc := &Blockchain{Blocks: []*Block{}}
	if emptyBc.IsValid() {
		t.Error("Empty blockchain should be invalid")
	}

	// Single genesis should be valid
	if !bc.IsValid() {
		t.Error("Blockchain with genesis should be valid")
	}

	// Add valid blocks
	bc.AddBlock("Valid block 1")
	bc.AddBlock("Valid block 2")

	if !bc.IsValid() {
		t.Error("Blockchain with valid blocks should be valid")
	}

	// Tamper with data - should invalidate
	originalData := bc.Blocks[1].Data
	bc.Blocks[1].Data = []byte("Tampered data")

	if bc.IsValid() {
		t.Error("Blockchain with tampered data should be invalid")
	}

	// Restore and check valid again
	bc.Blocks[1].Data = originalData
	if !bc.IsValid() {
		t.Error("Restored blockchain should be valid")
	}

	// Tamper with hash
	originalHash := bc.Blocks[1].Hash
	bc.Blocks[1].Hash = []byte("fakehash")

	if bc.IsValid() {
		t.Error("Blockchain with fake hash should be invalid")
	}

	// Restore
	bc.Blocks[1].Hash = originalHash
	if !bc.IsValid() {
		t.Error("Restored blockchain should be valid")
	}

	// Tamper with PrevHash
	originalPrevHash := bc.Blocks[2].PrevHash
	bc.Blocks[2].PrevHash = []byte("wrongprev")

	if bc.IsValid() {
		t.Error("Blockchain with wrong prev hash should be invalid")
	}

	// Restore
	bc.Blocks[2].PrevHash = originalPrevHash
	if !bc.IsValid() {
		t.Error("Restored blockchain should be valid")
	}
}

func TestGetLatestBlockEmpty(t *testing.T) {
	emptyBc := &Blockchain{Blocks: []*Block{}}
	latest := emptyBc.GetLatestBlock()
	if latest != nil {
		t.Error("GetLatestBlock on empty chain should return nil")
	}
}

func TestGetBlockByIndex(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock("Test block")

	// Valid indices
	genesis, err := bc.GetBlockByIndex(0)
	if err != nil {
		t.Errorf("Error getting genesis block: %v", err)
	}
	if string(genesis.Data) != "Genesis Block" {
		t.Error("Genesis block data incorrect")
	}

	block1, err := bc.GetBlockByIndex(1)
	if err != nil {
		t.Errorf("Error getting block 1: %v", err)
	}
	if string(block1.Data) != "Test block" {
		t.Error("Block 1 data incorrect")
	}

	// Invalid indices
	_, err = bc.GetBlockByIndex(-1)
	if err == nil {
		t.Error("Expected error for negative index")
	}

	_, err = bc.GetBlockByIndex(2)
	if err == nil {
		t.Error("Expected error for out-of-range index")
	}
}

func TestGetBlockByHash(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock("Test block")

	genesis := bc.Blocks[0]
	found := bc.GetBlockByHash(genesis.Hash)
	if found == nil {
		t.Error("Genesis block not found by hash")
	}
	if !bytes.Equal(found.Hash, genesis.Hash) {
		t.Error("Found block hash mismatch")
	}

	// Non-existing hash
	notFound := bc.GetBlockByHash([]byte("nonexistent"))
	if notFound != nil {
		t.Error("Non-existing hash should return nil")
	}
}

func TestGetChainLength(t *testing.T) {
	bc := NewBlockchain()
	if bc.GetChainLength() != 1 {
		t.Errorf("Expected length 1, got %d", bc.GetChainLength())
	}

	bc.AddBlock("Block 1")
	if bc.GetChainLength() != 2 {
		t.Errorf("Expected length 2, got %d", bc.GetChainLength())
	}
}

func TestPrintBlockChain(t *testing.T) {
	bc := NewBlockchain()
	bc.AddBlock("Test block")
	// Just call it to cover the function - output goes to stdout
	bc.PrintBlockChain()
}

func TestIsValidInvalidGenesis(t *testing.T) {
	bc := NewBlockchain()
	// Tamper with genesis hash
	originalHash := bc.Blocks[0].Hash
	bc.Blocks[0].Hash = []byte("invalid")

	if bc.IsValid() {
		t.Error("Blockchain with invalid genesis should be invalid")
	}

	// Restore
	bc.Blocks[0].Hash = originalHash
	if !bc.IsValid() {
		t.Error("Restored blockchain should be valid")
	}
}
