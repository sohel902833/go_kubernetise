package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"kubico/types"

	"github.com/google/uuid"
)

// ReplicaSetController manages ReplicaSets
type ReplicaSetController struct {
	apiServerURL string
	client       *http.Client
}

// NewReplicaSetController creates a new ReplicaSet controller
func NewReplicaSetController(apiServerURL string) *ReplicaSetController {
	return &ReplicaSetController{
		apiServerURL: apiServerURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start begins the controller reconciliation loop
func (c *ReplicaSetController) Start(stopCh <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("🎮 ReplicaSet Controller started")

	for {
		select {
		case <-ticker.C:
			if err := c.reconcile(); err != nil {
				log.Printf("❌ [REPLICASET] Reconciliation error: %v", err)
			}
		case <-stopCh:
			log.Println("🎮 ReplicaSet Controller stopped")
			return
		}
	}
}

// reconcile ensures the actual state matches desired state
func (c *ReplicaSetController) reconcile() error {
	log.Println("🔄 [REPLICASET] Starting reconciliation...")

	// Get all ReplicaSets
	replicaSets, err := c.getReplicaSets()
	if err != nil {
		return err
	}

	if len(replicaSets) == 0 {
		log.Println("✅ [REPLICASET] No ReplicaSets to manage")
		return nil
	}

	log.Printf("📋 [REPLICASET] Found %d ReplicaSet(s)", len(replicaSets))

	// Reconcile each ReplicaSet
	for _, rs := range replicaSets {
		if err := c.reconcileReplicaSet(rs); err != nil {
			log.Printf("❌ [REPLICASET] Error reconciling %s: %v", rs.Metadata.Name, err)
		}
	}

	return nil
}

// reconcileReplicaSet reconciles a single ReplicaSet
func (c *ReplicaSetController) reconcileReplicaSet(rs *types.ReplicaSet) error {
	log.Printf("🔍 [REPLICASET] Reconciling %s (desired: %d replicas)", 
		rs.Metadata.Name, rs.Spec.Replicas)

	// Get pods owned by this ReplicaSet
	pods, err := c.getPodsForReplicaSet(rs)
	if err != nil {
		return err
	}

	// Filter running and pending pods
	activePods := make([]*types.Pod, 0)
	for _, pod := range pods {
		if pod.Status.Phase != types.PodFailed && pod.Status.Phase != types.PodSucceeded {
			activePods = append(activePods, pod)
		}
	}

	currentReplicas := int32(len(activePods))
	desiredReplicas := rs.Spec.Replicas

	log.Printf("📊 [REPLICASET] %s: current=%d, desired=%d", 
		rs.Metadata.Name, currentReplicas, desiredReplicas)

	// Scale up
	if currentReplicas < desiredReplicas {
		diff := desiredReplicas - currentReplicas
		log.Printf("⬆️  [REPLICASET] Scaling up %s by %d replicas", rs.Metadata.Name, diff)
		
		for i := int32(0); i < diff; i++ {
			if err := c.createPod(rs); err != nil {
				log.Printf("❌ [REPLICASET] Failed to create pod: %v", err)
			} else {
				log.Printf("✅ [REPLICASET] Created new pod for %s", rs.Metadata.Name)
			}
		}
	}

	// Scale down
	if currentReplicas > desiredReplicas {
		diff := currentReplicas - desiredReplicas
		log.Printf("⬇️  [REPLICASET] Scaling down %s by %d replicas", rs.Metadata.Name, diff)
		
		for i := int32(0); i < diff && i < int32(len(activePods)); i++ {
			pod := activePods[i]
			if err := c.deletePod(pod.Metadata.Name); err != nil {
				log.Printf("❌ [REPLICASET] Failed to delete pod %s: %v", pod.Metadata.Name, err)
			} else {
				log.Printf("✅ [REPLICASET] Deleted pod %s", pod.Metadata.Name)
			}
		}
	}

	// Update ReplicaSet status
	rs.Status.Replicas = currentReplicas
	rs.Status.ReadyReplicas = c.countReadyPods(activePods)
	rs.Status.AvailableReplicas = rs.Status.ReadyReplicas

	if err := c.updateReplicaSetStatus(rs); err != nil {
		log.Printf("⚠️  [REPLICASET] Failed to update status: %v", err)
	}

	return nil
}

// createPod creates a new pod from the ReplicaSet template
func (c *ReplicaSetController) createPod(rs *types.ReplicaSet) error {
	// Generate unique pod name
	podName := fmt.Sprintf("%s-%s", rs.Metadata.Name, uuid.New().String()[:8])

	pod := &types.Pod{
		APIVersion: "v1",
		Kind:       "Pod",
		Metadata: types.Metadata{
			Name:      podName,
			Namespace: rs.Metadata.Namespace,
			Labels:    rs.Spec.Template.Metadata.Labels,
			CreatedAt: time.Now(),
			UID:       uuid.New().String(),
			OwnerReferences: []types.OwnerReference{
				{
					APIVersion: rs.APIVersion,
					Kind:       rs.Kind,
					Name:       rs.Metadata.Name,
					UID:        rs.Metadata.UID,
				},
			},
		},
		Spec: rs.Spec.Template.Spec,
		Status: types.PodStatus{
			Phase: types.PodPending,
		},
	}

	// Set default restart policy
	if pod.Spec.RestartPolicy == "" {
		pod.Spec.RestartPolicy = "Always"
	}

	data, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	resp, err := c.client.Post(
		fmt.Sprintf("%s/api/v1/pods", c.apiServerURL),
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create pod, status: %d", resp.StatusCode)
	}

	return nil
}

// deletePod deletes a pod
func (c *ReplicaSetController) deletePod(name string) error {
	req, err := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/api/v1/pods/%s", c.apiServerURL, name),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete pod, status: %d", resp.StatusCode)
	}

	return nil
}

// getReplicaSets fetches all ReplicaSets
func (c *ReplicaSetController) getReplicaSets() ([]*types.ReplicaSet, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/v1/replicasets", c.apiServerURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []*types.ReplicaSet `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Items, nil
}

// getPodsForReplicaSet gets all pods owned by a ReplicaSet
func (c *ReplicaSetController) getPodsForReplicaSet(rs *types.ReplicaSet) ([]*types.Pod, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/v1/pods", c.apiServerURL))
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

	// Filter pods that match the ReplicaSet selector and are owned by this RS
	matchingPods := make([]*types.Pod, 0)
	for _, pod := range result.Items {
		// Check if pod is owned by this ReplicaSet
		if c.isPodOwnedByReplicaSet(pod, rs) {
			matchingPods = append(matchingPods, pod)
		}
	}

	return matchingPods, nil
}

// isPodOwnedByReplicaSet checks if a pod is owned by the given ReplicaSet
func (c *ReplicaSetController) isPodOwnedByReplicaSet(pod *types.Pod, rs *types.ReplicaSet) bool {
	// Check owner references first
	for _, ownerRef := range pod.Metadata.OwnerReferences {
		if ownerRef.Kind == "ReplicaSet" && ownerRef.Name == rs.Metadata.Name {
			return true
		}
	}

	// Also check label selector match as backup
	if  len(rs.Spec.Selector) == 0 {
		return false
	}

	// All selector labels must match
	for key, value := range rs.Spec.Selector {
		if pod.Metadata.Labels == nil || pod.Metadata.Labels[key] != value {
			return false
		}
	}

	return true
}

// countReadyPods counts how many pods are ready
func (c *ReplicaSetController) countReadyPods(pods []*types.Pod) int32 {
	ready := int32(0)
	for _, pod := range pods {
		if pod.Status.Phase == types.PodRunning {
			// Check if all containers are ready
			allReady := true
			if len(pod.Status.ContainerStatuses) > 0 {
				for _, cs := range pod.Status.ContainerStatuses {
					if !cs.Ready {
						allReady = false
						break
					}
				}
			}
			if allReady {
				ready++
			}
		}
	}
	return ready
}

// updateReplicaSetStatus updates the ReplicaSet status
func (c *ReplicaSetController) updateReplicaSetStatus(rs *types.ReplicaSet) error {
	data, err := json.Marshal(rs)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/api/v1/replicasets/%s", c.apiServerURL, rs.Metadata.Name),
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to update status, code: %d", resp.StatusCode)
	}

	return nil
}