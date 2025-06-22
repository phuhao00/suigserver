// DEPRECATED: This file contains a simplified, non-actor based TCP server implementation.
// The main game server (`server/cmd/game/main.go`) uses an actor-based architecture
// with network handling in `server/internal/network/` and session management in `server/internal/actor/`.
// This file is kept for reference or potential alternative uses but is not part of the active game server.

package game

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net"
	"strings"
	"time"

	"github.com/phuhao00/suigserver/server/internal/sui"
	// For a more robust system, you'd likely use an actor system here as well
	// "github.com/asynkron/protoactor-go/actor"
	// "github.com/phuhao00/suigserver/server/internal/actor"
	// "github.com/phuhao00/suigserver/server/internal/actor/messages"
)

// Message represents a generic message structure for client-server communication.
// This is a very basic example. A real protocol would use something like Protocol Buffers.
type Message struct {
	Type    string      `json:"type"`    // e.g., "AUTH", "PLAYER_ACTION", "CHAT_MESSAGE"
	Payload interface{} `json:"payload"` // Data specific to the message type
}

// AuthPayload is an example payload for an "AUTH" message.
type AuthPayload struct {
	PlayerID string `json:"playerId"`
	Token    string `json:"token"`
}

type Server struct {
	listener  net.Listener
	suiClient *sui.Client
	// actorSystem *actor.ActorSystem // If using actors for connection handling
	// sessionManagerPID *actor.PID     // PID for an actor managing all sessions
	quit chan struct{}
}

func NewServer(suiClient *sui.Client /*actorSystem *actor.ActorSystem*/) *Server {
	return &Server{
		suiClient: suiClient,
		// actorSystem: actorSystem,
		quit: make(chan struct{}),
	}
}

func (s *Server) Run() {
	var err error
	s.listener, err = net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("Failed to start game server listener: %v", err)
	}
	log.Println("Game server started on :9000")

	// If using actors for session management:
	// props := actor.PropsFromProducer(func() actor.Actor { return actor.NewSessionManagerActor(s.actorSystem, s.suiClient) })
	// s.sessionManagerPID = s.actorSystem.Root.Spawn(props)
	// log.Printf("SessionManagerActor spawned with PID: %s", s.sessionManagerPID.Id)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				log.Println("Server shutting down, listener closed.")
				return
			default:
				log.Printf("Accept connection error: %v", err)
				// Consider a short delay before retrying on certain errors
				continue
			}
		}
		log.Printf("Accepted new connection from %s", conn.RemoteAddr())
		// If using actors:
		// s.actorSystem.Root.Send(s.sessionManagerPID, &messages.NewConnection{Conn: conn})
		// If handling directly (as in this simplified version):
		go s.handleConnection(conn)
	}
}

// handleConnection manages a single client connection.
// This is a simplified direct handling. In a production system, this would likely
// be handed off to a PlayerSessionActor or similar.
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		log.Printf("Closing connection from %s", conn.RemoteAddr())
		conn.Close()
	}()

	// TODO (Protocol Parsing): The current newline-delimited JSON is simple but not robust for production.
	// Consider alternatives like:
	//    1. Length-Prefixing: Send the length of the JSON message before the JSON data itself.
	//       The server reads the length, then reads that many bytes for the message.
	//    2. Binary Formats: Protocol Buffers, MessagePack, or FlatBuffers offer more efficient
	//       serialization and deserialization, and often include schema evolution support.
	//    3. WebSockets: If client is browser-based, WebSockets provide a message-based protocol.
	log.Println("[handleConnection] Using simple newline-delimited JSON. For production, consider a more robust protocol.")
	reader := bufio.NewReader(conn)
	isAuthenticated := false // Basic authentication state for this connection
	playerID := ""

	for {
		// Read until newline
		rawMsg, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Printf("Client %s disconnected.", conn.RemoteAddr())
			} else {
				log.Printf("Read error from %s: %v", conn.RemoteAddr(), err)
			}
			return // Terminate connection handling
		}

		rawMsg = strings.TrimSpace(rawMsg)
		if rawMsg == "" {
			continue
		}

		log.Printf("Received from %s: %s", conn.RemoteAddr(), rawMsg)

		var msg Message
		if err := json.Unmarshal([]byte(rawMsg), &msg); err != nil {
			log.Printf("Error unmarshaling message from %s: %v. Message: '%s'", conn.RemoteAddr(), err, rawMsg)
			s.sendErrorResponse(conn, "INVALID_JSON_FORMAT", "Message was not valid JSON.")
			continue
		}

		// TODO (Authentication): Current authentication is a placeholder. Proper authentication should involve:
		//    1. Secure Token Generation: Upon successful login (e.g., username/password, OAuth), issue a secure,
		//       session token (e.g., JWT).
		//    2. Token Verification: For each request (or upon connection for stateful protocols), validate the token's
		//       signature, expiration, and any claims. This might involve a database lookup or a call to an auth service.
		//    3. Secure Storage: Store session information securely if needed.
		//    4. HTTPS/TLS: Ensure transport layer security for all authentication exchanges.
		if !isAuthenticated {
			if msg.Type == "AUTH" {
				authPayloadMap, ok := msg.Payload.(map[string]interface{})
				if !ok {
					log.Printf("Invalid AUTH payload from %s: not a map.", conn.RemoteAddr())
					s.sendErrorResponse(conn, "AUTH_FAILED", "Invalid auth payload structure.")
					continue
				}
				pID, _ := authPayloadMap["playerId"].(string)
				token, _ := authPayloadMap["token"].(string)

				// Placeholder: In a real system, `token` would be validated against a session store or auth service.
				// `playerID` might be derived from the validated token, not taken directly from payload after auth.
				if pID != "" && token == "dummy_secret_token" { // Replace with real auth logic
					isAuthenticated = true
					playerID = pID // In real auth, playerID would come from validated token
					log.Printf("Player %s (%s) authenticated successfully (using placeholder auth).", playerID, conn.RemoteAddr())
					s.sendResponse(conn, "AUTH_SUCCESS", map[string]string{"playerId": playerID, "status": "Authenticated"})
				} else {
					log.Printf("Authentication failed for %s. PlayerID: '%s', Token: '%s' (placeholder auth).", conn.RemoteAddr(), pID, token)
					s.sendErrorResponse(conn, "AUTH_FAILED", "Invalid credentials.")
				}
			} else {
				log.Printf("Message type '%s' received from unauthenticated client %s. Ignoring.", msg.Type, conn.RemoteAddr())
				s.sendErrorResponse(conn, "NOT_AUTHENTICATED", "Please authenticate first.")
			}
			continue // Require authentication before processing other messages
		}

		// TODO (Message Handling): This switch statement is a basic router. For a scalable system:
		//    1. Actor-Based Handling: Each connection/player session could be managed by an actor (e.g., PlayerSessionActor).
		//       Messages would be forwarded to the relevant actor. (Protoactor-go setup is commented out).
		//    2. Command Pattern: Define command objects for each message type, processed by handlers.
		//    3. Service Layer: Route messages to specific service functions that encapsulate business logic.
		//    The choice depends on concurrency model, state management needs, and overall architecture.
		log.Printf("[handleConnection] Authenticated message type '%s' from player %s. Routing (placeholder)...", msg.Type, playerID)
		switch msg.Type {
		case "PLAYER_ACTION":
			log.Printf("Player %s (%s) performed action: %+v", playerID, conn.RemoteAddr(), msg.Payload)
			// Example: Interact with Sui for the action
			// actionData, _ := json.Marshal(msg.Payload)
			// s.suiClient.CallMoveFunction("game_logic_package", "handle_player_action", []interface{}{playerID, string(actionData)})
			s.sendResponse(conn, "ACTION_ACK", map[string]interface{}{"action": msg.Payload, "status": "Processed (Placeholder)"})
		case "CHAT_MESSAGE":
			chatPayload, _ := msg.Payload.(map[string]interface{})
			chatText, _ := chatPayload["text"].(string)
			log.Printf("Player %s (%s) says: %s", playerID, conn.RemoteAddr(), chatText)
			// In a real system, this would be broadcast to other relevant players (e.g., in the same room/area)
			s.sendResponse(conn, "CHAT_RECEIVED", map[string]string{"sender": playerID, "text": chatText, "status": "Received by server (not broadcast)"})

		default:
			log.Printf("Unknown message type '%s' from player %s (%s).", msg.Type, playerID, conn.RemoteAddr())
			s.sendErrorResponse(conn, "UNKNOWN_MESSAGE_TYPE", "Server does not understand message type: "+msg.Type)
		}
	}
}

func (s *Server) sendResponse(conn net.Conn, msgType string, payload interface{}) {
	resp := Message{Type: msgType, Payload: payload}
	jsonResp, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshaling response for %s: %v", conn.RemoteAddr(), err)
		// Attempt to send a basic error if marshaling fails for the intended response
		jsonErr, _ := json.Marshal(Message{Type: "SERVER_ERROR", Payload: "Error creating response."})
		conn.Write(jsonErr)
		conn.Write([]byte("\n")) // Ensure newline for client reader
		return
	}
	conn.Write(jsonResp)
	conn.Write([]byte("\n")) // Add newline delimiter
}

func (s *Server) sendErrorResponse(conn net.Conn, errCode string, errMsg string) {
	s.sendResponse(conn, "ERROR", map[string]string{"code": errCode, "message": errMsg})
}

func (s *Server) Stop() {
	log.Println("Stopping game server...")
	close(s.quit) // Signal Run loop to exit
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			log.Printf("Error closing listener: %v", err)
		}
	}
	// If using actor system for sessions:
	// if s.sessionManagerPID != nil {
	// 	s.actorSystem.Root.Stop(s.sessionManagerPID)
	// }
	// s.actorSystem.Shutdown() // Gracefully shut down actor system

	// Allow time for connections to close, though ideally, this is handled more gracefully
	// by tracking active connections or session actors.
	time.Sleep(1 * time.Second)
	log.Println("Game server stopped.")
}
