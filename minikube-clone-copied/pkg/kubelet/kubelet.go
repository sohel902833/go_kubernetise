package kubelet

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/sohel902833/minikube-clone/types"
)

// Kubelet manages pods on a node
type Kubelet struct {
	nodeName      string
	apiServerURL  string
	dockerClient  *client.Client
	httpClient    *http.Client
	runningPods   map[string][]string // pod name -> container IDs
}

// NewKubelet creates a new Kubelet
func NewKubelet(nodeName, apiServerURL string) (*Kubelet, error) {
	dockerClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Kubelet{
		nodeName:     nodeName,
		apiServerURL: apiServerURL,
		dockerClient: dockerClient,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		runningPods: make(map[string][]string),
	}, nil
}

// Start begins the kubelet main loop
func (k *Kubelet) Start(stopCh <-chan struct{}) error {
	// Register node with API server
	if err := k.registerNode(); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	log.Printf("Kubelet started on node %s", k.nodeName)

	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := k.syncPods(); err != nil {
				log.Printf("Error syncing pods: %v", err)
			}
		case <-stopCh:
			log.Println("Kubelet stopped")
			return nil
		}
	}
}

// registerNode registers this node with the API server
func (k *Kubelet) registerNode() error {
	node := types.Node{
		Metadata: types.Metadata{
			Name: k.nodeName,
		},
		Status: types.NodeStatus{
			Phase: types.NodeRunning,
			Addresses: []types.NodeAddress{
				{
					Type:    "InternalIP",
					Address: "127.0.0.1",
				},
			},
		},
	}

	data, err := json.Marshal(node)
	if err != nil {
		return err
	}

	resp, err := k.httpClient.Post(
		fmt.Sprintf("%s/api/v1/nodes", k.apiServerURL),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("failed to register node, status: %d", resp.StatusCode)
	}

	log.Printf("Node %s registered successfully", k.nodeName)
	return nil
}

// syncPods syncs the desired state with actual state
func (k *Kubelet) syncPods() error {
	// Get pods assigned to this node
	pods, err := k.getAssignedPods()
	if err != nil {
		return fmt.Errorf("failed to get assigned pods: %w", err)
	}

	// Process each pod
	for _, pod := range pods {
		if err := k.ensurePod(pod); err != nil {
			log.Printf("Error ensuring pod %s: %v", pod.Metadata.Name, err)
			k.updatePodStatus(pod, types.PodFailed, err.Error())
		}
	}

	return nil
}

// getAssignedPods gets all pods assigned to this node
func (k *Kubelet) getAssignedPods() ([]*types.Pod, error) {
	resp, err := k.httpClient.Get(fmt.Sprintf("%s/api/v1/pods", k.apiServerURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []*types.Pod `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Filter pods for this node
	assigned := make([]*types.Pod, 0)
	for _, pod := range result.Items {
		if pod.Spec.NodeName == k.nodeName {
			assigned = append(assigned, pod)
		}
	}

	return assigned, nil
}

// ensurePod ensures a pod is running
func (k *Kubelet) ensurePod(pod *types.Pod) error {
	ctx := context.Background()

	// Check if pod is already running
	if _, exists := k.runningPods[pod.Metadata.Name]; exists {
		return nil
	}

	containerIDs := make([]string, 0)

	// Start each container in the pod
	for _, containerSpec := range pod.Spec.Containers {
		containerID, err := k.startContainer(ctx, pod, containerSpec)
		if err != nil {
			// Cleanup any started containers
			k.cleanupContainers(ctx, containerIDs)
			return fmt.Errorf("failed to start container %s: %w", containerSpec.Name, err)
		}
		containerIDs = append(containerIDs, containerID)
		log.Printf("Started container %s (ID: %s) for pod %s", containerSpec.Name, containerID[:12], pod.Metadata.Name)
	}

	// Store running pod info
	k.runningPods[pod.Metadata.Name] = containerIDs

	// Update pod status to Running
	now := time.Now()
	pod.Status.StartTime = &now
	k.updatePodStatus(pod, types.PodRunning, "All containers started")

	return nil
}

// startContainer starts a single container
func (k *Kubelet) startContainer(ctx context.Context, pod *types.Pod, containerSpec types.Container) (string, error) {
	// Pull image
	log.Printf("Pulling image %s...", containerSpec.Image)
	reader, err := k.dockerClient.ImagePull(ctx, containerSpec.Image, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()
	io.Copy(io.Discard, reader)

	// Prepare container config
	config := &container.Config{
		Image: containerSpec.Image,
		Cmd:   containerSpec.Command,
		Labels: map[string]string{
			"pod":       pod.Metadata.Name,
			"container": containerSpec.Name,
		},
	}

	// Add environment variables
	if len(containerSpec.Env) > 0 {
		env := make([]string, 0, len(containerSpec.Env))
		for _, e := range containerSpec.Env {
			env = append(env, fmt.Sprintf("%s=%s", e.Name, e.Value))
		}
		config.Env = env
	}

	// Create container
	resp, err := k.dockerClient.ContainerCreate(ctx, config, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := k.dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}

// cleanupContainers stops and removes containers
func (k *Kubelet) cleanupContainers(ctx context.Context, containerIDs []string) {
	for _, id := range containerIDs {
		timeout := 10
		k.dockerClient.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
		k.dockerClient.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
	}
}

// updatePodStatus updates a pod's status on the API server
func (k *Kubelet) updatePodStatus(pod *types.Pod, phase types.PodPhase, message string) {
	pod.Status.Phase = phase
	pod.Status.Message = message
	pod.Status.HostIP = "127.0.0.1"

	data, err := json.Marshal(pod.Status)
	if err != nil {
		log.Printf("Failed to marshal pod status: %v", err)
		return
	}

	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/api/v1/pods/%s/status", k.apiServerURL, pod.Metadata.Name),
		bytes.NewBuffer(data),
	)
	if err != nil {
		log.Printf("Failed to create status update request: %v", err)
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := k.httpClient.Do(req)
	if err != nil {
		log.Printf("Failed to update pod status: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to update pod status, status code: %d", resp.StatusCode)
	}
}

// Close closes the kubelet and cleans up resources
func (k *Kubelet) Close() error {
	ctx := context.Background()
	
	// Stop all running containers
	for podName, containerIDs := range k.runningPods {
		log.Printf("Cleaning up pod %s", podName)
		k.cleanupContainers(ctx, containerIDs)
	}

	return k.dockerClient.Close()
}