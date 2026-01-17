package network

import (
	"net"
	"sync"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)

	if server.address != "127.0.0.1" {
		t.Errorf("Expected address 127.0.0.1, got %s", server.address)
	}

	if server.port != 0 {
		t.Errorf("Expected port 0, got %d", server.port)
	}

	if server.handler == nil {
		t.Error("Handler should not be nil")
	}
}

func TestServerStartStop(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	if server.listener == nil {
		t.Error("Listener should not be nil after start")
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}
}

func TestServerClientConnection(t *testing.T) {
	serverReceived := make(chan *Message, 1)
	clientReceived := make(chan *Message, 1)

	// Server handler
	serverHandler := func(peer *Peer, msg *Message) {
		if msg.Type == MessageTypePing {
			peer.Send(NewPongMessage())
			serverReceived <- msg
		}
	}

	server := NewServer("127.0.0.1", 0, serverHandler)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	// Get actual server address
	addr := server.listener.Addr().String()

	// Client handler
	clientHandler := func(peer *Peer, msg *Message) {
		if msg.Type == MessageTypePong {
			clientReceived <- msg
		}
	}

	client := NewClient(addr, clientHandler)
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Wait for connection to establish
	time.Sleep(50 * time.Millisecond)

	// Send ping
	if err := client.Send(NewPingMessage()); err != nil {
		t.Fatalf("Failed to send ping: %v", err)
	}

	// Wait for exchange
	select {
	case <-serverReceived:
		// Server received ping
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for server to receive ping")
	}

	select {
	case <-clientReceived:
		// Client received pong
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for client to receive pong")
	}
}

func TestServerMultipleClients(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	addr := server.listener.Addr().String()

	// Connect multiple clients
	numClients := 3
	clients := make([]*Client, numClients)
	var wg sync.WaitGroup
	wg.Add(numClients)

	for i := 0; i < numClients; i++ {
		go func(idx int) {
			defer wg.Done()
			client := NewClient(addr, func(peer *Peer, msg *Message) {})
			if err := client.Connect(); err != nil {
				t.Errorf("Failed to connect client: %v", err)
				return
			}
			clients[idx] = client
		}(i)
	}

	wg.Wait()

	// Wait for all connections to be established
	time.Sleep(100 * time.Millisecond)

	// Check peer count
	if server.GetPeerCount() != numClients {
		t.Errorf("Expected %d peers, got %d", numClients, server.GetPeerCount())
	}

	// Close all clients
	for _, client := range clients {
		if client != nil {
			client.Close()
		}
	}
}

func TestServerBroadcast(t *testing.T) {
	numClients := 3
	received := make(chan struct{}, numClients)

	handler := func(peer *Peer, msg *Message) {
		if msg.Type == MessageTypePing {
			received <- struct{}{}
		}
	}

	server := NewServer("127.0.0.1", 0, handler)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	addr := server.listener.Addr().String()

	// Connect clients
	clients := make([]*Client, numClients)
	for i := 0; i < numClients; i++ {
		client := NewClient(addr, handler)
		if err := client.Connect(); err != nil {
			t.Fatalf("Failed to connect client: %v", err)
		}
		clients[i] = client
		defer client.Close()
	}

	// Wait for connections to establish
	time.Sleep(100 * time.Millisecond)

	// Broadcast message
	server.Broadcast(NewPingMessage())

	// Wait for all clients to receive
	receivedCount := 0
	timeout := time.After(3 * time.Second)
	for {
		select {
		case <-received:
			receivedCount++
			if receivedCount == numClients {
				return // Success
			}
		case <-timeout:
			t.Errorf("Timeout waiting for broadcast. Received %d/%d", receivedCount, numClients)
			return
		}
	}
}

func TestClientDisconnect(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	addr := server.listener.Addr().String()

	client := NewClient(addr, func(peer *Peer, msg *Message) {})
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}

	if !client.IsConnected() {
		t.Error("Client should be connected")
	}

	client.Close()

	if client.IsConnected() {
		t.Error("Client should not be connected after close")
	}
}

func TestPeerSend(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	addr := server.listener.Addr().String()

	client := NewClient(addr, func(peer *Peer, msg *Message) {})
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Send multiple messages
	for i := 0; i < 10; i++ {
		if err := client.Send(NewPingMessage()); err != nil {
			t.Errorf("Failed to send message %d: %v", i, err)
		}
	}
}

func TestPeerUpdateLastSeen(t *testing.T) {
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	peer := NewPeer(conn1, "test-peer")
	initialTime := peer.GetInfo().LastSeen

	time.Sleep(10 * time.Millisecond)
	peer.UpdateLastSeen()

	updatedTime := peer.GetInfo().LastSeen
	if !updatedTime.After(initialTime) {
		t.Error("Last seen time should be updated")
	}
}

func TestPeerGetInfo(t *testing.T) {
	conn1, conn2 := net.Pipe()
	defer conn1.Close()
	defer conn2.Close()

	peer := NewPeer(conn1, "test-peer")
	info := peer.GetInfo()

	if info.ID != "test-peer" {
		t.Errorf("Expected ID test-peer, got %s", info.ID)
	}

	if !info.Connected {
		t.Error("Peer should be connected")
	}
}

func TestServerGetPeers(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	addr := server.listener.Addr().String()

	// Connect a client
	client := NewClient(addr, func(peer *Peer, msg *Message) {})
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Wait for connection
	time.Sleep(100 * time.Millisecond)

	peers := server.GetPeers()
	if len(peers) != 1 {
		t.Errorf("Expected 1 peer, got %d", len(peers))
	}

	if peers[0].ID == "" {
		t.Error("Peer ID should not be empty")
	}
}

func TestClientNotConnected(t *testing.T) {
	client := NewClient("127.0.0.1:9999", func(peer *Peer, msg *Message) {})

	if client.IsConnected() {
		t.Error("Client should not be connected")
	}

	err := client.Send(NewPingMessage())
	if err == nil {
		t.Error("Expected error when sending to disconnected client")
	}
}

func TestServerStartOnUsedPort(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server1 := NewServer("127.0.0.1", 12345, handler)

	if err := server1.Start(); err != nil {
		t.Fatalf("Failed to start first server: %v", err)
	}
	defer server1.Stop()

	server2 := NewServer("127.0.0.1", 12345, handler)
	if err := server2.Start(); err == nil {
		t.Error("Expected error when starting server on used port")
	}
}

func TestMessageExchange(t *testing.T) {
	serverReceived := make(chan *Message, 1)
	clientReceived := make(chan []PeerInfo, 1)

	// Server side
	serverHandler := func(peer *Peer, msg *Message) {
		if msg.Type == MessageTypeGetPeers {
			peers := []PeerInfo{
				{ID: "peer1", Address: "127.0.0.1", Port: 8000},
			}
			response, _ := NewPeersMessage(peers)
			peer.Send(response)
			serverReceived <- msg
		}
	}

	server := NewServer("127.0.0.1", 0, serverHandler)
	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	addr := server.listener.Addr().String()

	// Client side
	clientHandler := func(peer *Peer, msg *Message) {
		if msg.Type == MessageTypePeers {
			peers, _ := ParsePeersMessage(msg)
			clientReceived <- peers
		}
	}

	client := NewClient(addr, clientHandler)
	if err := client.Connect(); err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Wait for connection to establish
	time.Sleep(50 * time.Millisecond)

	// Send request
	if err := client.Send(NewGetPeersMessage()); err != nil {
		t.Fatalf("Failed to send get peers: %v", err)
	}

	// Wait for server to receive request
	select {
	case <-serverReceived:
		// Server received request
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for server to receive request")
	}

	// Wait for client to receive response
	select {
	case peers := <-clientReceived:
		if len(peers) != 1 {
			t.Errorf("Expected 1 peer, got %d", len(peers))
		}
	case <-time.After(2 * time.Second):
		t.Error("Timeout waiting for client to receive response")
	}
}

func TestConcurrentConnections(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}
	defer server.Stop()

	addr := server.listener.Addr().String()

	// Create many concurrent connections
	numConnections := 20
	clients := make([]*Client, numConnections)
	var wg sync.WaitGroup
	wg.Add(numConnections)

	for i := 0; i < numConnections; i++ {
		go func(idx int) {
			defer wg.Done()
			client := NewClient(addr, func(peer *Peer, msg *Message) {})
			if err := client.Connect(); err != nil {
				t.Errorf("Failed to connect: %v", err)
				return
			}
			clients[idx] = client
		}(i)
	}

	wg.Wait()

	// Wait a bit for all connections to be established
	time.Sleep(100 * time.Millisecond)

	if server.GetPeerCount() != numConnections {
		t.Errorf("Expected %d peers, got %d", numConnections, server.GetPeerCount())
	}

	// Close all clients
	for _, client := range clients {
		if client != nil {
			client.Close()
		}
	}
}

func TestServerContextCancellation(t *testing.T) {
	handler := func(peer *Peer, msg *Message) {}
	server := NewServer("127.0.0.1", 0, handler)

	if err := server.Start(); err != nil {
		t.Fatalf("Failed to start server: %v", err)
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("Failed to stop server: %v", err)
	}

	// Try to connect after stop
	addr := server.listener.Addr().String()
	client := NewClient(addr, func(peer *Peer, msg *Message) {})
	if err := client.Connect(); err == nil {
		t.Error("Expected error when connecting to stopped server")
	}
}
