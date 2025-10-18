package controller

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sohel902833/minikube-clone/types"
)

// Controller manages the desired state of the cluster
type Controller struct {
	apiServerURL string
	client       *http.Client
}

// NewController creates a new Controller
func NewController(apiServerURL string) *Controller {
	return &Controller{
		apiServerURL: apiServerURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Start begins the controller reconciliation loop
func (c *Controller) Start(stopCh <-chan struct{}) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	log.Println("Controller Manager started")

	for {
		select {
		case <-ticker.C:
			if err := c.reconcile(); err != nil {
				log.Printf("Reconciliation error: %v", err)
			}
		case <-stopCh:
			log.Println("Controller Manager stopped")
			return
		}
	}
}

// reconcile performs reconciliation of desired vs actual state
func (c *Controller) reconcile() error {
	// Check pod health
	if err := c.checkPodHealth(); err != nil {
		log.Printf("Pod health check error: %v", err)
	}

	// Check node health
	if err := c.checkNodeHealth(); err != nil {
		log.Printf("Node health check error: %v", err)
	}

	return nil
}

// checkPodHealth monitors pod health and takes corrective actions
func (c *Controller) checkPodHealth() error {
	pods, err := c.getPods()
	if err != nil {
		return err
	}

	for _, pod := range pods {
		// Check if pod has been pending for too long
		if pod.Status.Phase == types.PodPending {
			elapsed := time.Since(pod.Metadata.CreatedAt)
			if elapsed > 2*time.Minute {
				log.Printf("Pod %s has been pending for %v - might be stuck", pod.Metadata.Name, elapsed)
			}
		}

		// Check if pod failed
		if pod.Status.Phase == types.PodFailed {
			log.Printf("Pod %s failed: %s", pod.Metadata.Name, pod.Status.Message)
			// In a real system, we would implement retry logic here
		}
	}

	return nil
}

// checkNodeHealth monitors node health
func (c *Controller) checkNodeHealth() error {
	nodes, err := c.getNodes()
	if err != nil {
		return err
	}

	for _, node := range nodes {
		// Check node conditions
		for _, condition := range node.Status.Conditions {
			if condition.Type == "Ready" && condition.Status != "True" {
				log.Printf("Node %s is not ready: %s", node.Metadata.Name, condition.Message)
			}
		}

		// Check if node hasn't reported in a while
		if len(node.Status.Conditions) > 0 {
			lastHeartbeat := node.Status.Conditions[0].LastHeartbeatTime
			elapsed := time.Since(lastHeartbeat)
			if elapsed > 1*time.Minute {
				log.Printf("Node %s hasn't reported in %v", node.Metadata.Name, elapsed)
			}
		}
	}

	return nil
}

// getPods fetches all pods from the API server
func (c *Controller) getPods() ([]*types.Pod, error) {
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

	return result.Items, nil
}

// getNodes fetches all nodes from the API server
func (c *Controller) getNodes() ([]*types.Node, error) {
	resp, err := c.client.Get(fmt.Sprintf("%s/api/v1/nodes", c.apiServerURL))
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

	return result.Items, nil
}