package actor

import (
	"log"

	"github.com/asynkron/protoactor-go/actor"
	// "sui-mmo-server/server/internal/actor/messages"
)

// WorldManagerActor is responsible for managing the overall game world,
// such as global events, region management, or coordinating large-scale systems.
type WorldManagerActor struct {
	actorSystem *actor.ActorSystem
	// e.g., references to RegionActors, game event schedules, etc.
}

// NewWorldManagerActor creates a new WorldManagerActor.
func NewWorldManagerActor(system *actor.ActorSystem) actor.Actor {
	return &WorldManagerActor{
		actorSystem: system,
	}
}

// Receive is the message handling loop for the WorldManagerActor.
func (a *WorldManagerActor) Receive(ctx actor.Context) {
	switch msg := ctx.Message().(type) {
	case *actor.Started:
		log.Printf("[WorldManagerActor %s] Started.", ctx.Self().Id)
		// Initialization logic here, e.g., load world data, spawn region actors

	case *actor.Stopping:
		log.Printf("[WorldManagerActor %s] Stopping.", ctx.Self().Id)
		// Cleanup logic, e.g., save world state, stop child actors

	case *actor.Stopped:
		log.Printf("[WorldManagerActor %s] Stopped.", ctx.Self().Id)

	// case *messages.PlayerEnterWorld:
	//  log.Printf("[WorldManagerActor %s] Player %s (Session: %s) entering world.", ctx.Self().Id, msg.PlayerID, msg.SessionPID.Id)
	//  // TODO: Logic for when a player enters the world
	//  // - Find or assign to a region/zone actor
	//  // - Notify other systems or players if needed

	// case *messages.PlayerLeaveWorld:
	//  log.Printf("[WorldManagerActor %s] Player %s (Session: %s) leaving world.", ctx.Self().Id, msg.PlayerID, msg.SessionPID.Id)
	//  // TODO: Logic for when a player leaves the world
	//  // - Notify region/zone actor
	//  // - Cleanup any global state related to the player

	// case *messages.UpdateWorldState:
	//  log.Printf("[WorldManagerActor %s] Received UpdateWorldState: %+v", ctx.Self().Id, msg)
	//  // TODO: Handle world state updates

	default:
		log.Printf("[WorldManagerActor %s] Received unknown message: %+v", ctx.Self().Id, msg)
	}
}

// PropsForWorldManager creates actor.Props for WorldManagerActor.
func PropsForWorldManager(system *actor.ActorSystem) *actor.Props {
	return actor.PropsFromProducer(func() actor.Actor { return NewWorldManagerActor(system) })
}
