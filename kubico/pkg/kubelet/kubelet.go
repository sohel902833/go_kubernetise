package kubelet

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"

	"kubico/types"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Kubelet manages pods on a node
type Kubelet struct {
	nodeName      string
	apiServerURL  string
	dockerClient  *client.Client
	httpClient    *http.Client
	runningPods   map[string]*PodRuntime
}

// PodRuntime tracks runtime information about a pod
type PodRuntime struct {
	ContainerIDs    []string
	LastHealthCheck time.Time
	HealthStatus    map[string]bool // containerName -> healthy
	RestartCount    int32
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
		runningPods: make(map[string]*PodRuntime),
	}, nil
}

// Start begins the kubelet main loop
func (k *Kubelet) Start(stopCh <-chan struct{}) error {
	// Register node with API server
	if err := k.registerNode(); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	log.Printf("✅ Kubelet started on node %s", k.nodeName)

	syncTicker := time.NewTicker(3 * time.Second)
	healthTicker := time.NewTicker(5 * time.Second)
	defer syncTicker.Stop()
	defer healthTicker.Stop()

	for {
		select {
		case <-syncTicker.C:
			if err := k.syncPods(); err != nil {
				log.Printf("❌ [KUBELET-%s] Error syncing pods: %v", k.nodeName, err)
			}
		case <-healthTicker.C:
			k.performHealthChecks()
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
			Name:      k.nodeName,
			CreatedAt: time.Now(),
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
		fmt.Println("Erro while registering node",err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return fmt.Errorf("failed to register node, status: %d", resp.StatusCode)
	}

	log.Printf("✅ [KUBELET-%s] Node registered successfully", k.nodeName)
	return nil
}

// syncPods syncs the desired state with actual state
func (k *Kubelet) syncPods() error {
	log.Printf("🔄 [KUBELET-%s] Syncing pods...", k.nodeName)

	// Get pods assigned to this node
	pods, err := k.getAssignedPods()
	if err != nil {
		return fmt.Errorf("failed to get assigned pods: %w", err)
	}

	if len(pods) == 0 {
		log.Printf("✅ [KUBELET-%s] No pods assigned", k.nodeName)
		return nil
	}

	log.Printf("📋 [KUBELET-%s] Found %d pod(s) assigned to this node", k.nodeName, len(pods))

	// Check for pods that should be deleted
	k.cleanupOrphanedPods(pods)

	// Process each pod
	for _, pod := range pods {
		if err := k.reconcilePod(pod); err != nil {
			log.Printf("❌ [KUBELET-%s] Error reconciling pod %s: %v", k.nodeName, pod.Metadata.Name, err)
		}
	}

	return nil
}

// reconcilePod ensures a pod matches desired state
func (k *Kubelet) reconcilePod(pod *types.Pod) error {
	runtime, exists := k.runningPods[pod.Metadata.Name]

	// Check if containers are actually running
	if exists {
		allRunning := k.checkContainersRunning(runtime.ContainerIDs)
		if !allRunning {
			// Container died, need to restart
			log.Printf("⚠️  [KUBELET-%s] Container(s) for pod %s died, restarting...", 
				k.nodeName, pod.Metadata.Name)
			k.cleanupContainers(context.Background(), runtime.ContainerIDs)
			delete(k.runningPods, pod.Metadata.Name)
			exists = false
			runtime.RestartCount++
		}
	}

	if !exists {
		log.Printf("🚀 [KUBELET-%s] Starting pod '%s'...", k.nodeName, pod.Metadata.Name)
		if err := k.ensurePod(pod); err != nil {
			k.updatePodStatus(pod, types.PodFailed, err.Error())
			return err
		}
	}

	return nil
}

// checkContainersRunning verifies all containers are running
func (k *Kubelet) checkContainersRunning(containerIDs []string) bool {
	ctx := context.Background()
	for _, id := range containerIDs {
		inspect, err := k.dockerClient.ContainerInspect(ctx, id)
		if err != nil || !inspect.State.Running {
			return false
		}
	}
	return true
}

// cleanupOrphanedPods removes pods that no longer exist in desired state
func (k *Kubelet) cleanupOrphanedPods(desiredPods []*types.Pod) {
	desiredPodNames := make(map[string]bool)
	for _, pod := range desiredPods {
		desiredPodNames[pod.Metadata.Name] = true
	}

	for podName, runtime := range k.runningPods {
		if !desiredPodNames[podName] {
			log.Printf("🧹 [KUBELET-%s] Cleaning up orphaned pod: %s", k.nodeName, podName)
			k.cleanupContainers(context.Background(), runtime.ContainerIDs)
			delete(k.runningPods, podName)
		}
	}
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
	// Read the full body first
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(body, &result); err != nil {
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

	containerIDs := make([]string, 0)
	containerStatuses := make([]types.ContainerStatus, 0)

	// Start each container in the pod
	for _, containerSpec := range pod.Spec.Containers {
		log.Printf("   📦 [KUBELET-%s] Processing container: %s", k.nodeName, containerSpec.Name)
		
		containerID, err := k.startContainer(ctx, pod, containerSpec)
		if err != nil {
			// Cleanup any started containers
			k.cleanupContainers(ctx, containerIDs)
			return fmt.Errorf("failed to start container %s: %w", containerSpec.Name, err)
		}
		
		containerIDs = append(containerIDs, containerID)
		
		containerStatuses = append(containerStatuses, types.ContainerStatus{
			Name:         containerSpec.Name,
			Ready:        true,
			RestartCount: 0,
			State:        "Running",
			ContainerID:  containerID,
		})
		
		log.Printf("   ✅ [KUBELET-%s] Container %s started (ID: %s)", 
			k.nodeName, containerSpec.Name, containerID[:12])
	}

	// Store running pod info
	k.runningPods[pod.Metadata.Name] = &PodRuntime{
		ContainerIDs:    containerIDs,
		LastHealthCheck: time.Now(),
		HealthStatus:    make(map[string]bool),
		RestartCount:    0,
	}

	// Update pod status to Running
	now := time.Now()
	pod.Status.StartTime = &now
	pod.Status.ContainerStatuses = containerStatuses
	k.updatePodStatus(pod, types.PodRunning, "All containers started")

	log.Printf("✅ [KUBELET-%s] Pod '%s' is now running!", k.nodeName, pod.Metadata.Name)
	return nil
}

// startContainer starts a single container with detailed logging
func (k *Kubelet) startContainer(ctx context.Context, pod *types.Pod, containerSpec types.Container) (string, error) {
	// Pull image with detailed progress
	log.Printf("   🔽 [KUBELET-%s] Pulling image: %s", k.nodeName, containerSpec.Image)
	
	reader, err := k.dockerClient.ImagePull(ctx, containerSpec.Image, image.PullOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Stream pull progress
	scanner := bufio.NewScanner(reader)
	lastStatus := ""
	for scanner.Scan() {
		line := scanner.Text()
		
		var progress struct {
			Status         string `json:"status"`
			ProgressDetail struct {
				Current int64 `json:"current"`
				Total   int64 `json:"total"`
			} `json:"progressDetail"`
			Progress string `json:"progress"`
			ID       string `json:"id"`
		}
		
		if err := json.Unmarshal([]byte(line), &progress); err == nil {
			// Only log status changes to avoid spam
			currentStatus := fmt.Sprintf("%s %s", progress.Status, progress.ID)
			if currentStatus != lastStatus {
				if progress.Progress != "" {
					log.Printf("      %s %s: %s", progress.Status, progress.ID, progress.Progress)
				} else if progress.Status != "" {
					log.Printf("      %s %s", progress.Status, progress.ID)
				}
				lastStatus = currentStatus
			}
		}
	}

	log.Printf("   ✅ [KUBELET-%s] Image pulled successfully", k.nodeName)
	log.Printf("   🔨 [KUBELET-%s] Creating container...", k.nodeName)

	// Prepare container config
	config := &container.Config{
		Image: containerSpec.Image,
		Cmd:   containerSpec.Command,
		Labels: map[string]string{
			"pod":       pod.Metadata.Name,
			"container": containerSpec.Name,
			"managed-by": "mini-k8s",
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

	// Expose ports
	if len(containerSpec.Ports) > 0 {
		config.ExposedPorts = nat.PortSet{}
			for _, port := range containerSpec.Ports {
				portStr := fmt.Sprintf("%d/tcp", port.ContainerPort)
				config.ExposedPorts[nat.Port(portStr)] = struct{}{}
			}
		// config.ExposedPorts = make(map[string]struct{})
		// for _, port := range containerSpec.Ports {
		// 	portStr := fmt.Sprintf("%d/tcp", port.ContainerPort)
		// 	config.ExposedPorts[portStr] = struct{}{}
		// }
	}

	// Create container
	resp, err := k.dockerClient.ContainerCreate(ctx, config, nil, nil, nil, "")
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	log.Printf("   🚀 [KUBELET-%s] Starting container (ID: %s)...", k.nodeName, resp.ID[:12])

	// Start container
	if err := k.dockerClient.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return resp.ID, nil
}

// performHealthChecks performs health checks on all running pods
func (k *Kubelet) performHealthChecks() {
	log.Printf("🏥 [KUBELET-%s] Performing health checks...", k.nodeName)
	
	pods, err := k.getAssignedPods()
	if err != nil {
		log.Printf("❌ [KUBELET-%s] Failed to get pods for health check: %v", k.nodeName, err)
		return
	}

	for _, pod := range pods {
		runtime, exists := k.runningPods[pod.Metadata.Name]
		if !exists {
			continue
		}

		// Check each container
		for _, containerSpec := range pod.Spec.Containers {
			healthy := true

			// Liveness probe
			if containerSpec.LivenessProbe != nil {
				if !k.executeProbe(pod, containerSpec, containerSpec.LivenessProbe, "liveness") {
					healthy = false
					log.Printf("⚠️  [KUBELET-%s] Liveness probe failed for %s/%s", 
						k.nodeName, pod.Metadata.Name, containerSpec.Name)
					
					// Restart container on liveness failure
					k.restartPod(pod)
					break
				}
			}

			// Readiness probe
			if containerSpec.ReadinessProbe != nil {
				if !k.executeProbe(pod, containerSpec, containerSpec.ReadinessProbe, "readiness") {
					healthy = false
					log.Printf("⚠️  [KUBELET-%s] Readiness probe failed for %s/%s", 
						k.nodeName, pod.Metadata.Name, containerSpec.Name)
				}
			}

			runtime.HealthStatus[containerSpec.Name] = healthy
		}

		runtime.LastHealthCheck = time.Now()
	}
}

// executeProbe executes a health probe
func (k *Kubelet) executeProbe(pod *types.Pod, containerSpec types.Container, probe *types.Probe, probeType string) bool {
	// HTTP GET probe
	if probe.HTTPGet != nil {
		url := fmt.Sprintf("http://localhost:%d%s", probe.HTTPGet.Port, probe.HTTPGet.Path)
		
		client := &http.Client{
			Timeout: time.Duration(probe.TimeoutSeconds) * time.Second,
		}
		
		resp, err := client.Get(url)
		if err != nil {
			log.Printf("   ❌ [KUBELET-%s] %s probe failed: %v", k.nodeName, probeType, err)
			return false
		}
		defer resp.Body.Close()
		
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			log.Printf("   ✅ [KUBELET-%s] %s probe passed for %s/%s", 
				k.nodeName, probeType, pod.Metadata.Name, containerSpec.Name)
			return true
		}
		return false
	}

	// TCP Socket probe
	if probe.TCPSocket != nil {
		address := fmt.Sprintf("localhost:%d", probe.TCPSocket.Port)
		timeout := time.Duration(probe.TimeoutSeconds) * time.Second
		
		conn, err := net.DialTimeout("tcp", address, timeout)
		if err != nil {
			log.Printf("   ❌ [KUBELET-%s] %s probe failed: %v", k.nodeName, probeType, err)
			return false
		}
		conn.Close()
		
		log.Printf("   ✅ [KUBELET-%s] %s probe passed for %s/%s", 
			k.nodeName, probeType, pod.Metadata.Name, containerSpec.Name)
		return true
	}

	return true
}

// restartPod restarts a failed pod
func (k *Kubelet) restartPod(pod *types.Pod) {
	log.Printf("🔄 [KUBELET-%s] Restarting pod: %s", k.nodeName, pod.Metadata.Name)
	
	runtime, exists := k.runningPods[pod.Metadata.Name]
	if !exists {
		return
	}

	// Cleanup existing containers
	ctx := context.Background()
	k.cleanupContainers(ctx, runtime.ContainerIDs)
	delete(k.runningPods, pod.Metadata.Name)

	// Update restart count
	pod.Status.RestartCount++

	// Restart pod
	if err := k.ensurePod(pod); err != nil {
		log.Printf("❌ [KUBELET-%s] Failed to restart pod: %v", k.nodeName, err)
		k.updatePodStatus(pod, types.PodFailed, fmt.Sprintf("Restart failed: %v", err))
	}
}

// cleanupContainers stops and removes containers
func (k *Kubelet) cleanupContainers(ctx context.Context, containerIDs []string) {
	for _, id := range containerIDs {
		timeout := 10
		k.dockerClient.ContainerStop(ctx, id, container.StopOptions{Timeout: &timeout})
		k.dockerClient.ContainerRemove(ctx, id, container.RemoveOptions{Force: true})
		log.Printf("   🗑️  [KUBELET-%s] Removed container: %s", k.nodeName, id[:12])
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
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Failed to update pod status, status code: %d, body: %s", resp.StatusCode, string(body))
	}
}

// Close closes the kubelet and cleans up resources
func (k *Kubelet) Close() error {
	ctx := context.Background()
	
	// Stop all running containers
	for podName, runtime := range k.runningPods {
		log.Printf("🧹 [KUBELET-%s] Cleaning up pod %s", k.nodeName, podName)
		k.cleanupContainers(ctx, runtime.ContainerIDs)
	}

	return k.dockerClient.Close()
}