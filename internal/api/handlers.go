package api

import (
	"log"
	"time"

	"github.com/Takenobou/thamestracker/internal/scraper"
	"github.com/Takenobou/thamestracker/internal/storage"

	"github.com/gofiber/fiber/v2"
)

// GetBridgeLifts - Standard API for all lifts
func GetBridgeLifts(c *fiber.Ctx) error {
	var lifts []scraper.BridgeLift

	log.Println("Checking Redis cache...")
	err := storage.GetCache("bridge_lifts", &lifts)
	if err == nil {
		log.Println("Returning cached data ✅")
		return c.JSON(lifts)
	}

	log.Println("Cache miss ❌, scraping new data...")
	lifts, err = scraper.ScrapeBridgeLifts()
	if err != nil {
		log.Println("Error fetching bridge lifts:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve data"})
	}

	storage.SetCache("bridge_lifts", lifts, storage.DefaultTTL)
	return c.JSON(lifts)
}

// GetRareLifts - Returns only vessels that appear infrequently
func GetRareLifts(c *fiber.Ctx) error {
	var lifts []scraper.BridgeLift

	// Try cache first
	err := storage.GetCache("bridge_lifts", &lifts)
	if err != nil {
		log.Println("Cache miss ❌, scraping fresh data...")
		lifts, err = scraper.ScrapeBridgeLifts()
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
	var rareLifts []scraper.BridgeLift
	for _, lift := range lifts {
		if vesselCount[lift.Vessel] <= 2 {
			rareLifts = append(rareLifts, lift)
		}
	}

	return c.JSON(rareLifts)
}

// GetShips handles the /ships API endpoint
func GetShips(c *fiber.Ctx) error {
	var ships []scraper.Ship

	log.Println("Checking Redis cache for ships...")

	// Try cache first
	err := storage.GetCache("ships_in_port", &ships)
	if err == nil {
		log.Println("Returning cached ships ✅")
		return c.JSON(ships)
	}

	log.Println("Cache miss ❌, scraping fresh ship data...")

	// Scrape ship data
	ships, err = scraper.ScrapeShips()
	if err != nil {
		log.Println("Error fetching ship data:", err)
		return c.Status(500).JSON(fiber.Map{"error": "Failed to retrieve ship data"})
	}

	// Store in cache for 30 minutes
	storage.SetCache("ships_in_port", ships, 30*time.Minute)

	return c.JSON(ships)
}
