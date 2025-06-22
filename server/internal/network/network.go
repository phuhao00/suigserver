package network

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"encoding/binary"
	"github.com/asynkron/protoactor-go/actor"
	sessionactor "github.com/phuhao00/suigserver/server/internal/actor" // Alias for the actor package
	"github.com/phuhao00/suigserver/server/internal/actor/messages"
)

const (
	// MaxMessageSize defines the maximum allowed size for a single message payload.
	// This helps prevent DoS attacks with overly large messages. E.g., 1MB.
	MaxMessageSize = 1 * 1024 * 1024
	// LengthPrefixSize is the size in bytes of the message length prefix.
	// Using uint32 for length, so 4 bytes.
	LengthPrefixSize = 4
)

// TCPServer manages TCP client connections and interfaces with the actor system.
type TCPServer struct {
	listener        net.Listener
	port            int
	actorSystem     *actor.ActorSystem
	wg              sync.WaitGroup
	shutdown        chan struct{}
	roomManagerPID  *actor.PID // PID of the RoomManagerActor
	worldManagerPID *actor.PID // PID of the WorldManagerActor, to be passed to SessionActor
}

// NewTCPServer creates a new TCPServer.
// It now requires worldManagerPID.
func NewTCPServer(port int, system *actor.ActorSystem, roomManagerPID *actor.PID, worldManagerPID *actor.PID) *TCPServer {
	log.Printf("Initializing TCP Server for port %d...\n", port)
	if roomManagerPID == nil {
		log.Panicf("TCPServer: RoomManagerPID cannot be nil")
	}
	if worldManagerPID == nil {
		log.Panicf("TCPServer: WorldManagerPID cannot be nil")
	}
	return &TCPServer{
		port:            port,
		actorSystem:     system,
		shutdown:        make(chan struct{}),
		roomManagerPID:  roomManagerPID,
		worldManagerPID: worldManagerPID,
	}
}

// Start begins listening for TCP connections.
func (s *TCPServer) Start() error {
	listenAddr := ":" + strconv.Itoa(s.port)
	var err error
	s.listener, err = net.Listen("tcp", listenAddr)
	if err != nil {
		log.Printf("Error starting TCP server on port %d: %v\n", s.port, err)
		return err
	}
	log.Printf("TCP Server started and listening on %s\n", listenAddr)

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

func (s *TCPServer) acceptConnections() {
	defer s.wg.Done()
	log.Println("TCP accept loop started.")
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				log.Println("TCP accept loop shutting down.")
				return
			default:
				log.Printf("Error accepting connection: %v\n", err)
				if ne, ok := err.(net.Error); ok && !ne.Temporary() {
					log.Printf("Permanent error in accept: %v. Shutting down accept loop.", err)
					return
				}
				continue
			}
		}
		log.Printf("Accepted new connection from %s\n", conn.RemoteAddr())

		s.wg.Add(1)
		// Each connection will have its own actor to manage its lifecycle and communication.
		// This "ConnectionActor" will then interact with a PlayerSessionActor.
		// For now, we'll directly spawn a conceptual PlayerSessionActor or a handler here.
		// In a more robust design, TCPServer might spawn a ConnectionHandlerActor per connection.
		go s.handleConnection(conn)
	}
}

// handleConnection is responsible for a single client connection.
// It creates a PlayerSessionActor (or a similar actor) for this connection
// and then mainly acts as a bridge for reading from the socket and writing to it.
func (s *TCPServer) handleConnection(conn net.Conn) {
	defer s.wg.Done() // Decrement counter when this connection handler exits

	// TODO: Replace with actual PlayerSessionActor props once defined in actors package
	// playerSessionProps := actor.PropsFromProducer(func() actor.Actor { return actors.NewPlayerSessionActor(conn, s.actorSystem) })
	// The above line is incorrect as actors should not be passed `conn` directly in constructor usually,
	// but rather receive it via a message after starting.
	if s.roomManagerPID == nil {
		log.Panicf("[%s] CRITICAL: RoomManagerPID is not set in TCPServer. Cannot spawn PlayerSessionActor correctly.", conn.RemoteAddr())
		// conn.Close() - Panicking, so this won't be reached.
		return
	}
	if s.worldManagerPID == nil {
		log.Panicf("[%s] CRITICAL: WorldManagerPID is not set in TCPServer. Cannot spawn PlayerSessionActor correctly.", conn.RemoteAddr())
		// conn.Close() - Panicking, so this won't be reached.
		return
	}

	// PlayerSessionActor now requires worldManagerPID as well.
	playerSessionProps := sessionactor.Props(s.actorSystem, s.roomManagerPID, s.worldManagerPID)
	playerSessionPID := s.actorSystem.Root.Spawn(playerSessionProps)
	log.Printf("[%s] Spawned PlayerSessionActor with PID: %s", conn.RemoteAddr(), playerSessionPID.String())

	// Send ClientConnected message to the PlayerSessionActor
	// This message includes the net.Conn so the actor can use it.
	connectedMsg := &messages.ClientConnected{
		Conn: conn,
		// SelfPID: ctx.Self(), // If TCPServer's handleConnection was an actor itself. Not the case here.
	}
	s.actorSystem.Root.Send(playerSessionPID, connectedMsg)

	// Goroutine for reading from the client and forwarding messages to PlayerSessionActor
	// reader := bufio.NewReader(conn) // Replaced by direct read for length-prefixing
	for {
		// Implement proper message framing: Length-Prefixing
		// 1. Read the length prefix (e.g., 4 bytes for uint32)
		lenBuf := make([]byte, LengthPrefixSize)
		_, err := io.ReadFull(conn, lenBuf)
		if err != nil {
			s.handleReadError(conn, playerSessionPID, err, "reading length prefix")
			return
		}

		messageLength := binary.BigEndian.Uint32(lenBuf)

		// 2. Validate message length
		if messageLength == 0 {
			log.Printf("[%s] Received message with zero length. Ignoring.", conn.RemoteAddr())
			continue // Or treat as an error/disconnect
		}
		if messageLength > MaxMessageSize {
			log.Printf("[%s] Message length %d exceeds MaxMessageSize %d. Closing connection.",
				conn.RemoteAddr(), messageLength, MaxMessageSize)
			s.actorSystem.Root.Send(playerSessionPID, &messages.ClientDisconnected{Reason: "Message too large"})
			conn.Close()
			return
		}

		// 3. Read the message payload
		payloadBuf := make([]byte, messageLength)
		_, err = io.ReadFull(conn, payloadBuf)
		if err != nil {
			s.handleReadError(conn, playerSessionPID, err, "reading payload")
			return
		}

		log.Printf("[%s] Received %d bytes. Payload: '%s'\n", conn.RemoteAddr(), messageLength, string(payloadBuf))

		if playerSessionPID != nil {
			// The payloadBuf is what PlayerSessionActor expects (e.g., JSON string)
			s.actorSystem.Root.Send(playerSessionPID, &messages.ClientMessage{Payload: payloadBuf})
		} else {
			// This case should ideally not be reached if PIDs are managed correctly
			log.Printf("[%s] Warning: No PlayerSessionPID. Cannot process message.", conn.RemoteAddr())
			// Not echoing back anymore as the protocol is more defined.
		}

		// Check for server shutdown signal
		select {
		case <-s.shutdown:
			log.Printf("[%s] Server shutting down, closing connection handler.", conn.RemoteAddr())
			if playerSessionPID != nil {
				s.actorSystem.Root.Send(playerSessionPID, &messages.ClientDisconnected{Reason: "Server shutdown"})
			}
			conn.Close()
			return
		default:
			// continue reading
		}
	}
}

func (s *TCPServer) handleReadError(conn net.Conn, sessionPID *actor.PID, err error, context string) {
	errMsg := ""
	if err == io.EOF {
		log.Printf("[%s] Connection closed by client (EOF) while %s.\n", conn.RemoteAddr(), context)
		errMsg = "EOF"
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		log.Printf("[%s] Connection timeout while %s.\n", conn.RemoteAddr(), context)
		errMsg = "Timeout"
	} else {
		log.Printf("[%s] Error reading from connection while %s: %v\n", conn.RemoteAddr(), context, err)
		errMsg = err.Error()
	}

	if sessionPID != nil {
		s.actorSystem.Root.Send(sessionPID, &messages.ClientDisconnected{Reason: errMsg})
	}
	conn.Close() // Ensure connection is closed
}

// Stop gracefully shuts down the TCP server.
func (s *TCPServer) Stop() {
	log.Println("Attempting to stop TCP Server...")
	close(s.shutdown) // Signal all goroutines (acceptConnections, handleConnection) to stop

	if s.listener != nil {
		if err := s.listener.Close(); err != nil { // This will cause Accept() to return an error
			log.Printf("Error closing TCP listener: %v\n", err)
		} else {
			log.Println("TCP listener closed.")
		}
	} else {
		log.Println("TCP listener was not active or already closed.")
	}

	// Wait for all connection handlers and the accept loop goroutine to finish.
	// A timeout can be added here to prevent hanging indefinitely if a goroutine is stuck.
	shutdownCompleted := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(shutdownCompleted)
	}()

	select {
	case <-shutdownCompleted:
		log.Println("TCP Server all goroutines finished.")
	case <-time.After(10 * time.Second): // Timeout for graceful shutdown
		log.Println("TCP Server shutdown timed out waiting for goroutines.")
	}
	log.Println("TCP Server stopped successfully.")
}

// trimNewlineCharsBytes is no longer needed with length-prefixing, but kept for reference if other parts use it.
func trimNewlineCharsBytes(b []byte) []byte {
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	if len(b) > 0 && b[len(b)-1] == '\r' {
		b = b[:len(b)-1]
	}
	return b
}

// Note: The PlayerSessionActor (server/internal/actor/session_actor.go)
// is responsible for:
// 1. Receiving `ClientConnected` (with net.Conn) and `ClientMessage` (with raw payload []byte).
// 2. Parsing `ClientMessage.Payload` (e.g., from JSON []byte) into game-specific commands/messages.
// 3. Interacting with other actors (RoomActor, WorldManagerActor, etc.).
// 4. Sending `messages.ForwardToClient` (which contains payload []byte for the client).
//    The PlayerSessionActor needs to prepend the length prefix before writing to net.Conn.
// 5. Handling `ClientDisconnected` for cleanup.
