package consensus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/aliexe/blockChain/internal/blockchain"
	"github.com/aliexe/blockChain/internal/network"
)

// SyncManager manages blockchain synchronization between nodes
type SyncManager struct {
	localChain    *blockchain.Blockchain
	networkClient *network.Client
	syncing       bool
	syncMu        sync.RWMutex
	progress      *SyncProgress
	progressMu    sync.RWMutex
}

// SyncProgress tracks synchronization progress
type SyncProgress struct {
	TotalBlocks      int
	ReceivedBlocks   int
	CurrentHeight    int
	TargetHeight     int
	StartTime        time.Time
	LastUpdateTime   time.Time
	Errors           []string
	BytesReceived    int64
	BlocksPerSecond  float64
}

// SyncConfig holds synchronization configuration
type SyncConfig struct {
	MaxConcurrentRequests int
	BlockSize             int
	Timeout               time.Duration
	RetryAttempts         int
	VerifyBlocks          bool
}

// DefaultSyncConfig returns default synchronization configuration
func DefaultSyncConfig() SyncConfig {
	return SyncConfig{
		MaxConcurrentRequests: 5,
		BlockSize:             100,
		Timeout:               30 * time.Second,
		RetryAttempts:         3,
		VerifyBlocks:          true,
	}
}

// NewSyncManager creates a new synchronization manager
func NewSyncManager(localChain *blockchain.Blockchain) *SyncManager {
	return &SyncManager{
		localChain: localChain,
		syncing:    false,
		progress:   &SyncProgress{},
	}
}

// SetNetworkClient sets the network client for peer communication
func (sm *SyncManager) SetNetworkClient(client *network.Client) {
	sm.networkClient = client
}

// IsSyncing returns whether synchronization is in progress
func (sm *SyncManager) IsSyncing() bool {
	sm.syncMu.RLock()
	defer sm.syncMu.RUnlock()
	return sm.syncing
}

// SyncWithPeer synchronizes the blockchain with a specific peer
func (sm *SyncManager) SyncWithPeer(ctx context.Context, peerAddr string, config SyncConfig) error {
	sm.syncMu.Lock()
	if sm.syncing {
		sm.syncMu.Unlock()
		return fmt.Errorf("synchronization already in progress")
	}
	sm.syncing = true
	sm.syncMu.Unlock()

	defer func() {
		sm.syncMu.Lock()
		sm.syncing = false
		sm.syncMu.Unlock()
	}()

	// Initialize progress
	sm.progressMu.Lock()
	sm.progress = &SyncProgress{
		StartTime:      time.Now(),
		LastUpdateTime: time.Now(),
		CurrentHeight:  sm.localChain.GetChainLength() - 1,
	}
	sm.progressMu.Unlock()

	// Find common ancestor
	commonIndex, err := sm.findCommonAncestor(ctx, peerAddr)
	if err != nil {
		return fmt.Errorf("failed to find common ancestor: %w", err)
	}

	// Request missing blocks
	if err := sm.requestBlocks(ctx, peerAddr, commonIndex, config); err != nil {
		return fmt.Errorf("failed to request blocks: %w", err)
	}

	// Verify and integrate blocks
	if err := sm.verifyAndIntegrateBlocks(config.VerifyBlocks); err != nil {
		return fmt.Errorf("failed to verify blocks: %w", err)
	}

	return nil
}

// findCommonAncestor finds the last common block with a peer
func (sm *SyncManager) findCommonAncestor(ctx context.Context, peerAddr string) (int, error) {
	// Start from the tip of our chain and work backwards
	localHeight := sm.localChain.GetChainLength() - 1

	// Request peer's chain height
	peerHeight, err := sm.getPeerChainHeight(ctx, peerAddr)
	if err != nil {
		return -1, fmt.Errorf("failed to get peer chain height: %w", err)
	}

	// Find common ancestor by comparing hashes
	commonIndex := -1
	checkIndex := min(localHeight, peerHeight)

	for checkIndex >= 0 {
		localBlock, err := sm.localChain.GetBlockByIndex(checkIndex)
		if err != nil {
			return -1, fmt.Errorf("failed to get local block %d: %w", checkIndex, err)
		}

		peerBlock, err := sm.getPeerBlock(ctx, peerAddr, checkIndex)
		if err != nil {
			return -1, fmt.Errorf("failed to get peer block %d: %w", checkIndex, err)
		}

		// Compare hashes
		if string(localBlock.Hash) == string(peerBlock.Hash) {
			commonIndex = checkIndex
			break
		}

		checkIndex--
	}

	if commonIndex == -1 {
		return -1, fmt.Errorf("no common ancestor found")
	}

	sm.progressMu.Lock()
	sm.progress.CurrentHeight = commonIndex
	sm.progressMu.Unlock()

	return commonIndex, nil
}

// requestBlocks requests missing blocks from a peer
func (sm *SyncManager) requestBlocks(ctx context.Context, peerAddr string, fromIndex int, config SyncConfig) error {
	// Get peer's chain height
	peerHeight, err := sm.getPeerChainHeight(ctx, peerAddr)
	if err != nil {
		return fmt.Errorf("failed to get peer chain height: %w", err)
	}

	sm.progressMu.Lock()
	sm.progress.TargetHeight = peerHeight
	sm.progress.TotalBlocks = peerHeight - fromIndex - 1
	sm.progressMu.Unlock()

	// Request blocks in batches
	currentIndex := fromIndex + 1
	tempBlocks := make([]*blockchain.Block, 0, config.BlockSize)

	for currentIndex <= peerHeight {
		batchSize := syncMin(config.BlockSize, peerHeight-currentIndex+1)

		blocks, err := sm.getPeerBlocks(ctx, peerAddr, currentIndex, batchSize)
		if err != nil {
			return fmt.Errorf("failed to get blocks starting at %d: %w", currentIndex, err)
		}

		if len(blocks) == 0 {
			break
		}

		// Store blocks temporarily for verification
		tempBlocks = append(tempBlocks, blocks...)

		sm.progressMu.Lock()
		sm.progress.ReceivedBlocks = currentIndex - fromIndex
		sm.progress.BytesReceived += estimateBlocksSize(blocks)
		sm.progress.BlocksPerSecond = sm.calculateBlocksPerSecond()
		sm.progress.LastUpdateTime = time.Now()
		sm.progressMu.Unlock()

		currentIndex += len(blocks)

		// Verify batch periodically
		if len(tempBlocks) >= config.BlockSize || currentIndex > peerHeight {
			if err := sm.verifyBatch(tempBlocks, fromIndex); err != nil {
				return fmt.Errorf("batch verification failed: %w", err)
			}
			tempBlocks = tempBlocks[:0] // Clear batch
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

// verifyAndIntegrateBlocks verifies and integrates received blocks
func (sm *SyncManager) verifyAndIntegrateBlocks(verify bool) error {
	sm.progressMu.RLock()
	currentHeight := sm.progress.CurrentHeight
	sm.progressMu.RUnlock()

	// Get all blocks after the common ancestor
	blocks := make([]*blockchain.Block, 0)
	for i := currentHeight + 1; i < sm.localChain.GetChainLength(); i++ {
		block, err := sm.localChain.GetBlockByIndex(i)
		if err != nil {
			return fmt.Errorf("failed to get block %d: %w", i, err)
		}
		blocks = append(blocks, block)
	}

	// Verify each block
	if verify {
		for i, block := range blocks {
			if !block.IsValidProof() {
				return fmt.Errorf("block %d has invalid proof of work", currentHeight+1+i)
			}

			// Verify hash linking
			if i > 0 {
				prevBlock := blocks[i-1]
				if string(block.PrevHash) != string(prevBlock.Hash) {
					return fmt.Errorf("block %d has invalid previous hash", currentHeight+1+i)
				}
			} else {
				// Check against local chain
				prevBlock, err := sm.localChain.GetBlockByIndex(currentHeight)
				if err != nil {
					return fmt.Errorf("failed to get previous block: %w", err)
				}
				if string(block.PrevHash) != string(prevBlock.Hash) {
					return fmt.Errorf("block %d has invalid previous hash", currentHeight+1)
				}
			}
		}
	}

	return nil
}

// verifyBatch verifies a batch of blocks
func (sm *SyncManager) verifyBatch(blocks []*blockchain.Block, startIndex int) error {
	for i, block := range blocks {
		// Verify proof of work
		if !block.IsValidProof() {
			return fmt.Errorf("block %d has invalid proof of work", startIndex+i+1)
		}

		// Verify hash linking
		if i == 0 {
			// First block should link to our chain
			prevBlock, err := sm.localChain.GetBlockByIndex(startIndex)
			if err != nil {
				return fmt.Errorf("failed to get previous block: %w", err)
			}
			if string(block.PrevHash) != string(prevBlock.Hash) {
				return fmt.Errorf("block %d has invalid previous hash", startIndex+1)
			}
		} else {
			// Subsequent blocks should link to previous block in batch
			prevBlock := blocks[i-1]
			if string(block.PrevHash) != string(prevBlock.Hash) {
				return fmt.Errorf("block %d has invalid previous hash", startIndex+i+1)
			}
		}

		// Add block to local chain
		sm.localChain.AddBlock(string(block.Data))
	}

	return nil
}

// getPeerChainHeight gets the chain height from a peer
func (sm *SyncManager) getPeerChainHeight(ctx context.Context, peerAddr string) (int, error) {
	if sm.networkClient == nil {
		return -1, fmt.Errorf("network client not configured")
	}

	// Create a get blockchain info message
	msg := network.NewMessage(network.MessageTypeGetBlockchain, nil)

	// Send message to peer and wait for response
	responseChan := make(chan *network.Message, 1)
	errChan := make(chan error, 1)

	go func() {
		client := network.NewClient(peerAddr, func(peer *network.Peer, msg *network.Message) {
			if msg.Type == network.MessageTypeBlockchain {
				responseChan <- msg
			}
		})

		if err := client.Connect(); err != nil {
			errChan <- fmt.Errorf("failed to connect to peer %s: %w", peerAddr, err)
			return
		}
		defer client.Close()

		if err := client.Send(msg); err != nil {
			errChan <- fmt.Errorf("failed to send message to peer %s: %w", peerAddr, err)
			return
		}
	}()

	// Wait for response or timeout
	select {
	case <-ctx.Done():
		return -1, ctx.Err()
	case err := <-errChan:
		return -1, err
	case response := <-responseChan:
		// Parse the blockchain info from response
		var chainInfo struct {
			Height int `json:"height"`
		}
		if err := json.Unmarshal(response.Payload, &chainInfo); err != nil {
			return -1, fmt.Errorf("failed to parse chain info: %w", err)
		}
		return chainInfo.Height, nil
	case <-time.After(30 * time.Second):
		return -1, fmt.Errorf("timeout waiting for chain height from peer %s", peerAddr)
	}
}

// getPeerBlock gets a specific block from a peer
func (sm *SyncManager) getPeerBlock(ctx context.Context, peerAddr string, index int) (*blockchain.Block, error) {
	if sm.networkClient == nil {
		return nil, fmt.Errorf("network client not configured")
	}

	// Create a get blocks message
	msg, err := network.NewGetBlocksMessage(uint32(index), 1)
	if err != nil {
		return nil, fmt.Errorf("failed to create get blocks message: %w", err)
	}

	// Send message to peer and wait for response
	responseChan := make(chan *network.Message, 1)
	errChan := make(chan error, 1)

	go func() {
		client := network.NewClient(peerAddr, func(peer *network.Peer, msg *network.Message) {
			if msg.Type == network.MessageTypeBlocks {
				responseChan <- msg
			}
		})

		if err := client.Connect(); err != nil {
			errChan <- fmt.Errorf("failed to connect to peer %s: %w", peerAddr, err)
			return
		}
		defer client.Close()

		if err := client.Send(msg); err != nil {
			errChan <- fmt.Errorf("failed to send message to peer %s: %w", peerAddr, err)
			return
		}
	}()

	// Wait for response or timeout
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errChan:
		return nil, err
	case response := <-responseChan:
		// Parse the block from response
		var blocks []*blockchain.Block
		if err := json.Unmarshal(response.Payload, &blocks); err != nil {
			return nil, fmt.Errorf("failed to parse blocks: %w", err)
		}
		if len(blocks) == 0 {
			return nil, fmt.Errorf("no blocks received from peer %s", peerAddr)
		}
		return blocks[0], nil
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout waiting for block from peer %s", peerAddr)
	}
}

// getPeerBlocks gets a range of blocks from a peer
func (sm *SyncManager) getPeerBlocks(ctx context.Context, peerAddr string, startIndex, count int) ([]*blockchain.Block, error) {
	blocks := make([]*blockchain.Block, 0, count)

	for i := 0; i < count; i++ {
		block, err := sm.getPeerBlock(ctx, peerAddr, startIndex+i)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// GetProgress returns the current synchronization progress
func (sm *SyncManager) GetProgress() *SyncProgress {
	sm.progressMu.RLock()
	defer sm.progressMu.RUnlock()

	// Return a copy to avoid race conditions
	progress := &SyncProgress{
		TotalBlocks:      sm.progress.TotalBlocks,
		ReceivedBlocks:   sm.progress.ReceivedBlocks,
		CurrentHeight:    sm.progress.CurrentHeight,
		TargetHeight:     sm.progress.TargetHeight,
		StartTime:        sm.progress.StartTime,
		LastUpdateTime:   sm.progress.LastUpdateTime,
		Errors:           make([]string, len(sm.progress.Errors)),
		BytesReceived:    sm.progress.BytesReceived,
		BlocksPerSecond:  sm.progress.BlocksPerSecond,
	}
	copy(progress.Errors, sm.progress.Errors)

	return progress
}

// calculateBlocksPerSecond calculates the current synchronization speed
func (sm *SyncManager) calculateBlocksPerSecond() float64 {
	sm.progressMu.RLock()
	defer sm.progressMu.RUnlock()

	if sm.progress.ReceivedBlocks == 0 {
		return 0
	}

	elapsed := time.Since(sm.progress.StartTime).Seconds()
	if elapsed == 0 {
		return 0
	}

	return float64(sm.progress.ReceivedBlocks) / elapsed
}

// estimateBlocksSize estimates the size of a block slice
func estimateBlocksSize(blocks []*blockchain.Block) int64 {
	var size int64
	for _, block := range blocks {
		size += int64(len(block.Data)) + 100 // Rough estimate
	}
	return size
}

// CancelSync cancels the current synchronization
func (sm *SyncManager) CancelSync() {
	sm.syncMu.Lock()
	defer sm.syncMu.Unlock()

	if sm.syncing {
		sm.syncing = false
		sm.progressMu.Lock()
		sm.progress.Errors = append(sm.progress.Errors, "Synchronization cancelled")
		sm.progressMu.Unlock()
	}
}

// GetSyncStats returns synchronization statistics
func (sm *SyncManager) GetSyncStats() map[string]interface{} {
	sm.progressMu.RLock()
	defer sm.progressMu.RUnlock()

	elapsed := time.Since(sm.progress.StartTime)
	percentage := 0.0
	if sm.progress.TotalBlocks > 0 {
		percentage = float64(sm.progress.ReceivedBlocks) / float64(sm.progress.TotalBlocks) * 100
	}

	return map[string]interface{}{
		"syncing":          sm.syncing,
		"total_blocks":     sm.progress.TotalBlocks,
		"received_blocks":  sm.progress.ReceivedBlocks,
		"current_height":   sm.progress.CurrentHeight,
		"target_height":    sm.progress.TargetHeight,
		"progress_percent": percentage,
		"elapsed_time":     elapsed.String(),
		"blocks_per_sec":   sm.progress.BlocksPerSecond,
		"bytes_received":   sm.progress.BytesReceived,
		"error_count":      len(sm.progress.Errors),
	}
}

func syncMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}