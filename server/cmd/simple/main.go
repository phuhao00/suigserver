package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/phuhao00/suigserver/server/internal/simple"
)

func main() {
	var port = flag.Int("port", 8080, "Port to run the server on")
	flag.Parse()

	log.Println("Starting Simple Game Server...")

	// Create and start server
	server := simple.NewSimpleServer(*port)
	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Simple Game Server started on port %d", *port)
	log.Println("Press Ctrl+C to shut down")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	server.Stop()
	log.Println("Server stopped")
}
