package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/api"
	"github.com/Takenobou/thamestracker/internal/helpers/cache"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/service"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger.InitLogger()
	config.LoadConfig()

	cacheClient := cache.NewRedisCache(config.AppConfig.Redis.Address)
	svc := service.NewService(httpclient.DefaultClient, cacheClient)
	handler := api.NewAPIHandler(svc)

	app := fiber.New()
	api.SetupRoutes(app, handler)

	serverAddr := fmt.Sprintf(":%d", config.AppConfig.Server.Port)
	logger.Logger.Infof("Server running, address: %s", serverAddr)

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownCh
		logger.Logger.Infof("Shutdown signal received, shutting down gracefully...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := app.Shutdown(); err != nil {
			logger.Logger.Errorf("Error during server shutdown: %v", err)
		}
		<-ctx.Done()
		logger.Logger.Infof("Server has been shut down.")
		os.Exit(0)
	}()

	if err := app.Listen(serverAddr); err != nil {
		logger.Logger.Errorf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
