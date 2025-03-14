package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/scraper/bridge"
	"github.com/Takenobou/thamestracker/internal/scraper/ships"
)

func main() {
	// Load configuration before anything else
	config.LoadConfig()

	if len(os.Args) < 2 {
		log.Fatalf("Usage: %s [bridge-lifts | ships | arrivals | departures | forecast]", os.Args[0])
	}

	switch os.Args[1] {
	case "bridge-lifts":
		lifts, err := bridge.ScrapeBridgeLifts()
		if err != nil {
			log.Fatalf("Failed to scrape bridge lifts: %v", err)
		}
		printJSON(lifts)

	case "ships":
		shipList, err := ships.ScrapeShips("inport") // ✅ Fix: Pass "inport" explicitly
		if err != nil {
			log.Fatalf("Failed to scrape ships in port: %v", err)
		}
		printJSON(shipList)

	case "arrivals":
		arrivalList, err := ships.ScrapeShips("arrivals") // ✅ Now handles arrivals
		if err != nil {
			log.Fatalf("Failed to scrape ship arrivals: %v", err)
		}
		printJSON(arrivalList)

	case "departures":
		departureList, err := ships.ScrapeShips("departures") // ✅ Now handles departures
		if err != nil {
			log.Fatalf("Failed to scrape ship departures: %v", err)
		}
		printJSON(departureList)

	case "forecast":
		forecastList, err := ships.ScrapeShips("forecast") // ✅ Now handles forecasts
		if err != nil {
			log.Fatalf("Failed to scrape ship forecasts: %v", err)
		}
		printJSON(forecastList)

	default:
		log.Fatalf("Unknown command: %s. Use 'bridge-lifts', 'ships', 'arrivals', 'departures', or 'forecast'", os.Args[1])
	}
}

func printJSON(data interface{}) {
	output, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(output))
}
