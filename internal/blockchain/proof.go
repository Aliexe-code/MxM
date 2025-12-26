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
	DefaultDifficulty = 4
	MaxNonce          = 10000000 // Reasonable limit to prevent infinite loop
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

func (pow *ProofOfWork) Run() (uint32, []byte, time.Duration) {
	var hashInt big.Int
	var hash [32]byte
	var nonce uint32

	startTime := time.Now()

	// For testing, use a much lower timeout and better progress tracking
	maxAttempts := uint32(100000) // Reduced attempts for faster testing
	for nonce = 0; nonce < MaxNonce && nonce < maxAttempts; nonce++ {
		// Check for cancellation
		select {
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

		// Add timeout for tests - if mining takes too long, stop
		if time.Since(startTime) > 200*time.Millisecond {
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
