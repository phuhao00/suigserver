package network

import (
	"io"
	// "log" // Replaced by utils.LogX
	"net"
	"strconv"
	"sync"
	"time"

	"encoding/binary"

	"github.com/asynkron/protoactor-go/actor"
	sessionactor "github.com/phuhao00/suigserver/server/internal/actor" // Alias for the actor package
	"github.com/phuhao00/suigserver/server/internal/actor/messages"
	"github.com/phuhao00/suigserver/server/internal/sui"   // For sui.SuiClient
	"github.com/phuhao00/suigserver/server/internal/utils" // Logger
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
	roomManagerPID  *actor.PID     // PID of the RoomManagerActor
	worldManagerPID *actor.PID     // PID of the WorldManagerActor
	suiClient       *sui.SuiClient // SUI client instance
	// Auth Configs
	enableDummyAuth bool
	dummyToken      string
	dummyPlayerID   string
}

// NewTCPServer creates a new TCPServer.
// It now requires worldManagerPID, suiClient, and dummy auth configurations.
func NewTCPServer(
	port int,
	system *actor.ActorSystem,
	roomManagerPID *actor.PID,
	worldManagerPID *actor.PID,
	suiClient *sui.SuiClient,
	enableDummyAuth bool,
	dummyToken string,
	dummyPlayerID string,
) *TCPServer {
	utils.LogInfof("Initializing TCP Server for port %d...", port)
	if roomManagerPID == nil {
		utils.LogFatalf("TCPServer: RoomManagerPID cannot be nil")
	}
	if worldManagerPID == nil {
		utils.LogFatalf("TCPServer: WorldManagerPID cannot be nil")
	}
	if suiClient == nil {
		utils.LogFatalf("TCPServer: suiClient cannot be nil")
	}
	// Note: dummyToken and dummyPlayerID can be empty if enableDummyAuth is false.
	// Add checks if they must be non-empty when enableDummyAuth is true, if necessary.
	return &TCPServer{
		port:            port,
		actorSystem:     system,
		shutdown:        make(chan struct{}),
		roomManagerPID:  roomManagerPID,
		worldManagerPID: worldManagerPID,
		suiClient:       suiClient,
		enableDummyAuth: enableDummyAuth,
		dummyToken:      dummyToken,
		dummyPlayerID:   dummyPlayerID,
	}
}

// Start begins listening for TCP connections.
func (s *TCPServer) Start() error {
	listenAddr := ":" + strconv.Itoa(s.port)
	var err error
	s.listener, err = net.Listen("tcp", listenAddr)
	if err != nil {
		utils.LogErrorf("Error starting TCP server on port %d: %v", s.port, err)
		return err
	}
	utils.LogInfof("TCP Server started and listening on %s", listenAddr)

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

func (s *TCPServer) acceptConnections() {
	defer s.wg.Done()
	utils.LogInfo("TCP accept loop started.")
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.shutdown:
				utils.LogInfo("TCP accept loop shutting down.")
				return
			default:
				utils.LogWarnf("Error accepting connection: %v", err)
				if ne, ok := err.(net.Error); ok && !ne.Temporary() {
					utils.LogErrorf("Permanent error in accept: %v. Shutting down accept loop.", err)
					return
				}
				continue
			}
		}
		utils.LogInfof("Accepted new connection from %s", conn.RemoteAddr())

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
	clientAddr := conn.RemoteAddr().String()
	utils.LogDebugf("Handling new connection for %s", clientAddr)

	// Note: PlayerSessionActor props are now used directly.
	// The old TODO about replacing them was based on an earlier structure.
	if s.roomManagerPID == nil {
		utils.LogFatalf("[%s] CRITICAL: RoomManagerPID is not set in TCPServer. Cannot spawn PlayerSessionActor correctly.", clientAddr)
		return
	}
	if s.worldManagerPID == nil {
		utils.LogFatalf("[%s] CRITICAL: WorldManagerPID is not set in TCPServer. Cannot spawn PlayerSessionActor correctly.", clientAddr)
		return
	}
	if s.suiClient == nil {
		utils.LogFatalf("[%s] CRITICAL: SuiClient is not set in TCPServer. Cannot spawn PlayerSessionActor correctly.", clientAddr)
		return
	}

	// PlayerSessionActor now requires worldManagerPID, suiClient, and auth configs.
	playerSessionProps := sessionactor.PropsForPlayerSession(
		s.actorSystem,
		s.roomManagerPID,
		s.worldManagerPID,
		s.suiClient,
		s.enableDummyAuth,
		s.dummyToken,
		s.dummyPlayerID,
	)
	playerSessionPID := s.actorSystem.Root.Spawn(playerSessionProps)
	utils.LogInfof("[%s] Spawned PlayerSessionActor with PID: %s", clientAddr, playerSessionPID.String())

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
			utils.LogWarnf("[%s] Received message with zero length. Ignoring.", clientAddr)
			continue // Or treat as an error/disconnect
		}
		if messageLength > MaxMessageSize {
			utils.LogWarnf("[%s] Message length %d exceeds MaxMessageSize %d. Closing connection.",
				clientAddr, messageLength, MaxMessageSize)
			s.actorSystem.Root.Send(playerSessionPID, &messages.ClientDisconnected{Reason: "Message too large"})
			conn.Close()
			return
		}

		// 3. Read the message payload
		payloadBuf := make([]byte, messageLength)
		_, err = io.ReadFull(conn, payloadBuf)
		if err != nil {
			s.handleReadError(conn, playerSessionPID, err, "reading payload") // handleReadError uses utils.Log
			return
		}

		utils.LogDebugf("[%s] Received %d bytes. Payload: '%s'", clientAddr, messageLength, string(payloadBuf))

		if playerSessionPID != nil {
			// The payloadBuf is what PlayerSessionActor expects (e.g., JSON string)
			s.actorSystem.Root.Send(playerSessionPID, &messages.ClientMessage{Payload: payloadBuf})
		} else {
			// This case should ideally not be reached if PIDs are managed correctly
			utils.LogWarnf("[%s] Warning: No PlayerSessionPID. Cannot process message.", clientAddr)
			// Not echoing back anymore as the protocol is more defined.
		}

		// Check for server shutdown signal
		select {
		case <-s.shutdown:
			utils.LogInfof("[%s] Server shutting down, closing connection handler.", clientAddr)
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
	clientAddr := conn.RemoteAddr().String()
	errMsg := ""
	if err == io.EOF {
		utils.LogInfof("[%s] Connection closed by client (EOF) while %s.", clientAddr, context)
		errMsg = "EOF"
	} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
		utils.LogWarnf("[%s] Connection timeout while %s.", clientAddr, context)
		errMsg = "Timeout"
	} else {
		utils.LogErrorf("[%s] Error reading from connection while %s: %v", clientAddr, context, err)
		errMsg = err.Error()
	}

	if sessionPID != nil {
		s.actorSystem.Root.Send(sessionPID, &messages.ClientDisconnected{Reason: errMsg})
	}
	conn.Close() // Ensure connection is closed
}

// Stop gracefully shuts down the TCP server.
func (s *TCPServer) Stop() {
	utils.LogInfo("Attempting to stop TCP Server...")
	close(s.shutdown) // Signal all goroutines (acceptConnections, handleConnection) to stop

	if s.listener != nil {
		if err := s.listener.Close(); err != nil { // This will cause Accept() to return an error
			utils.LogErrorf("Error closing TCP listener: %v", err)
		} else {
			utils.LogInfo("TCP listener closed.")
		}
	} else {
		utils.LogInfo("TCP listener was not active or already closed.")
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
		utils.LogInfo("TCP Server all goroutines finished.")
	case <-time.After(10 * time.Second): // Timeout for graceful shutdown
		utils.LogWarn("TCP Server shutdown timed out waiting for goroutines.")
	}
	utils.LogInfo("TCP Server stopped successfully.")
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
