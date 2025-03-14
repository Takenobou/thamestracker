package main

import (
	"fmt"
	"log"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/api"

	"github.com/gofiber/fiber/v2"
)

func main() {
	config.LoadConfig()

	app := fiber.New()
	api.SetupRoutes(app)

	serverAddr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	log.Printf("Server running on http://localhost%s\n", serverAddr)

	log.Fatal(app.Listen(serverAddr))
}
