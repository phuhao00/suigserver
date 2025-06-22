package simple

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// SimpleServer is a basic game server without external dependencies
type SimpleServer struct {
	port     int
	listener net.Listener
	clients  map[string]*Client
	rooms    map[string]*Room
	mu       sync.RWMutex
	quit     chan struct{}
}

// Client represents a connected player
type Client struct {
	conn   net.Conn
	id     string
	name   string
	room   *Room
	server *SimpleServer
	sendCh chan string
	quitCh chan struct{}
}

// Room represents a game room
type Room struct {
	id      string
	name    string
	clients map[string]*Client
	mu      sync.RWMutex
}

// NewSimpleServer creates a new simple server
func NewSimpleServer(port int) *SimpleServer {
	return &SimpleServer{
		port:    port,
		clients: make(map[string]*Client),
		rooms:   make(map[string]*Room),
		quit:    make(chan struct{}),
	}
}

// Start starts the server
func (s *SimpleServer) Start() error {
	var err error
	s.listener, err = net.Listen("tcp", ":"+strconv.Itoa(s.port))
	if err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	log.Printf("Simple Game Server started on port %d", s.port)

	// Create a default room
	defaultRoom := &Room{
		id:      "lobby",
		name:    "Lobby",
		clients: make(map[string]*Client),
	}
	s.rooms["lobby"] = defaultRoom

	go s.acceptConnections()
	return nil
}

// acceptConnections handles incoming connections
func (s *SimpleServer) acceptConnections() {
	for {
		select {
		case <-s.quit:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				log.Printf("Error accepting connection: %v", err)
				continue
			}
			go s.handleConnection(conn)
		}
	}
}

// handleConnection handles a single client connection
func (s *SimpleServer) handleConnection(conn net.Conn) {
	clientID := fmt.Sprintf("client_%d", time.Now().UnixNano())
	client := &Client{
		conn:   conn,
		id:     clientID,
		server: s,
		sendCh: make(chan string, 10),
		quitCh: make(chan struct{}),
	}

	s.mu.Lock()
	s.clients[clientID] = client
	s.mu.Unlock()

	log.Printf("Client %s connected from %s", clientID, conn.RemoteAddr())

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()

	// Send welcome message
	client.send("Welcome to the Simple Game Server!")
	client.send("Commands: /auth <name>, /join <room>, /say <message>, /quit")

	// Wait for client to disconnect
	<-client.quitCh

	// Cleanup
	s.mu.Lock()
	delete(s.clients, clientID)
	s.mu.Unlock()

	if client.room != nil {
		client.leaveRoom()
	}

	conn.Close()
	log.Printf("Client %s disconnected", clientID)
}

// send sends a message to the client
func (c *Client) send(message string) {
	select {
	case c.sendCh <- message:
	case <-c.quitCh:
	default:
		// Channel full, drop message
	}
}

// writePump sends messages to the client
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message := <-c.sendCh:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := c.conn.Write([]byte(message + "\n")); err != nil {
				log.Printf("Error writing to client %s: %v", c.id, err)
				close(c.quitCh)
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if _, err := c.conn.Write([]byte("\n")); err != nil {
				close(c.quitCh)
				return
			}
		case <-c.quitCh:
			return
		}
	}
}

// readPump reads messages from the client
func (c *Client) readPump() {
	defer close(c.quitCh)

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	reader := bufio.NewReader(c.conn)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			log.Printf("Error reading from client %s: %v", c.id, err)
			return
		}

		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		message := strings.TrimSpace(line)
		if message == "" {
			continue
		}

		c.handleMessage(message)
	}
}

// handleMessage processes client messages
func (c *Client) handleMessage(message string) {
	parts := strings.Fields(message)
	if len(parts) == 0 {
		return
	}

	command := parts[0]
	log.Printf("Client %s sent command: %s", c.id, command)

	switch command {
	case "/auth":
		if len(parts) >= 2 {
			c.name = parts[1]
			c.send(fmt.Sprintf("Authenticated as %s", c.name))
		} else {
			c.send("Usage: /auth <name>")
		}

	case "/join":
		if c.name == "" {
			c.send("Please authenticate first with /auth <name>")
			return
		}
		if len(parts) >= 2 {
			roomID := parts[1]
			c.joinRoom(roomID)
		} else {
			c.send("Usage: /join <room>")
		}

	case "/say":
		if c.name == "" {
			c.send("Please authenticate first with /auth <name>")
			return
		}
		if c.room == nil {
			c.send("You must be in a room to chat. Use /join <room>")
			return
		}
		if len(parts) >= 2 {
			chatMessage := strings.Join(parts[1:], " ")
			c.room.broadcast(c, fmt.Sprintf("[%s]: %s", c.name, chatMessage))
		} else {
			c.send("Usage: /say <message>")
		}

	case "/quit":
		c.send("Goodbye!")
		close(c.quitCh)

	default:
		c.send("Unknown command. Available commands: /auth, /join, /say, /quit")
	}
}

// joinRoom joins a room
func (c *Client) joinRoom(roomID string) {
	// Leave current room if any
	if c.room != nil {
		c.leaveRoom()
	}

	c.server.mu.Lock()
	room, exists := c.server.rooms[roomID]
	if !exists {
		// Create new room
		room = &Room{
			id:      roomID,
			name:    roomID,
			clients: make(map[string]*Client),
		}
		c.server.rooms[roomID] = room
		log.Printf("Created new room: %s", roomID)
	}
	c.server.mu.Unlock()

	room.mu.Lock()
	room.clients[c.id] = c
	room.mu.Unlock()

	c.room = room
	c.send(fmt.Sprintf("Joined room: %s", roomID))
	room.broadcast(c, fmt.Sprintf("%s joined the room", c.name))
}

// leaveRoom leaves the current room
func (c *Client) leaveRoom() {
	if c.room == nil {
		return
	}

	c.room.mu.Lock()
	delete(c.room.clients, c.id)
	clientCount := len(c.room.clients)
	c.room.mu.Unlock()

	c.room.broadcast(c, fmt.Sprintf("%s left the room", c.name))

	// Remove empty rooms (except lobby)
	if clientCount == 0 && c.room.id != "lobby" {
		c.server.mu.Lock()
		delete(c.server.rooms, c.room.id)
		c.server.mu.Unlock()
		log.Printf("Removed empty room: %s", c.room.id)
	}

	c.room = nil
}

// broadcast sends a message to all clients in the room
func (r *Room) broadcast(sender *Client, message string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, client := range r.clients {
		if client != sender {
			client.send(message)
		}
	}
}

// Stop stops the server
func (s *SimpleServer) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}

	s.mu.Lock()
	for _, client := range s.clients {
		close(client.quitCh)
	}
	s.mu.Unlock()

	log.Println("Simple Game Server stopped")
}
