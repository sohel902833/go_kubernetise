package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"kubico/pkg/controller"
)

func main() {
	// Parse command line flags
	apiServer := flag.String("api-server", "http://localhost:8080", "API server URL")
	flag.Parse()

	log.Println("🎮 Starting Mini Kubernetes ReplicaSet Controller...")
	log.Printf("🔗 Connecting to API server at %s", *apiServer)

	// Create ReplicaSet controller
	rsController := controller.NewReplicaSetController(*apiServer)

	// Setup signal handling for graceful shutdown
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start controller in a goroutine
	go rsController.Start(stopCh)

	// Wait for termination signal
	<-sigCh
	log.Println("⏳ Received termination signal, shutting down...")
	close(stopCh)

	log.Println("✅ ReplicaSet Controller stopped")
}