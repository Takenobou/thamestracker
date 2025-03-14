package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/scraper"
)

func main() {
	// Load configuration before anything else
	config.LoadConfig()

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s [bridge-lifts | ships]", os.Args[0])
	}

	switch os.Args[1] {
	case "bridge-lifts":
		lifts, err := scraper.ScrapeBridgeLifts()
		if err != nil {
			log.Fatalf("Failed to scrape bridge lifts: %v", err)
		}
		printJSON(lifts)

	case "ships":
		ships, err := scraper.ScrapeShips()
		if err != nil {
			log.Fatalf("Failed to scrape ships in port: %v", err)
		}
		printJSON(ships)

	default:
		log.Fatalf("Unknown command: %s. Use 'bridge-lifts' or 'ships'", os.Args[1])
	}
}

// printJSON formats and prints JSON output
func printJSON(data interface{}) {
	output, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(output))
}
