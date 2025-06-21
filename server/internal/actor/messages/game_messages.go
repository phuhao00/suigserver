package messages

// This file can contain messages for broader game logic,
// interactions with systems like WorldManagerActor, CombatEngineActor (if they become actors), etc.

// Example: Player issued a command that needs processing by game logic
// type PlayerCommand struct {
// 	PlayerID  string
// 	SessionPID *actor.PID
// 	Command   string      // e.g., "/spawn_npc", "/start_event"
// 	Args      []string
// }

// Example: Message to a WorldManagerActor to update some world state
// type UpdateWorldState struct {
// 	RegionID string
// 	NewState interface{}
// }

// Example: Message to initiate combat between two entities
// type InitiateCombat struct {
//  AttackerID string
//  AttackerPID *actor.PID // Could be a player or NPC actor
//  DefenderID string
//  DefenderPID *actor.PID // Could be a player or NPC actor
// }

// Example: Combat outcome message
// type CombatResult struct {
//  WinnerID string
//  LoserID string
//  DamageDealt int
//  Rewards map[string]interface{} //
// }

// For now, this file will be mostly a placeholder.
// Specific messages will be added as these higher-level actor interactions are designed.

// Ping is a simple message that can be used for health checks or keep-alives.
type Ping struct {
	Timestamp int64
}

// Pong is the response to a Ping.
type Pong struct {
	Timestamp    int64
	ResponseTime int64
}
