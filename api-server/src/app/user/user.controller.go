package user

import (
	"context"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sohel902833/go-kubernetise-api-server/src/database"
)

// Controller struct
type Controller struct{}

// Factory function
func UserController() *Controller {
	return &Controller{}
}

// GetAll creates dummy users, stores in Postgres and Redis
func (ctr *Controller) GetAll(c *fiber.Ctx) error {
	ctx := context.Background()

	// Dummy users
	users := []struct {
		Name  string
		Email string
	}{
		{"Sohrab", "sohrab@example.com"},
		{"Shumi", "shumi@example.com"},
		{"Isac", "isac@example.com"},
	}

	storedUsers := []map[string]interface{}{}

	for _, u := range users {
		// 1️⃣ Store in PostgreSQL using Ent
		user, err := database.Client.User.
			Create().
			SetName(u.Name).
			SetEmail(u.Email).
			Save(ctx)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to save user in Postgres: %v", err),
			})
		}

		// 2️⃣ Store in Redis cache (JSON string)
		key := fmt.Sprintf("user:%d", user.ID)
		err = database.RedisClient.Set(ctx, key, fmt.Sprintf("%s|%s", user.Name, user.Email), 10*time.Minute).Err()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": fmt.Sprintf("failed to save user in Redis: %v", err),
			})
		}

		storedUsers = append(storedUsers, map[string]interface{}{
			"id":    user.ID,
			"name":  user.Name,
			"email": user.Email,
		})
	}

	return c.JSON(fiber.Map{
		"message": "dummy users stored in Postgres and Redis",
		"data":    storedUsers,
	})
}
