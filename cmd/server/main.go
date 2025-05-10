package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Takenobou/thamestracker/internal/api"
	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/cache"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	bridgeScraper "github.com/Takenobou/thamestracker/internal/scraper/bridge"
	vesselScraper "github.com/Takenobou/thamestracker/internal/scraper/vessels"
	"github.com/Takenobou/thamestracker/internal/service"
	"github.com/Takenobou/thamestracker/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger.InitLogger()
	config.LoadConfig()
	// initialize storage cache client with loaded config
	storage.CacheClient = cache.NewRedisCache(config.AppConfig.Redis.Address)

	cacheClient := cache.NewRedisCache(config.AppConfig.Redis.Address)
	// wrap HTTP client in circuit breaker
	breakerClient := httpclient.NewBreakerClient(httpclient.DefaultClient,
		config.AppConfig.CircuitBreaker.MaxFailures,
		config.AppConfig.CircuitBreaker.CoolOffSeconds)
	svc := service.NewService(
		breakerClient,
		cacheClient,
		bridgeScraper.BridgeScraperImpl{},
		vesselScraper.VesselScraperImpl{},
	)
	handler := api.NewAPIHandler(svc)

	app := fiber.New()
	// per-IP rate limiter middleware
	app.Use(limiter.New(limiter.Config{
		Max:        config.AppConfig.RequestsPerMin,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).
				JSON(fiber.Map{"error": "Rate limit exceeded"})
		},
	}))
	// structured request logging middleware
	app.Use(logger.RequestLogger())
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
		// use ShutdownWithContext to respect timeout and exit promptly
		if err := app.ShutdownWithContext(ctx); err != nil {
			logger.Logger.Errorf("Error during server shutdown: %v", err)
		}
		logger.Logger.Infof("Server has been shut down.")
		os.Exit(0)
	}()

	if err := app.Listen(serverAddr); err != nil {
		logger.Logger.Errorf("Failed to start server: %v", err)
		os.Exit(1)
	}
}
