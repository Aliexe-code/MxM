package network

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// PeerInfo represents information about a peer
type PeerInfo struct {
	ID        string    `json:"id"`
	Address   string    `json:"address"`
	Port      int       `json:"port"`
	LastSeen  time.Time `json:"last_seen"`
	Version   string    `json:"version"`
	Connected bool      `json:"connected"`
}

// Peer represents a connected peer
type Peer struct {
	info      PeerInfo
	conn      net.Conn
	sendChan  chan *Message
	closeChan chan struct{}
	mu        sync.RWMutex
}

// NewPeer creates a new peer
func NewPeer(conn net.Conn, id string) *Peer {
	remoteAddr := conn.RemoteAddr().String()
	host, port, _ := net.SplitHostPort(remoteAddr)
	portNum := 0
	fmt.Sscanf(port, "%d", &portNum)

	// Configure TCP keepalive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
		tcpConn.SetNoDelay(true) // Disable Nagle's algorithm for better latency
	}

	return &Peer{
		info: PeerInfo{
			ID:        id,
			Address:   host,
			Port:      portNum,
			LastSeen:  time.Now(),
			Connected: true,
		},
		conn:      conn,
		sendChan:  make(chan *Message, 100),
		closeChan: make(chan struct{}),
	}
}

// GetInfo returns peer information
func (p *Peer) GetInfo() PeerInfo {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.info
}

// UpdateLastSeen updates the last seen timestamp
func (p *Peer) UpdateLastSeen() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.info.LastSeen = time.Now()
}

// Send sends a message to the peer
func (p *Peer) Send(msg *Message) error {
	select {
	case p.sendChan <- msg:
		return nil
	case <-p.closeChan:
		return fmt.Errorf("peer is closed")
	case <-time.After(5 * time.Second):
		return fmt.Errorf("send timeout")
	}
}

// Close closes the peer connection
func (p *Peer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.info.Connected {
		return nil
	}

	// Signal goroutines to stop
	close(p.closeChan)
	p.info.Connected = false

	// Drain send channel to prevent deadlock
	go func() {
		for range p.sendChan {
		}
	}()

	// Close connection with timeout
	done := make(chan error, 1)
	go func() {
		done <- p.conn.Close()
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		// Force close if timeout
		return p.conn.Close()
	}
}

// IsConnected returns whether the peer is connected
func (p *Peer) IsConnected() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.info.Connected
}

// startSender starts the sender goroutine
func (p *Peer) startSender() {
	go func() {
		for {
			select {
			case msg := <-p.sendChan:
				data, err := msg.Serialize()
				if err != nil {
					fmt.Printf("Error serializing message: %v\n", err)
					continue
				}

				// Set write deadline to prevent blocking
				if err := p.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
					fmt.Printf("Error setting write deadline: %v\n", err)
					p.Close()
					return
				}

				if _, err := p.conn.Write(data); err != nil {
					fmt.Printf("Error sending message: %v\n", err)
					p.Close()
					return
				}
			case <-p.closeChan:
				return
			}
		}
	}()
}

// startReceiver starts the receiver goroutine
func (p *Peer) startReceiver(handler MessageHandler) error {
	go func() {
		defer p.Close()

		for {
			data, err := p.readMessage()
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Error reading message: %v\n", err)
				}
				return
			}

			msg, err := Deserialize(data)
			if err != nil {
				fmt.Printf("Error deserializing message: %v\n", err)
				continue
			}

			p.UpdateLastSeen()

			if handler != nil {
				handler(p, msg)
			}
		}
	}()
	return nil
}

// readMessage reads a complete message from the connection
func (p *Peer) readMessage() ([]byte, error) {
	// Set read deadline to prevent blocking
	if err := p.conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read header first
	header := make([]byte, HeaderSize)
	_, err := io.ReadFull(p.conn, header)
	if err != nil {
		return nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Parse length from header
	buf := bytes.NewReader(header)
	var version uint8
	var msgType uint8
	var length uint32
	var checksum uint32

	if err := binary.Read(buf, binary.BigEndian, &version); err != nil {
		return nil, fmt.Errorf("failed to read version: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &msgType); err != nil {
		return nil, fmt.Errorf("failed to read message type: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &length); err != nil {
		return nil, fmt.Errorf("failed to read length: %w", err)
	}
	if err := binary.Read(buf, binary.BigEndian, &checksum); err != nil {
		return nil, fmt.Errorf("failed to read checksum: %w", err)
	}

	// Validate length
	if length > MaxMessageSize {
		return nil, fmt.Errorf("message size %d exceeds maximum %d", length, MaxMessageSize)
	}

	// Read signature length (2 bytes)
	sigLenBytes := make([]byte, 2)
	if _, err := io.ReadFull(p.conn, sigLenBytes); err != nil {
		return nil, fmt.Errorf("failed to read signature length: %w", err)
	}
	sigLen := binary.BigEndian.Uint16(sigLenBytes)

	// Read signature
	signature := make([]byte, sigLen)
	if sigLen > 0 {
		if _, err := io.ReadFull(p.conn, signature); err != nil {
			return nil, fmt.Errorf("failed to read signature: %w", err)
		}
	}

	// Read node ID length (2 bytes)
	nodeIDLenBytes := make([]byte, 2)
	if _, err := io.ReadFull(p.conn, nodeIDLenBytes); err != nil {
		return nil, fmt.Errorf("failed to read node ID length: %w", err)
	}
	nodeIDLen := binary.BigEndian.Uint16(nodeIDLenBytes)

	// Read node ID
	nodeID := make([]byte, nodeIDLen)
	if nodeIDLen > 0 {
		if _, err := io.ReadFull(p.conn, nodeID); err != nil {
			return nil, fmt.Errorf("failed to read node ID: %w", err)
		}
	}

	// Read payload
	payload := make([]byte, length)
	_, err = io.ReadFull(p.conn, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to read payload: %w", err)
	}

	// Combine all parts
	fullMessage := make([]byte, 0, HeaderSize+2+uint32(sigLen)+2+uint32(nodeIDLen)+length)
	fullMessage = append(fullMessage, header...)
	fullMessage = append(fullMessage, sigLenBytes...)
	fullMessage = append(fullMessage, signature...)
	fullMessage = append(fullMessage, nodeIDLenBytes...)
	fullMessage = append(fullMessage, nodeID...)
	fullMessage = append(fullMessage, payload...)

	return fullMessage, nil
}

// MessageHandler handles incoming messages
type MessageHandler func(peer *Peer, msg *Message)

// Server represents a TCP server
type Server struct {
	address     string
	port        int
	listener    net.Listener
	peers       map[string]*Peer
	peersMu     sync.RWMutex
	handler     MessageHandler
	peerCounter int
	mu          sync.Mutex
	closeChan   chan struct{}
	maxPeers    int
	rateLimiter map[string]time.Time
	rateMu      sync.RWMutex
}

// NewServer creates a new TCP server
func NewServer(address string, port int, handler MessageHandler) *Server {
	return &Server{
		address:     address,
		port:        port,
		peers:       make(map[string]*Peer),
		handler:     handler,
		closeChan:   make(chan struct{}),
		maxPeers:    100,
		rateLimiter: make(map[string]time.Time),
	}
}

// Start starts the server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.address, s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	s.listener = listener

	fmt.Printf("Server listening on %s\n", addr)

	go s.acceptConnections()

	return nil
}

// Stop stops the server
func (s *Server) Stop() error {
	close(s.closeChan)

	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
	}

	s.peersMu.Lock()
	for _, peer := range s.peers {
		peer.Close()
	}
	s.peersMu.Unlock()

	return nil
}

// acceptConnections accepts incoming connections
func (s *Server) acceptConnections() {
	for {
		select {
		case <-s.closeChan:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.closeChan:
					return
				default:
					fmt.Printf("Error accepting connection: %v\n", err)
					continue
				}
			}

			s.handleConnection(conn)
		}
	}
}

// handleConnection handles a new connection
func (s *Server) handleConnection(conn net.Conn) {
	remoteAddr := conn.RemoteAddr().String()

	// Check connection limit
	s.peersMu.RLock()
	peerCount := len(s.peers)
	s.peersMu.RUnlock()

	if peerCount >= s.maxPeers {
		fmt.Printf("Connection rejected: max peers (%d) reached from %s\n", s.maxPeers, remoteAddr)
		conn.Close()
		return
	}

	// Check rate limiting
	s.rateMu.Lock()
	lastConn, exists := s.rateLimiter[remoteAddr]
	if exists && time.Since(lastConn) < 1*time.Second {
		s.rateMu.Unlock()
		fmt.Printf("Connection rejected: rate limited from %s\n", remoteAddr)
		conn.Close()
		return
	}
	s.rateLimiter[remoteAddr] = time.Now()
	s.rateMu.Unlock()

	s.mu.Lock()
	s.peerCounter++
	peerID := fmt.Sprintf("peer-%d", s.peerCounter)
	s.mu.Unlock()

	peer := NewPeer(conn, peerID)
	peer.startSender()

	if err := peer.startReceiver(s.handler); err != nil {
		fmt.Printf("Error starting receiver: %v\n", err)
		peer.Close()
		return
	}

	s.peersMu.Lock()
	s.peers[peerID] = peer
	s.peersMu.Unlock()

	fmt.Printf("New peer connected: %s (%s)\n", peerID, conn.RemoteAddr())
}

// Broadcast sends a message to all connected peers
func (s *Server) Broadcast(msg *Message) {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()

	for _, peer := range s.peers {
		if peer.IsConnected() {
			if err := peer.Send(msg); err != nil {
				fmt.Printf("Error broadcasting to peer %s: %v\n", peer.GetInfo().ID, err)
			}
		}
	}
}

// GetPeers returns all connected peers
func (s *Server) GetPeers() []PeerInfo {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()

	peers := make([]PeerInfo, 0, len(s.peers))
	for _, peer := range s.peers {
		if peer.IsConnected() {
			peers = append(peers, peer.GetInfo())
		}
	}
	return peers
}

// GetPeerCount returns the number of connected peers
func (s *Server) GetPeerCount() int {
	s.peersMu.RLock()
	defer s.peersMu.RUnlock()

	count := 0
	for _, peer := range s.peers {
		if peer.IsConnected() {
			count++
		}
	}
	return count
}

// Client represents a TCP client
type Client struct {
	serverAddr string
	peer       *Peer
	handler    MessageHandler
}

// NewClient creates a new TCP client
func NewClient(serverAddr string, handler MessageHandler) *Client {
	return &Client{
		serverAddr: serverAddr,
		handler:    handler,
	}
}

// Connect connects to the server
func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", c.serverAddr)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.peer = NewPeer(conn, "client")
	c.peer.startSender()

	if err := c.peer.startReceiver(c.handler); err != nil {
		return fmt.Errorf("failed to start receiver: %w", err)
	}

	fmt.Printf("Connected to server: %s\n", c.serverAddr)
	return nil
}

// Send sends a message to the server
func (c *Client) Send(msg *Message) error {
	if c.peer == nil {
		return fmt.Errorf("not connected")
	}
	return c.peer.Send(msg)
}

// Close closes the client connection
func (c *Client) Close() error {
	if c.peer != nil {
		return c.peer.Close()
	}
	return nil
}

// IsConnected returns whether the client is connected
func (c *Client) IsConnected() bool {
	if c.peer == nil {
		return false
	}
	return c.peer.IsConnected()
}
