package api

import (
	"time"

	"kubico/pkg/storage"
	"kubico/types"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Handler handles API requests
type Handler struct {
	store *storage.PostgresStore
}

// NewHandler creates a new Handler
func NewHandler(store *storage.PostgresStore) *Handler {
	return &Handler{
		store: store,
	}
}

// SetupRoutes configures all API routes
func (h *Handler) SetupRoutes(app *fiber.App) {
	api := app.Group("/api/v1")

	// Pod endpoints
	api.Post("/pods", h.CreatePod)
	api.Get("/pods", h.ListPods)
	api.Get("/pods/:name", h.GetPod)
	api.Delete("/pods/:name", h.DeletePod)
	api.Put("/pods/:name/status", h.UpdatePodStatus)

	// Node endpoints
	api.Post("/nodes", h.CreateNode)
	api.Get("/nodes", h.ListNodes)
	api.Get("/nodes/:name", h.GetNode)
	api.Put("/nodes/:name/status", h.UpdateNodeStatus)
	api.Delete("/nodes/:name", h.DeleteNode)


		// ReplicaSet endpoints
	api.Post("/replicasets", h.CreateReplicaSet)
	api.Get("/replicasets", h.ListReplicaSets)
	api.Get("/replicasets/:name", h.GetReplicaSet)
	api.Delete("/replicasets/:name", h.DeleteReplicaSet)
	api.Put("/replicasets/:name", h.UpdateReplicaSet)

	// Health check
	app.Get("/healthz", h.HealthCheck)
}

// CreatePod creates a new pod
func (h *Handler) CreatePod(c *fiber.Ctx) error {
	var pod types.Pod
	if err := c.BodyParser(&pod); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Set default values
	if pod.APIVersion == "" {
		pod.APIVersion = "v1"
	}
	if pod.Kind == "" {
		pod.Kind = "Pod"
	}
	if pod.Metadata.Namespace == "" {
		pod.Metadata.Namespace = "default"
	}

	pod.Metadata.CreatedAt = time.Now()
	pod.Status.Phase = types.PodPending

	if err := h.store.CreatePod(&pod); err != nil {
		if err == storage.ErrAlreadyExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "pod already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(pod)
}

// ListPods returns all pods
func (h *Handler) ListPods(c *fiber.Ctx) error {
	pods,err := h.store.ListPods()
	if(err!=nil){
		 return c.JSON(fiber.Map{
			 "message":"Error while getting pods",
		 })
	}
	return c.JSON(fiber.Map{
		"items": pods,
		"count": len(pods),
	})
}

// GetPod returns a specific pod
func (h *Handler) GetPod(c *fiber.Ctx) error {
	name := c.Params("name")
	pod, err := h.store.GetPod(name)
	if err != nil {
		if err == storage.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "pod not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(pod)
}

// DeletePod deletes a pod
func (h *Handler) DeletePod(c *fiber.Ctx) error {
	name := c.Params("name")
	if err := h.store.DeletePod(name); err != nil {
		if err == storage.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "pod not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "pod deleted successfully",
	})
}

// UpdatePodStatus updates a pod's status
func (h *Handler) UpdatePodStatus(c *fiber.Ctx) error {
	name := c.Params("name")
	var updatedPod types.Pod

	if err := c.BodyParser(&updatedPod); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	pod, err := h.store.GetPod(name)
	if err != nil {
		if err == storage.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "pod not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Update NodeName if provided
	if updatedPod.Spec.NodeName != "" {
		pod.Spec.NodeName = updatedPod.Spec.NodeName
	}

	// Update Status fields
	pod.Status = updatedPod.Status
	if err := h.store.UpdatePod(pod); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(pod)
}

// CreateNode registers a new node
func (h *Handler) CreateNode(c *fiber.Ctx) error {
	var node types.Node
	if err := c.BodyParser(&node); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	node.Metadata.CreatedAt = time.Now()
	node.Status.Phase = types.NodeRunning

	if err := h.store.CreateNode(&node); err != nil {
		if err == storage.ErrAlreadyExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "node already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(node)
}

// ListNodes returns all nodes
func (h *Handler) ListNodes(c *fiber.Ctx) error {
	nodes,err := h.store.ListNodes()
	if(err!=nil){
		 return c.JSON(fiber.Map{
			 "message":"Error while getting nodes",
		 })
	}
	return c.JSON(fiber.Map{
		"items": nodes,
		"count": len(nodes),
	})
}

// GetNode returns a specific node
func (h *Handler) GetNode(c *fiber.Ctx) error {
	name := c.Params("name")
	node, err := h.store.GetNode(name)
	if err != nil {
		if err == storage.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "node not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(node)
}

// UpdateNodeStatus updates a node's status
func (h *Handler) UpdateNodeStatus(c *fiber.Ctx) error {
	name := c.Params("name")
	var status types.NodeStatus

	if err := c.BodyParser(&status); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	node, err := h.store.GetNode(name)
	if err != nil {
		if err == storage.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "node not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	node.Status = status
	if err := h.store.UpdateNode(node); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(node)
}

// DeleteNode deletes a node
func (h *Handler) DeleteNode(c *fiber.Ctx) error {
	name := c.Params("name")
	if err := h.store.DeleteNode(name); err != nil {
		if err == storage.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "node not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "node deleted successfully",
	})
}

// HealthCheck returns API server health status
func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "healthy",
		"time":   time.Now(),
	})
}



// CreateReplicaSet creates a new ReplicaSet
func (h *Handler) CreateReplicaSet(c *fiber.Ctx) error {
	var rs types.ReplicaSet
	if err := c.BodyParser(&rs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Set defaults
	if rs.APIVersion == "" {
		rs.APIVersion = "apps/v1"
	}
	if rs.Kind == "" {
		rs.Kind = "ReplicaSet"
	}
	if rs.Metadata.Namespace == "" {
		rs.Metadata.Namespace = "default"
	}
	if rs.Metadata.UID == "" {
		rs.Metadata.UID = uuid.New().String()
	}

	rs.Metadata.CreatedAt = time.Now()
	rs.Status.Replicas = 0
	rs.Status.ReadyReplicas = 0
	rs.Status.AvailableReplicas = 0

	if err := h.store.CreateReplicaSet(&rs); err != nil {
		if err == storage.ErrAlreadyExists {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "replicaset already exists",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(rs)
}

// ListReplicaSets returns all ReplicaSets
func (h *Handler) ListReplicaSets(c *fiber.Ctx) error {
	replicaSets, err := h.store.ListReplicaSets()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"items": replicaSets,
		"count": len(replicaSets),
	})
}

// GetReplicaSet returns a specific ReplicaSet
func (h *Handler) GetReplicaSet(c *fiber.Ctx) error {
	name := c.Params("name")
	rs, err := h.store.GetReplicaSet(name)
	if err != nil {
		if err == storage.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "replicaset not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(rs)
}

// UpdateReplicaSet updates a ReplicaSet
func (h *Handler) UpdateReplicaSet(c *fiber.Ctx) error {
	name := c.Params("name")
	
	var rs types.ReplicaSet
	if err := c.BodyParser(&rs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	rs.Metadata.Name = name

	if err := h.store.UpdateReplicaSet(&rs); err != nil {
		if err == storage.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "replicaset not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(rs)
}

// DeleteReplicaSet deletes a ReplicaSet (and optionally its pods)
func (h *Handler) DeleteReplicaSet(c *fiber.Ctx) error {
	name := c.Params("name")
	
	// Get the ReplicaSet first
	rs, err := h.store.GetReplicaSet(name)
	if err != nil {
		if err == storage.ErrNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "replicaset not found",
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Delete all pods owned by this ReplicaSet
	pods, _ := h.store.GetPodsByLabels(rs.Spec.Selector)
	for _, pod := range pods {
		for _, ownerRef := range pod.Metadata.OwnerReferences {
			if ownerRef.Kind == "ReplicaSet" && ownerRef.Name == name {
				h.store.DeletePod(pod.Metadata.Name)
				break
			}
		}
	}

	// Delete the ReplicaSet
	if err := h.store.DeleteReplicaSet(name); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "replicaset deleted successfully",
	})
}
