package actor

import (
	"log"
	"sync"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/phuhao00/suigserver/server/internal/actor/messages"
)

// WorldManagerActor is responsible for managing the overall game world,
// such as global events, region management, or coordinating large-scale systems.
// It also keeps track of currently active players in the world.
type WorldManagerActor struct {
	actorSystem   *actor.ActorSystem
	activePlayers map[string]*actor.PID // Map PlayerID to PlayerSessionActor PID
	mu            sync.RWMutex          // To protect concurrent access to activePlayers
	// e.g., references to RegionActors, game event schedules, etc.
	// regionManagerPID *actor.PID // Example: PID for a RegionManagerActor
}

// NewWorldManagerActor creates a new WorldManagerActor.
func NewWorldManagerActor(system *actor.ActorSystem) actor.Actor {
	return &WorldManagerActor{
		actorSystem:   system,
		activePlayers: make(map[string]*actor.PID),
		// regionManagerPID: nil, // Initialize or discover later
	}
}

// Receive is the message handling loop for the WorldManagerActor.
func (a *WorldManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("[WorldManagerActor %s] Started.", ctx.Self().Id)
		// Initialization logic here, e.g., load world data, spawn region actors
		// Example: Spawn a RegionManagerActor
		// regionManagerProps := PropsForRegionManager(a.actorSystem)
		// a.regionManagerPID = ctx.Spawn(regionManagerProps)
		// ctx.Watch(a.regionManagerPID)
		// log.Printf("[WorldManagerActor %s] Spawned RegionManagerActor: %s", ctx.Self().Id, a.regionManagerPID.Id)

	case *actor.Stopping:
		log.Printf("[WorldManagerActor %s] Stopping.", ctx.Self().Id)
		// Cleanup logic, e.g., save world state, stop child actors
		// if a.regionManagerPID != nil {
		// 	ctx.Stop(a.regionManagerPID)
		// }
		log.Printf("[WorldManagerActor %s] Currently active players at shutdown: %d", ctx.Self().Id, len(a.activePlayers))

	case *actor.Stopped:
		log.Printf("[WorldManagerActor %s] Stopped.", ctx.Self().Id)

	case *messages.PlayerEnteredWorld:
		a.handlePlayerEnteredWorld(ctx, msg)

	case *messages.PlayerLeftWorld:
		a.handlePlayerLeftWorld(ctx, msg)

	// case *messages.UpdateWorldState:
	//  log.Printf("[WorldManagerActor %s] Received UpdateWorldState: %+v", ctx.Self().Id, msg)
	//  // TODO: Handle world state updates from game logic or other systems.
	//  // This could involve:
	//  // - Updating global game parameters.
	//  // - Triggering world events (e.g., weather changes, NPC invasions).
	//  // - Persisting changes to a database.
	//  // - Forwarding relevant updates to RegionManagerActors or directly to players.
	//  // Example:
	//  // if update, ok := msg.Data.(MyWorldParameterUpdate); ok {
	//  //    a.worldParameters[update.Key] = update.Value
	//  //    log.Printf("[WorldManagerActor] World parameter '%s' updated to '%v'", update.Key, update.Value)
	//  //    a.persistWorldParameters()
	//  //    a.broadcastWorldUpdateToRegions(&messages.WorldParameterChanged{Key: update.Key, Value: update.Value})
	//  // }

	default:
		log.Printf("[WorldManagerActor %s] Received unknown message: %T %+v", ctx.Self().Id, msg, msg)
	}
}

func (a *WorldManagerActor) handlePlayerEnteredWorld(ctx actor.Context, msg *messages.PlayerEnteredWorld) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.activePlayers[msg.PlayerID]; exists {
		log.Printf("[WorldManagerActor %s] Player %s (PID: %s) already marked as active. Ignoring duplicate PlayerEnteredWorld.",
			ctx.Self().Id, msg.PlayerID, msg.PlayerPID.Id)
		return
	}

	a.activePlayers[msg.PlayerID] = msg.PlayerPID
	log.Printf("[WorldManagerActor %s] Player %s (PID: %s) entered world. Total active players: %d",
		ctx.Self().Id, msg.PlayerID, msg.PlayerPID.Id, len(a.activePlayers))

	// TODO: Further logic for when a player enters the world:
	// 1. Assign to a default region/zone or determine based on player's last location.
	//    Example: ctx.Send(a.regionManagerPID, &messages.AssignPlayerToRegion{PlayerID: msg.PlayerID, PlayerPID: msg.PlayerPID})
	// 2. Load persistent player world data if not already handled by SessionActor/PlayerDataActor.
	// 3. Notify nearby players or systems about the new player's presence if necessary (e.g., via region actor).
	// 4. Send initial world state or welcome pack to the player (e.g. list of nearby interactables, current global events).
	//    Example: ctx.Send(msg.PlayerPID, &messages.WorldWelcomeInfo{...})
}

func (a *WorldManagerActor) handlePlayerLeftWorld(ctx actor.Context, msg *messages.PlayerLeftWorld) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.activePlayers[msg.PlayerID]; !exists {
		log.Printf("[WorldManagerActor %s] Player %s (PID: %s) not found in active players list. Ignoring PlayerLeftWorld.",
			ctx.Self().Id, msg.PlayerID, msg.PlayerPID.Id)
		return
	}

	delete(a.activePlayers, msg.PlayerID)
	log.Printf("[WorldManagerActor %s] Player %s (PID: %s) left world. Total active players: %d",
		ctx.Self().Id, msg.PlayerID, msg.PlayerPID.Id, len(a.activePlayers))

	// TODO: Further logic for when a player leaves the world:
	// 1. Notify the player's current region/zone actor to remove them.
	//    Example: if playerRegionPID := a.getPlayerRegion(msg.PlayerID); playerRegionPID != nil {
	//                  ctx.Send(playerRegionPID, &messages.PlayerExitedRegion{PlayerID: msg.PlayerID})
	//             }
	// 2. Trigger saving of player's world-specific persistent data (e.g., last location in world).
	// 3. Clean up any global resources or subscriptions associated with the player in the world context.
}

// PropsForWorldManager creates actor.Props for WorldManagerActor.
func PropsForWorldManager(system *actor.ActorSystem) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor { return NewWorldManagerActor(system) })
}
