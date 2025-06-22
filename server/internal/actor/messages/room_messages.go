package messages

import "github.com/asynkron/protoactor-go/actor"

// --- Room Management Messages (typically to a RoomManagerActor) ---

// CreateRoomRequest is sent to a RoomManagerActor to request a new room.
type CreateRoomRequest struct {
	RoomID   string // Optional, can be auto-generated
	RoomName string
	MaxPlayers int
	// Other room parameters (e.g., map ID, game mode)
	RequesterPID *actor.PID // PID of the actor requesting room creation (e.g. a PlayerSessionActor)
}

// CreateRoomResponse is sent back from RoomManagerActor.
type CreateRoomResponse struct {
	RoomID  string
	RoomPID *actor.PID // PID of the newly created RoomActor
	Success bool
	Error   string
}

// FindRoomRequest is sent to RoomManagerActor to find a suitable room.
type FindRoomRequest struct {
	Criteria interface{} // e.g., map ID, game mode, not full
	PlayerPID *actor.PID
}

// FindRoomResponse provides a room PID or indicates no suitable room found.
type FindRoomResponse struct {
	RoomID  string
	RoomPID *actor.PID
	Found   bool
	Error   string
}

// --- Room Interaction Messages (typically to a specific RoomActor) ---

// JoinRoomRequest is sent to a RoomActor for a player to join.
type JoinRoomRequest struct {
	PlayerID   string
	PlayerPID  *actor.PID // PID of the PlayerSessionActor wishing to join
	// CharacterData interface{} // Potentially some character info
}

// JoinRoomResponse is sent by the RoomActor back to the PlayerSessionActor.
type JoinRoomResponse struct {
	RoomID    string
	Success   bool
	Error     string
	// CurrentRoomState interface{} // Snapshot of room state (e.g., other players)
}

// LeaveRoomRequest is sent to a RoomActor.
type LeaveRoomRequest struct {
	PlayerID  string
	PlayerPID *actor.PID
}

// PlayerJoinedRoomBroadcast is sent by RoomActor to other players in the room.
type PlayerJoinedRoomBroadcast struct {
	PlayerID string
	// CharacterData interface{}
	Timestamp int64
}

// PlayerLeftRoomBroadcast is sent by RoomActor to other players in the room.
type PlayerLeftRoomBroadcast struct {
	PlayerID string
	Timestamp int64
}

// BroadcastToRoom is a generic message to send a payload to all occupants of a room.
// The RoomActor will iterate its members and forward the `ActualMessage`.
type BroadcastToRoom struct {
	ExcludeSender bool       // Whether to exclude the original sender of the action
	SenderPID     *actor.PID // Optional: PID of the original sender
	ActualMessage interface{}  // The message to be broadcast (e.g., ChatMessage, PlayerAction)
}

// RoomChatMessage is an example of an ActualMessage for BroadcastToRoom.
type RoomChatMessage struct {
	SenderID   string
	SenderName string // Optional display name
	Message    string
	Timestamp  int64
}

// UpdateRoomPlayerCount is sent by a RoomActor to its RoomManagerActor
// to inform it about changes in the player count.
type UpdateRoomPlayerCount struct {
	RoomID         string
	CurrentPlayers int
	MaxPlayers     int // Optional, can be useful for manager to know if it changed
}

// PlayerActionInRoom is another example for BroadcastToRoom, representing a game action.
type PlayerActionInRoom struct {
	PlayerID   string
	ActionType string // e.g., "ATTACK", "USE_ITEM"
	TargetID   string // Optional, if action has a target
	Params     map[string]interface{}
	Timestamp  int64
}
