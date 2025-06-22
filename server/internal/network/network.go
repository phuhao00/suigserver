package network

import (
	"bufio"
	"io"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	sessionactor "github.com/phuhao00/suigserver/server/internal/actor" // Alias for the actor package
	"github.com/phuhao00/suigserver/server/internal/actor/messages"
)

// TCPServer manages TCP client connections and interfaces with the actor system.
type TCPServer struct {
	listener       net.Listener
	port           int
	actorSystem    *actor.ActorSystem
	wg             sync.WaitGroup
	shutdown       chan struct{}
	roomManagerPID *actor.PID // PID of the RoomManagerActor
}

// NewTCPServer creates a new TCPServer.
func NewTCPServer(port int, system *actor.ActorSystem, roomManagerPID *actor.PID) *TCPServer {
	log.Printf("Initializing TCP Server for port %d...\n", port)
	return &TCPServer{
		port:           port,
		actorSystem:    system,
		shutdown:       make(chan struct{}),
		roomManagerPID: roomManagerPID,
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
		log.Printf("[%s] CRITICAL: RoomManagerPID is not set in TCPServer. Cannot spawn PlayerSessionActor correctly.", conn.RemoteAddr())
		conn.Close() // Close connection as we can't proceed
		return
	}
	playerSessionProps := sessionactor.Props(s.actorSystem, s.roomManagerPID) // Pass RoomManagerPID
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
	reader := bufio.NewReader(conn)
	for {
		// TODO: Implement proper message framing (e.g., length-prefixing + payload)
		// For now, using newline-delimited messages as a simple placeholder.
		// Consider a timeout for reading to prevent dead connections from holding resources.
		// conn.SetReadDeadline(time.Now().Add(60 * time.Second)) // Example read deadline

		line, err := reader.ReadBytes('\n') // Reads until \n, includes \n in 'line'
		if err != nil {
			if err == io.EOF {
				log.Printf("[%s] Connection closed by client (EOF).\n", conn.RemoteAddr())
			} else if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Printf("[%s] Connection timeout.\n", conn.RemoteAddr())
			} else {
				log.Printf("[%s] Error reading from connection: %v\n", conn.RemoteAddr(), err)
			}
			// Notify PlayerSessionActor about disconnection
			if playerSessionPID != nil {
				s.actorSystem.Root.Send(playerSessionPID, &messages.ClientDisconnected{Reason: err.Error()})
				// Optionally wait for PlayerSessionActor to confirm cleanup before closing conn,
				// or just stop it.
				// s.actorSystem.Root.Stop(playerSessionPID) // Stop the actor
			}
			conn.Close() // Ensure connection is closed
			return       // Exit handleConnection goroutine
		}

		// Trim newline character(s)
		trimmedLine := trimNewlineCharsBytes(line)
		log.Printf("[%s] Received raw: '%s'\n", conn.RemoteAddr(), string(trimmedLine))

		if playerSessionPID != nil {
			s.actorSystem.Root.Send(playerSessionPID, &messages.ClientMessage{Payload: trimmedLine})
		} else {
			// Fallback echo if no actor system linkage (for very basic testing)
			log.Printf("[%s] Warning: No PlayerSessionPID, echoing back.\n", conn.RemoteAddr())
			if _, writeErr := conn.Write(append([]byte("Echo: "), trimmedLine...)); writeErr != nil {
				log.Printf("[%s] Error writing echo: %v", conn.RemoteAddr(), writeErr)
				// No need to notify PlayerSessionActor here as it's not involved.
				conn.Close()
				return
			}
			if _, writeErr := conn.Write([]byte("\n")); writeErr != nil { // Add newline back for client
				log.Printf("[%s] Error writing newline for echo: %v", conn.RemoteAddr(), writeErr)
				conn.Close()
				return
			}
		}

		// Check for server shutdown signal
		select {
		case <-s.shutdown:
			log.Printf("[%s] Server shutting down, closing connection handler.", conn.RemoteAddr())
			// Notify PlayerSessionActor about server shutdown initiated disconnection
			if playerSessionPID != nil {
				s.actorSystem.Root.Send(playerSessionPID, &messages.ClientDisconnected{Reason: "Server shutdown"})
				// s.actorSystem.Root.Stop(playerSessionPID) // Stop the actor
			}
			conn.Close()
			return
		default:
			// continue reading
		}
	}
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

// trimNewlineCharsBytes removes trailing \n and \r from a byte slice.
func trimNewlineCharsBytes(b []byte) []byte {
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	if len(b) > 0 && b[len(b)-1] == '\r' {
		b = b[:len(b)-1]
	}
	return b
}

// Note: The PlayerSessionActor (to be created in `server/internal/actor/session_actor.go`)
// will be responsible for:
// 1. Receiving `ClientConnected` and `ClientMessage`.
// 2. Parsing `ClientMessage.Payload` into game-specific commands/messages.
// 3. Interacting with other actors (RoomActor, WorldActor, etc.).
// 4. Sending `messages.ForwardToClient` back to a mechanism that can write to `net.Conn`.
//    This could be done by the PlayerSessionActor holding `net.Conn` and writing directly,
//    or by sending a message to a dedicated "ConnectionWriterActor" if more complex write management is needed.
//    For simplicity, PlayerSessionActor might write directly.
// 5. Handling `ClientDisconnected` for cleanup.
