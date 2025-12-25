package blockchain

import (
	"bytes"
	"fmt"
	"testing"
	"time"
)

func TestBlockchainIntegration(t *testing.T) {
	bc := NewBlockchain()

	transactions := []string{
		"Alice sends 1 BTC to Bob",
		"Bob sends 0.5 BTC to maged",
		"maged sends 0.2 BTC to aymen",
		"aymen send 0.1 BTC to yussef",
	}
	for i, tx := range transactions {
		err := bc.AddBlock(tx)
		if err != nil {
			t.Errorf("Error adding block %d:%v", i+1, err)
		}
		if !bc.IsValid() {
			t.Errorf("Blockchain should be valid after adding block %d", i+1)
		}
	}
	if bc.GetChainLength() != 5 {
		t.Errorf("Expected chain length 5, got %d", bc.GetChainLength())
	}
	for i := 1; i < bc.GetChainLength(); i++ {
		currentBlock := bc.Blocks[i]
		previousBlock := bc.Blocks[i-1]
		if !bytes.Equal(currentBlock.PrevHash, previousBlock.Hash) {
			t.Errorf("Block %d should link to block %d", i, i-1)
		}
	}
}

func TestBlockchainPerformance(t *testing.T) {
	bc := NewBlockchain()
	start := time.Now()
	for i := 0; i < 1000; i++ {
		bc.AddBlock(fmt.Sprintf("Transaction %d", i))
	}
	duration := time.Since(start)
	if duration > time.Second {
		t.Errorf("Adding 1000 blocks took too long:%v", duration)
	}
	if !bc.IsValid() {
		t.Error("Large blockchain should still be valid")
	}
	t.Logf("Added 1000 blocks in %v", duration)
}
