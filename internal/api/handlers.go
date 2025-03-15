package api

import (
	"log"
	"strings"
	"time"

	"github.com/Takenobou/thamestracker/internal/models"
	bridgeScraper "github.com/Takenobou/thamestracker/internal/scraper/bridge"
	shipScraper "github.com/Takenobou/thamestracker/internal/scraper/ships"
	"github.com/Takenobou/thamestracker/internal/storage"
	"github.com/gofiber/fiber/v2"
)

// GetBridgeLifts returns all scheduled bridge lifts (or unique ones if ?unique=true is set)
func GetBridgeLifts(c *fiber.Ctx) error {
	var lifts []models.BridgeLift

	log.Println("Checking Redis cache for bridge lifts...")
	err := storage.GetCache("bridge_lifts", &lifts)
	if err == nil {
		log.Println("Returning cached bridge lifts ✅")
	} else {
		log.Println("Cache miss ❌, scraping fresh bridge lift data...")
		lifts, err = bridgeScraper.ScrapeBridgeLifts()
		if err != nil {
			log.Println("Error fetching bridge lifts:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve bridge lift data"})
		}
		storage.SetCache("bridge_lifts", lifts, 15*time.Minute)
	}

	// Filter unique lifts if query parameter is set
	unique := c.Query("unique", "false")
	if unique == "true" {
		lifts = FilterUniqueLifts(lifts, 4)
	}

	return c.JSON(lifts)
}

// GetShips handles all ship data queries by type using a query parameter.
// Use ?type=all (default), ?type=inport, ?type=arrivals, ?type=departures, or ?type=forecast.
func GetShips(c *fiber.Ctx) error {
	shipType := c.Query("type", "all")
	var cacheKey string
	if shipType == "all" {
		cacheKey = "all_ships"
	} else {
		cacheKey = "ships_" + strings.ToLower(shipType)
	}
	return GetShipData(shipType, cacheKey, c)
}

// GetShipData fetches ships from the API (or Redis cache) and applies filtering based on query parameters.
func GetShipData(shipType, cacheKey string, c *fiber.Ctx) error {
	var ships []models.Ship

	// Check Redis cache first
	err := storage.GetCache(cacheKey, &ships)
	if err == nil {
		log.Println("Returning cached ship data ✅")
	} else {
		// Cache miss - scrape fresh data
		log.Println("Cache miss ❌, scraping fresh data...")
		ships, err = shipScraper.ScrapeShips(shipType)
		if err != nil {
			log.Println("Error fetching ship data:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve ship data"})
		}
		storage.SetCache(cacheKey, ships, 30*time.Minute)
	}

	// **Apply Filtering** based on additional query parameters.
	filteredShips := FilterShips(ships, c)
	return c.JSON(filteredShips)
}
