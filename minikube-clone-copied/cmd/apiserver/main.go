package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/sohel902833/minikube-clone/pkg/api"
	"github.com/sohel902833/minikube-clone/pkg/storage"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "API server port")
	flag.Parse()

	// Initialize storage
	store := storage.NewStore()

	// Create Fiber app with config
	app := fiber.New(fiber.Config{
		AppName:      "Mini Kubernetes API Server",
		ServerHeader: "MiniK8s",
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New()) // Recover from panics
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	// Setup API routes
	handler := api.NewHandler(store)
	handler.SetupRoutes(app)

	// Welcome route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Mini Kubernetes API Server",
			"version": "v1",
			"status":  "running",
		})
	})

	// Start server in a goroutine
	go func() {
		log.Printf("Starting API server on port %s", *port)
		if err := app.Listen(":" + *port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down API server...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("Error shutting down server: %v", err)
	}

	log.Println("API server stopped")
}

// customErrorHandler handles errors globally
func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error": err.Error(),
		"code":  code,
	})
}