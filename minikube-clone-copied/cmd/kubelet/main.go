package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sohel902833/minikube-clone/pkg/kubelet"
)

func main() {
	// Parse command line flags
	nodeName := flag.String("node-name", "node1", "Name of this node")
	apiServer := flag.String("api-server", "http://localhost:8080", "API server URL")
	flag.Parse()

	log.Println("Starting Mini Kubernetes Kubelet...")
	log.Printf("Node name: %s", *nodeName)
	log.Printf("API server: %s", *apiServer)

	// Create kubelet
	kube, err := kubelet.NewKubelet(*nodeName, *apiServer)
	if err != nil {
		log.Fatalf("Failed to create kubelet: %v", err)
	}
	defer kube.Close()

	// Setup signal handling for graceful shutdown
	stopCh := make(chan struct{})
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start kubelet in a goroutine
	go func() {
		if err := kube.Start(stopCh); err != nil {
			log.Fatalf("Kubelet error: %v", err)
		}
	}()

	// Wait for termination signal
	<-sigCh
	log.Println("Received termination signal, shutting down...")
	close(stopCh)

	// Give it a moment to cleanup
	log.Println("Kubelet stopped")
}