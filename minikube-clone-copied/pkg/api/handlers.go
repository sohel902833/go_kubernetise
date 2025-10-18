package api

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sohel902833/minikube-clone/pkg/storage"
	"github.com/sohel902833/minikube-clone/types"
)

// Handler handles API requests
type Handler struct {
	store *storage.Store
}

// NewHandler creates a new Handler
func NewHandler(store *storage.Store) *Handler {
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
	pods := h.store.ListPods()
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
	var status types.PodStatus

	if err := c.BodyParser(&status); err != nil {
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

	pod.Status = status
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
	nodes := h.store.ListNodes()
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