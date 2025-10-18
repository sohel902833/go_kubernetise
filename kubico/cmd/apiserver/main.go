package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"kubico/pkg/api"
	"kubico/pkg/storage"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	// Parse command line flags
	port := flag.String("port", "8080", "API server port")
	dbURL := flag.String("db-url", "postgres://minik8s:minik8s123@localhost:5436/minik8s?sslmode=disable", "PostgreSQL connection URL")
	flag.Parse()

	// Initialize PostgreSQL storage
	log.Println("🔌 Connecting to PostgreSQL...")
	store, err := storage.NewPostgresStore(*dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer store.Close()
	log.Println("Connected to PostgreSQL")

	// Create Fiber app with config
	app := fiber.New(fiber.Config{
		AppName:      "Mini Kubernetes API Server",
		ServerHeader: "MiniK8s",
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New()) // Recover from panics
	app.Use(logger.New(logger.Config{
		Format:     "[${time}] ${status} - ${method} ${path} (${latency})\n",
		TimeFormat: "15:04:05",
		TimeZone:   "Local",
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
			"features": []string{
				"Pods",
				"ReplicaSets",
				"Nodes",
				"Health Checks",
				"Auto-scaling",
			},
		})
	})

	// Start server in a goroutine
	go func() {
		log.Printf("API Server starting on port %s", *port)
		log.Printf("Endpoints:")
		log.Printf("   - Health:       http://localhost:%s/healthz", *port)
		log.Printf("   - Pods:         http://localhost:%s/api/v1/pods", *port)
		log.Printf("   - ReplicaSets:  http://localhost:%s/api/v1/replicasets", *port)
		log.Printf("   - Nodes:        http://localhost:%s/api/v1/nodes", *port)
		
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