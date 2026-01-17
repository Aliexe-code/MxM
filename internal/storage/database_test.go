package storage

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aliexe/blockChain/internal/blockchain"
)

func TestDatabaseNew(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Verify database file exists
	if _, err := os.Stat(tempFile); os.IsNotExist(err) {
		t.Error("Database file was not created")
	}

	// Verify schema version
	version, err := ds.GetSchemaVersion()
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}
	if version != "1" {
		t.Errorf("Expected schema version 1, got %s", version)
	}
}

func TestDatabaseSaveAndLoad(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create a blockchain with some blocks
	bc := blockchain.NewBlockchain()
	bc.AddBlock("First transaction")
	bc.AddBlock("Second transaction")

	// Save the blockchain
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Verify blockchain exists
	if !ds.Exists() {
		t.Error("Blockchain was not saved to database")
	}

	// Load the blockchain
	loadedBC, err := ds.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load blockchain: %v", err)
	}

	// Verify loaded blockchain
	if loadedBC.GetChainLength() != bc.GetChainLength() {
		t.Errorf("Expected chain length %d, got %d", bc.GetChainLength(), loadedBC.GetChainLength())
	}

	// Verify validity
	if !loadedBC.IsValid() {
		t.Error("Loaded blockchain is invalid")
	}
}

func TestDatabaseSaveWithMining(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create blockchain with mined blocks
	bc := blockchain.NewBlockchain()
	_, err = bc.AddBlockWithMining("Mined block 1", "miner1", 2)
	if err != nil {
		t.Fatalf("Failed to add mined block: %v", err)
	}

	// Save
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Load
	loadedBC, err := ds.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load blockchain: %v", err)
	}

	// Verify mining rewards
	stats := loadedBC.GetMiningStats()
	totalRewards := stats["total_rewards"].(float64)
	if totalRewards == 0 {
		t.Error("Mining rewards were not preserved")
	}

	// Verify miner rewards
	minerRewards, err := ds.GetMinerRewards("miner1")
	if err != nil {
		t.Fatalf("Failed to get miner rewards: %v", err)
	}
	if minerRewards == 0 {
		t.Error("Miner rewards are zero")
	}
}

func TestDatabaseLoadEmpty(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Try to load non-existent blockchain
	// Since database is empty, LoadBlockchain will fail validation
	// This is expected behavior
	_, err = ds.LoadBlockchain()
	if err == nil {
		t.Error("Expected error when loading empty database")
	}
}

func TestDatabaseGetBlockByIndex(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Block 1")
	bc.AddBlock("Block 2")
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Get block by index
	block, err := ds.GetBlockByIndex(1)
	if err != nil {
		t.Fatalf("Failed to get block by index: %v", err)
	}

	if string(block.Data) != "Block 1" {
		t.Errorf("Expected data 'Block 1', got '%s'", string(block.Data))
	}

	// Try to get non-existent block
	_, err = ds.GetBlockByIndex(999)
	if err == nil {
		t.Error("Expected error when getting non-existent block")
	}
}

func TestDatabaseGetBlockByHash(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Block 1")
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Get block by hash
	block1 := bc.Blocks[1]
	block, err := ds.GetBlockByHash(block1.Hash)
	if err != nil {
		t.Fatalf("Failed to get block by hash: %v", err)
	}

	if string(block.Data) != "Block 1" {
		t.Errorf("Expected data 'Block 1', got '%s'", string(block.Data))
	}

	// Try to get non-existent block
	_, err = ds.GetBlockByHash([]byte("nonexistent"))
	if err == nil {
		t.Error("Expected error when getting non-existent block by hash")
	}
}

func TestDatabaseGetBlocksByTimeRange(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create blockchain with specific timestamps
	bc := blockchain.NewBlockchain()
	now := time.Now().Unix()
	bc.Blocks[0].Timestamp = now - 3600 // 1 hour ago
	bc.AddBlock("Block 1")
	bc.Blocks[1].Timestamp = now - 1800 // 30 min ago
	bc.AddBlock("Block 2")
	bc.Blocks[2].Timestamp = now - 900 // 15 min ago

	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Query blocks in time range
	start := time.Unix(now-2400, 0) // 40 min ago
	end := time.Unix(now-600, 0)    // 10 min ago
	blocks, err := ds.GetBlocksByTimeRange(start, end)
	if err != nil {
		t.Fatalf("Failed to get blocks by time range: %v", err)
	}

	if len(blocks) != 2 {
		t.Errorf("Expected 2 blocks in range, got %d", len(blocks))
	}
}

func TestDatabaseGetChainLength(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Test empty database
	length, err := ds.GetChainLength()
	if err != nil {
		t.Fatalf("Failed to get chain length: %v", err)
	}
	if length != 0 {
		t.Errorf("Expected chain length 0, got %d", length)
	}

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Block 1")
	bc.AddBlock("Block 2")
	bc.AddBlock("Block 3")
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Get chain length
	length, err = ds.GetChainLength()
	if err != nil {
		t.Fatalf("Failed to get chain length: %v", err)
	}
	if length != 4 {
		t.Errorf("Expected chain length 4, got %d", length)
	}
}

func TestDatabaseGetMinerRewards(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create blockchain with mining rewards
	bc := blockchain.NewBlockchain()
	bc.AddBlockWithMining("Block 1", "miner1", 2)
	bc.AddBlockWithMining("Block 2", "miner2", 2)
	bc.AddBlockWithMining("Block 3", "miner1", 3)
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Get rewards for miner1
	rewards1, err := ds.GetMinerRewards("miner1")
	if err != nil {
		t.Fatalf("Failed to get miner1 rewards: %v", err)
	}
	if rewards1 == 0 {
		t.Error("Expected non-zero rewards for miner1")
	}

	// Get rewards for miner2
	rewards2, err := ds.GetMinerRewards("miner2")
	if err != nil {
		t.Fatalf("Failed to get miner2 rewards: %v", err)
	}
	if rewards2 == 0 {
		t.Error("Expected non-zero rewards for miner2")
	}

	// Get rewards for non-existent miner
	rewards3, err := ds.GetMinerRewards("miner3")
	if err != nil {
		t.Fatalf("Failed to get miner3 rewards: %v", err)
	}
	if rewards3 != 0 {
		t.Error("Expected zero rewards for non-existent miner")
	}
}

func TestDatabaseGetMiningStats(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create blockchain with mining rewards
	bc := blockchain.NewBlockchain()
	bc.AddBlockWithMining("Block 1", "miner1", 2)
	bc.AddBlockWithMining("Block 2", "miner2", 2)
	bc.AddBlockWithMining("Block 3", "miner1", 3)
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Get mining stats
	stats, err := ds.GetMiningStats()
	if err != nil {
		t.Fatalf("Failed to get mining stats: %v", err)
	}

	totalBlocks := stats["total_blocks"].(int)
	if totalBlocks != 4 {
		t.Errorf("Expected 4 blocks, got %d", totalBlocks)
	}

	totalRewards := stats["total_rewards"].(float64)
	if totalRewards == 0 {
		t.Error("Expected non-zero total rewards")
	}

	minerRewards := stats["miner_rewards"].(map[string]float64)
	if _, ok := minerRewards["miner1"]; !ok {
		t.Error("Expected miner1 in rewards")
	}
	if _, ok := minerRewards["miner2"]; !ok {
		t.Error("Expected miner2 in rewards")
	}

	minerBlocks := stats["miner_blocks"].(map[string]int)
	if minerBlocks["miner1"] != 2 {
		t.Errorf("Expected 2 blocks for miner1, got %d", minerBlocks["miner1"])
	}
}

func TestDatabaseDelete(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create and save blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Block 1")
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save blockchain: %v", err)
	}

	// Delete
	if err := ds.Delete(); err != nil {
		t.Fatalf("Failed to delete storage: %v", err)
	}

	// Verify blockchain is deleted
	if ds.Exists() {
		t.Error("Blockchain was not deleted")
	}

	// Verify chain length is 0
	length, err := ds.GetChainLength()
	if err != nil {
		t.Fatalf("Failed to get chain length: %v", err)
	}
	if length != 0 {
		t.Errorf("Expected chain length 0 after delete, got %d", length)
	}
}

func TestDatabaseConcurrentAccess(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create initial blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Initial block")
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save initial blockchain: %v", err)
	}

	// Concurrent saves and loads
	done := make(chan bool, 10)

	// Start 5 writers
	for i := 0; i < 5; i++ {
		go func(index int) {
			loadedBC, err := ds.LoadBlockchain()
			if err != nil {
				t.Errorf("Failed to load blockchain: %v", err)
				done <- false
				return
			}
			loadedBC.AddBlock(fmt.Sprintf("Concurrent block %d", index))
			if err := ds.SaveBlockchain(loadedBC); err != nil {
				t.Errorf("Concurrent save failed: %v", err)
				done <- false
				return
			}
			done <- true
		}(i)
	}

	// Start 5 readers
	for i := 0; i < 5; i++ {
		go func() {
			_, err := ds.LoadBlockchain()
			if err != nil {
				t.Errorf("Concurrent load failed: %v", err)
				done <- false
				return
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	successCount := 0
	for i := 0; i < 10; i++ {
		if <-done {
			successCount++
		}
	}

	// At least some operations should succeed
	if successCount < 5 {
		t.Errorf("Too many concurrent operations failed: %d/10", successCount)
	}

	// Verify final blockchain is valid
	loadedBC, err := ds.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load final blockchain: %v", err)
	}

	if !loadedBC.IsValid() {
		t.Error("Final blockchain is invalid after concurrent access")
	}
}

func TestDatabaseLargeBlockchain(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Create blockchain with many blocks
	bc := blockchain.NewBlockchain()
	for i := 0; i < 100; i++ {
		bc.AddBlock(fmt.Sprintf("Block %d with some data content", i))
	}

	// Save and load
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save large blockchain: %v", err)
	}

	loadedBC, err := ds.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load large blockchain: %v", err)
	}

	if loadedBC.GetChainLength() != 101 {
		t.Errorf("Expected chain length 101, got %d", loadedBC.GetChainLength())
	}
}

func TestDatabaseMultipleSaveLoadCycles(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Multiple save/load cycles
	for i := 0; i < 5; i++ {
		bc := blockchain.NewBlockchain()
		bc.AddBlock(fmt.Sprintf("Cycle %d - Block 1", i))
		bc.AddBlock(fmt.Sprintf("Cycle %d - Block 2", i))

		if err := ds.SaveBlockchain(bc); err != nil {
			t.Fatalf("Failed to save blockchain in cycle %d: %v", i, err)
		}

		loadedBC, err := ds.LoadBlockchain()
		if err != nil {
			t.Fatalf("Failed to load blockchain in cycle %d: %v", i, err)
		}

		if loadedBC.GetChainLength() != 3 {
			t.Errorf("Expected chain length 3 in cycle %d, got %d", i, loadedBC.GetChainLength())
		}
	}
}

func TestDatabaseGetDBPath(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	if ds.GetDBPath() != tempFile {
		t.Errorf("Expected DB path %s, got %s", tempFile, ds.GetDBPath())
	}
}

func TestDatabaseClose(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}

	// Close the database
	if err := ds.Close(); err != nil {
		t.Fatalf("Failed to close database: %v", err)
	}

	// Try to use closed database (should fail)
	_, err = ds.LoadBlockchain()
	if err == nil {
		t.Error("Expected error when using closed database")
	}
}

func TestDatabaseTransactionRollback(t *testing.T) {
	tempFile := t.TempDir() + "/test.db"
	ds, err := NewDatabaseStorage(tempFile)
	if err != nil {
		t.Fatalf("Failed to create database storage: %v", err)
	}
	defer ds.Close()

	// Save initial blockchain
	bc := blockchain.NewBlockchain()
	bc.AddBlock("Block 1")
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save initial blockchain: %v", err)
	}

	// Try to save invalid blockchain (SaveBlockchain doesn't validate, it just saves)
	// The validation happens on load, so we need to test that the database can be cleared
	invalidBC := &blockchain.Blockchain{
		Blocks: []*blockchain.Block{
			{
				Timestamp:  time.Now().Unix(),
				Data:       []byte("Invalid"),
				PrevHash:   []byte{},
				Hash:       []byte("wrong"),
				Nonce:      0,
				Difficulty: 4,
			},
		},
	}

	// This will save the invalid blockchain
	err = ds.SaveBlockchain(invalidBC)
	if err != nil {
		t.Fatalf("Failed to save invalid blockchain: %v", err)
	}

	// Now try to load it - should fail validation
	_, err = ds.LoadBlockchain()
	if err == nil {
		t.Error("Expected error when loading invalid blockchain")
	}

	// Delete and save valid blockchain again
	if err := ds.Delete(); err != nil {
		t.Fatalf("Failed to delete blockchain: %v", err)
	}

	bc = blockchain.NewBlockchain()
	bc.AddBlock("Block 1")
	if err := ds.SaveBlockchain(bc); err != nil {
		t.Fatalf("Failed to save valid blockchain: %v", err)
	}

	// Verify we can load it now
	loadedBC, err := ds.LoadBlockchain()
	if err != nil {
		t.Fatalf("Failed to load blockchain: %v", err)
	}

	if loadedBC.GetChainLength() != 2 {
		t.Errorf("Expected chain length 2, got %d", loadedBC.GetChainLength())
	}
}
