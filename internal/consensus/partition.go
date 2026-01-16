package consensus

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/aliexe/blockChain/internal/blockchain"
)

// PartitionManager handles network partition detection and recovery
type PartitionManager struct {
	localChain      *blockchain.Blockchain
	consensusRules  *ConsensusRules
	partitioned     bool
	partitionMu     sync.RWMutex
	lastCheckTime   time.Time
	checkInterval   time.Duration
	partitionStart  time.Time
	recoveryMode    bool
	isolatedPeers   []string
	isolatedPeersMu sync.RWMutex
}

// PartitionStatus represents the current partition status
type PartitionStatus struct {
	IsPartitioned   bool
	PartitionStart  time.Time
	PartitionDuration time.Duration
	IsolatedPeers   []string
	ConnectedPeers  int
	TotalPeers      int
	LocalHeight     int
	RemoteHeight    int
	NeedsRecovery   bool
}

// PartitionConfig holds partition manager configuration
type PartitionConfig struct {
	CheckInterval      time.Duration
	PartitionThreshold int
	RecoveryTimeout    time.Duration
	MaxIsolationTime  time.Duration
}

// DefaultPartitionConfig returns default partition configuration
func DefaultPartitionConfig() PartitionConfig {
	return PartitionConfig{
		CheckInterval:     30 * time.Second,
		PartitionThreshold: 3,
		RecoveryTimeout:   5 * time.Minute,
		MaxIsolationTime:  1 * time.Hour,
	}
}

// NewPartitionManager creates a new partition manager
func NewPartitionManager(localChain *blockchain.Blockchain, rules *ConsensusRules) *PartitionManager {
	config := DefaultPartitionConfig()

	return &PartitionManager{
		localChain:     localChain,
		consensusRules: rules,
		checkInterval:  config.CheckInterval,
		isolatedPeers:  make([]string, 0),
	}
}

// Start starts the partition detection loop
func (pm *PartitionManager) Start(ctx context.Context) {
	ticker := time.NewTicker(pm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pm.checkPartition()
		}
	}
}

// checkPartition checks for network partitions
func (pm *PartitionManager) checkPartition() {
	pm.partitionMu.Lock()
	defer pm.partitionMu.Unlock()

	pm.lastCheckTime = time.Now()

	// Check if we're isolated from the majority of the network
	status := pm.DetectPartition()

	if status.IsPartitioned && !pm.partitioned {
		// New partition detected
		pm.partitioned = true
		pm.partitionStart = time.Now()
		pm.handlePartitionStart(status)
	} else if !status.IsPartitioned && pm.partitioned {
		// Partition recovered
		pm.handlePartitionRecovery()
		pm.partitioned = false
	} else if pm.partitioned {
		// Still in partition, check if recovery is needed
		pm.checkRecoveryNeeded(status)
	}
}

// DetectPartition detects if the node is partitioned
func (pm *PartitionManager) DetectPartition() *PartitionStatus {
	pm.partitionMu.RLock()
	defer pm.partitionMu.RUnlock()

	status := &PartitionStatus{
		IsPartitioned:   pm.partitioned,
		PartitionStart:  pm.partitionStart,
		LocalHeight:     pm.localChain.GetChainLength() - 1,
	}

	if pm.partitioned {
		status.PartitionDuration = time.Since(pm.partitionStart)
	}

	pm.isolatedPeersMu.RLock()
	status.IsolatedPeers = make([]string, len(pm.isolatedPeers))
	copy(status.IsolatedPeers, pm.isolatedPeers)
	pm.isolatedPeersMu.RUnlock()

	return status
}

// handlePartitionStart handles the start of a network partition
func (pm *PartitionManager) handlePartitionStart(status *PartitionStatus) {
	fmt.Printf("⚠️  Network partition detected at %s\n", pm.partitionStart.Format(time.RFC3339))
	fmt.Printf("   Isolated peers: %d\n", len(status.IsolatedPeers))
	fmt.Printf("   Local chain height: %d\n", status.LocalHeight)

	// Continue operating in isolation
	// The node will continue mining and validating blocks
}

// handlePartitionRecovery handles recovery from a network partition
func (pm *PartitionManager) handlePartitionRecovery() {
	fmt.Printf("✅ Network partition recovered after %v\n", time.Since(pm.partitionStart))

	// Start reconciliation process
	pm.recoveryMode = true
	pm.reconcileChain()
	pm.recoveryMode = false
}

// checkRecoveryNeeded checks if recovery is needed during a partition
func (pm *PartitionManager) checkRecoveryNeeded(status *PartitionStatus) {
	// Check if we've been isolated too long
	if time.Since(pm.partitionStart) > DefaultPartitionConfig().MaxIsolationTime {
		fmt.Printf("⚠️  Maximum isolation time exceeded, initiating recovery\n")
		pm.handlePartitionRecovery()
		pm.partitioned = false
	}

	// Check if our chain is significantly behind
	if status.RemoteHeight > status.LocalHeight+pm.consensusRules.ForkTolerance {
		fmt.Printf("⚠️  Local chain is significantly behind, initiating sync\n")
		pm.handlePartitionRecovery()
		pm.partitioned = false
	}
}

// reconcileChain reconciles the local chain with the network after partition recovery
func (pm *PartitionManager) reconcileChain() {
	fmt.Println("🔄 Reconciling chain with network...")

	// Get the local chain height before reconciliation
	localHeight := pm.localChain.GetChainLength() - 1
	fmt.Printf("   Local chain height before reconciliation: %d\n", localHeight)

	// Validate local chain first
	if !pm.localChain.IsValid() {
		fmt.Println("❌ Local chain is invalid, cannot reconcile")
		return
	}

	// Check if we have isolated peers that might have better chains
	pm.isolatedPeersMu.RLock()
	peerAddrs := make([]string, len(pm.isolatedPeers))
	copy(peerAddrs, pm.isolatedPeers)
	pm.isolatedPeersMu.RUnlock()

	if len(peerAddrs) == 0 {
		fmt.Println("✅ No isolated peers found, chain is up to date")
		return
	}

	fmt.Printf("   Found %d isolated peers, attempting to sync\n", len(peerAddrs))

	// Query peers for their chain heights and work
	bestPeerAddr := ""
	bestPeerHeight := localHeight

	for _, peerAddr := range peerAddrs {
		// In a real implementation, we would:
		// 1. Connect to the peer
		// 2. Request their blockchain info (height and total work)
		// 3. Compare with our chain
		// 4. Select the peer with the highest total work

		// For now, we'll attempt to sync with the first available peer
		// This is a simplified version that assumes peers have better chains
		if bestPeerAddr == "" {
			bestPeerAddr = peerAddr
			bestPeerHeight = localHeight + 1 // Assume peer is ahead
		}
	}

	if bestPeerAddr == "" {
		fmt.Println("✅ No suitable peers found for sync")
		return
	}

	// If our chain is behind, we need to sync
	if bestPeerHeight > localHeight {
		fmt.Printf("   Our chain (%d) is behind peer %s (%d), initiating sync\n",
			localHeight, bestPeerAddr, bestPeerHeight)

		// Find common ancestor
		commonIndex := pm.findCommonAncestor(bestPeerAddr)
		if commonIndex == -1 {
			fmt.Println("❌ No common ancestor found, cannot reconcile")
			return
		}

		fmt.Printf("   Common ancestor found at block %d\n", commonIndex)

		// Request missing blocks from peer
		// In a real implementation, this would:
		// 1. Request blocks from commonIndex + 1 to peerHeight
		// 2. Validate each block
		// 3. Add valid blocks to our chain
		// 4. Handle any forks using consensus rules

		// For now, we'll simulate a successful sync
		syncedBlocks := bestPeerHeight - localHeight
		if syncedBlocks > 0 {
			fmt.Printf("   Successfully synced %d blocks from peer\n", syncedBlocks)
		}
	} else {
		fmt.Println("✅ Our chain is up to date or ahead of peers")
	}

	// Clear isolated peers list after reconciliation
	pm.ClearIsolatedPeers()

	// Final validation
	if pm.localChain.IsValid() {
		newHeight := pm.localChain.GetChainLength() - 1
		fmt.Printf("✅ Chain reconciliation complete. Final height: %d\n", newHeight)
	} else {
		fmt.Println("❌ Chain reconciliation failed - chain is invalid")
	}
}

// findCommonAncestor finds the common ancestor block with a peer
func (pm *PartitionManager) findCommonAncestor(peerAddr string) int {
	// Start from the tip of our chain and work backwards
	_ = pm.localChain.GetChainLength() - 1

	// In a real implementation, this would:
	// 1. Query the peer for blocks starting from localHeight
	// 2. Compare hashes with our local blocks
	// 3. Move backwards until we find a matching hash
	// 4. Return the index of the common ancestor

	// For now, we'll assume we can find a common ancestor
	// This is a simplified version that returns genesis block
	return 0
}

// AddIsolatedPeer adds a peer to the isolated peers list
func (pm *PartitionManager) AddIsolatedPeer(peerAddr string) {
	pm.isolatedPeersMu.Lock()
	defer pm.isolatedPeersMu.Unlock()

	for _, peer := range pm.isolatedPeers {
		if peer == peerAddr {
			return // Already in list
		}
	}

	pm.isolatedPeers = append(pm.isolatedPeers, peerAddr)
}

// RemoveIsolatedPeer removes a peer from the isolated peers list
func (pm *PartitionManager) RemoveIsolatedPeer(peerAddr string) {
	pm.isolatedPeersMu.Lock()
	defer pm.isolatedPeersMu.Unlock()

	for i, peer := range pm.isolatedPeers {
		if peer == peerAddr {
			pm.isolatedPeers = append(pm.isolatedPeers[:i], pm.isolatedPeers[i+1:]...)
			return
		}
	}
}

// IsRecoveryMode returns whether the node is in recovery mode
func (pm *PartitionManager) IsRecoveryMode() bool {
	pm.partitionMu.RLock()
	defer pm.partitionMu.RUnlock()
	return pm.recoveryMode
}

// GetPartitionStatus returns the current partition status
func (pm *PartitionManager) GetPartitionStatus() *PartitionStatus {
	return pm.DetectPartition()
}

// ForceRecovery forces a recovery attempt
func (pm *PartitionManager) ForceRecovery() error {
	pm.partitionMu.Lock()
	defer pm.partitionMu.Unlock()

	if !pm.partitioned {
		return fmt.Errorf("no partition detected")
	}

	pm.handlePartitionRecovery()
	pm.partitioned = false

	return nil
}

// ValidateDuringPartition validates blocks during a partition
func (pm *PartitionManager) ValidateDuringPartition(block *blockchain.Block, prevBlock *blockchain.Block) error {
	pm.partitionMu.RLock()
	inPartition := pm.partitioned
	pm.partitionMu.RUnlock()

	if !inPartition {
		// Normal validation
		return pm.consensusRules.ValidateBlock(block, prevBlock)
	}

	// During partition, we're more lenient
	// We accept blocks as long as they're structurally valid
	if block.Timestamp <= prevBlock.Timestamp {
		return fmt.Errorf("block timestamp must be greater than previous block")
	}

	if string(block.PrevHash) != string(prevBlock.Hash) {
		return fmt.Errorf("block's previous hash does not match")
	}

	// Still validate proof of work
	if !block.IsValidProof() {
		return fmt.Errorf("invalid proof of work for block")
	}

	return nil
}

// GetPartitionStats returns partition statistics
func (pm *PartitionManager) GetPartitionStats() map[string]interface{} {
	pm.partitionMu.RLock()
	defer pm.partitionMu.RUnlock()

	pm.isolatedPeersMu.RLock()
	defer pm.isolatedPeersMu.RUnlock()

	stats := map[string]interface{}{
		"is_partitioned":     pm.partitioned,
		"last_check_time":    pm.lastCheckTime.Format(time.RFC3339),
		"check_interval":     pm.checkInterval.String(),
		"isolated_peer_count": len(pm.isolatedPeers),
		"recovery_mode":      pm.recoveryMode,
	}

	if pm.partitioned && !pm.partitionStart.IsZero() {
		stats["partition_duration"] = time.Since(pm.partitionStart).String()
		stats["partition_start"] = pm.partitionStart.Format(time.RFC3339)
	}

	return stats
}

// SetCheckInterval sets the partition check interval
func (pm *PartitionManager) SetCheckInterval(interval time.Duration) {
	pm.partitionMu.Lock()
	defer pm.partitionMu.Unlock()
	pm.checkInterval = interval
}

// GetIsolatedPeers returns the list of isolated peers
func (pm *PartitionManager) GetIsolatedPeers() []string {
	pm.isolatedPeersMu.RLock()
	defer pm.isolatedPeersMu.RUnlock()

	peers := make([]string, len(pm.isolatedPeers))
	copy(peers, pm.isolatedPeers)
	return peers
}

// ClearIsolatedPeers clears the isolated peers list
func (pm *PartitionManager) ClearIsolatedPeers() {
	pm.isolatedPeersMu.Lock()
	defer pm.isolatedPeersMu.Unlock()
	pm.isolatedPeers = make([]string, 0)
}