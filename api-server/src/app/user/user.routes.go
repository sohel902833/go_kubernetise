package user

import "github.com/gofiber/fiber/v2"

func RegisterRoutes(parent fiber.Router){
	 ctr:=UserController();
	 parent.Get("/",ctr.GetAll);
}