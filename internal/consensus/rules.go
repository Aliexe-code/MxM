package consensus

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/aliexe/blockChain/internal/blockchain"
	"github.com/aliexe/blockChain/internal/transactions"
)

// ConsensusRules defines the consensus rules for the blockchain
type ConsensusRules struct {
	rulesMu sync.RWMutex

	// Difficulty adjustment
	TargetBlockTime    time.Duration
	AdjustmentInterval int
	MinDifficulty      int
	MaxDifficulty      int

	// Block validation
	MaxBlockSize      int
	MaxTxCount        int
	CoinbaseMaturity  int

	// Fork resolution
	ForkTolerance     int
	MinConfirmations  int
}

// DefaultConsensusRules returns the default consensus rules
func DefaultConsensusRules() *ConsensusRules {
	return &ConsensusRules{
		TargetBlockTime:    2 * time.Minute,
		AdjustmentInterval: 10,
		MinDifficulty:      1,
		MaxDifficulty:      32,
		MaxBlockSize:       1_000_000, // 1MB
		MaxTxCount:         10000,
		CoinbaseMaturity:   100,
		ForkTolerance:      6,
		MinConfirmations:   6,
	}
}

// ValidateBlock validates a block against consensus rules
func (cr *ConsensusRules) ValidateBlock(block *blockchain.Block, prevBlock *blockchain.Block) error {
	cr.rulesMu.RLock()
	defer cr.rulesMu.RUnlock()

	// Validate basic block structure
	// Allow same timestamp for blocks mined in quick succession
	if block.Timestamp < prevBlock.Timestamp {
		return fmt.Errorf("block timestamp (%d) must be greater than or equal to previous block (%d)",
			block.Timestamp, prevBlock.Timestamp)
	}

	// Validate block size
	if len(block.Data) > cr.MaxBlockSize {
		return fmt.Errorf("block size (%d) exceeds maximum (%d)", len(block.Data), cr.MaxBlockSize)
	}

	// Validate difficulty
	if block.Difficulty < cr.MinDifficulty || block.Difficulty > cr.MaxDifficulty {
		return fmt.Errorf("invalid difficulty (%d): must be between %d and %d",
			block.Difficulty, cr.MinDifficulty, cr.MaxDifficulty)
	}

	// Validate proof of work
	if !block.IsValidProof() {
		return fmt.Errorf("invalid proof of work for block")
	}

	// Validate hash linking
	if string(block.PrevHash) != string(prevBlock.Hash) {
		return fmt.Errorf("block's previous hash does not match")
	}

	// Validate transactions in the block
	// Note: Currently blocks store data as bytes, not structured transactions
	// This validation will be enhanced when Block struct is refactored
	// For now, we perform basic data integrity checks
	if len(block.Data) > 0 {
		// Try to parse as transaction JSON for validation
		tx, err := transactions.FromJSON(string(block.Data))
		if err == nil {
			// Successfully parsed as transaction, validate it
			if err := tx.ValidateBasic(); err != nil {
				return fmt.Errorf("invalid transaction in block: %w", err)
			}
		}
		// If parsing fails, we assume it's non-transaction data and skip validation
		// This is acceptable for the current implementation
	}

	return nil
}

// ValidateChain validates the entire blockchain
func (cr *ConsensusRules) ValidateChain(bc *blockchain.Blockchain) error {
	cr.rulesMu.RLock()
	defer cr.rulesMu.RUnlock()

	if !bc.IsValid() {
		return fmt.Errorf("blockchain basic validation failed")
	}

	// Validate each block against consensus rules
	for i := 1; i < bc.GetChainLength(); i++ {
		block, err := bc.GetBlockByIndex(i)
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", i, err)
		}

		prevBlock, err := bc.GetBlockByIndex(i - 1)
		if err != nil {
			return fmt.Errorf("failed to get previous block %d: %w", i-1, err)
		}

		if err := cr.ValidateBlock(block, prevBlock); err != nil {
			return fmt.Errorf("block %d validation failed: %w", i, err)
		}
	}

	return nil
}

// CalculateNewDifficulty calculates the new difficulty based on recent block times
func (cr *ConsensusRules) CalculateNewDifficulty(bc *blockchain.Blockchain) (int, error) {
	cr.rulesMu.RLock()
	defer cr.rulesMu.RUnlock()

	if bc.GetChainLength() < cr.AdjustmentInterval {
		return cr.MinDifficulty, nil
	}

	// Get the last AdjustmentInterval blocks
	startIndex := bc.GetChainLength() - cr.AdjustmentInterval
	recentBlocks := make([]*blockchain.Block, 0, cr.AdjustmentInterval)

	for i := startIndex; i < bc.GetChainLength(); i++ {
		block, err := bc.GetBlockByIndex(i)
		if err != nil {
			return 0, fmt.Errorf("failed to get block %d: %w", i, err)
		}
		recentBlocks = append(recentBlocks, block)
	}

	// Calculate average block time
	var totalTime time.Duration
	for i := 1; i < len(recentBlocks); i++ {
		blockTime := time.Duration(recentBlocks[i].Timestamp - recentBlocks[i-1].Timestamp)
		totalTime += blockTime
	}

	avgBlockTime := totalTime / time.Duration(cr.AdjustmentInterval-1)
	currentDifficulty := recentBlocks[len(recentBlocks)-1].Difficulty

	// Calculate deviation percentage from target
	deviation := float64(avgBlockTime) / float64(cr.TargetBlockTime)

	// Proportional difficulty adjustment
	// If blocks are too fast (deviation < 1.0), increase difficulty
	// If blocks are too slow (deviation > 1.0), decrease difficulty
	// The adjustment is proportional to the deviation
	adjustment := 0

	switch {
	case deviation < 0.5:
		// Blocks are significantly too fast, increase by 2
		adjustment = 2
	case deviation < 0.75:
		// Blocks are moderately too fast, increase by 1
		adjustment = 1
	case deviation < 1.25:
		// Within tolerance range, no change
		adjustment = 0
	case deviation < 1.5:
		// Blocks are moderately too slow, decrease by 1
		adjustment = -1
	default:
		// Blocks are significantly too slow, decrease by 2
		adjustment = -2
	}

	newDifficulty := currentDifficulty + adjustment

	// Clamp to min/max bounds
	newDifficulty = rulesMax(cr.MinDifficulty, newDifficulty)
	newDifficulty = rulesMin(cr.MaxDifficulty, newDifficulty)

	return newDifficulty, nil
}

// SelectBestChain selects the best chain from multiple candidates
func (cr *ConsensusRules) SelectBestChain(localChain *blockchain.Blockchain, candidateChains []*blockchain.Blockchain) (*blockchain.Blockchain, error) {
	cr.rulesMu.RLock()
	defer cr.rulesMu.RUnlock()

	bestChain := localChain
	bestWork := calculateTotalWork(localChain)

	for _, candidate := range candidateChains {
		// Validate candidate chain
		if err := cr.ValidateChain(candidate); err != nil {
			continue // Skip invalid chains
		}

		// Calculate total work
		candidateWork := calculateTotalWork(candidate)

		// Select chain with more work
		if candidateWork.Cmp(bestWork) > 0 {
			bestChain = candidate
			bestWork = candidateWork
		}
	}

	return bestChain, nil
}

// ResolveFork resolves a blockchain fork by comparing total work
func (cr *ConsensusRules) ResolveFork(localChain *blockchain.Blockchain, forkChain *blockchain.Blockchain) error {
	cr.rulesMu.RLock()
	defer cr.rulesMu.RUnlock()

	// Find common ancestor
	commonIndex := localChain.FindCommonAncestor(forkChain)
	if commonIndex == -1 {
		return fmt.Errorf("no common ancestor found, cannot resolve fork")
	}

	// Calculate total work for both chains from common ancestor
	localWork := calculateTotalWorkFromIndex(localChain, commonIndex)
	forkWork := calculateTotalWorkFromIndex(forkChain, commonIndex)

	// Accept chain with more work (longest chain rule)
	if forkWork.Cmp(localWork) > 0 {
		// Replace local chain with fork chain
		return cr.replaceChain(localChain, forkChain, commonIndex)
	}

	return nil
}

// replaceChain replaces the blockchain with another chain
func (cr *ConsensusRules) replaceChain(localChain *blockchain.Blockchain, newChain *blockchain.Blockchain, commonIndex int) error {
	// Validate the new chain
	if err := cr.ValidateChain(newChain); err != nil {
		return fmt.Errorf("new chain validation failed: %w", err)
	}

	// Use the blockchain's built-in chain replacement method
	// This handles:
	// - Replacing blocks from common ancestor onwards
	// - Updating mining rewards
	// - Recalculating total rewards
	// - Maintaining blockchain integrity
	return localChain.ResolveFork(newChain)
}

// GetBlockWork calculates the work required to mine a block
func GetBlockWork(block *blockchain.Blockchain) *big.Int {
	return calculateTotalWork(block)
}

// calculateTotalWork calculates the total work of a blockchain
func calculateTotalWork(bc *blockchain.Blockchain) *big.Int {
	return calculateTotalWorkFromIndex(bc, 0)
}

// calculateTotalWorkFromIndex calculates the total work from a specific index
func calculateTotalWorkFromIndex(bc *blockchain.Blockchain, fromIndex int) *big.Int {
	totalWork := big.NewInt(0)

	for i := fromIndex; i < bc.GetChainLength(); i++ {
		block, err := bc.GetBlockByIndex(i)
		if err != nil {
			continue
		}

		// Work = 2^difficulty
		work := big.NewInt(1)
		work.Lsh(work, uint(block.Difficulty))
		totalWork.Add(totalWork, work)
	}

	return totalWork
}

// GetChainHeight returns the height of the blockchain
func (cr *ConsensusRules) GetChainHeight(bc *blockchain.Blockchain) int {
	return bc.GetChainLength() - 1
}

// GetChainTip returns the tip block of the blockchain
func (cr *ConsensusRules) GetChainTip(bc *blockchain.Blockchain) *blockchain.Block {
	return bc.GetLatestBlock()
}

// IsChainLonger checks if one chain is longer than another
func (cr *ConsensusRules) IsChainLonger(chainA, chainB *blockchain.Blockchain) bool {
	return chainA.GetChainLength() > chainB.GetChainLength()
}

// HasMoreWork checks if one chain has more work than another
func (cr *ConsensusRules) HasMoreWork(chainA, chainB *blockchain.Blockchain) bool {
	workA := calculateTotalWork(chainA)
	workB := calculateTotalWork(chainB)
	return workA.Cmp(workB) > 0
}

// GetForkPoint finds the point where two chains diverge
func (cr *ConsensusRules) GetForkPoint(chainA, chainB *blockchain.Blockchain) (int, error) {
	commonIndex := chainA.FindCommonAncestor(chainB)
	if commonIndex == -1 {
		return -1, fmt.Errorf("no common ancestor found")
	}
	return commonIndex, nil
}

// ValidateDifficultyTransition validates that difficulty transitions are valid
func (cr *ConsensusRules) ValidateDifficultyTransition(bc *blockchain.Blockchain, newBlock *blockchain.Block) error {
	cr.rulesMu.RLock()
	defer cr.rulesMu.RUnlock()

	if bc.GetChainLength() < cr.AdjustmentInterval {
		return nil // No difficulty adjustment needed yet
	}

	// Check if this is an adjustment block
	if (bc.GetChainLength() % cr.AdjustmentInterval) != 0 {
		// Difficulty should remain the same
		prevBlock := bc.GetLatestBlock()
		if prevBlock == nil {
			return fmt.Errorf("failed to get previous block")
		}

		if newBlock.Difficulty != prevBlock.Difficulty {
			return fmt.Errorf("difficulty changed at non-adjustment block: expected %d, got %d",
				prevBlock.Difficulty, newBlock.Difficulty)
		}

		return nil
	}

	// Calculate expected difficulty
	expectedDifficulty, err := cr.CalculateNewDifficulty(bc)
	if err != nil {
		return fmt.Errorf("failed to calculate new difficulty: %w", err)
	}

	if newBlock.Difficulty != expectedDifficulty {
		return fmt.Errorf("invalid difficulty at adjustment block: expected %d, got %d",
			expectedDifficulty, newBlock.Difficulty)
	}

	return nil
}

// GetConsensusInfo returns information about the consensus rules
func (cr *ConsensusRules) GetConsensusInfo() map[string]interface{} {
	cr.rulesMu.RLock()
	defer cr.rulesMu.RUnlock()

	return map[string]interface{}{
		"target_block_time":    cr.TargetBlockTime.String(),
		"adjustment_interval":  cr.AdjustmentInterval,
		"min_difficulty":       cr.MinDifficulty,
		"max_difficulty":       cr.MaxDifficulty,
		"max_block_size":       cr.MaxBlockSize,
		"max_tx_count":         cr.MaxTxCount,
		"coinbase_maturity":    cr.CoinbaseMaturity,
		"fork_tolerance":       cr.ForkTolerance,
		"min_confirmations":    cr.MinConfirmations,
	}
}

func rulesMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func rulesMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}