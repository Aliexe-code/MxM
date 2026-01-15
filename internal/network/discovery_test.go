package network

import (
	"net"
	"testing"
	"time"
)

func TestNewDiscovery(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)

	bootstrapPeers := []string{"127.0.0.1:8001", "127.0.0.1:8002"}
	discovery := NewDiscovery(server, bootstrapPeers)

	if discovery.server != server {
		t.Error("Server should be set")
	}

	if len(discovery.bootstrapPeers) != len(bootstrapPeers) {
		t.Errorf("Expected %d bootstrap peers, got %d", len(bootstrapPeers), len(discovery.bootstrapPeers))
	}

	if discovery.knownPeers == nil {
		t.Error("Known peers map should be initialized")
	}
}

func TestDiscoveryStartStop(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	bootstrapPeers := []string{}
	discovery := NewDiscovery(server, bootstrapPeers)

	discovery.Start()
	time.Sleep(100 * time.Millisecond)

	discovery.Stop()
}

func TestDiscoveryAddKnownPeer(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	discovery := NewDiscovery(server, []string{})

	addr := "127.0.0.1:8000"
	discovery.addKnownPeer(addr)

	peers := discovery.getKnownPeers()
	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}

	if peers[0] != addr {
		t.Errorf("Expected peer %s, got %s", addr, peers[0])
	}
}

func TestDiscoveryUpdatePeerReputation(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	discovery := NewDiscovery(server, []string{})

	addr := "127.0.0.1:8000"
	discovery.addKnownPeer(addr)

	// Increase reputation
	discovery.updatePeerReputation(addr, 10)
	peers := discovery.GetKnownPeers()
	if peers[addr].Score != 110 {
		t.Errorf("Expected score 110, got %d", peers[addr].Score)
	}

	// Decrease reputation
	discovery.updatePeerReputation(addr, -5)
	peers = discovery.GetKnownPeers()
	if peers[addr].Score != 105 {
		t.Errorf("Expected score 105, got %d", peers[addr].Score)
	}

	// Increase fail count (negative delta also increases fail count)
	discovery.updatePeerReputation(addr, -1)
	peers = discovery.GetKnownPeers()
	if peers[addr].FailCount != 2 {
		t.Errorf("Expected fail count 2, got %d", peers[addr].FailCount)
	}
}

func TestDiscoveryUpdatePeerReputationRemoveLowScore(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	discovery := NewDiscovery(server, []string{})

	addr := "127.0.0.1:8000"
	discovery.addKnownPeer(addr)

	// Decrease score below zero
	for i := 0; i < 110; i++ {
		discovery.updatePeerReputation(addr, -1)
	}

	peers := discovery.GetKnownPeers()
	if _, exists := peers[addr]; exists {
		t.Error("Peer with low score should be removed")
	}
}

func TestDiscoveryGetKnownPeers(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	discovery := NewDiscovery(server, []string{})

	peers := []string{"127.0.0.1:8000", "127.0.0.1:8001", "127.0.0.1:8002"}
	for _, peer := range peers {
		discovery.addKnownPeer(peer)
	}

	knownPeers := discovery.getKnownPeers()
	if len(knownPeers) != len(peers) {
		t.Errorf("Expected %d peers, got %d", len(peers), len(knownPeers))
	}
}

func TestDiscoveryGetStats(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	bootstrapPeers := []string{"127.0.0.1:8001"}
	discovery := NewDiscovery(server, bootstrapPeers)

	// Add some known peers
	discovery.addKnownPeer("127.0.0.1:8000")
	discovery.addKnownPeer("127.0.0.1:8002")

	stats := discovery.GetStats()

	// Bootstrap peers are not added to knownPeers until connected
	if stats["total_peers"] != 2 {
		t.Errorf("Expected total_peers 2, got %v", stats["total_peers"])
	}

	if stats["bootstrap_peers"] != 1 {
		t.Errorf("Expected bootstrap_peers 1, got %v", stats["bootstrap_peers"])
	}

	if stats["max_peers"] != DefaultMaxPeers {
		t.Errorf("Expected max_peers %d, got %v", DefaultMaxPeers, stats["max_peers"])
	}
}

func TestDiscoveryHandleMessagePing(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	discovery := NewDiscovery(server, []string{})

	conn1, conn2 := makeTestConn()
	defer conn1.Close()
	defer conn2.Close()

	peer := NewPeer(conn1, "test-peer")
	peer.startSender()

	msg := NewPingMessage()
	discovery.handleMessage(peer, msg)

	// Give time for pong to be sent
	time.Sleep(50 * time.Millisecond)
}

func TestDiscoveryHandleMessageGetPeers(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	discovery := NewDiscovery(server, []string{})

	// Add some known peers
	discovery.addKnownPeer("127.0.0.1:8000")
	discovery.addKnownPeer("127.0.0.1:8001")

	conn1, conn2 := makeTestConn()
	defer conn1.Close()
	defer conn2.Close()

	peer := NewPeer(conn1, "test-peer")
	peer.startSender()

	msg := NewGetPeersMessage()
	discovery.handleMessage(peer, msg)

	// Give time for response to be sent
	time.Sleep(50 * time.Millisecond)
}

func TestDiscoveryHandleMessagePeers(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	discovery := NewDiscovery(server, []string{})

	conn1, conn2 := makeTestConn()
	defer conn1.Close()
	defer conn2.Close()

	peer := NewPeer(conn1, "test-peer")
	peer.startSender()

	peers := []PeerInfo{
		{ID: "peer1", Address: "127.0.0.1", Port: 8000},
		{ID: "peer2", Address: "127.0.0.1", Port: 8001},
	}
	msg, _ := NewPeersMessage(peers)

	initialCount := len(discovery.getKnownPeers())
	discovery.handleMessage(peer, msg)
	finalCount := len(discovery.getKnownPeers())

	if finalCount <= initialCount {
		t.Errorf("Expected peer count to increase, got %d -> %d", initialCount, finalCount)
	}
}

func TestDiscoveryProcessNewPeers(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	discovery := NewDiscovery(server, []string{})

	peers := []PeerInfo{
		{ID: "peer1", Address: "127.0.0.1", Port: 8000},
		{ID: "peer2", Address: "127.0.0.1", Port: 8001},
		{ID: "peer3", Address: "127.0.0.1", Port: 8002},
	}

	initialCount := len(discovery.getKnownPeers())
	discovery.processNewPeers(peers)
	finalCount := len(discovery.getKnownPeers())

	if finalCount-initialCount != len(peers) {
		t.Errorf("Expected %d new peers, got %d", len(peers), finalCount-initialCount)
	}
}

func TestDiscoveryGetPeerList(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	discovery := NewDiscovery(server, []string{})

	// Add some known peers
	discovery.addKnownPeer("127.0.0.1:8000")
	discovery.addKnownPeer("127.0.0.1:8001")

	peerList := discovery.getPeerList()

	if len(peerList) < 2 {
		t.Errorf("Expected at least 2 peers, got %d", len(peerList))
	}
}

// Helper function to create test connections
func makeTestConn() (net.Conn, net.Conn) {
	type connPair struct {
		c1 net.Conn
		c2 net.Conn
	}

	pair := make(chan connPair)

	go func() {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			panic(err)
		}
		defer listener.Close()

		addr := listener.Addr().String()

		conn1, err := net.Dial("tcp", addr)
		if err != nil {
			panic(err)
		}

		conn2, err := listener.Accept()
		if err != nil {
			panic(err)
		}

		pair <- connPair{c1: conn1, c2: conn2}
	}()

	result := <-pair
	return result.c1, result.c2
}