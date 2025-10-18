package scheduler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sohel902833/minikube-clone/types"
)

// Scheduler assigns pods to nodes
type Scheduler struct {
	apiServerURL string
	client       *http.Client
	lastNodeIdx  int
}

// NewScheduler creates a new Scheduler
func NewScheduler(apiServerURL string) *Scheduler {
	return &Scheduler{
		apiServerURL: apiServerURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		lastNodeIdx: 0,
	}
}

// Start begins the scheduling loop
func (s *Scheduler) Start(stopCh <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	log.Println("Scheduler started")

	for {
		select {
		case <-ticker.C:
			if err := s.scheduleOnce(); err != nil {
				log.Printf("Scheduling error: %v", err)
			}
		case <-stopCh:
			log.Println("Scheduler stopped")
			return
		}
	}
}

// scheduleOnce performs one scheduling cycle
func (s *Scheduler) scheduleOnce() error {
	// Get unscheduled pods
	pods, err := s.getUnscheduledPods()
	if err != nil {
		return fmt.Errorf("failed to get unscheduled pods: %w", err)
	}

	if len(pods) == 0 {
		return nil
	}

	// Get available nodes
	nodes, err := s.getAvailableNodes()
	if err != nil {
		return fmt.Errorf("failed to get available nodes: %w", err)
	}

	if len(nodes) == 0 {
		log.Println("No available nodes for scheduling")
		return nil
	}

	// Schedule each pod
	for _, pod := range pods {
		node := s.selectNode(nodes)
		if node != nil {
			if err := s.bindPodToNode(pod, node); err != nil {
				log.Printf("Failed to bind pod %s to node %s: %v", pod.Metadata.Name, node.Metadata.Name, err)
			} else {
				log.Printf("Successfully scheduled pod %s to node %s", pod.Metadata.Name, node.Metadata.Name)
			}
		}
	}

	return nil
}

// getUnscheduledPods fetches pods without node assignment
func (s *Scheduler) getUnscheduledPods() ([]*types.Pod, error) {
	resp, err := s.client.Get(fmt.Sprintf("%s/api/v1/pods", s.apiServerURL))
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

	// Filter unscheduled pods
	unscheduled := make([]*types.Pod, 0)
	for _, pod := range result.Items {
		if pod.Spec.NodeName == "" && pod.Status.Phase == types.PodPending {
			unscheduled = append(unscheduled, pod)
		}
	}

	return unscheduled, nil
}

// getAvailableNodes fetches all ready nodes
func (s *Scheduler) getAvailableNodes() ([]*types.Node, error) {
	resp, err := s.client.Get(fmt.Sprintf("%s/api/v1/nodes", s.apiServerURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Items []*types.Node `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Filter ready nodes
	ready := make([]*types.Node, 0)
	for _, node := range result.Items {
		if node.Status.Phase == types.NodeRunning {
			ready = append(ready, node)
		}
	}

	return ready, nil
}

// selectNode selects a node for the pod using round-robin
func (s *Scheduler) selectNode(nodes []*types.Node) *types.Node {
	if len(nodes) == 0 {
		return nil
	}

	// Round-robin scheduling
	node := nodes[s.lastNodeIdx%len(nodes)]
	s.lastNodeIdx++

	return node
}

// bindPodToNode assigns a pod to a node
func (s *Scheduler) bindPodToNode(pod *types.Pod, node *types.Node) error {
	// Update pod spec with node name
	pod.Spec.NodeName = node.Metadata.Name

	// Send update to API server
	data, err := json.Marshal(pod)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf("%s/api/v1/pods/%s/status", s.apiServerURL, pod.Metadata.Name),
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to bind pod, status code: %d", resp.StatusCode)
	}

	return nil
}