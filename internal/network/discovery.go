package network

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const (
	DefaultBootstrapPeers     = 3
	DefaultMaxPeers           = 50
	DefaultPeerHealthInterval = 30 * time.Second
	DefaultPeerTimeout        = 5 * time.Minute
	DefaultDiscoveryInterval  = 1 * time.Minute
)

// PeerReputation tracks peer reputation
type PeerReputation struct {
	Score       int       `json:"score"`
	LastContact time.Time `json:"last_contact"`
	FailCount   int       `json:"fail_count"`
}

// Discovery manages peer discovery and health
type Discovery struct {
	server         *Server
	bootstrapPeers []string
	knownPeers     map[string]*PeerReputation
	peersMu        sync.RWMutex
	maxPeers       int
	healthInterval time.Duration
	peerTimeout    time.Duration
	discoveryInterval time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
}

// NewDiscovery creates a new peer discovery manager
func NewDiscovery(server *Server, bootstrapPeers []string) *Discovery {
	ctx, cancel := context.WithCancel(context.Background())

	return &Discovery{
		server:            server,
		bootstrapPeers:    bootstrapPeers,
		knownPeers:        make(map[string]*PeerReputation),
		maxPeers:          DefaultMaxPeers,
		healthInterval:    DefaultPeerHealthInterval,
		peerTimeout:       DefaultPeerTimeout,
		discoveryInterval: DefaultDiscoveryInterval,
		ctx:               ctx,
		cancel:            cancel,
	}
}

// Start starts the discovery service
func (d *Discovery) Start() {
	fmt.Println("Starting peer discovery service...")

	// Connect to bootstrap peers
	d.connectToBootstrapPeers()

	// Start health check loop
	go d.healthCheckLoop()

	// Start discovery loop
	go d.discoveryLoop()
}

// Stop stops the discovery service
func (d *Discovery) Stop() {
	d.cancel()
	fmt.Println("Peer discovery service stopped")
}

// connectToBootstrapPeers connects to bootstrap peers
func (d *Discovery) connectToBootstrapPeers() {
	for _, addr := range d.bootstrapPeers {
		d.connectToPeer(addr)
	}
}

// connectToPeer connects to a peer
func (d *Discovery) connectToPeer(addr string) error {
	d.peersMu.RLock()
	_, exists := d.knownPeers[addr]
	d.peersMu.RUnlock()

	if exists {
		return fmt.Errorf("peer %s already known", addr)
	}

	client := NewClient(addr, func(peer *Peer, msg *Message) {
		d.handleMessage(peer, msg)
	})

	if err := client.Connect(); err != nil {
		d.updatePeerReputation(addr, -1)
		return fmt.Errorf("failed to connect to peer %s: %w", addr, err)
	}

	d.addKnownPeer(addr)
	fmt.Printf("Connected to bootstrap peer: %s\n", addr)

	// Request peers from this peer
	if err := client.Send(NewGetPeersMessage()); err != nil {
		fmt.Printf("Failed to request peers from %s: %v\n", addr, err)
	}

	return nil
}

// addKnownPeer adds a peer to the known peers list
func (d *Discovery) addKnownPeer(addr string) {
	d.peersMu.Lock()
	defer d.peersMu.Unlock()

	if _, exists := d.knownPeers[addr]; !exists {
		d.knownPeers[addr] = &PeerReputation{
			Score:       100,
			LastContact: time.Now(),
			FailCount:   0,
		}
	}
}

// updatePeerReputation updates a peer's reputation
func (d *Discovery) updatePeerReputation(addr string, delta int) {
	d.peersMu.Lock()
	defer d.peersMu.Unlock()

	if rep, exists := d.knownPeers[addr]; exists {
		rep.Score += delta
		rep.LastContact = time.Now()

		if delta < 0 {
			rep.FailCount++
		} else {
			rep.FailCount = 0
		}

		// Remove peers with very low reputation
		if rep.Score < 0 {
			delete(d.knownPeers, addr)
			fmt.Printf("Removed peer %s due to low reputation\n", addr)
		}
	}
}

// getKnownPeers returns all known peers
func (d *Discovery) getKnownPeers() []string {
	d.peersMu.RLock()
	defer d.peersMu.RUnlock()

	peers := make([]string, 0, len(d.knownPeers))
	for addr := range d.knownPeers {
		peers = append(peers, addr)
	}
	return peers
}

// healthCheckLoop periodically checks peer health
func (d *Discovery) healthCheckLoop() {
	ticker := time.NewTicker(d.healthInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.checkPeerHealth()
		}
	}
}

// checkPeerHealth checks the health of all known peers
func (d *Discovery) checkPeerHealth() {
	d.peersMu.RLock()
	peers := make([]string, 0, len(d.knownPeers))
	for addr := range d.knownPeers {
		peers = append(peers, addr)
	}
	d.peersMu.RUnlock()

	now := time.Now()
	for _, addr := range peers {
		d.peersMu.RLock()
		rep, exists := d.knownPeers[addr]
		d.peersMu.RUnlock()

		if !exists {
			continue
		}

		// Check if peer has timed out
		if now.Sub(rep.LastContact) > d.peerTimeout {
			fmt.Printf("Peer %s timed out\n", addr)
			d.updatePeerReputation(addr, -10)
		}

		// Send ping to check connectivity
		client := NewClient(addr, func(peer *Peer, msg *Message) {
			if msg.Type == MessageTypePong {
				d.updatePeerReputation(addr, 1)
			}
		})

		if err := client.Connect(); err != nil {
			d.updatePeerReputation(addr, -1)
			continue
		}

		if err := client.Send(NewPingMessage()); err != nil {
			d.updatePeerReputation(addr, -1)
			client.Close()
			continue
		}

		// Wait for pong with timeout
		select {
		case <-time.After(5 * time.Second):
			d.updatePeerReputation(addr, -1)
		case <-d.ctx.Done():
			return
		}

		client.Close()
	}
}

// discoveryLoop periodically discovers new peers
func (d *Discovery) discoveryLoop() {
	ticker := time.NewTicker(d.discoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.discoverPeers()
		}
	}
}

// discoverPeers discovers new peers from known peers
func (d *Discovery) discoverPeers() {
	peers := d.getKnownPeers()
	if len(peers) == 0 {
		return
	}

	// Request peers from a subset of known peers
	for _, addr := range peers {
		if d.server.GetPeerCount() >= d.maxPeers {
			break
		}

		client := NewClient(addr, func(peer *Peer, msg *Message) {
			d.handleMessage(peer, msg)
		})

		if err := client.Connect(); err != nil {
			d.updatePeerReputation(addr, -1)
			continue
		}

		if err := client.Send(NewGetPeersMessage()); err != nil {
			fmt.Printf("Failed to request peers from %s: %v\n", addr, err)
			client.Close()
			continue
		}

		client.Close()
	}
}

// handleMessage handles incoming messages
func (d *Discovery) handleMessage(peer *Peer, msg *Message) {
	switch msg.Type {
	case MessageTypePing:
		// Respond with pong
		if err := peer.Send(NewPongMessage()); err != nil {
			fmt.Printf("Failed to send pong: %v\n", err)
		}
	case MessageTypePong:
		// Update peer reputation
		d.updatePeerReputation(peer.GetInfo().Address, 1)
	case MessageTypeGetPeers:
		// Send list of known peers
		peers := d.getPeerList()
		msg, err := NewPeersMessage(peers)
		if err != nil {
			fmt.Printf("Failed to create peers message: %v\n", err)
			return
		}
		if err := peer.Send(msg); err != nil {
			fmt.Printf("Failed to send peers: %v\n", err)
		}
	case MessageTypePeers:
		// Process received peers
		peers, err := ParsePeersMessage(msg)
		if err != nil {
			fmt.Printf("Failed to parse peers message: %v\n", err)
			return
		}
		d.processNewPeers(peers)
	}
}

// getPeerList returns a list of peer information
func (d *Discovery) getPeerList() []PeerInfo {
	peers := d.server.GetPeers()

	d.peersMu.RLock()
	for addr := range d.knownPeers {
		found := false
		for _, p := range peers {
			if p.Address == addr {
				found = true
				break
			}
		}
		if !found {
			peers = append(peers, PeerInfo{
				Address:   addr,
				Connected: false,
				LastSeen:  time.Now(),
			})
		}
	}
	d.peersMu.RUnlock()

	return peers
}

// processNewPeers processes newly discovered peers
func (d *Discovery) processNewPeers(peers []PeerInfo) {
	for _, peerInfo := range peers {
		addr := fmt.Sprintf("%s:%d", peerInfo.Address, peerInfo.Port)

		// Skip if already known
		d.peersMu.RLock()
		_, exists := d.knownPeers[addr]
		d.peersMu.RUnlock()

		if exists {
			continue
		}

		// Add to known peers
		d.addKnownPeer(addr)

		// Connect if we have capacity
		if d.server.GetPeerCount() < d.maxPeers {
			if err := d.connectToPeer(addr); err != nil {
				fmt.Printf("Failed to connect to discovered peer %s: %v\n", addr, err)
			}
		}
	}
}

// GetStats returns discovery statistics
func (d *Discovery) GetStats() map[string]interface{} {
	d.peersMu.RLock()
	defer d.peersMu.RUnlock()

	totalPeers := len(d.knownPeers)
	connectedPeers := d.server.GetPeerCount()
	goodReputation := 0

	for _, rep := range d.knownPeers {
		if rep.Score >= 50 {
			goodReputation++
		}
	}

	return map[string]interface{}{
		"total_peers":       totalPeers,
		"connected_peers":   connectedPeers,
		"good_reputation":   goodReputation,
		"max_peers":         d.maxPeers,
		"bootstrap_peers":   len(d.bootstrapPeers),
	}
}

// GetKnownPeers returns all known peers with their reputation
func (d *Discovery) GetKnownPeers() map[string]*PeerReputation {
	d.peersMu.RLock()
	defer d.peersMu.RUnlock()

	peers := make(map[string]*PeerReputation, len(d.knownPeers))
	for addr, rep := range d.knownPeers {
		peers[addr] = &PeerReputation{
			Score:       rep.Score,
			LastContact: rep.LastContact,
			FailCount:   rep.FailCount,
		}
	}
	return peers
}
