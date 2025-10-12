package routes

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sohel902833/go-kubernetise-api-server/src/app/user"
)

type Feature struct{
	 Path string
	 Register func(r fiber.Router)
}

func RegisterAll(parent fiber.Router){
	 features:=[]Feature{
		{
			Path: "/users",
			Register: user.RegisterRoutes,
		},
	 }
	 for _,f:=range features {
		 group:=parent.Group(f.Path);
		 f.Register(group);
	 }
}