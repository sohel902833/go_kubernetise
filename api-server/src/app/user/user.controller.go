package user

import (
	"github.com/gofiber/fiber/v2"
)
type Controller struct{
	
}


func UserController()*Controller{
	return &Controller{}
}

func (ctr *Controller) GetAll(c *fiber.Ctx) error {
	users := []User{
		{ID: 1, Name: "Sohrab", Email: "sohrab@example.com"},
		{ID: 2, Name: "Shumi", Email: "shumi@example.com"},
		{ID: 3, Name: "Isac", Email: "isac@example.com"},
	}

	return c.JSON(users)
}