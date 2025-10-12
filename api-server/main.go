package main

import (
	"log"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/sohel902833/go-kubernetise-api-server/src/database"
	"github.com/sohel902833/go-kubernetise-api-server/src/initialization"
	"github.com/sohel902833/go-kubernetise-api-server/src/routes"
)

func main() {
	app := fiber.New()
	initialization.Init();
	// Connect DB and Redis
	database.Connect()

	// Define a test route
	app.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "Welcome to Go Fiber REST API 🚀",
		})
	})
	app.Use(logger.New())
	//base api group

	api:=app.Group("/api");
	v1:=api.Group("/v1");
	routes.RegisterAll(v1);

	// Start the server
	log.Fatal(app.Listen(":3000"))
}
