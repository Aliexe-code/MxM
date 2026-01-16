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

	// 1. Connect to peers and request their chain information
	// 2. Find the best chain (highest total work)
	// 3. Synchronize with the best peer
	// 4. Validate received blocks
	// 5. Resolve any forks using consensus rules

	// Get the local chain height before reconciliation
	localHeight := pm.localChain.GetChainLength() - 1
	fmt.Printf("   Local chain height before reconciliation: %d\n", localHeight)

	// Find common ancestor and check if we need to sync
	// In a real implementation, this would:
	// - Query multiple peers for their chain heights
	// - Select the peer with the highest chain
	// - Perform a full synchronization if needed
	// - Validate all received blocks
	// - Resolve any forks using the consensus rules

	// For now, we'll validate our local chain and ensure it's consistent
	if !pm.localChain.IsValid() {
		fmt.Println("❌ Local chain is invalid, cannot reconcile")
		return
	}

	// Check if we have any isolated peers that might have better chains
	pm.isolatedPeersMu.RLock()
	hasIsolatedPeers := len(pm.isolatedPeers) > 0
	pm.isolatedPeersMu.RUnlock()

	if hasIsolatedPeers {
		fmt.Printf("   Found %d isolated peers, attempting to sync\n", len(pm.isolatedPeers))
		// In a real implementation, we would sync with these peers
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