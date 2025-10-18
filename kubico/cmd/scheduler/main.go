package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"kubico/pkg/scheduler"
)

func main() {
	// Parse command line flags
	apiServer := flag.String("api-server", "http://localhost:8080", "API server URL")
	flag.Parse()

	log.Println("Starting Mini Kubernetes Scheduler...")
	log.Printf("Connecting to API server at %s", *apiServer)

	// Create scheduler
	sched := scheduler.NewScheduler(*apiServer)

	// Setup signal handling for graceful shutdown
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start scheduler in a goroutine
	go sched.Start(stopCh)

	// Wait for termination signal
	<-sigCh
	log.Println("Received termination signal, shutting down...")
	close(stopCh)

	log.Println("Scheduler stopped")
}