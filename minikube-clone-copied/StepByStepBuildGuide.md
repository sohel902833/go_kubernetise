# Mini Kubernetes Clone - Step-by-Step Build Guide

This guide will help you build the project incrementally, understanding each component before moving to the next.

## 🎯 Learning Path Overview

```
Phase 1: Foundation (Days 1-2)
    ↓
Phase 2: Core Storage & API (Days 3-5)
    ↓
Phase 3: Scheduling (Days 6-7)
    ↓
Phase 4: Container Management (Days 8-10)
    ↓
Phase 5: CLI & Polish (Days 11-12)
```

---

## Phase 1: Project Foundation & Basic Types

### Day 1: Setup & Understanding Data Structures

**Goal:** Understand what data we need to represent in our mini Kubernetes.

#### Step 1.1: Create Project Structure

```bash
mkdir minikube-clone && cd minikube-clone
go mod init github.com/sohel902833/minikube-clone
mkdir -p types pkg cmd
```

#### Step 1.2: Define Basic Types (types/types.go)

Start with **just the Pod structure**:

```go
package types

import "time"

// Start simple - just Pod
type Pod struct {
    Metadata PodMetadata `json:"metadata"`
    Spec     PodSpec     `json:"spec"`
    Status   PodStatus   `json:"status"`
}

type PodMetadata struct {
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"createdAt"`
}

type PodSpec struct {
    Containers []Container `json:"containers"`
    NodeName   string      `json:"nodeName"` // Which node runs this pod
}

type Container struct {
    Name  string `json:"name"`
    Image string `json:"image"`
}

type PodStatus struct {
    Phase   string `json:"phase"` // Pending, Running, Failed
    Message string `json:"message"`
}
```

**🎓 Understanding Check:**

- A **Pod** is like a wrapper around one or more containers
- **Metadata** = information about the pod (name, when created)
- **Spec** = what you want (desired state)
- **Status** = what's actually happening (current state)

#### Step 1.3: Test Your Types

Create `types/types_test.go`:

```go
package types

import (
    "encoding/json"
    "testing"
    "time"
)

func TestPodSerialization(t *testing.T) {
    pod := Pod{
        Metadata: PodMetadata{
            Name:      "test-pod",
            CreatedAt: time.Now(),
        },
        Spec: PodSpec{
            Containers: []Container{
                {Name: "nginx", Image: "nginx:latest"},
            },
        },
        Status: PodStatus{
            Phase: "Pending",
        },
    }

    // Can we convert to JSON?
    data, err := json.Marshal(pod)
    if err != nil {
        t.Fatalf("Failed to marshal: %v", err)
    }

    // Can we convert back?
    var pod2 Pod
    if err := json.Unmarshal(data, &pod2); err != nil {
        t.Fatalf("Failed to unmarshal: %v", err)
    }

    if pod2.Metadata.Name != "test-pod" {
        t.Errorf("Expected name 'test-pod', got '%s'", pod2.Metadata.Name)
    }
}
```

Run: `go test ./types`

**✅ Checkpoint:** You should understand:

- What a Pod contains
- How to create and serialize a Pod
- The difference between Spec (desired) and Status (actual)

---

## Phase 2: Storage & API Server

### Day 2-3: In-Memory Storage

**Goal:** Build a thread-safe place to store pods.

#### Step 2.1: Simple Storage (pkg/storage/store.go)

Start **without thread safety** first:

```go
package storage

import (
    "errors"
    "github.com/sohel902833/minikube-clone/types"
)

var ErrNotFound = errors.New("not found")
var ErrAlreadyExists = errors.New("already exists")

// Simple version - no mutex yet
type Store struct {
    pods map[string]*types.Pod
}

func NewStore() *Store {
    return &Store{
        pods: make(map[string]*types.Pod),
    }
}

func (s *Store) CreatePod(pod *types.Pod) error {
    if _, exists := s.pods[pod.Metadata.Name]; exists {
        return ErrAlreadyExists
    }
    s.pods[pod.Metadata.Name] = pod
    return nil
}

func (s *Store) GetPod(name string) (*types.Pod, error) {
    pod, exists := s.pods[name]
    if !exists {
        return nil, ErrNotFound
    }
    return pod, nil
}

func (s *Store) ListPods() []*types.Pod {
    pods := make([]*types.Pod, 0, len(s.pods))
    for _, pod := range s.pods {
        pods = append(pods, pod)
    }
    return pods
}

func (s *Store) DeletePod(name string) error {
    if _, exists := s.pods[name]; !exists {
        return ErrNotFound
    }
    delete(s.pods, name)
    return nil
}
```

#### Step 2.2: Test Storage

Create `pkg/storage/store_test.go`:

```go
package storage

import (
    "testing"
    "github.com/sohel902833/minikube-clone/types"
)

func TestStoreBasicOperations(t *testing.T) {
    store := NewStore()

    // Create a pod
    pod := &types.Pod{
        Metadata: types.PodMetadata{Name: "test-pod"},
    }

    err := store.CreatePod(pod)
    if err != nil {
        t.Fatalf("Failed to create pod: %v", err)
    }

    // Get the pod
    retrieved, err := store.GetPod("test-pod")
    if err != nil {
        t.Fatalf("Failed to get pod: %v", err)
    }
    if retrieved.Metadata.Name != "test-pod" {
        t.Errorf("Wrong pod retrieved")
    }

    // List pods
    pods := store.ListPods()
    if len(pods) != 1 {
        t.Errorf("Expected 1 pod, got %d", len(pods))
    }

    // Delete pod
    err = store.DeletePod("test-pod")
    if err != nil {
        t.Fatalf("Failed to delete pod: %v", err)
    }

    // Should be gone
    _, err = store.GetPod("test-pod")
    if err != ErrNotFound {
        t.Errorf("Pod should not exist")
    }
}
```

Run: `go test ./pkg/storage`

**🎓 Understanding Check:**

- Storage is just a map with helper functions
- We use pointers (`*types.Pod`) to avoid copying large objects
- Error handling is important (what if pod doesn't exist?)

#### Step 2.3: Add Thread Safety

**Why?** Multiple requests might access storage simultaneously.

Update `pkg/storage/store.go`:

```go
package storage

import (
    "errors"
    "sync"
    "github.com/sohel902833/minikube-clone/types"
)

type Store struct {
    pods map[string]*types.Pod
    mu   sync.RWMutex  // Add this!
}

func (s *Store) CreatePod(pod *types.Pod) error {
    s.mu.Lock()           // Lock for writing
    defer s.mu.Unlock()   // Unlock when done

    if _, exists := s.pods[pod.Metadata.Name]; exists {
        return ErrAlreadyExists
    }
    s.pods[pod.Metadata.Name] = pod
    return nil
}

func (s *Store) GetPod(name string) (*types.Pod, error) {
    s.mu.RLock()          // Lock for reading (allows multiple readers)
    defer s.mu.RUnlock()

    pod, exists := s.pods[name]
    if !exists {
        return nil, ErrNotFound
    }
    return pod, nil
}

// Update ListPods and DeletePod similarly
```

**🎓 Understanding Check:**

- `RWMutex` allows multiple readers OR one writer
- `Lock()` for write operations (Create, Update, Delete)
- `RLock()` for read operations (Get, List)
- `defer` ensures unlock happens even if function returns early

**✅ Checkpoint:** Run tests again to ensure thread safety didn't break anything.

---

### Day 4-5: API Server with Fiber

**Goal:** Create HTTP endpoints to interact with storage.

#### Step 3.1: Install Fiber

```bash
go get github.com/gofiber/fiber/v2
```

#### Step 3.2: Simple API Server (cmd/apiserver/main.go)

Start with **just one endpoint**:

```go
package main

import (
    "log"
    "github.com/gofiber/fiber/v2"
    "github.com/sohel902833/minikube-clone/pkg/storage"
)

func main() {
    // Create storage
    store := storage.NewStore()

    // Create Fiber app
    app := fiber.New(fiber.Config{
        AppName: "Mini K8s API",
    })

    // Health check endpoint
    app.Get("/healthz", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{
            "status": "healthy",
        })
    })

    log.Println("Starting API server on :8080")
    log.Fatal(app.Listen(":8080"))
}
```

**Test it:**

```bash
go run cmd/apiserver/main.go
# In another terminal:
curl http://localhost:8080/healthz
```

**🎓 Understanding Check:**

- Fiber is like Express.js for Go
- `fiber.Map` is shorthand for `map[string]interface{}`
- `c.JSON()` automatically sets Content-Type and serializes

#### Step 3.3: Add Pod Endpoints

Create `pkg/api/handlers.go`:

```go
package api

import (
    "time"
    "github.com/gofiber/fiber/v2"
    "github.com/sohel902833/minikube-clone/pkg/storage"
    "github.com/sohel902833/minikube-clone/types"
)

type Handler struct {
    store *storage.Store
}

func NewHandler(store *storage.Store) *Handler {
    return &Handler{store: store}
}

// CreatePod handles POST /api/v1/pods
func (h *Handler) CreatePod(c *fiber.Ctx) error {
    var pod types.Pod

    // Parse JSON body
    if err := c.BodyParser(&pod); err != nil {
        return c.Status(400).JSON(fiber.Map{
            "error": "Invalid JSON",
        })
    }

    // Set defaults
    pod.Metadata.CreatedAt = time.Now()
    pod.Status.Phase = "Pending"

    // Save to storage
    if err := h.store.CreatePod(&pod); err != nil {
        return c.Status(500).JSON(fiber.Map{
            "error": err.Error(),
        })
    }

    return c.Status(201).JSON(pod)
}

// ListPods handles GET /api/v1/pods
func (h *Handler) ListPods(c *fiber.Ctx) error {
    pods := h.store.ListPods()
    return c.JSON(fiber.Map{
        "items": pods,
        "count": len(pods),
    })
}

// GetPod handles GET /api/v1/pods/:name
func (h *Handler) GetPod(c *fiber.Ctx) error {
    name := c.Params("name")
    pod, err := h.store.GetPod(name)
    if err != nil {
        return c.Status(404).JSON(fiber.Map{
            "error": "Pod not found",
        })
    }
    return c.JSON(pod)
}
```

#### Step 3.4: Wire Up Routes

Update `cmd/apiserver/main.go`:

```go
package main

import (
    "log"
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/logger"
    "github.com/sohel902833/minikube-clone/pkg/api"
    "github.com/sohel902833/minikube-clone/pkg/storage"
)

func main() {
    store := storage.NewStore()
    handler := api.NewHandler(store)

    app := fiber.New()

    // Middleware for logging
    app.Use(logger.New())

    // Routes
    app.Get("/healthz", func(c *fiber.Ctx) error {
        return c.JSON(fiber.Map{"status": "healthy"})
    })

    apiV1 := app.Group("/api/v1")
    apiV1.Post("/pods", handler.CreatePod)
    apiV1.Get("/pods", handler.ListPods)
    apiV1.Get("/pods/:name", handler.GetPod)

    log.Println("API Server running on :8080")
    log.Fatal(app.Listen(":8080"))
}
```

#### Step 3.5: Test the API

```bash
# Start server
go run cmd/apiserver/main.go

# In another terminal:

# Create a pod
curl -X POST http://localhost:8080/api/v1/pods \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {"name": "test-pod"},
    "spec": {
      "containers": [{"name": "nginx", "image": "nginx:latest"}]
    }
  }'

# List pods
curl http://localhost:8080/api/v1/pods

# Get specific pod
curl http://localhost:8080/api/v1/pods/test-pod
```

**✅ Checkpoint:** You should have a working REST API that can:

- Create pods
- List pods
- Get a specific pod

---

## Phase 3: Scheduler

### Day 6-7: Pod Scheduling

**Goal:** Automatically assign pods to nodes.

#### Step 4.1: Add Node Support

First, add Node types to `types/types.go`:

```go
type Node struct {
    Metadata NodeMetadata `json:"metadata"`
    Status   NodeStatus   `json:"status"`
}

type NodeMetadata struct {
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"createdAt"`
}

type NodeStatus struct {
    Phase string `json:"phase"` // Running, NotReady
}
```

#### Step 4.2: Add Node Storage

Update `pkg/storage/store.go`:

```go
type Store struct {
    pods  map[string]*types.Pod
    nodes map[string]*types.Node  // Add this
    mu    sync.RWMutex
}

func NewStore() *Store {
    return &Store{
        pods:  make(map[string]*types.Pod),
        nodes: make(map[string]*types.Node),
    }
}

func (s *Store) CreateNode(node *types.Node) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.nodes[node.Metadata.Name]; exists {
        return ErrAlreadyExists
    }
    s.nodes[node.Metadata.Name] = node
    return nil
}

func (s *Store) ListNodes() []*types.Node {
    s.mu.RLock()
    defer s.mu.RUnlock()

    nodes := make([]*types.Node, 0, len(s.nodes))
    for _, node := range s.nodes {
        nodes = append(nodes, node)
    }
    return nodes
}

func (s *Store) GetUnscheduledPods() []*types.Pod {
    s.mu.RLock()
    defer s.mu.RUnlock()

    unscheduled := make([]*types.Pod, 0)
    for _, pod := range s.pods {
        // Pod has no node AND is pending
        if pod.Spec.NodeName == "" && pod.Status.Phase == "Pending" {
            unscheduled = append(unscheduled, pod)
        }
    }
    return unscheduled
}
```

#### Step 4.3: Simple Scheduler (pkg/scheduler/scheduler.go)

```go
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

type Scheduler struct {
    apiServerURL string
    client       *http.Client
    lastNodeIdx  int  // For round-robin
}

func NewScheduler(apiServerURL string) *Scheduler {
    return &Scheduler{
        apiServerURL: apiServerURL,
        client:       &http.Client{Timeout: 10 * time.Second},
        lastNodeIdx:  0,
    }
}

func (s *Scheduler) Start(stopCh <-chan struct{}) {
    ticker := time.NewTicker(3 * time.Second)
    defer ticker.Stop()

    log.Println("Scheduler started")

    for {
        select {
        case <-ticker.C:
            s.schedule()
        case <-stopCh:
            log.Println("Scheduler stopped")
            return
        }
    }
}

func (s *Scheduler) schedule() {
    // 1. Get unscheduled pods
    pods, err := s.getUnscheduledPods()
    if err != nil {
        log.Printf("Error getting pods: %v", err)
        return
    }

    if len(pods) == 0 {
        return // Nothing to schedule
    }

    // 2. Get available nodes
    nodes, err := s.getNodes()
    if err != nil {
        log.Printf("Error getting nodes: %v", err)
        return
    }

    if len(nodes) == 0 {
        log.Println("No nodes available")
        return
    }

    // 3. Assign pods to nodes (round-robin)
    for _, pod := range pods {
        node := nodes[s.lastNodeIdx % len(nodes)]
        s.lastNodeIdx++

        pod.Spec.NodeName = node.Metadata.Name

        if err := s.updatePod(pod); err != nil {
            log.Printf("Error updating pod: %v", err)
        } else {
            log.Printf("Scheduled pod %s to node %s",
                pod.Metadata.Name, node.Metadata.Name)
        }
    }
}

func (s *Scheduler) getUnscheduledPods() ([]*types.Pod, error) {
    resp, err := s.client.Get(s.apiServerURL + "/api/v1/pods")
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

    // Filter unscheduled
    unscheduled := make([]*types.Pod, 0)
    for _, pod := range result.Items {
        if pod.Spec.NodeName == "" && pod.Status.Phase == "Pending" {
            unscheduled = append(unscheduled, pod)
        }
    }

    return unscheduled, nil
}

func (s *Scheduler) getNodes() ([]*types.Node, error) {
    resp, err := s.client.Get(s.apiServerURL + "/api/v1/nodes")
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

func (s *Scheduler) updatePod(pod *types.Pod) error {
    data, err := json.Marshal(pod)
    if err != nil {
        return err
    }

    req, err := http.NewRequest(
        "PUT",
        fmt.Sprintf("%s/api/v1/pods/%s", s.apiServerURL, pod.Metadata.Name),
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
        return fmt.Errorf("update failed: %d", resp.StatusCode)
    }

    return nil
}
```

**🎓 Understanding Check:**

- Scheduler runs in a loop (every 3 seconds)
- It finds pods without a node assignment
- It picks a node using round-robin
- It updates the pod with the node name

#### Step 4.4: Add UpdatePod to API

Update `pkg/api/handlers.go`:

```go
func (h *Handler) UpdatePod(c *fiber.Ctx) error {
    name := c.Params("name")

    var pod types.Pod
    if err := c.BodyParser(&pod); err != nil {
        return c.Status(400).JSON(fiber.Map{"error": "Invalid JSON"})
    }

    // Ensure name matches
    pod.Metadata.Name = name

    if err := h.store.UpdatePod(&pod); err != nil {
        return c.Status(404).JSON(fiber.Map{"error": err.Error()})
    }

    return c.JSON(pod)
}
```

Add to storage:

```go
func (s *Store) UpdatePod(pod *types.Pod) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.pods[pod.Metadata.Name]; !exists {
        return ErrNotFound
    }

    s.pods[pod.Metadata.Name] = pod
    return nil
}
```

Register route in `cmd/apiserver/main.go`:

```go
apiV1.Put("/pods/:name", handler.UpdatePod)
```

#### Step 4.5: Test Scheduler

Create `cmd/scheduler/main.go`:

```go
package main

import (
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/sohel902833/minikube-clone/pkg/scheduler"
)

func main() {
    sched := scheduler.NewScheduler("http://localhost:8080")

    stopCh := make(chan struct{})
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

    go sched.Start(stopCh)

    <-sigCh
    log.Println("Shutting down...")
    close(stopCh)
}
```

**Test it:**

Terminal 1:

```bash
go run cmd/apiserver/main.go
```

Terminal 2:

```bash
# Create a node first
curl -X POST http://localhost:8080/api/v1/nodes \
  -H "Content-Type: application/json" \
  -d '{"metadata": {"name": "node1"}, "status": {"phase": "Running"}}'

# Start scheduler
go run cmd/scheduler/main.go
```

Terminal 3:

```bash
# Create a pod
curl -X POST http://localhost:8080/api/v1/pods \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {"name": "my-pod"},
    "spec": {"containers": [{"name": "nginx", "image": "nginx"}]}
  }'

# Wait a few seconds, then check
curl http://localhost:8080/api/v1/pods/my-pod
# Should show nodeName: "node1"
```

**✅ Checkpoint:** Pods should automatically get assigned to nodes!

---

## Next Steps Preview

### Phase 4: Kubelet (Days 8-10)

- Install Docker SDK for Go
- Pull and run containers
- Report status back to API

### Phase 5: CLI Tool (Days 11-12)

- Build `minictl` with Cobra
- Commands: get, apply, delete
- Pretty table output

---

## 📚 Learning Resources at Each Phase

### Phase 1-2:

- Go basics: structs, pointers, interfaces
- JSON marshaling/unmarshaling
- Maps and slices
- Mutex and concurrency

### Phase 3:

- HTTP clients in Go
- REST API design
- Fiber framework basics

### Phase 4:

- Docker concepts
- Container lifecycle
- Docker SDK for Go

### Phase 5:

- CLI design patterns
- Cobra framework
- YAML parsing

---

## 🎯 Daily Checklist Template

```markdown
### Day X: [Component Name]

**What I Built:**

- [ ] Core functionality
- [ ] Tests
- [ ] Integration with existing code

**What I Learned:**

- Concept 1
- Concept 2

**Challenges:**

- Problem and how I solved it

**Tomorrow:**

- Next feature to build
```

---

## 💡 Pro Tips

1. **Run tests frequently**: `go test ./...`
2. **Use `fmt.Printf` for debugging**: Don't be afraid to add logs
3. **Commit after each working phase**: Git is your friend
4. **Read error messages carefully**: Go's errors are usually helpful
5. **Take breaks**: Building systems takes time

---

## 🆘 When You Get Stuck

1. **Check the tests**: Do they pass?
2. **Use curl**: Test API endpoints directly
3. **Add logging**: See what's actually happening
4. **Read the code**: Understand before extending
5. **Ask questions**: Google, Stack Overflow, Go forums

---

Ready to start? Begin with **Phase 1, Step 1.1**! 🚀
