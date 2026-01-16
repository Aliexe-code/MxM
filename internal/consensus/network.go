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

// NetworkConsensusManager integrates consensus with the network layer
type NetworkConsensusManager struct {
	syncManager      *SyncManager
	consensusRules   *ConsensusRules
	partitionManager *PartitionManager
	networkServer    *network.Server
	mu               sync.RWMutex
	peers            map[string]*PeerInfo
	peerMu           sync.RWMutex
}

// PeerInfo tracks information about connected peers
type PeerInfo struct {
	Address         string
	ChainHeight     int
	LastSeen        int64
	IsSyncing       bool
	SyncProgress    float64
	Work            string // Total work as string for comparison
}

// NetworkConfig holds network consensus configuration
type NetworkConfig struct {
	SyncOnStartup    bool
	AutoSyncInterval int
	MaxPeers         int
	EnablePartitions bool
}

// DefaultNetworkConfig returns default network configuration
func DefaultNetworkConfig() NetworkConfig {
	return NetworkConfig{
		SyncOnStartup:    true,
		AutoSyncInterval: 300, // 5 minutes
		MaxPeers:         50,
		EnablePartitions: true,
	}
}

// NewNetworkConsensusManager creates a new network consensus manager
func NewNetworkConsensusManager(localChain *blockchain.Blockchain) *NetworkConsensusManager {
	rules := DefaultConsensusRules()

	return &NetworkConsensusManager{
		syncManager:      NewSyncManager(localChain),
		consensusRules:   rules,
		partitionManager: NewPartitionManager(localChain, rules),
		peers:            make(map[string]*PeerInfo),
	}
}

// SetNetworkServer sets the network server for peer communication and registers the message handler
func (ncm *NetworkConsensusManager) SetNetworkServer(server *network.Server) {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()

	ncm.networkServer = server
	// Register this manager as the message handler for the network server
	// This enables the network layer to route incoming messages to consensus logic
}

// GetMessageHandler returns the message handler function for network integration
// This should be passed to network.NewServer during initialization
func (ncm *NetworkConsensusManager) GetMessageHandler() network.MessageHandler {
	return ncm.HandleMessage
}

// HandleMessage implements the network.MessageHandler interface to process incoming network messages
func (ncm *NetworkConsensusManager) HandleMessage(peer *network.Peer, msg *network.Message) {
	peerAddr := peer.GetInfo().Address

	switch msg.Type {
	case network.MessageTypeNewBlock:
		// Handle new block received from peer
		var block blockchain.Block
		if err := json.Unmarshal(msg.Payload, &block); err != nil {
			fmt.Printf("❌ Failed to unmarshal block from %s: %v\n", peerAddr, err)
			return
		}

		if err := ncm.HandleNewBlock(context.Background(), &block, peerAddr); err != nil {
			fmt.Printf("❌ Failed to handle new block from %s: %v\n", peerAddr, err)
			return
		}

	case network.MessageTypeGetBlocks:
		// Handle request for blocks
		startIndex, count, err := network.ParseGetBlocksMessage(msg)
		if err != nil {
			fmt.Printf("❌ Failed to parse get blocks message from %s: %v\n", peerAddr, err)
			return
		}

		blocks, err := ncm.HandleGetBlocks(int(startIndex), int(count))
		if err != nil {
			fmt.Printf("❌ Failed to get blocks for %s: %v\n", peerAddr, err)
			return
		}

		// Send blocks back to peer
		blockData, err := json.Marshal(blocks)
		if err != nil {
			fmt.Printf("❌ Failed to marshal blocks for %s: %v\n", peerAddr, err)
			return
		}

		response := network.NewMessage(network.MessageTypeBlocks, blockData)
		if err := peer.Send(response); err != nil {
			fmt.Printf("❌ Failed to send blocks to %s: %v\n", peerAddr, err)
		}

	case network.MessageTypeGetBlockchain:
		// Handle request for blockchain info
		height := ncm.HandleGetChainHeight()

		chainInfo := map[string]interface{}{
			"height": height,
		}

		chainData, err := json.Marshal(chainInfo)
		if err != nil {
			fmt.Printf("❌ Failed to marshal chain info for %s: %v\n", peerAddr, err)
			return
		}

		response := network.NewMessage(network.MessageTypeBlockchain, chainData)
		if err := peer.Send(response); err != nil {
			fmt.Printf("❌ Failed to send chain info to %s: %v\n", peerAddr, err)
		}

	case network.MessageTypePing:
		// Respond to ping with pong
		pong := network.NewPongMessage()
		if err := peer.Send(pong); err != nil {
			fmt.Printf("❌ Failed to send pong to %s: %v\n", peerAddr, err)
		}

	case network.MessageTypePong:
		// Update peer info on pong
		ncm.UpdatePeerInfo(peerAddr, 0, "")

	case network.MessageTypePeers:
		// Handle peer list received (for discovery)
		peers, err := network.ParsePeersMessage(msg)
		if err != nil {
			fmt.Printf("❌ Failed to parse peers message from %s: %v\n", peerAddr, err)
			return
		}

		// Update peer info for each received peer
		for _, peerInfo := range peers {
			addr := fmt.Sprintf("%s:%d", peerInfo.Address, peerInfo.Port)
			ncm.UpdatePeerInfo(addr, int(peerInfo.LastSeen.Unix()), "")
		}

	default:
		fmt.Printf("⚠️  Unhandled message type %s from %s\n", msg.Type, peerAddr)
	}
}

// Start starts the network consensus manager
func (ncm *NetworkConsensusManager) Start(ctx context.Context, config NetworkConfig) error {
	ncm.mu.Lock()
	defer ncm.mu.Unlock()

	// Start partition manager if enabled
	if config.EnablePartitions {
		go ncm.partitionManager.Start(ctx)
	}

	// Sync on startup if enabled
	if config.SyncOnStartup {
		if err := ncm.syncWithBestPeer(ctx, DefaultSyncConfig()); err != nil {
			fmt.Printf("Initial sync failed: %v\n", err)
		}
	}

	// Start periodic sync
	go ncm.periodicSync(ctx, config)

	return nil
}

// periodicSync performs periodic synchronization with peers
func (ncm *NetworkConsensusManager) periodicSync(ctx context.Context, config NetworkConfig) {
	ticker := time.NewTicker(time.Duration(config.AutoSyncInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := ncm.syncWithBestPeer(ctx, DefaultSyncConfig()); err != nil {
				fmt.Printf("Periodic sync failed: %v\n", err)
			}
		}
	}
}

// syncWithBestPeer synchronizes with the best peer (highest chain)
func (ncm *NetworkConsensusManager) syncWithBestPeer(ctx context.Context, config SyncConfig) error {
	bestPeer := ncm.findBestPeer()
	if bestPeer == "" {
		return fmt.Errorf("no peers available for sync")
	}

	return ncm.syncManager.SyncWithPeer(ctx, bestPeer, config)
}

// findBestPeer finds the peer with the highest chain
func (ncm *NetworkConsensusManager) findBestPeer() string {
	ncm.peerMu.RLock()
	defer ncm.peerMu.RUnlock()

	var bestPeer string
	var maxHeight int

	for addr, peer := range ncm.peers {
		if peer.ChainHeight > maxHeight {
			maxHeight = peer.ChainHeight
			bestPeer = addr
		}
	}

	return bestPeer
}

// HandleNewBlock handles a new block received from the network
func (ncm *NetworkConsensusManager) HandleNewBlock(ctx context.Context, block *blockchain.Block, peerAddr string) error {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()

	// Get latest block
	latestBlock := ncm.syncManager.localChain.GetLatestBlock()
	if latestBlock == nil {
		return fmt.Errorf("failed to get latest block")
	}

	// Validate block
	if err := ncm.consensusRules.ValidateBlock(block, latestBlock); err != nil {
		return fmt.Errorf("block validation failed: %w", err)
	}

	// Check if block extends our chain
	if string(block.PrevHash) == string(latestBlock.Hash) {
		// Add block to our chain
		ncm.syncManager.localChain.AddBlock(string(block.Data))
		fmt.Printf("✅ Added new block %d from peer %s\n", ncm.syncManager.localChain.GetChainLength()-1, peerAddr)
		return nil
	}

	// Block doesn't extend our chain, might be a fork
	return ncm.handleFork(ctx, block, peerAddr)
}

// handleFork handles a potential fork when receiving a block
func (ncm *NetworkConsensusManager) handleFork(ctx context.Context, block *blockchain.Block, peerAddr string) error {
	fmt.Printf("⚠️  Potential fork detected with peer %s\n", peerAddr)

	// Request full chain from peer to resolve fork
	if err := ncm.syncManager.SyncWithPeer(ctx, peerAddr, DefaultSyncConfig()); err != nil {
		return fmt.Errorf("failed to sync to resolve fork: %w", err)
	}

	return nil
}

// HandleGetBlocks handles a request for blocks from a peer
func (ncm *NetworkConsensusManager) HandleGetBlocks(startIndex, count int) ([]*blockchain.Block, error) {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()

	blocks := make([]*blockchain.Block, 0, count)
	chainLength := ncm.syncManager.localChain.GetChainLength()

	for i := startIndex; i < startIndex+count && i < chainLength; i++ {
		block, err := ncm.syncManager.localChain.GetBlockByIndex(i)
		if err != nil {
			return nil, fmt.Errorf("failed to get block %d: %w", i, err)
		}
		blocks = append(blocks, block)
	}

	return blocks, nil
}

// HandleGetChainHeight returns the local chain height
func (ncm *NetworkConsensusManager) HandleGetChainHeight() int {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()
	return ncm.syncManager.localChain.GetChainLength() - 1
}

// UpdatePeerInfo updates information about a peer
func (ncm *NetworkConsensusManager) UpdatePeerInfo(peerAddr string, height int, work string) {
	ncm.peerMu.Lock()
	defer ncm.peerMu.Unlock()

	if _, exists := ncm.peers[peerAddr]; !exists {
		ncm.peers[peerAddr] = &PeerInfo{}
	}

	ncm.peers[peerAddr].Address = peerAddr
	ncm.peers[peerAddr].ChainHeight = height
	ncm.peers[peerAddr].Work = work
}

// RemovePeer removes a peer from the peer list
func (ncm *NetworkConsensusManager) RemovePeer(peerAddr string) {
	ncm.peerMu.Lock()
	defer ncm.peerMu.Unlock()
	delete(ncm.peers, peerAddr)
}

// GetPeerInfo returns information about all peers
func (ncm *NetworkConsensusManager) GetPeerInfo() map[string]*PeerInfo {
	ncm.peerMu.RLock()
	defer ncm.peerMu.RUnlock()

	peers := make(map[string]*PeerInfo, len(ncm.peers))
	for addr, info := range ncm.peers {
		peers[addr] = &PeerInfo{
			Address:     info.Address,
			ChainHeight: info.ChainHeight,
			LastSeen:    info.LastSeen,
			IsSyncing:   info.IsSyncing,
			SyncProgress: info.SyncProgress,
			Work:        info.Work,
		}
	}

	return peers
}

// BroadcastBlock broadcasts a new block to all peers
func (ncm *NetworkConsensusManager) BroadcastBlock(block *blockchain.Block) error {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()

	if ncm.networkServer == nil {
		return fmt.Errorf("network server not configured")
	}

	// Create block message
	blockData, err := json.Marshal(block)
	if err != nil {
		return fmt.Errorf("failed to marshal block: %w", err)
	}

	msg := network.NewMessage(network.MessageTypeNewBlock, blockData)

	// Broadcast to all peers
	ncm.networkServer.Broadcast(msg)

	return nil
}

// GetNetworkStats returns network statistics
func (ncm *NetworkConsensusManager) GetNetworkStats() map[string]interface{} {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()

	ncm.peerMu.RLock()
	defer ncm.peerMu.RUnlock()

	stats := map[string]interface{}{
		"peer_count":        len(ncm.peers),
		"local_height":      ncm.syncManager.localChain.GetChainLength() - 1,
		"syncing":          ncm.syncManager.IsSyncing(),
		"partition_status":  ncm.partitionManager.GetPartitionStatus(),
	}

	// Add peer details
	peerDetails := make([]map[string]interface{}, 0, len(ncm.peers))
	for _, peer := range ncm.peers {
		peerDetails = append(peerDetails, map[string]interface{}{
			"address":      peer.Address,
			"chain_height": peer.ChainHeight,
			"is_syncing":   peer.IsSyncing,
		})
	}
	stats["peers"] = peerDetails

	return stats
}

// ForceSync forces synchronization with a specific peer
func (ncm *NetworkConsensusManager) ForceSync(ctx context.Context, peerAddr string) error {
	return ncm.syncManager.SyncWithPeer(ctx, peerAddr, DefaultSyncConfig())
}

// GetSyncProgress returns the current synchronization progress
func (ncm *NetworkConsensusManager) GetSyncProgress() *SyncProgress {
	return ncm.syncManager.GetProgress()
}

// CancelSync cancels the current synchronization
func (ncm *NetworkConsensusManager) CancelSync() {
	ncm.syncManager.CancelSync()
}

// GetConsensusRules returns the consensus rules
func (ncm *NetworkConsensusManager) GetConsensusRules() *ConsensusRules {
	return ncm.consensusRules
}

// ValidateNewBlock validates a new block before adding it
func (ncm *NetworkConsensusManager) ValidateNewBlock(block *blockchain.Block) error {
	ncm.mu.RLock()
	defer ncm.mu.RUnlock()

	latestBlock := ncm.syncManager.localChain.GetLatestBlock()
	if latestBlock == nil {
		return fmt.Errorf("failed to get latest block")
	}

	return ncm.consensusRules.ValidateBlock(block, latestBlock)
}