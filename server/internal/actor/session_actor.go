package actor

import (
	"fmt"
	"log"
	"net"
	"strings" // For basic message parsing, will be replaced by proper protocol

	"github.com/asynkron/protoactor-go/actor"
	"github.com/phuhao00/suigserver/server/internal/actor/messages"
	// "sui-mmo-server/server/internal/models" // For player data
)

// PlayerSessionActor manages a single client's connection and game session.
type PlayerSessionActor struct {
	conn           net.Conn
	actorSystem    *actor.ActorSystem // To interact with other actors
	playerID       string             // Set after authentication
	roomPID        *actor.PID         // PID of the room the player is currently in
	roomManagerPID *actor.PID         // PID of the RoomManagerActor
	// other player-specific state
}

// NewPlayerSessionActor creates a new PlayerSessionActor instance.
// It's a constructor function used with actor.PropsFromProducer.
func NewPlayerSessionActor(system *actor.ActorSystem, roomManagerPID *actor.PID) actor.Actor {
	return &PlayerSessionActor{
		actorSystem:    system,
		roomManagerPID: roomManagerPID,
	}
}

// Receive is the main message handling loop for the PlayerSessionActor.
func (a *PlayerSessionActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("[%s] PlayerSessionActor started.", ctx.Self().Id)

	case *actor.Stopping:
		log.Printf("[%s] PlayerSessionActor stopping.", ctx.Self().Id)
		if a.conn != nil {
			a.conn.Close() // Ensure connection is closed when actor stops
		}

	case *actor.Stopped:
		log.Printf("[%s] PlayerSessionActor stopped.", ctx.Self().Id)

	case *messages.ClientConnected:
		log.Printf("[%s] Received ClientConnected from %s", ctx.Self().Id, msg.Conn.RemoteAddr())
		a.conn = msg.Conn
		// TODO: Implement heartbeat mechanism
		// TODO: Request authentication, e.g., by sending a message to the client or waiting for auth token

		// Example: Send a welcome message to the client
		welcomeMsg := &messages.ForwardToClient{Payload: []byte("Welcome to the MMO Server! Please authenticate.\n")}
		a.handleForwardToClient(welcomeMsg)

	case *messages.ClientMessage:
		log.Printf("[%s] Received ClientMessage: %s", ctx.Self().Id, string(msg.Payload))
		a.handleClientPayload(ctx, msg.Payload)

	case *messages.ForwardToClient:
		a.handleForwardToClient(msg)

	case *messages.ClientDisconnected:
		log.Printf("[%s] Received ClientDisconnected: %s. Cleaning up.", ctx.Self().Id, msg.Reason)
		// If in a room, notify the room actor
		if a.roomPID != nil {
			ctx.Send(a.roomPID, &messages.LeaveRoomRequest{PlayerID: a.playerID, PlayerPID: ctx.Self()})
		}
		// TODO: Notify other relevant systems (e.g., save player data)
		if a.conn != nil {
			a.conn.Close() // Ensure conn is closed
		}
		ctx.Stop(ctx.Self()) // Stop this actor instance

	case *messages.AuthenticatePlayer: // This message might be sent by this actor to itself after parsing, or from an AuthActor
		log.Printf("[%s] Authenticating player %s", ctx.Self().Id, msg.PlayerID)
		// TODO: Actual authentication logic here (e.g., check token against DB/auth service)
		// For now, assume success
		a.playerID = msg.PlayerID
		authResponse := &messages.PlayerAuthenticated{
			PlayerID: msg.PlayerID,
			Success:  true,
			// PlayerActorPID: pidForPlayerData, // If there's a separate actor for persistent player data
		}
		// This message could be sent to itself to update state or to the original requester (if any)
		ctx.Respond(authResponse) // If using Request/Response pattern
		log.Printf("[%s] Player %s authenticated.", ctx.Self().Id, a.playerID)

		// Example: After auth, maybe try to join a default room or send to lobby actor
		// if a.roomManagerPID != nil {
		//    ctx.Request(a.roomManagerPID, &messages.FindRoomRequest{PlayerPID: ctx.Self(), Criteria: "default_lobby"})
		// }

	case *messages.FindRoomResponse: // Response from RoomManagerActor
		log.Printf("[%s] Received FindRoomResponse: Found=%t, RoomID=%s, RoomPID=%s, Error=%s",
			ctx.Self().Id, msg.Found, msg.RoomID, msg.RoomPID, msg.Error)
		if msg.Found && msg.RoomPID != nil {
			// Room found, now send a JoinRoomRequest to the RoomActor
			joinReq := &messages.JoinRoomRequest{
				PlayerID:  a.playerID,
				PlayerPID: ctx.Self(),
			}
			ctx.Request(msg.RoomPID, joinReq) // Request to join the actual room
			// The response to this (JoinRoomResponse) will be handled by the next case.
		} else {
			// Room not found or error occurred
			errMsg := fmt.Sprintf("Error finding room: %s\n", msg.Error)
			if !msg.Found {
				errMsg = fmt.Sprintf("Room not found: %s\n", msg.Error) // Or based on criteria in original request
			}
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte(errMsg)})
		}

	case *messages.JoinRoomResponse: // Response from a RoomActor
		if msg.Success {
			// Assuming the sender of JoinRoomResponse is the RoomActor itself.
			a.roomPID = ctx.Sender()
			log.Printf("[%s] Player %s successfully joined room %s (RoomActor PID: %s)", ctx.Self().Id, a.playerID, msg.RoomID, a.roomPID.Id)
			// Notify client they joined the room
			joinNotification := &messages.ForwardToClient{Payload: []byte("Successfully joined room: " + msg.RoomID + "\n")}
			a.handleForwardToClient(joinNotification)
		} else {
			log.Printf("[%s] Player %s failed to join room %s: %s", ctx.Self().Id, a.playerID, msg.RoomID, msg.Error)
			// Notify client about failure
			failNotification := &messages.ForwardToClient{Payload: []byte("Failed to join room " + msg.RoomID + ": " + msg.Error + "\n")}
			a.handleForwardToClient(failNotification)
		}

	case *messages.RoomChatMessage: // Received from a RoomActor to be forwarded to this client
		chatPayload := []byte("[" + msg.SenderName + "]: " + msg.Message + "\n")
		a.handleForwardToClient(&messages.ForwardToClient{Payload: chatPayload})

	default:
		log.Printf("[%s] PlayerSessionActor received unknown message: %+v", ctx.Self().Id, msg)
	}
}

// handleClientPayload parses the raw payload from the client and decides what to do.
func (a *PlayerSessionActor) handleClientPayload(ctx actor.Context, payload []byte) {
	// This is where a proper command/protocol parser would go.
	// For now, simple string matching.
	command := strings.TrimSpace(string(payload))
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return
	}

	log.Printf("[%s] Parsed command: %s, Parts: %v", ctx.Self().Id, parts[0], parts)

	switch strings.ToLower(parts[0]) {
	case "auth": // Example: "auth <player_id> <token>"
		if len(parts) >= 2 { // Simplified: using player_id as token for now
			// In a real system, this would go to an AuthenticationActor or service
			// For now, handle directly for simplicity
			authMsg := &messages.AuthenticatePlayer{
				PlayerID: parts[1],
				Token:    "dummy_token", // parts[2] if token exists
			}
			// Send to self to process authentication flow
			ctx.Send(ctx.Self(), authMsg)
		} else {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Usage: auth <player_id>\n")})
		}
	case "join": // Example: "join <room_id>"
		if !a.isAuthenticated() {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Error: Not authenticated. Use 'auth <player_id>'.\n")})
			return
		}
		if len(parts) == 2 {
			roomID := parts[1]
			// TODO: Need a way to get RoomManagerActor PID or specific RoomActor PID
			// For now, assume we have a known RoomManager PID or a way to discover Room PIDs.
			// This is a placeholder for actual room discovery/management.
			log.Printf("[%s] Player %s attempting to join room %s", ctx.Self().Id, a.playerID, roomID)

			// --- Placeholder for Room Discovery/Creation ---
			// In a real system, you'd ask a RoomManagerActor or use a discovery mechanism.
			// For this example, let's assume a RoomActor with a predictable name/ID might exist
			// or we try to spawn one directly if we knew its fixed name (not typical for dynamic rooms).

			// Example: Directly find or spawn a known room actor (for testing purposes)
			// This is NOT a scalable approach for dynamic rooms.
			log.Printf("[%s] Player %s attempting to join room %s", ctx.Self().Id, a.playerID, roomID)

			if a.roomManagerPID == nil {
				log.Printf("[%s] RoomManagerPID not configured for PlayerSessionActor. Cannot join room.", ctx.Self().Id)
				a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Error: Room manager is not available.\n")})
				return
			}

			// Send FindRoomRequest to RoomManagerActor
			// The criteria could be more complex, here we use roomID as the criteria.
			ctx.Request(a.roomManagerPID, &messages.FindRoomRequest{
				Criteria:  roomID, // Player is requesting a specific room ID
				PlayerPID: ctx.Self(),
			})
			// The response (FindRoomResponse) will be handled in this actor's Receive method.
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte(fmt.Sprintf("Attempting to find and join room '%s'...\n", roomID))})

		} else {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Usage: join <room_id>\n")})
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
				SenderName: a.playerID, // Could be a character name
				Message:    chatMessage,
			}
			// Send to RoomActor for broadcasting
			ctx.Send(a.roomPID, &messages.BroadcastToRoom{
				SenderPID:     ctx.Self(),
				ActualMessage: roomChatMessage,
			})
		} else {
			a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte("Usage: say <message>\n")})
		}

	default:
		errMsg := "Unknown command: " + parts[0] + "\n"
		a.handleForwardToClient(&messages.ForwardToClient{Payload: []byte(errMsg)})
	}
}

// handleForwardToClient sends a message payload to the connected client.
func (a *PlayerSessionActor) handleForwardToClient(msg *messages.ForwardToClient) {
	if a.conn == nil {
		log.Printf("PlayerSessionActor: No connection available to forward message.")
		return
	}
	if _, err := a.conn.Write(msg.Payload); err != nil {
		log.Printf("PlayerSessionActor: Error writing to client %s: %v", a.conn.RemoteAddr(), err)
		// If write fails, connection might be dead. Consider stopping the actor.
		// This could also be where a ClientDisconnected message is generated and sent to self.
		// For now, just log. The read loop in TCPServer should detect EOF/errors eventually.
	}
}

func (a *PlayerSessionActor) isAuthenticated() bool {
	return a.playerID != ""
}

// Props creates actor.Props for PlayerSessionActor.
// This is how other actors or the system will spawn PlayerSessionActors.
func Props(actorSystem *actor.ActorSystem, roomManagerPID *actor.PID) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor { return NewPlayerSessionActor(actorSystem, roomManagerPID) })
}
