package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/scraper/bridge"
	"github.com/Takenobou/thamestracker/internal/scraper/ships"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger.InitLogger()
	config.LoadConfig()

	if len(os.Args) < 2 {
		logger.Logger.Errorf("Usage error. Usage: %s [bridge-lifts | ships | arrivals | departures | forecast]", os.Args[0])
		os.Exit(1)
	}

	switch os.Args[1] {
	case "bridge-lifts":
		lifts, err := bridge.ScrapeBridgeLifts()
		if err != nil {
			logger.Logger.Errorf("Failed to scrape bridge lifts: %v", err)
			os.Exit(1)
		}
		printJSON(lifts)

	case "ships":
		shipList, err := ships.ScrapeShips("inport")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape ships in port: %v", err)
			os.Exit(1)
		}
		printJSON(shipList)

	case "arrivals":
		arrivalList, err := ships.ScrapeShips("arrivals")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape ship arrivals: %v", err)
			os.Exit(1)
		}
		printJSON(arrivalList)

	case "departures":
		departureList, err := ships.ScrapeShips("departures")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape ship departures: %v", err)
			os.Exit(1)
		}
		printJSON(departureList)

	case "forecast":
		forecastList, err := ships.ScrapeShips("forecast")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape ship forecasts: %v", err)
			os.Exit(1)
		}
		printJSON(forecastList)

	default:
		logger.Logger.Errorf("Unknown command: %s. Usage: Use 'bridge-lifts', 'ships', 'arrivals', 'departures', or 'forecast'", os.Args[1])
		os.Exit(1)
	}
}

func printJSON(data interface{}) {
	output, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(output))
}
