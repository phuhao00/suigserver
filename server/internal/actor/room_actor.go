package actor

import (
	"log"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/phuhao00/suigserver/server/internal/actor/messages"
	// "sui-mmo-server/server/internal/models" // For Room model if needed
)

// RoomActor manages the state and interactions within a single game room.
type RoomActor struct {
	actorSystem    *actor.ActorSystem
	roomID         string
	roomName       string
	maxPlayers     int
	players        map[string]*actor.PID // Map PlayerID to PlayerSessionActor PID
	roomManagerPID *actor.PID            // PID of the RoomManagerActor to send updates
	// other room-specific state, e.g., game state, NPCs, etc.
}

// NewRoomActor creates a new RoomActor instance.
// It now requires roomManagerPID to send updates like player count.
func NewRoomActor(roomID, roomName string, maxPlayers int, system *actor.ActorSystem, roomManagerPID *actor.PID) actor.Actor {
	return &RoomActor{
		actorSystem:    system,
		roomID:         roomID,
		roomName:       roomName,
		maxPlayers:     maxPlayers,
		players:        make(map[string]*actor.PID),
		roomManagerPID: roomManagerPID,
	}
}

// Receive is the message handling loop for the RoomActor.
func (a *RoomActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("[RoomActor %s - %s] Started. Max players: %d.", a.roomID, ctx.Self().Id, a.maxPlayers)
		a.notifyManagerPlayerCountChanged(ctx) // Notify manager on start (0 players)

	case *actor.Stopping:
		log.Printf("[RoomActor %s - %s] Stopping. Notifying players...", a.roomID, ctx.Self().Id)
		// Notify all players that the room is closing
		shutdownMsg := &messages.ForwardToClient{Payload: []byte("Room '" + a.roomName + "' is shutting down.\n")}
		// Create a temporary list of PIDs to avoid issues if a player leaves during this broadcast
		playerPIDsToNotify := make([]*actor.PID, 0, len(a.players))
		for _, pid := range a.players {
			playerPIDsToNotify = append(playerPIDsToNotify, pid)
		}

		for _, playerPID := range playerPIDsToNotify {
			// Check if playerPID is still valid/part of the room before sending,
			// though in Stopping, it's less likely to change rapidly.
			// For robustness, one might re-check `a.players` map.
			ctx.Send(playerPID, shutdownMsg)
		}
		// RoomManager will be notified via actor.Terminated message as it Watches this room.

	case *actor.Stopped:
		log.Printf("[RoomActor %s - %s] Stopped.", a.roomID, ctx.Self().Id)
		// The RoomManagerActor should handle the actor.Terminated message for this room.

	case *messages.JoinRoomRequest:
		a.handleJoinRoomRequest(ctx, msg)

	case *messages.LeaveRoomRequest:
		a.handleLeaveRoomRequest(ctx, msg)

	case *messages.BroadcastToRoom:
		a.handleBroadcastToRoom(ctx, msg)

	default:
		log.Printf("[RoomActor %s - %s] Received unknown message: %T %+v", a.roomID, ctx.Self().Id, msg, msg)
	}
}

func (a *RoomActor) handleJoinRoomRequest(ctx actor.Context, msg *messages.JoinRoomRequest) {
	log.Printf("[RoomActor %s] Join request from Player %s (PID: %s)", a.roomID, msg.PlayerID, msg.PlayerPID.Id)

	if len(a.players) >= a.maxPlayers {
		log.Printf("[RoomActor %s] Join failed for %s: Room is full (%d/%d).", a.roomID, msg.PlayerID, len(a.players), a.maxPlayers)
		ctx.Respond(&messages.JoinRoomResponse{
			RoomID:  a.roomID,
			Success: false,
			Error:   "Room is full.",
		})
		return
	}

	if _, exists := a.players[msg.PlayerID]; exists {
		log.Printf("[RoomActor %s] Join failed for %s: Player already in room.", a.roomID, msg.PlayerID)
		ctx.Respond(&messages.JoinRoomResponse{
			RoomID:  a.roomID,
			Success: false,
			Error:   "Player already in room.",
		})
		return
	}

	a.players[msg.PlayerID] = msg.PlayerPID
	log.Printf("[RoomActor %s] Player %s joined. Total players: %d/%d", a.roomID, msg.PlayerID, len(a.players), a.maxPlayers)

	// Notify RoomManager about player count change
	a.notifyManagerPlayerCountChanged(ctx)

	// Respond to the joining player
	ctx.Respond(&messages.JoinRoomResponse{
		RoomID:  a.roomID,
		Success: true,
		// CurrentRoomState: a.getRoomStateSnapshot(), // TODO: Could send current players list etc.
	})

	// Broadcast to other players in the room that a new player joined
	joinBroadcast := &messages.PlayerJoinedRoomBroadcast{
		PlayerID:  msg.PlayerID,
		Timestamp: time.Now().Unix(),
		// CharacterData: msg.CharacterData, // If character data was part of JoinRoomRequest
	}
	// Send to all other players (exclude the new player from *this* specific broadcast)
	a.broadcastMessage(ctx, msg.PlayerPID, joinBroadcast)
}

func (a *RoomActor) handleLeaveRoomRequest(ctx actor.Context, msg *messages.LeaveRoomRequest) {
	log.Printf("[RoomActor %s] Leave request from Player %s (PID: %s)", a.roomID, msg.PlayerID, msg.PlayerPID.Id)

	if actualPID, exists := a.players[msg.PlayerID]; exists {
		// Verify if the PID matches, for security or consistency
		if msg.PlayerPID != nil && actualPID.Equal(msg.PlayerPID) {
			delete(a.players, msg.PlayerID)
			log.Printf("[RoomActor %s] Player %s left. Total players: %d/%d", a.roomID, msg.PlayerID, len(a.players), a.maxPlayers)

			// Notify RoomManager about player count change
			a.notifyManagerPlayerCountChanged(ctx)

			// Broadcast to remaining players
			leaveBroadcast := &messages.PlayerLeftRoomBroadcast{
				PlayerID:  msg.PlayerID,
				Timestamp: time.Now().Unix(),
			}
			a.broadcastMessage(ctx, nil, leaveBroadcast) // Send to all remaining

			// No explicit response needed for LeaveRoomRequest usually, but can be added if protocol requires
		} else {
			log.Printf("[RoomActor %s] Mismatched or nil PID for leave request. PlayerID: %s, RequestPID: %v, StoredPID: %s. Not removing.",
				a.roomID, msg.PlayerID, msg.PlayerPID, actualPID.Id)
		}
	} else {
		log.Printf("[RoomActor %s] Player %s not found in room for leave request.", a.roomID, msg.PlayerID)
	}

	// Optional: If room becomes empty and is configured to auto-terminate (not implemented here)
	// if len(a.players) == 0 && a.autoTerminateIfEmpty {
	//    log.Printf("[RoomActor %s] Room empty, self-terminating...", a.roomID)
	//    ctx.Stop(ctx.Self()) // This will trigger Terminated message to RoomManager
	// }
}

func (a *RoomActor) handleBroadcastToRoom(ctx actor.Context, msg *messages.BroadcastToRoom) {
	// Example: For RoomChatMessage, log sender and message
	if chatMsg, ok := msg.ActualMessage.(*messages.RoomChatMessage); ok {
		log.Printf("[RoomActor %s] Broadcasting chat from %s: '%s'", a.roomID, chatMsg.SenderName, chatMsg.Message)
	} else {
		log.Printf("[RoomActor %s] Broadcasting generic message of type %T", a.roomID, msg.ActualMessage)
	}

	var senderPID *actor.PID
	if msg.ExcludeSender {
		senderPID = msg.SenderPID
	}
	a.broadcastMessage(ctx, senderPID, msg.ActualMessage)
}

// broadcastMessage sends a message to all players in the room, optionally excluding one PID.
func (a *RoomActor) broadcastMessage(ctx actor.Context, excludePID *actor.PID, message interface{}) {
	if len(a.players) == 0 {
		return // No one to broadcast to
	}
	log.Printf("[RoomActor %s] Broadcasting message type %T to %d players (excluding: %v)",
		a.roomID, message, len(a.players), excludePID != nil)

	for playerID, playerPID := range a.players {
		if excludePID != nil && playerPID.Equal(excludePID) {
			continue // Skip the excluded player
		}
		// The message being sent here must be something the PlayerSessionActor understands
		// and can forward to its client. If it's already a ForwardToClient message, that's fine.
		// If it's a structured message like RoomChatMessage, PlayerSessionActor needs a case for it.
		ctx.Send(playerPID, message)
	}
}

// notifyManagerPlayerCountChanged sends an update to the RoomManagerActor.
func (a *RoomActor) notifyManagerPlayerCountChanged(ctx actor.Context) {
	if a.roomManagerPID == nil {
		log.Printf("[RoomActor %s] RoomManagerPID not set. Cannot notify about player count change.", a.roomID)
		return
	}
	playerCountUpdate := &messages.UpdateRoomPlayerCount{
		RoomID:         a.roomID,
		CurrentPlayers: len(a.players),
		MaxPlayers:     a.maxPlayers, // Send current maxPlayers, in case it's dynamic in future
	}
	ctx.Send(a.roomManagerPID, playerCountUpdate)
	log.Printf("[RoomActor %s] Notified RoomManager. Current players: %d/%d", a.roomID, len(a.players), a.maxPlayers)
}

// Props creates actor.Props for RoomActor.
// It now requires roomManagerPID.
func Props(roomID, roomName string, maxPlayers int, system *actor.ActorSystem, roomManagerPID *actor.PID) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor { return NewRoomActor(roomID, roomName, maxPlayers, system, roomManagerPID) })
}
