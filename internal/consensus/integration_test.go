package consensus

import (
	"fmt"
	"testing"

	"github.com/aliexe/blockChain/internal/blockchain"
)

// TestMultiNodeConsensus tests consensus across multiple nodes
func TestMultiNodeConsensus(t *testing.T) {
	// Create three independent blockchains
	bc1 := blockchain.NewBlockchain()
	bc2 := blockchain.NewBlockchain()
	bc3 := blockchain.NewBlockchain()

	// Mine blocks on bc1
	for i := 0; i < 3; i++ {
		_, err := bc1.AddBlockWithMining(
			fmt.Sprintf("Block %d from node1", i),
			"node1",
			2,
		)
		if err != nil {
			t.Fatalf("Failed to mine block %d: %v", i, err)
		}
	}

	// Verify bc1 is valid
	if !bc1.IsValid() {
		t.Error("Node1 chain is invalid")
	}

	// Simulate consensus: all nodes should agree on the same chain
	// In a real scenario, nodes would sync via network
	height1 := bc1.GetChainLength()
	height2 := bc2.GetChainLength()
	height3 := bc3.GetChainLength()

	// bc1 should have more blocks than bc2 and bc3
	if height1 <= height2 || height1 <= height3 {
		t.Errorf("Expected node1 to have more blocks: node1=%d, node2=%d, node3=%d", height1, height2, height3)
	}

	// Test consensus rules
	rules := DefaultConsensusRules()

	// Verify bc1 passes consensus rules
	if err := rules.ValidateChain(bc1); err != nil {
		t.Errorf("Node1 chain failed consensus validation: %v", err)
	}

	// Test fork resolution by creating a fork
	bc4 := blockchain.NewBlockchain()
	bc5 := blockchain.NewBlockchain()

	// Add same blocks to both chains
	for i := 0; i < 2; i++ {
		err := bc4.AddBlock(fmt.Sprintf("Common block %d", i))
		if err != nil {
			t.Fatalf("Failed to add block to bc4: %v", err)
		}
		err = bc5.AddBlock(fmt.Sprintf("Common block %d", i))
		if err != nil {
			t.Fatalf("Failed to add block to bc5: %v", err)
		}
	}

	// Add different blocks to create a fork
	_, err := bc4.AddBlockWithMining("Fork block A", "miner-a", 2)
	if err != nil {
		t.Fatalf("Failed to mine fork block A: %v", err)
	}

	_, err = bc5.AddBlockWithMining("Fork block B", "miner-b", 2)
	if err != nil {
		t.Fatalf("Failed to mine fork block B: %v", err)
	}

	// Both chains should be valid
	if !bc4.IsValid() {
		t.Error("Fork chain A is invalid")
	}
	if !bc5.IsValid() {
		t.Error("Fork chain B is invalid")
	}

	// Test fork resolution
	bestChain, err := rules.SelectBestChain(bc4, []*blockchain.Blockchain{bc5})
	if err != nil {
		t.Fatalf("Failed to select best chain: %v", err)
	}

	// The chain with more work should be selected
	// Both chains have 4 blocks (1 genesis + 2 common + 1 fork)
	if bestChain.GetChainLength() != 4 {
		t.Errorf("Expected best chain to have 4 blocks, got %d", bestChain.GetChainLength())
	}
}

// TestForkResolution tests fork resolution across nodes
func TestForkResolution(t *testing.T) {
	// Create two blockchains with same genesis
	bc1 := blockchain.NewBlockchain()
	bc2 := blockchain.NewBlockchain()

	// Add same blocks to both chains
	for i := 0; i < 2; i++ {
		err := bc1.AddBlock(fmt.Sprintf("Common block %d", i))
		if err != nil {
			t.Fatalf("Failed to add block to bc1: %v", err)
		}
		err = bc2.AddBlock(fmt.Sprintf("Common block %d", i))
		if err != nil {
			t.Fatalf("Failed to add block to bc2: %v", err)
		}
	}

	// Mine different blocks on each chain (creating a fork)
	_, err := bc1.AddBlockWithMining("Fork block A", "miner-a", 2)
	if err != nil {
		t.Fatalf("Failed to mine fork block A: %v", err)
	}

	_, err = bc2.AddBlockWithMining("Fork block B", "miner-b", 2)
	if err != nil {
		t.Fatalf("Failed to mine fork block B: %v", err)
	}

	// Both chains should be valid
	if !bc1.IsValid() {
		t.Error("Fork chain A is invalid")
	}
	if !bc2.IsValid() {
		t.Error("Fork chain B is invalid")
	}

	// Test fork resolution using consensus rules
	rules := DefaultConsensusRules()
	bestChain, err := rules.SelectBestChain(bc1, []*blockchain.Blockchain{bc2})
	if err != nil {
		t.Fatalf("Failed to select best chain: %v", err)
	}

	// The chain with more work should be selected
	if bestChain.GetChainLength() != 4 {
		t.Errorf("Expected best chain to have 4 blocks, got %d", bestChain.GetChainLength())
	}

	// Test fork point detection
	forkPoint, err := rules.GetForkPoint(bc1, bc2)
	if err != nil {
		t.Fatalf("Failed to get fork point: %v", err)
	}

	if forkPoint != 2 {
		t.Errorf("Expected fork point to be 2, got %d", forkPoint)
	}
}

// TestPartitionRecovery tests recovery from network partition
func TestPartitionRecovery(t *testing.T) {
	// Create two blockchains
	bc1 := blockchain.NewBlockchain()
	bc2 := blockchain.NewBlockchain()

	// Add same blocks to both chains
	for i := 0; i < 2; i++ {
		err := bc1.AddBlock(fmt.Sprintf("Common block %d", i))
		if err != nil {
			t.Fatalf("Failed to add block to bc1: %v", err)
		}
		err = bc2.AddBlock(fmt.Sprintf("Common block %d", i))
		if err != nil {
			t.Fatalf("Failed to add block to bc2: %v", err)
		}
	}

	// Mine additional blocks on bc1 (simulating partition)
	for i := 0; i < 2; i++ {
		_, err := bc1.AddBlockWithMining(
			fmt.Sprintf("Partition block %d", i),
			"node1",
			2,
		)
		if err != nil {
			t.Fatalf("Failed to mine during partition: %v", err)
		}
	}

	height1 := bc1.GetChainLength()
	height2 := bc2.GetChainLength()

	t.Logf("During partition: bc1=%d, bc2=%d", height1, height2)

	// Both chains should still be valid
	if !bc1.IsValid() {
		t.Error("Chain A is invalid after partition")
	}
	if !bc2.IsValid() {
		t.Error("Chain B is invalid after partition")
	}

	// Simulate partition recovery by reconciling chains
	rules := DefaultConsensusRules()

	// The longer chain should win in fork resolution
	bestChain, err := rules.SelectBestChain(bc1, []*blockchain.Blockchain{bc2})
	if err != nil {
		t.Fatalf("Failed to select best chain: %v", err)
	}

	// bc1 should be selected as it has more blocks
	if bestChain.GetChainLength() != 5 {
		t.Errorf("Expected best chain to have 5 blocks, got %d", bestChain.GetChainLength())
	}
}