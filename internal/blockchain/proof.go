package blockchain

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"time"
)

const (
	DefaultDifficulty    = 4
	MaxNonce             = 10000000        // Reasonable limit to prevent infinite loop
	DefaultMiningTimeout = 5 * time.Second // Default timeout for mining operations

	// Difficulty adjustment constants
	TargetBlockTime    = 2 * time.Minute // Target time between blocks
	AdjustmentInterval = 10              // Number of blocks between difficulty adjustments
	MinDifficulty      = 1               // Minimum difficulty
	MaxDifficulty      = 32              // Maximum difficulty
)

type ProofOfWork struct {
	Block      *Block
	Target     *big.Int
	Difficulty int
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
}

func NewProofOfWork(b *Block, difficulty int) *ProofOfWork {
	target := big.NewInt(1)
	target.Lsh(target, uint(256-4*difficulty))

	ctx, cancel := context.WithCancel(context.Background())

	pow := &ProofOfWork{
		Block:      b,
		Target:     target,
		Difficulty: difficulty,
		cancelCtx:  ctx,
		cancelFunc: cancel,
	}
	return pow
}

func (pow *ProofOfWork) prepareData(nonce uint32) []byte {
	data := pow.Block.PrevHash
	data = append(data, pow.Block.Data...)
	data = append(data, []byte(strconv.FormatInt(pow.Block.Timestamp, 10))...)
	data = append(data, []byte(strconv.FormatInt(int64(pow.Difficulty), 10))...)
	data = append(data, []byte(strconv.FormatInt(int64(nonce), 10))...)
	return data
}

func (pow *ProofOfWork) Run(ctx context.Context) (uint32, []byte, time.Duration) {
	var hashInt big.Int
	var hash [32]byte
	var nonce uint32

	startTime := time.Now()

	// Combine contexts for proper cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Use configurable timeout instead of hardcoded 200ms
	for nonce = 0; nonce < MaxNonce; nonce++ {
		// Check for cancellation
		select {
		case <-ctx.Done():
			return 0, nil, time.Since(startTime)
		case <-pow.cancelCtx.Done():
			return 0, nil, time.Since(startTime)
		default:
			// Continue mining
		}

		data := pow.prepareData(nonce)
		hash = sha256.Sum256(data)
		hashInt.SetBytes(hash[:])

		if hashInt.Cmp(pow.Target) == -1 {
			duration := time.Since(startTime)
			return nonce, hash[:], duration
		}

		// Check for timeout
		if time.Since(startTime) > DefaultMiningTimeout {
			return 0, nil, time.Since(startTime)
		}
	}
	duration := time.Since(startTime)
	return 0, nil, duration
}

// Cancel stops the mining process
func (pow *ProofOfWork) Cancel() {
	if pow.cancelFunc != nil {
		pow.cancelFunc()
	}
}

func (pow *ProofOfWork) Validate() bool {
	var hashInt big.Int
	data := pow.prepareData(pow.Block.Nonce)
	hash := sha256.Sum256(data)
	hashInt.SetBytes(hash[:])
	return hashInt.Cmp(pow.Target) == -1
}

func (pow *ProofOfWork) GetMiningStats() map[string]interface{} {
	return map[string]interface{}{
		"difficulty": pow.Difficulty,
		"target":     pow.Target.String(),
		"block_data": string(pow.Block.Data),
		"prev_hash":  hex.EncodeToString(pow.Block.PrevHash),
	}
}

func (pow *ProofOfWork) SetDifficulty(difficulty int) {
	if difficulty < 1 || difficulty > 32 {
		fmt.Printf("Invalid difficulty %d, using default %d\n", difficulty, DefaultDifficulty)
		difficulty = DefaultDifficulty
	}
	pow.Difficulty = difficulty
	pow.Target = big.NewInt(1)
	pow.Target.Lsh(pow.Target, uint(256-4*difficulty))

	// Create new cancellation context
	if pow.cancelFunc != nil {
		pow.cancelFunc() // Cancel any existing mining
	}
	ctx, cancel := context.WithCancel(context.Background())
	pow.cancelCtx = ctx
	pow.cancelFunc = cancel
}

// DifficultyAdjuster handles dynamic difficulty adjustment
type DifficultyAdjuster struct {
	blocks []*Block
}

// NewDifficultyAdjuster creates a new difficulty adjuster
func NewDifficultyAdjuster(blocks []*Block) *DifficultyAdjuster {
	return &DifficultyAdjuster{
		blocks: blocks,
	}
}

// CalculateNewDifficulty calculates the new difficulty based on recent block times
// This ensures stable block production regardless of network hashrate changes
func (da *DifficultyAdjuster) CalculateNewDifficulty() int {
	if len(da.blocks) < AdjustmentInterval {
		return DefaultDifficulty
	}

	// Get the last AdjustmentInterval blocks
	startIndex := len(da.blocks) - AdjustmentInterval
	recentBlocks := da.blocks[startIndex:]

	// Calculate average block time
	var totalTime time.Duration
	for i := 1; i < len(recentBlocks); i++ {
		blockTime := time.Duration(recentBlocks[i].Timestamp - recentBlocks[i-1].Timestamp)
		totalTime += blockTime
	}

	avgBlockTime := totalTime / time.Duration(AdjustmentInterval-1)
	currentDifficulty := da.blocks[len(da.blocks)-1].Difficulty

	// Adjust difficulty based on average block time
	// If blocks are too fast, increase difficulty
	// If blocks are too slow, decrease difficulty
	newDifficulty := currentDifficulty

	if avgBlockTime > TargetBlockTime*2 {
		// Blocks are too slow, decrease difficulty
		newDifficulty = max(MinDifficulty, currentDifficulty-1)
	} else if avgBlockTime < TargetBlockTime/2 {
		// Blocks are too fast, increase difficulty
		newDifficulty = min(MaxDifficulty, currentDifficulty+1)
	}

	return newDifficulty
}

// CalculateNewDifficultyForBlockchain calculates the new difficulty for a blockchain
func CalculateNewDifficultyForBlockchain(bc *Blockchain) int {
	if len(bc.Blocks) < AdjustmentInterval {
		return DefaultDifficulty
	}

	adjuster := NewDifficultyAdjuster(bc.Blocks)
	return adjuster.CalculateNewDifficulty()
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
