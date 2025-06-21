package actor

import (
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"sui-mmo-server/server/internal/actor/messages"
	// "sui-mmo-server/server/internal/models" // For Room model if needed
)

// RoomActor manages the state and interactions within a single game room.
type RoomActor struct {
	actorSystem *actor.ActorSystem
	roomID      string
	roomName    string
	maxPlayers  int
	players     map[string]*actor.PID // Map PlayerID to PlayerSessionActor PID
	// other room-specific state, e.g., game state, NPCs, etc.
}

// NewRoomActor creates a new RoomActor instance.
func NewRoomActor(roomID, roomName string, maxPlayers int, system *actor.ActorSystem) actor.Actor {
	return &RoomActor{
		actorSystem: system,
		roomID:      roomID,
		roomName:    roomName,
		maxPlayers:  maxPlayers,
		players:     make(map[string]*actor.PID),
	}
}

// Receive is the message handling loop for the RoomActor.
func (a *RoomActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("[RoomActor %s - %s] Started.", a.roomID, ctx.Self().Id)

	case *actor.Stopping:
		log.Printf("[RoomActor %s - %s] Stopping. Notifying players...", a.roomID, ctx.Self().Id)
		// Notify all players that the room is closing
		shutdownMsg := &messages.ForwardToClient{Payload: []byte("Room '" + a.roomName + "' is shutting down.\n")}
		for playerID, playerPID := range a.players {
			log.Printf("[RoomActor %s] Sending shutdown message to player %s (PID: %s)", a.roomID, playerID, playerPID.Id)
			ctx.Send(playerPID, shutdownMsg)
			// Also tell PlayerSessionActor to leave this room (or it might get stuck if room just vanishes)
			// This part is tricky; PlayerSessionActor should ideally handle room disappearing.
		}

	case *actor.Stopped:
		log.Printf("[RoomActor %s - %s] Stopped.", a.roomID, ctx.Self().Id)

	case *messages.JoinRoomRequest:
		a.handleJoinRoomRequest(ctx, msg)

	case *messages.LeaveRoomRequest:
		a.handleLeaveRoomRequest(ctx, msg)

	case *messages.BroadcastToRoom:
		a.handleBroadcastToRoom(ctx, msg)

	default:
		log.Printf("[RoomActor %s - %s] Received unknown message: %+v", a.roomID, ctx.Self().Id, msg)
	}
}

func (a *RoomActor) handleJoinRoomRequest(ctx actor.Context, msg *messages.JoinRoomRequest) {
	log.Printf("[RoomActor %s] Join request from Player %s (PID: %s)", a.roomID, msg.PlayerID, msg.PlayerPID.Id)

	if len(a.players) >= a.maxPlayers {
		ctx.Respond(&messages.JoinRoomResponse{
			RoomID:  a.roomID,
			Success: false,
			Error:   "Room is full.",
		})
		return
	}

	if _, exists := a.players[msg.PlayerID]; exists {
		ctx.Respond(&messages.JoinRoomResponse{
			RoomID:  a.roomID,
			Success: false,
			Error:   "Player already in room.",
		})
		return
	}

	a.players[msg.PlayerID] = msg.PlayerPID
	log.Printf("[RoomActor %s] Player %s joined. Total players: %d", a.roomID, msg.PlayerID, len(a.players))

	// Respond to the joining player
	ctx.Respond(&messages.JoinRoomResponse{
		RoomID:  a.roomID,
		Success: true,
		// CurrentRoomState: a.getRoomStateSnapshot(), // Could send current players list etc.
	})

	// Broadcast to other players in the room that a new player joined
	joinBroadcast := &messages.PlayerJoinedRoomBroadcast{
		PlayerID:  msg.PlayerID,
		Timestamp: time.Now().Unix(),
	}
	a.broadcastMessage(ctx, msg.PlayerPID, joinBroadcast) // Exclude the new player from this specific broadcast
}

func (a *RoomActor) handleLeaveRoomRequest(ctx actor.Context, msg *messages.LeaveRoomRequest) {
	log.Printf("[RoomActor %s] Leave request from Player %s (PID: %s)", a.roomID, msg.PlayerID, msg.PlayerPID.Id)

	if actualPID, exists := a.players[msg.PlayerID]; exists {
		// Verify if the PID matches, for security or consistency, though PlayerID is primary key here
		if actualPID.Equal(msg.PlayerPID) {
			delete(a.players, msg.PlayerID)
			log.Printf("[RoomActor %s] Player %s left. Total players: %d", a.roomID, msg.PlayerID, len(a.players))

			// Broadcast to remaining players
			leaveBroadcast := &messages.PlayerLeftRoomBroadcast{
				PlayerID:  msg.PlayerID,
				Timestamp: time.Now().Unix(),
			}
			a.broadcastMessage(ctx, nil, leaveBroadcast) // Send to all remaining

			// Optionally, respond to the leaving player if this message expects a response
			// ctx.Respond(&messages.LeaveRoomResponse{Success: true})
		} else {
			log.Printf("[RoomActor %s] Mismatched PID for leave request. PlayerID: %s, RequestPID: %s, StoredPID: %s",
				a.roomID, msg.PlayerID, msg.PlayerPID.Id, actualPID.Id)
			// ctx.Respond(&messages.LeaveRoomResponse{Success: false, Error: "PID mismatch"})
		}
	} else {
		log.Printf("[RoomActor %s] Player %s not found in room for leave request.", a.roomID, msg.PlayerID)
		// ctx.Respond(&messages.LeaveRoomResponse{Success: false, Error: "Player not in room"})
	}

	// If room becomes empty, it might auto-terminate or notify a manager
	// if len(a.players) == 0 && a.autoTerminate {
	//    log.Printf("[RoomActor %s] Room empty, self-terminating.", a.roomID)
	//    ctx.Stop(ctx.Self())
	// }
}

func (a *RoomActor) handleBroadcastToRoom(ctx actor.Context, msg *messages.BroadcastToRoom) {
	log.Printf("[RoomActor %s] Broadcasting message: %+v", a.roomID, msg.ActualMessage)
	var senderPID *actor.PID
	if msg.ExcludeSender {
		senderPID = msg.SenderPID
	}
	a.broadcastMessage(ctx, senderPID, msg.ActualMessage)
}

// broadcastMessage sends a message to all players in the room, optionally excluding one PID.
func (a *RoomActor) broadcastMessage(ctx actor.Context, excludePID *actor.PID, message interface{}) {
	for playerID, playerPID := range a.players {
		if excludePID != nil && playerPID.Equal(excludePID) {
			continue // Skip the excluded player
		}
		// Important: The message being sent here must be something the PlayerSessionActor understands
		// and can forward to its client. E.g., if `message` is RoomChatMessage, PlayerSessionActor
		// should have a case for it.
		log.Printf("[RoomActor %s] Sending broadcast message type %T to player %s (PID: %s)", a.roomID, message, playerID, playerPID.Id)
		ctx.Send(playerPID, message)
	}
}

// Props creates actor.Props for RoomActor.
func Props(roomID, roomName string, maxPlayers int, system *actor.ActorSystem) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor { return NewRoomActor(roomID, roomName, maxPlayers, system) })
}
