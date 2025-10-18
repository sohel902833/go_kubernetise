package storage

import (
	"errors"
	"sync"

	"github.com/sohel902833/minikube-clone/types"
)

var (
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
)

// Store provides thread-safe in-memory storage
type Store struct {
	pods  map[string]*types.Pod
	nodes map[string]*types.Node
	mu    sync.RWMutex
}

// NewStore creates a new Store instance
func NewStore() *Store {
	return &Store{
		pods:  make(map[string]*types.Pod),
		nodes: make(map[string]*types.Node),
	}
}

// CreatePod adds a new pod to the store
func (s *Store) CreatePod(pod *types.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pods[pod.Metadata.Name]; exists {
		return ErrAlreadyExists
	}

	s.pods[pod.Metadata.Name] = pod
	return nil
}

// GetPod retrieves a pod by name
func (s *Store) GetPod(name string) (*types.Pod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pod, exists := s.pods[name]
	if !exists {
		return nil, ErrNotFound
	}

	return pod, nil
}

// ListPods returns all pods
func (s *Store) ListPods() []*types.Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pods := make([]*types.Pod, 0, len(s.pods))
	for _, pod := range s.pods {
		pods = append(pods, pod)
	}

	return pods
}

// UpdatePod updates an existing pod
func (s *Store) UpdatePod(pod *types.Pod) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pods[pod.Metadata.Name]; !exists {
		return ErrNotFound
	}

	s.pods[pod.Metadata.Name] = pod
	return nil
}

// DeletePod removes a pod from the store
func (s *Store) DeletePod(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pods[name]; !exists {
		return ErrNotFound
	}

	delete(s.pods, name)
	return nil
}

// GetUnscheduledPods returns pods without a node assignment
func (s *Store) GetUnscheduledPods() []*types.Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()

	unscheduled := make([]*types.Pod, 0)
	for _, pod := range s.pods {
		if pod.Spec.NodeName == "" && pod.Status.Phase == types.PodPending {
			unscheduled = append(unscheduled, pod)
		}
	}

	return unscheduled
}

// GetPodsByNode returns all pods assigned to a specific node
func (s *Store) GetPodsByNode(nodeName string) []*types.Pod {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pods := make([]*types.Pod, 0)
	for _, pod := range s.pods {
		if pod.Spec.NodeName == nodeName {
			pods = append(pods, pod)
		}
	}

	return pods
}

// CreateNode adds a new node to the store
func (s *Store) CreateNode(node *types.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[node.Metadata.Name]; exists {
		return ErrAlreadyExists
	}

	s.nodes[node.Metadata.Name] = node
	return nil
}

// GetNode retrieves a node by name
func (s *Store) GetNode(name string) (*types.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, exists := s.nodes[name]
	if !exists {
		return nil, ErrNotFound
	}

	return node, nil
}

// ListNodes returns all nodes
func (s *Store) ListNodes() []*types.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]*types.Node, 0, len(s.nodes))
	for _, node := range s.nodes {
		nodes = append(nodes, node)
	}

	return nodes
}

// UpdateNode updates an existing node
func (s *Store) UpdateNode(node *types.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[node.Metadata.Name]; !exists {
		return ErrNotFound
	}

	s.nodes[node.Metadata.Name] = node
	return nil
}

// DeleteNode removes a node from the store
func (s *Store) DeleteNode(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.nodes[name]; !exists {
		return ErrNotFound
	}

	delete(s.nodes, name)
	return nil
}

// GetReadyNodes returns all nodes in Ready state
func (s *Store) GetReadyNodes() []*types.Node {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ready := make([]*types.Node, 0)
	for _, node := range s.nodes {
		if node.Status.Phase == types.NodeRunning {
			ready = append(ready, node)
		}
	}

	return ready
}