package messages

import "github.com/asynkron/protoactor-go/actor"

// AuthenticatePlayer is sent to a PlayerSessionActor with credentials or a token.
type AuthenticatePlayer struct {
	Token    string
	PlayerID string // Or other identifying information
}

// PlayerAuthenticated is sent back from PlayerSessionActor or an AuthActor
// to confirm authentication, potentially including the player's main data PID.
type PlayerAuthenticated struct {
	PlayerID       string
	PlayerActorPID *actor.PID // PID for an actor managing this player's persistent state/data
	Success        bool
	Error          string
}

// LoadPlayerRequest is sent to a data management actor to load player data.
type LoadPlayerRequest struct {
	PlayerID string
}

// LoadPlayerResponse contains the loaded player data or an error.
type LoadPlayerResponse struct {
	PlayerID string
	// PlayerData *models.Player // Assuming models.Player struct exists
	Data    interface{} // Generic data for now
	Success bool
	Error   string
}

// SavePlayerRequest is sent to a data management actor to save player data.
type SavePlayerRequest struct {
	PlayerID string
	// PlayerData *models.Player
	Data interface{}
}

// SavePlayerResponse indicates the result of a save operation.
type SavePlayerResponse struct {
	PlayerID string
	Success  bool
	Error    string
}

// PlayerEnterWorld is sent by PlayerSessionActor after successful authentication
// to a WorldManagerActor or similar, to signal the player is entering the game world.
type PlayerEnterWorld struct {
	PlayerID    string
	SessionPID  *actor.PID // The PID of the PlayerSessionActor
	CharacterID string     // If characters are separate from player accounts
}

// PlayerLeaveWorld is sent when a player session ends or they log out.
type PlayerLeaveWorld struct {
	PlayerID   string
	SessionPID *actor.PID
}

// PlayerEnteredWorld is sent by PlayerSessionActor after successful authentication
// to a WorldManagerActor or similar, to signal the player has entered the game world.
type PlayerEnteredWorld struct {
	PlayerID  string
	PlayerPID *actor.PID // The PID of the PlayerSessionActor
}

// PlayerLeftWorld is sent when a player session ends or they log out.
type PlayerLeftWorld struct {
	PlayerID  string
	PlayerPID *actor.PID
}
