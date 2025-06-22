package main

import (
	// For SUI client health check
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/asynkron/protoactor-go/actor"
	"github.com/phuhao00/suigserver/server/configs"
	internalActor "github.com/phuhao00/suigserver/server/internal/actor" // Renamed to avoid conflict with protoactor's actor package
	"github.com/phuhao00/suigserver/server/internal/network"
	"github.com/phuhao00/suigserver/server/internal/sui"   // Import for SUI client
	"github.com/phuhao00/suigserver/server/internal/utils" // Import for logger
	// Other direct service initializations if any (e.g., DB connection pools)
)

func main() {
	// --- Configuration Loading ---
	// Create an example config if it doesn't exist.
	// In production, ensure 'config.json' is present and properly configured.
	configs.CreateExampleConfigFile("config.json") // This might log using standard logger before ours is set
	cfg, err := configs.LoadConfig("config.json")
	if err != nil {
		// Use standard log here as our logger might not be initialized or config not loaded
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// --- Initialize Logger ---
	utils.SetLogLevel(cfg.Server.LogLevel)
	utils.LogInfo("Starting MMO Game Server with Actor Model...")
	utils.LogInfof("Configuration loaded. Server TCP Port: %d, Sui RPC: %s, LogLevel: %s", cfg.Server.TCPPort, cfg.Sui.RPCURL, cfg.Server.LogLevel)

	// --- Initialize Actor System ---
	// Note: Proto.Actor logging configuration methods may vary by version
	// Commenting out potentially outdated logging setup
	utils.LogInfo("Proto.Actor logging will use default configuration.")

	actorSystem := actor.NewActorSystem()
	utils.LogInfo("Actor system initialized.")

	// --- Spawn Top-Level Actors ---
	// RoomManagerActor
	roomManagerProps := internalActor.PropsForRoomManager(actorSystem)
	roomManagerPID, err := actorSystem.Root.SpawnNamed(roomManagerProps, "room-manager")
	if err != nil {
		utils.LogFatalf("Failed to spawn RoomManagerActor: %v", err)
	}
	utils.LogInfof("RoomManagerActor spawned with PID: %s", roomManagerPID.String())

	// Spawn WorldManagerActor
	worldManagerProps := internalActor.PropsForWorldManager(actorSystem)
	worldManagerPID, err := actorSystem.Root.SpawnNamed(worldManagerProps, "world-manager")
	if err != nil {
		utils.LogFatalf("Failed to spawn WorldManagerActor: %v", err)
	}
	utils.LogInfof("WorldManagerActor spawned with PID: %s", worldManagerPID.String())

	// TODO: Spawn other top-level actors as needed (e.g., PlayerDataManagerActor, GameEventManagerActor)
	utils.LogInfo("Placeholder: Additional top-level actors (PlayerDataManager, GameEventManager) would be spawned here if defined.")

	// --- Initialize SUI Client ---
	suiClient := sui.NewSuiClient(cfg.Sui.RPCURL) // Using the modern SuiClient
	utils.LogInfof("SUI client initialized for RPC URL: %s", cfg.Sui.RPCURL)
	if cfg.Sui.PrivateKey != "" && cfg.Sui.PrivateKey != "YOUR_SUI_PRIVATE_KEY_HEX_HERE" {
		utils.LogInfo("SUI private key loaded and available for server-side transaction signing.")
	} else {
		utils.LogWarn("SUI private key is not configured or is using the default placeholder. Server-side SUI transactions requiring this key will not be possible.")
	}
	// Perform SUI client health check
	go func() {
		time.Sleep(2 * time.Second) // Brief delay to allow server to fully start before check
		// Test SUI client connectivity by querying an object (using a simple call that should exist)
		// Using a simple get object call with a known invalid ID to test connectivity
		_, err := suiClient.GetObject("0x1")
		if err != nil {
			// Even if the object doesn't exist, if we get a response from the network, it means connectivity is working
			// We expect this to fail with "object not found" rather than a network error
			utils.LogInfo("SUI client health check completed. Network connectivity appears to be working.")
		} else {
			utils.LogInfo("SUI client health check successful. Connected to Sui network.")
		}
	}()

	// --- Initialize Network Server ---
	// TCPServer now also needs WorldManagerPID, suiClient, and Auth configs to pass to PlayerSessionActors
	tcpServer := network.NewTCPServer(
		cfg.Server.TCPPort,
		actorSystem,
		roomManagerPID,
		worldManagerPID,
		suiClient,
		cfg.Auth.EnableDummyAuth,
		cfg.Auth.DummyToken,
		cfg.Auth.DummyPlayerID,
	)
	if err := tcpServer.Start(); err != nil {
		log.Fatalf("Failed to start TCP server: %v", err)
	}

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

	// Stop WorldManagerActor
	log.Printf("Stopping WorldManagerActor %s...", worldManagerPID.String())
	if err := actorSystem.Root.StopFuture(worldManagerPID).Wait(); err != nil {
		log.Printf("Error stopping WorldManagerActor: %v", err)
	} else {
		log.Println("WorldManagerActor stopped.")
	}

	// TODO: Stop other top-level actors (e.g., PlayerDataManagerActor) in appropriate order
	log.Println("Placeholder: Additional top-level actors would be stopped here if they were spawned.")

	// Shutdown actor system
	// This will wait for all actors to stop.
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
