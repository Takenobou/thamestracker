package api

import (
	"log"
	"time"

	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/Takenobou/thamestracker/internal/scraper/bridge"
	"github.com/Takenobou/thamestracker/internal/scraper/ships"
	"github.com/Takenobou/thamestracker/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// GetBridgeLifts - Returns all scheduled bridge lifts
func GetBridgeLifts(c *fiber.Ctx) error {
	var lifts []models.BridgeLift

	log.Println("Checking Redis cache for bridge lifts...")
	err := storage.GetCache("bridge_lifts", &lifts)
	if err == nil {
		log.Println("Returning cached bridge lifts ✅")
		return c.JSON(lifts)
	}

	log.Println("Cache miss ❌, scraping fresh bridge lift data...")
	lifts, err = bridge.ScrapeBridgeLifts()
	if err != nil {
		log.Println("Error fetching bridge lifts:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve bridge lift data"})
	}

	storage.SetCache("bridge_lifts", lifts, storage.DefaultTTL)
	return c.JSON(lifts)
}

// GetRareLifts - Returns only vessels that appear infrequently
func GetRareLifts(c *fiber.Ctx) error {
	var lifts []models.BridgeLift

	// Try cache first
	err := storage.GetCache("bridge_lifts", &lifts)
	if err != nil {
		log.Println("Cache miss ❌, scraping fresh data...")
		lifts, err = bridge.ScrapeBridgeLifts()
		if err != nil {
			log.Println("Error fetching bridge lifts:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve data"})
		}
		// Store in cache
		storage.SetCache("bridge_lifts", lifts, storage.DefaultTTL)
	}

	// Count occurrences of each vessel
	vesselCount := make(map[string]int)
	for _, lift := range lifts {
		vesselCount[lift.Vessel]++
	}

	// Filter rare vessels (e.g., appears ≤ 2 times)
	var rareLifts []models.BridgeLift
	for _, lift := range lifts {
		if vesselCount[lift.Vessel] <= 2 {
			rareLifts = append(rareLifts, lift)
		}
	}

	return c.JSON(rareLifts)
}

// GetShipData - Generic handler for fetching ship data
func GetShipData(shipType string, cacheKey string, c *fiber.Ctx) error {
	var shipList []models.Ship

	log.Printf("Checking Redis cache for %s...", shipType)
	err := storage.GetCache(cacheKey, &shipList)
	if err == nil {
		log.Printf("Returning cached %s ✅", shipType)
		return c.JSON(shipList)
	}

	log.Printf("Cache miss ❌, scraping fresh %s data...", shipType)
	shipList, err = ships.ScrapeShips(shipType)
	if err != nil {
		log.Printf("Error fetching %s data: %v", shipType, err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve data"})
	}

	// Store in cache for 30 minutes
	storage.SetCache(cacheKey, shipList, 30*time.Minute)

	return c.JSON(shipList)
}

func GetShips(c *fiber.Ctx) error {
	return GetShipData("inport", "ships_in_port", c)
}

// GetArrivals - Fetch currently arriving ships
func GetArrivals(c *fiber.Ctx) error {
	return GetShipData("arrivals", "ship_arrivals", c)
}

// GetDepartures - Fetch currently departing ships
func GetDepartures(c *fiber.Ctx) error {
	return GetShipData("departures", "ship_departures", c)
}

// GetForecast - Fetch upcoming ship movements
func GetForecast(c *fiber.Ctx) error {
	return GetShipData("forecast", "ship_forecast", c)
}
