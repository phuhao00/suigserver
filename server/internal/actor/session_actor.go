package actor

import (
package actor

import (
	"fmt"
	"log"
	"net"
	"strings" // For basic message parsing, will be replaced by proper protocol
	"time"    // For heartbeat

	"github.com/asynkron/protoactor-go/actor"
	"github.com/phuhao00/suigserver/server/internal/actor/messages"
)

// PlayerSessionActor manages a single client's connection and game session.
type PlayerSessionActor struct {
	conn            net.Conn
	actorSystem     *actor.ActorSystem // To interact with other actors
	playerID        string             // Set after authentication
	roomPID         *actor.PID         // PID of the room the player is currently in
	roomManagerPID  *actor.PID         // PID of the RoomManagerActor
	worldManagerPID *actor.PID         // PID of the WorldManagerActor, to be injected or discovered
	// other player-specific state

	lastActivity    time.Time // Time of last message from client or significant activity
	heartbeatStopCh chan struct{} // Channel to stop heartbeat goroutine (if any server-side ping)
}

// NewPlayerSessionActor creates a new PlayerSessionActor instance.
// roomManagerPID and worldManagerPID should be passed in or discovered.
// For now, worldManagerPID is passed in. A discovery mechanism (e.g. actor registry) is better for complex apps.
func NewPlayerSessionActor(system *actor.ActorSystem, roomManagerPID *actor.PID, worldManagerPID *actor.PID) actor.Actor {
	return &PlayerSessionActor{
		actorSystem:     system,
		roomManagerPID:  roomManagerPID,
		worldManagerPID: worldManagerPID, // Store this for later use
		heartbeatStopCh: make(chan struct{}),
	}
}

const (
	// clientActivityTimeout is the duration after which a client is disconnected if no messages are received.
	clientActivityTimeout = 90 * time.Second
	// authTimeout is the time allowed for a client to authenticate after connecting.
	authTimeout = 60 * time.Second
)

// Receive is the main message handling loop for the PlayerSessionActor.
func (a *PlayerSessionActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("[%s] PlayerSessionActor started.", ctx.Self().Id)
		// An initial timeout for ClientConnected could be set here if TCPServer doesn't guarantee it.
		// For now, we assume ClientConnected will arrive shortly.
		// If ClientConnected doesn't arrive, this actor might become a zombie.
		// Consider a timeout here if ClientConnected is not guaranteed.

	case *actor.Stopping:
		log.Printf("[%s] PlayerSessionActor stopping: %s", ctx.Self().Id, a.playerID)
		if a.conn != nil {
			a.conn.Close() // Ensure connection is closed when actor stops
		}
		a.cleanupResources(ctx) // Cleanup heartbeat resources and notify other systems

	case *actor.Stopped:
		log.Printf("[%s] PlayerSessionActor stopped: %s", ctx.Self().Id, a.playerID)

	case *actor.ReceiveTimeout:
		log.Printf("[%s] ReceiveTimeout for player %s: No client activity or authentication in time. Stopping session.", ctx.Self().Id, a.playerID)
		if a.conn != nil {
			// Try to inform client, though connection might already be dead
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Timeout due to inactivity or failed authentication. Disconnecting.\n")})
			a.conn.Close()
		}
		ctx.Stop(ctx.Self())

	case *messages.ClientConnected:
		log.Printf("[%s] Received ClientConnected from %s", ctx.Self().Id, msg.Conn.RemoteAddr())
		a.conn = msg.Conn
		a.lastActivity = time.Now()
		ctx.SetReceiveTimeout(authTimeout) // Client has this much time to send auth command

		// Request authentication
		authRequestMsg := &messages.ForwardToClient{Payload: []byte("Welcome! Please authenticate using 'auth <player_id> <token>'.\n")}
		a.handleForwardToClient(authRequestMsg)

	case *messages.ClientMessage:
		log.Printf("[%s] Received ClientMessage from player %s: %s", ctx.Self().Id, a.playerID, string(msg.Payload))
		a.lastActivity = time.Now() // Update last activity time on any client message

		if !a.isAuthenticated() {
			// If not authenticated, keep the authTimeout active.
			// Any message resets it, giving client more time for the 'auth' command.
			ctx.SetReceiveTimeout(authTimeout)
		} else {
			// If authenticated, switch to clientActivityTimeout for general inactivity.
			ctx.SetReceiveTimeout(clientActivityTimeout)
		}
		a.handleClientPayload(ctx, msg.Payload)

	case *messages.ForwardToClient:
		a.handleForwardToClient(msg)

	case *messages.ClientDisconnected:
		log.Printf("[%s] Received ClientDisconnected for player %s: %s. Cleaning up.", ctx.Self().Id, a.playerID, msg.Reason)
		// If in a room, notify the room actor
		if a.roomPID != nil {
			ctx.Send(a.roomPID, &messages.LeaveRoomRequest{PlayerID: a.playerID, PlayerPID: ctx.Self()})
		}
		// Other cleanup is handled in the *actor.Stopping case, which will be triggered by ctx.Stop(ctx.Self())
		if a.conn != nil {
			a.conn.Close() // Ensure conn is closed
		}
		ctx.Stop(ctx.Self()) // Stop this actor instance

	case *messages.AuthenticatePlayer:
		log.Printf("[%s] Authenticating player %s with token '%s'", ctx.Self().Id, msg.PlayerID, msg.Token)
		// Actual authentication logic placeholder
		// In a real app, this would involve checking against a database or auth service.
		// The token should be securely handled.
		success := false
		expectedToken := "dummy_token" // Example token
		if msg.Token == expectedToken {
			a.playerID = msg.PlayerID // Set playerID upon successful authentication
			success = true
			a.lastActivity = time.Now()
			ctx.CancelReceiveTimeout()                   // Authentication successful, cancel auth timeout
			ctx.SetReceiveTimeout(clientActivityTimeout) // Start general client activity timeout
			log.Printf("[%s] Player %s authenticated successfully.", ctx.Self().Id, a.playerID)

			// Notify WorldManager that player has entered
			// The WorldManagerPID should be available to the PlayerSessionActor,
			// e.g., passed during creation or retrieved from a well-known actor registry.
			if a.worldManagerPID != nil {
				log.Printf("[%s] Notifying WorldManager that player %s has entered.", ctx.Self().Id, a.playerID)
				ctx.Send(a.worldManagerPID, &messages.PlayerEnteredWorld{PlayerID: a.playerID, PlayerPID: ctx.Self()})
			} else {
				log.Printf("[%s] WorldManagerPID not set for player %s. Cannot notify about entering world.", ctx.Self().Id, a.playerID)
			}

		} else {
			log.Printf("[%s] Player %s authentication failed (invalid token).", ctx.Self().Id, msg.PlayerID)
			authFailMsg := &messages.ForwardToClient{Payload: []byte("Authentication failed. Invalid credentials.\n")}
			a.handleForwardToClient(authFailMsg)
			// Keep auth timeout active for another attempt, or disconnect after N failed attempts (not implemented here).
			ctx.SetReceiveTimeout(authTimeout)
		}

		authResponse := &messages.PlayerAuthenticated{
			PlayerID: msg.PlayerID,
			Success:  success,
		}
		// Respond to original requester if it was a Request, or just update state.
		// If 'auth' command was parsed and sent to self, ctx.Respond works.
		if ctx.Sender() != nil {
			ctx.Respond(authResponse)
		}


	case *messages.FindRoomResponse: // Response from RoomManagerActor
		log.Printf("[%s] Player %s received FindRoomResponse: Found=%t, RoomID=%s, RoomPID=%s, Error=%s",
			ctx.Self().Id, a.playerID, msg.Found, msg.RoomID, msg.RoomPID, msg.Error)
		if msg.Found && msg.RoomPID != nil {
			joinReq := &messages.JoinRoomRequest{
				PlayerID:  a.playerID,
				PlayerPID: ctx.Self(),
			}
			ctx.Request(msg.RoomPID, joinReq) // Request to join the actual room
		} else {
			errMsg := fmt.Sprintf("Error finding room: %s\n", msg.Error)
			if !msg.Found {
				errMsg = fmt.Sprintf("Room not found for criteria.\n") // msg.Error might contain more details
			}
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte(errMsg)})
		}

	case *messages.JoinRoomResponse: // Response from a RoomActor
		if msg.Success {
			a.roomPID = ctx.Sender() // Assume sender is the RoomActor
			log.Printf("[%s] Player %s successfully joined room %s (RoomActor PID: %s)", ctx.Self().Id, a.playerID, msg.RoomID, a.roomPID.Id)
			joinNotification := &messages.ForwardToClient{Payload: []byte("Successfully joined room: " + msg.RoomID + "\n")}
			a.handleForwardToClient(joinNotification)
		} else {
			log.Printf("[%s] Player %s failed to join room %s: %s", ctx.Self().Id, a.playerID, msg.RoomID, msg.Error)
			failNotification := &messages.ForwardToClient{Payload: []byte("Failed to join room " + msg.RoomID + ": " + msg.Error + "\n")}
			a.handleForwardToClient(failNotification)
		}

	case *messages.RoomChatMessage: // Received from a RoomActor to be forwarded to this client
		chatPayload := []byte("[" + msg.SenderName + "]: " + msg.Message + "\n")
		a.handleForwardToClient(&messages.ForwardToClient{Payload: chatPayload})

	// Example: If client sends specific PING messages for keep-alive
	// case *messages.Ping:
	// 	log.Printf("[%s] Received Ping from client %s.", ctx.Self().Id, a.playerID)
	// 	a.lastActivity = time.Now()
	// 	ctx.SetReceiveTimeout(clientActivityTimeout) // Reset timeout
	// 	// Optionally send a Pong back
	// 	// pongMsg := &messages.ForwardToClient{Payload: []byte("PONG\n")}
	// 	// a.handleForwardToClient(pongMsg)

	default:
		log.Printf("[%s] PlayerSessionActor %s received unknown message: %T %+v", ctx.Self().Id, a.playerID, msg, msg)
	}
}

// cleanupResources performs necessary cleanup when the actor is stopping.
// This includes stopping any heartbeat timers and notifying other systems.
func (a *PlayerSessionActor) cleanupResources(ctx actor.Context) {
	log.Printf("[%s] Cleaning up resources for player %s.", ctx.Self().Id, a.playerID)
	ctx.CancelReceiveTimeout() // Cancel any pending receive timeout

	// If a server-side ping mechanism (e.g., using time.Ticker) was implemented, it would be stopped here.
	// close(a.heartbeatStopCh) // Signal any dedicated goroutine to stop

	// Notify other relevant systems if player was authenticated
	if a.playerID != "" {
		// Notify WorldManagerActor that player has left
		// This is important for tracking players in the game world.
		if a.worldManagerPID != nil {
			log.Printf("[%s] Notifying WorldManager that player %s has left.", ctx.Self().Id, a.playerID)
			ctx.Send(a.worldManagerPID, &messages.PlayerLeftWorld{PlayerID: a.playerID, PlayerPID: ctx.Self()})
		} else {
			log.Printf("[%s] WorldManagerPID not set for player %s. Cannot notify WorldManager about leaving.", ctx.Self().Id, a.playerID)
		}

		// Placeholder for saving player data
		// This would typically involve sending a message to a PlayerDataManagerActor or a similar service.
		log.Printf("[%s] Player %s disconnected. Placeholder: Trigger save player data mechanism.", ctx.Self().Id, a.playerID)
	}
}

// handleClientPayload parses the raw payload from the client and decides what to do.
// RoomManagerPID is injected. RoomActor PIDs are obtained via RoomManagerActor.
// WorldManagerPID is also injected (or could be discovered via a registry).
func (a *PlayerSessionActor) handleClientPayload(ctx actor.Context, payload []byte) {
	command := strings.TrimSpace(string(payload))
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	log.Printf("[%s] Player %s parsed command: %s, Parts: %v", ctx.Self().Id, a.playerID, parts[0], parts)

	switch strings.ToLower(parts[0]) {
	case "auth": // Example: "auth <player_id> <token>"
		if len(parts) >= 3 { // Expecting "auth <player_id> <token>"
			// Token might contain spaces if not handled carefully by client; joining parts[2:] is safer.
			token := strings.Join(parts[2:], " ")
			authMsg := &messages.AuthenticatePlayer{
				PlayerID: parts[1],
				Token:    token,
			}
			// Send to self to process authentication flow.
			// Using RequestSelf ensures that the Sender field is set for ctx.Respond.
			ctx.Request(ctx.Self(), authMsg)
		} else {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Usage: auth <player_id> <token>\n")})
		}
	case "join": // Example: "join <room_criteria_or_id>"
		if !a.isAuthenticated() {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Error: Not authenticated. Use 'auth <player_id> <token>'.\n")})
			return
		}
		if len(parts) >= 2 {
			criteria := strings.Join(parts[1:], " ") // Room criteria can be multi-word
			log.Printf("[%s] Player %s attempting to find and join room with criteria: '%s'", ctx.Self().Id, a.playerID, criteria)

			if a.roomManagerPID == nil {
				log.Printf("[%s] RoomManagerPID not configured for PlayerSessionActor. Cannot join room.", ctx.Self().Id)
				a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Error: Room manager is not available.\n")})
				return
			}

			// Send FindRoomRequest to RoomManagerActor
			ctx.Request(a.roomManagerPID, &messages.FindRoomRequest{
				Criteria:  criteria,
				PlayerPID: ctx.Self(),
			})
			// Response (FindRoomResponse) will be handled in Receive()
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte(fmt.Sprintf("Attempting to find and join room '%s'...\n", criteria))})

		} else {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Usage: join <room_criteria_or_id>\n")})
		}
	case "say": // Example: "say <message>"
		if !a.isAuthenticated() {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Error: Not authenticated.\n")})
			return
		}
		if a.roomPID == nil {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Error: Not in a room. Use 'join <room_id>'.\n")})
			return
		}
		if len(parts) > 1 {
			chatMessage := strings.Join(parts[1:], " ")
			roomChatMessage := &messages.RoomChatMessage{
				SenderID:   a.playerID,
				SenderName: a.playerID, // Could be a character name or display name
				Message:    chatMessage,
			}
			// Send to RoomActor for broadcasting
			ctx.Send(a.roomPID, &messages.BroadcastToRoom{
				SenderPID:     ctx.Self(), // So room can identify sender if needed (e.g. not broadcast back)
				ActualMessage: roomChatMessage,
			})
		} else {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Usage: say <message>\n")})
		}
	case "ping": // Client-initiated ping for keep-alive
		// The reception of any message, including "ping", updates `a.lastActivity`
		// and resets `clientActivityTimeout` in the Receive method's ClientMessage case.
		log.Printf("[%s] Received 'ping' from client %s.", ctx.Self().Id, a.playerID)
		a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("PONG\n")}) // Send PONG back as acknowledgement

	default:
		errMsg := "Unknown command: " + parts[0] + "\n"
		a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte(errMsg)})
	}
}

// handleForwardToClient sends a message payload to the connected client.
func (a *PlayerSessionActor) handleForwardToClient(msg *messages.ForwardToClient) {
	if a.conn == nil {
		log.Printf("[%s] PlayerSessionActor %s: No connection available to forward message.", ctx.Self().Id, a.playerID)
		return
	}
	if _, err := a.conn.Write(msg.Payload); err != nil {
		log.Printf("[%s] PlayerSessionActor %s: Error writing to client %s: %v", ctx.Self().Id, a.playerID, a.conn.RemoteAddr(), err)
		// If write fails, connection might be dead.
		// Consider sending ClientDisconnected to self to trigger cleanup.
		// For now, just log. The read loop in TCPServer or ReceiveTimeout should eventually handle it.
	}
}

func (a *PlayerSessionActor) isAuthenticated() bool {
	return a.playerID != ""
}
