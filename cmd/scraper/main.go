package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/scraper/bridge"
	"github.com/Takenobou/thamestracker/internal/scraper/vessels"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger.InitLogger()
	config.LoadConfig()

	if len(os.Args) < 2 {
		logger.Logger.Errorf("Usage error. Usage: %s [bridge-lifts | vessels | arrivals | departures | forecast]", os.Args[0])
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

	case "vessels":
		vesselList, err := vessels.ScrapeVessels("inport")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape vessels in port: %v", err)
			os.Exit(1)
		}
		printJSON(vesselList)

	case "arrivals":
		arrivalList, err := vessels.ScrapeVessels("arrivals")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape vessel arrivals: %v", err)
			os.Exit(1)
		}
		printJSON(arrivalList)

	case "departures":
		departureList, err := vessels.ScrapeVessels("departures")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape vessel departures: %v", err)
			os.Exit(1)
		}
		printJSON(departureList)

	case "forecast":
		forecastList, err := vessels.ScrapeVessels("forecast")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape vessel forecasts: %v", err)
			os.Exit(1)
		}
		printJSON(forecastList)

	default:
		logger.Logger.Errorf("Unknown command: %s. Usage: Use 'bridge-lifts', 'vessels', 'arrivals', 'departures', or 'forecast'", os.Args[1])
		os.Exit(1)
	}
}

func printJSON(data interface{}) {
	output, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(output))
}
