package actor

import (
	"fmt"
	"log"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
	"sui-mmo-server/server/internal/actor/messages"
)

// RoomManagerActor manages the lifecycle and discovery of RoomActors.
type RoomManagerActor struct {
	actorSystem *actor.ActorSystem
	rooms       map[string]*actor.PID // Map RoomID to RoomActor PID
	mu          sync.RWMutex          // To protect concurrent access to the rooms map
	nextRoomNum int                   // For generating unique room IDs if not provided
}

// NewRoomManagerActor creates a new RoomManagerActor.
func NewRoomManagerActor(system *actor.ActorSystem) actor.Actor {
	return &RoomManagerActor{
		actorSystem: system,
		rooms:       make(map[string]*actor.PID),
		nextRoomNum: 1,
	}
}

// Receive is the message handling loop for RoomManagerActor.
func (a *RoomManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("[RoomManagerActor %s] Started.", ctx.Self().Id)
		// Example: Pre-spawn a default room
		// defaultRoomID := "default_lobby"
		// defaultRoomProps := Props(defaultRoomID, "Default Lobby", 50, a.actorSystem) // Using RoomActor's Props
		// defaultRoomPID := ctx.SpawnNamed(defaultRoomProps, "room-"+defaultRoomID)
		// a.rooms[defaultRoomID] = defaultRoomPID
		// log.Printf("[RoomManagerActor %s] Pre-spawned default room '%s' with PID %s", ctx.Self().Id, defaultRoomID, defaultRoomPID.Id)

	case *actor.Stopping:
		log.Printf("[RoomManagerActor %s] Stopping. Stopping all managed rooms...", ctx.Self().Id)
		a.mu.RLock()
		for roomID, roomPID := range a.rooms {
			log.Printf("[RoomManagerActor %s] Sending stop signal to room %s (PID: %s)", ctx.Self().Id, roomID, roomPID.Id)
			ctx.Stop(roomPID) // Request room actor to stop
		}
		a.mu.RUnlock()

	case *actor.Stopped:
		log.Printf("[RoomManagerActor %s] Stopped.", ctx.Self().Id)

	case *messages.CreateRoomRequest:
		a.handleCreateRoomRequest(ctx, msg)

	case *messages.FindRoomRequest:
		a.handleFindRoomRequest(ctx, msg)

		// TODO: Add message for when a RoomActor stops, so RoomManager can clean it up from the map.
		// This typically involves the RoomActor sending a Terminated message to its parent/watcher (RoomManagerActor).
		// case *actor.Terminated:
		//     a.handleRoomTerminated(ctx, msg)

	default:
		log.Printf("[RoomManagerActor %s] Received unknown message: %+v", ctx.Self().Id, msg)
	}
}

func (a *RoomManagerActor) handleCreateRoomRequest(ctx actor.Context, msg *messages.CreateRoomRequest) {
	a.mu.Lock()
	defer a.mu.Unlock()

	roomID := msg.RoomID
	if roomID == "" {
		roomID = fmt.Sprintf("room-%d", a.nextRoomNum)
		a.nextRoomNum++
	}

	if _, exists := a.rooms[roomID]; exists {
		log.Printf("[RoomManagerActor %s] Room %s already exists.", ctx.Self().Id, roomID)
		if msg.RequesterPID != nil { // Check if a response is expected
			ctx.Send(msg.RequesterPID, &messages.CreateRoomResponse{
				RoomID:  roomID,
				Success: false,
				Error:   "Room ID already exists.",
			})
		}
		return
	}

	roomName := msg.RoomName
	if roomName == "" {
		roomName = "Room " + roomID
	}
	maxPlayers := msg.MaxPlayers
	if maxPlayers <= 0 {
		maxPlayers = 10 // Default max players
	}

	// Use RoomActor's Props function
	roomProps := Props(roomID, roomName, maxPlayers, a.actorSystem)
	// Spawn the RoomActor as a child of RoomManagerActor for supervision, or use ctx.Spawn/ctx.SpawnNamed for system-level.
	// Spawning as child: roomPID := ctx.SpawnChild(roomProps, "room-"+roomID)
	// Spawning at root level (or context of where RoomManager is, if not root):
	roomPID := ctx.SpawnNamed(roomProps, "room-"+roomID) // Name must be unique system-wide for SpawnNamed at root
	// To ensure unique names if not child, could use: roomPID := ctx.Spawn(roomProps) and then the name is auto-generated.
	// If using SpawnNamed, ensure "room-"+roomID is unique or it will panic.

	a.rooms[roomID] = roomPID
	// ctx.Watch(roomPID) // Watch the child room actor for termination

	log.Printf("[RoomManagerActor %s] Created room '%s' (ID: %s, PID: %s) with max players %d.",
		ctx.Self().Id, roomName, roomID, roomPID.Id, maxPlayers)

	if msg.RequesterPID != nil {
		ctx.Send(msg.RequesterPID, &messages.CreateRoomResponse{
			RoomID:  roomID,
			RoomPID: roomPID,
			Success: true,
		})
	}
}

func (a *RoomManagerActor) handleFindRoomRequest(ctx actor.Context, msg *messages.FindRoomRequest) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	// TODO: Implement actual finding logic based on msg.Criteria.
	// For now, just returns the first available room or a specific one if ID is in criteria.
	var foundRoomID string
	var foundRoomPID *actor.PID

	// Example: if criteria is a string, assume it's a roomID
	if roomIDCriterion, ok := msg.Criteria.(string); ok {
		if pid, exists := a.rooms[roomIDCriterion]; exists {
			foundRoomID = roomIDCriterion
			foundRoomPID = pid
		}
	} else {
		// Fallback: find first room (not a good strategy for production)
		for id, pid := range a.rooms {
			// TODO: Could check if room is not full by querying the RoomActor (would be async)
			foundRoomID = id
			foundRoomPID = pid
			break
		}
	}

	if foundRoomPID != nil {
		log.Printf("[RoomManagerActor %s] Found room %s (PID: %s) for player %s.", ctx.Self().Id, foundRoomID, foundRoomPID.Id, msg.PlayerPID.Id)
		if msg.PlayerPID != nil {
			ctx.Send(msg.PlayerPID, &messages.FindRoomResponse{
				RoomID:  foundRoomID,
				RoomPID: foundRoomPID,
				Found:   true,
			})
		}
	} else {
		log.Printf("[RoomManagerActor %s] No suitable room found for player %s.", ctx.Self().Id, msg.PlayerPID.Id)
		if msg.PlayerPID != nil {
			ctx.Send(msg.PlayerPID, &messages.FindRoomResponse{
				Found: false,
				Error: "No suitable room found.",
			})
		}
	}
}

/*
func (a *RoomManagerActor) handleRoomTerminated(ctx actor.Context, terminated *actor.Terminated) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for roomID, roomPID := range a.rooms {
		if roomPID.Equal(terminated.Who) {
			delete(a.rooms, roomID)
			log.Printf("[RoomManagerActor %s] Room %s (PID: %s) terminated and removed from manager.", ctx.Self().Id, roomID, terminated.Who.Id)
			break
		}
	}
}
*/

// PropsForRoomManager creates actor.Props for RoomManagerActor.
func PropsForRoomManager(system *actor.ActorSystem) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor { return NewRoomManagerActor(system) })
}
