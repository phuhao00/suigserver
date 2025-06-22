package game

import (
	"log"
	"net"
	"time"

	"github.com/phuhao00/suigserver/server/internal/sui"
)

type Server struct {
	listener  net.Listener
	suiClient *sui.Client
	quit      chan struct{}
}

func NewServer(suiClient *sui.Client) *Server {
	return &Server{
		suiClient: suiClient,
		quit:      make(chan struct{}),
	}
}

func (s *Server) Run() {
	var err error
	s.listener, err = net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("listen error: %v", err)
	}
	log.Println("Game server started on :9000")

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.quit:
				return
			default:
				log.Printf("accept error: %v", err)
				continue
			}
		}
		go s.handleConn(conn)
	}
}

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

	// TODO: Protocol Parsing: Implement a more robust protocol than simple newline-delimited JSON.
	//       Consider length-prefixing messages or using a binary format like Protocol Buffers.
	//       For now, we'll use newline-delimited JSON for simplicity.
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

		// TODO: Authentication: Implement proper authentication logic.
		//       This could involve checking a token against a database or auth service.
		//       For now, a simple "AUTH" message type is handled.
		if !isAuthenticated {
			if msg.Type == "AUTH" {
				authPayloadMap, ok := msg.Payload.(map[string]interface{})
				if !ok {
					log.Printf("Invalid AUTH payload from %s: not a map.", conn.RemoteAddr())
					s.sendErrorResponse(conn, "AUTH_FAILED", "Invalid auth payload structure.")
					continue
				}
				// Example: Simple token check
				pID, _ := authPayloadMap["playerId"].(string)
				token, _ := authPayloadMap["token"].(string)

				if pID != "" && token == "dummy_secret_token" { // Replace with real auth
					isAuthenticated = true
					playerID = pID
					log.Printf("Player %s (%s) authenticated successfully.", playerID, conn.RemoteAddr())
					s.sendResponse(conn, "AUTH_SUCCESS", map[string]string{"playerId": playerID, "status": "Authenticated"})
				} else {
					log.Printf("Authentication failed for %s. PlayerID: '%s', Token: '%s'", conn.RemoteAddr(), pID, token)
					s.sendErrorResponse(conn, "AUTH_FAILED", "Invalid credentials.")
				}
			} else {
				log.Printf("Message type '%s' received from unauthenticated client %s. Ignoring.", msg.Type, conn.RemoteAddr())
				s.sendErrorResponse(conn, "NOT_AUTHENTICATED", "Please authenticate first.")
			}
			continue // Require authentication before processing other messages
		}

		// TODO: Message Handling: Route authenticated messages to appropriate game logic handlers or actors.
		//       This is where commands like "MOVE", "ATTACK", "CHAT" would be processed.
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
		if err != nil {
			log.Printf("read error: %v", err)
			return
		}
		log.Printf("Received: %s", string(buf[:n]))
		// 示例：与Sui交互
		s.suiClient.CallMoveFunction("game", "on_message", []interface{}{string(buf[:n])})
		conn.Write([]byte("pong"))
	}
}

func (s *Server) Stop() {
	close(s.quit)
	if s.listener != nil {
		s.listener.Close()
	}
	time.Sleep(time.Second) // 等待连接关闭
}
