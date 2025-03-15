package main

import (
	"fmt"
	"log"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/api"
	"github.com/Takenobou/thamestracker/internal/helpers/cache"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/service"
	"github.com/gofiber/fiber/v2"
)

func main() {
	config.LoadConfig()

	// Instantiate dependencies.
	cacheClient := cache.NewRedisCache(config.AppConfig.Redis.Address)
	svc := service.NewService(httpclient.DefaultClient, cacheClient)
	handler := api.NewAPIHandler(svc)

	app := fiber.New()
	api.SetupRoutes(app, handler)

	serverAddr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	log.Printf("Server running on http://localhost%s\n", serverAddr)

	log.Fatal(app.Listen(serverAddr))
}
