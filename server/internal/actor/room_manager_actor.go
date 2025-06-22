package actor

import (
	"fmt"
	"log"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/phuhao00/suigserver/server/internal/actor/messages"
)

// RoomManagerActor manages the lifecycle and discovery of RoomActors.
type RoomManagerActor struct {
	actorSystem *actor.ActorSystem
	rooms       map[string]*actor.PID // Map RoomID to RoomActor PID
	roomInfo    map[string]RoomInfo   // Map RoomID to RoomInfo (name, maxPlayers, currentPlayers)
	mu          sync.RWMutex          // To protect concurrent access to the rooms map and roomInfo
	nextRoomNum int                   // For generating unique room IDs if not provided
}

// RoomInfo holds metadata about a room.
type RoomInfo struct {
	ID             string
	Name           string
	MaxPlayers     int
	CurrentPlayers int
	PID            *actor.PID
}

// NewRoomManagerActor creates a new RoomManagerActor.
func NewRoomManagerActor(system *actor.ActorSystem) actor.Actor {
	return &RoomManagerActor{
		actorSystem: system,
		rooms:       make(map[string]*actor.PID),
		roomInfo:    make(map[string]RoomInfo),
		nextRoomNum: 1,
	}
}

// Receive is the message handling loop for RoomManagerActor.
func (a *RoomManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("[RoomManagerActor %s] Started.", ctx.Self().Id)
		// Example: Pre-spawn a default room
		// a.createDefaultRoom(ctx)

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

	case *actor.Terminated:
		// This message is received when a child/watched actor stops.
		a.handleRoomTerminated(ctx, msg)

	case *messages.UpdateRoomPlayerCount:
		a.handleUpdateRoomPlayerCount(ctx, msg)

	default:
		log.Printf("[RoomManagerActor %s] Received unknown message: %T %+v", ctx.Self().Id, msg, msg)
	}
}

// createDefaultRoom is an example helper to pre-spawn a room.
func (a *RoomManagerActor) createDefaultRoom(ctx actor.Context) {
	defaultRoomID := "lobby"
	roomName := "Default Lobby"
	maxPlayers := 50

	// Use RoomActor's Props function (ensure RoomActor.Props is defined and accessible)
	// Assuming RoomActor.Props exists like: func Props(roomID, roomName string, maxPlayers int, actorSystem *actor.ActorSystem, roomManagerPID *actor.PID) *actor.Props
	roomProps := Props(defaultRoomID, roomName, maxPlayers, a.actorSystem, ctx.Self())
	roomPID := ctx.SpawnNamed(roomProps, "room-"+defaultRoomID)

	a.mu.Lock()
	a.rooms[defaultRoomID] = roomPID
	a.roomInfo[defaultRoomID] = RoomInfo{
		ID:             defaultRoomID,
		Name:           roomName,
		MaxPlayers:     maxPlayers,
		CurrentPlayers: 0, // Initially empty
		PID:            roomPID,
	}
	a.mu.Unlock()

	ctx.Watch(roomPID) // Watch the room actor for termination
	log.Printf("[RoomManagerActor %s] Pre-spawned default room '%s' (ID: %s, PID: %s)", ctx.Self().Id, roomName, defaultRoomID, roomPID.Id)
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
		if msg.RequesterPID != nil {
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

	// Pass RoomManager's PID (ctx.Self()) to the RoomActor so it can send updates (e.g. player count)
	roomProps := Props(roomID, roomName, maxPlayers, a.actorSystem, ctx.Self())
	roomPID := ctx.SpawnNamed(roomProps, "room-"+roomID) // Ensure "room-"+roomID is unique

	a.rooms[roomID] = roomPID
	a.roomInfo[roomID] = RoomInfo{
		ID:             roomID,
		Name:           roomName,
		MaxPlayers:     maxPlayers,
		CurrentPlayers: 0, // New room starts with 0 players
		PID:            roomPID,
	}
	ctx.Watch(roomPID) // Watch the child room actor for termination

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

	var foundRoom RoomInfo
	found := false

	// Attempt to find by specific ID if criteria is a string
	if roomIDCriterion, ok := msg.Criteria.(string); ok && roomIDCriterion != "" {
		if info, exists := a.roomInfo[roomIDCriterion]; exists {
			// Check if room is full (basic check, RoomActor itself is the source of truth for instantaneous count)
			if info.CurrentPlayers < info.MaxPlayers {
				foundRoom = info
				found = true
			} else {
				log.Printf("[RoomManagerActor %s] Room %s found but is full (%d/%d players).", ctx.Self().Id, info.ID, info.CurrentPlayers, info.MaxPlayers)
				if msg.PlayerPID != nil {
					ctx.Send(msg.PlayerPID, &messages.FindRoomResponse{
						Found: false,
						Error: fmt.Sprintf("Room '%s' is full.", info.Name),
					})
				}
				return // Early exit as specific room is full
			}
		}
	} else {
		// Fallback: find the first available non-full room (simple matchmaking)
		// More sophisticated matchmaking would consider criteria like game mode, map, player rank etc.
		for _, info := range a.roomInfo {
			if info.CurrentPlayers < info.MaxPlayers {
				foundRoom = info
				found = true
				break // Found a suitable room
			}
		}
	}

	if found {
		log.Printf("[RoomManagerActor %s] Found room %s (Name: %s, PID: %s) for player %s. Players: %d/%d.",
			ctx.Self().Id, foundRoom.ID, foundRoom.Name, foundRoom.PID.Id, msg.PlayerPID.Id, foundRoom.CurrentPlayers, foundRoom.MaxPlayers)
		if msg.PlayerPID != nil {
			ctx.Send(msg.PlayerPID, &messages.FindRoomResponse{
				RoomID:  foundRoom.ID,
				RoomPID: foundRoom.PID,
				Found:   true,
			})
		}
	} else {
		log.Printf("[RoomManagerActor %s] No suitable room found for player %s with criteria '%v'.", ctx.Self().Id, msg.PlayerPID.Id, msg.Criteria)
		if msg.PlayerPID != nil {
			ctx.Send(msg.PlayerPID, &messages.FindRoomResponse{
				Found: false,
				Error: "No suitable room found or the specified room is full/does not exist.",
			})
		}
	}
}

func (a *RoomManagerActor) handleRoomTerminated(ctx actor.Context, terminated *actor.Terminated) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for roomID, roomPID := range a.rooms {
		if roomPID.Equal(terminated.Who) {
			delete(a.rooms, roomID)
			delete(a.roomInfo, roomID)
			log.Printf("[RoomManagerActor %s] Room %s (PID: %s) terminated and removed from manager.", ctx.Self().Id, roomID, terminated.Who.Id)
			// No need to Unwatch, it's automatic for terminated actors.
			break
		}
	}
}

func (a *RoomManagerActor) handleUpdateRoomPlayerCount(ctx actor.Context, msg *messages.UpdateRoomPlayerCount) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if info, exists := a.roomInfo[msg.RoomID]; exists {
		info.CurrentPlayers = msg.CurrentPlayers
		a.roomInfo[msg.RoomID] = info
		log.Printf("[RoomManagerActor %s] Updated player count for room %s to %d/%d.", ctx.Self().Id, msg.RoomID, info.CurrentPlayers, info.MaxPlayers)
	} else {
		log.Printf("[RoomManagerActor %s] Received player count update for unknown room %s.", ctx.Self().Id, msg.RoomID)
	}
}

// PropsForRoomManager creates actor.Props for RoomManagerActor.
func PropsForRoomManager(system *actor.ActorSystem) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor { return NewRoomManagerActor(system) })
}
