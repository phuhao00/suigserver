package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"sui-mmo-server/server/configs"
	internalActor "sui-mmo-server/server/internal/actor" // Renamed to avoid conflict with protoactor's actor package
	"sui-mmo-server/server/internal/network"
	// Other direct service initializations if any (e.g., DB connection pools)
)

func main() {
	log.Println("Starting MMO Game Server with Actor Model...")

	// --- Configuration Loading ---
	// Create an example config if it doesn't exist.
	// In production, ensure 'config.json' is present and properly configured.
	configs.CreateExampleConfigFile("config.json")
	cfg, err := configs.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Configuration loaded. Server TCP Port: %d, Sui RPC: %s", cfg.Server.TCPPort, cfg.Sui.RPCURL)

	// --- Initialize Actor System ---
	actorSystem := actor.NewActorSystem()
	log.Println("Actor system initialized.")

	// --- Spawn Top-Level Actors ---
	// RoomManagerActor
	roomManagerProps := internalActor.PropsForRoomManager(actorSystem)
	roomManagerPID := actorSystem.Root.SpawnNamed(roomManagerProps, "room-manager")
	log.Printf("RoomManagerActor spawned with PID: %s", roomManagerPID.String())

	// TODO: Spawn other top-level actors as needed (e.g., WorldManagerActor, PlayerDataManagerActor)
	// Example:
	// dbCacheLayer := game.NewDBCacheLayer() // Assuming this is still a service
	// dbCacheLayer.Start() // Start DB connections

	// suiService := sui.NewSuiService(cfg.Sui.RPCURL, cfg.Sui.PrivateKey) // Example Sui service

	// --- Initialize Network Server ---
	// The TCPServer needs the actor system and the RoomManagerPID to pass to PlayerSessionActors
	tcpServer := network.NewTCPServer(cfg.Server.TCPPort, actorSystem, roomManagerPID)
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}
	// Note: TCPServer now runs its accept loop in a goroutine.

	log.Println("MMO Game Server successfully initialized and running.")
	log.Println("Press Ctrl+C to shut down.")

	// --- Graceful Shutdown ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a shutdown signal is received

	log.Println("Shutting down MMO Game Server...")

	// Stop TCPServer first to prevent new connections and allow existing handlers to finish
	tcpServer.Stop() // This should handle its goroutines

	// Stop top-level actors
	// Order might matter if actors message each other during shutdown.
	// Proto.Actor's Stop will send a Stopping message, then wait for the actor to process it and stop.
	log.Printf("Stopping RoomManagerActor %s...", roomManagerPID.String())
	if err := actorSystem.Root.StopFuture(roomManagerPID).Wait(); err != nil {
		log.Printf("Error stopping RoomManagerActor: %v", err)
	} else {
		log.Println("RoomManagerActor stopped.")
	}

	// TODO: Stop other top-level actors

	// Shutdown actor system
	// This will wait for all actors to stop. A timeout can be added.
	log.Println("Shutting down actor system...")
	actorSystem.Shutdown() // Waits for all actors to stop
	// It's good practice to use actorSystem.ProcessRegistry.AddutdownHook if you need complex shutdown sequences or timeouts.
	// For example:
	// done := make(chan bool)
	// go func() {
	// 	actorSystem.Shutdown()
	// 	close(done)
	// }()
	// select {
	// case <-done:
	// 	log.Println("Actor system shut down gracefully.")
	// case <-time.After(15 * time.Second): // Adjust timeout as needed
	// 	log.Println("Actor system shutdown timed out.")
	// }

	// dbCacheLayer.Stop() // Stop any other services

	// A small delay to allow logs to flush, if necessary.
	time.Sleep(1 * time.Second)
	log.Println("MMO Game Server shut down gracefully.")
}
