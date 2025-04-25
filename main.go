package main

import (
	"fmt"
	"log/slog"

	"github.com/Takenobou/thamestracker/internal/api"
	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/cache"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/service"
	"github.com/gofiber/fiber/v2"
)

func main() {
	config.LoadConfig()

	cacheClient := cache.NewRedisCache(config.AppConfig.Redis.Address)
	svc := service.NewService(httpclient.DefaultClient, cacheClient)
	handler := api.NewAPIHandler(svc)

	app := fiber.New()
	api.SetupRoutes(app, handler)

	serverAddr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	logger.Logger.Info("Server running", slog.String("address", serverAddr))

	if err := app.Listen(serverAddr); err != nil {
		logger.Logger.Error("Failed to start server", "error", err)
	}
}
