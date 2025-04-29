package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/cache"
	"github.com/Takenobou/thamestracker/internal/helpers/httpclient"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"

	"github.com/Takenobou/thamestracker/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	logger.InitLogger()
	config.LoadConfig()

	// initialize service layer
	cacheClient := cache.NewRedisCache(config.AppConfig.Redis.Address)
	svc := service.NewService(httpclient.DefaultClient, cacheClient)

	if len(os.Args) < 2 {
		logger.Logger.Errorf("Usage error. Usage: %s [bridge-lifts | vessels | arrivals | departures | forecast | ics | bridge-ics | vessels-ics]", os.Args[0])
		os.Exit(1)
	}

	switch os.Args[1] {
	case "ics":
		// Fetch combined calendar ICS feed from local server
		url := fmt.Sprintf("http://localhost:%d/calendar.ics", config.AppConfig.Server.Port)
		resp, err := httpclient.DefaultClient.Get(url)
		if err != nil {
			logger.Logger.Errorf("Failed to fetch calendar ICS: %v", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return

	case "bridge-ics":
		// Fetch bridge-lifts calendar ICS feed
		url := fmt.Sprintf("http://localhost:%d/bridge-lifts/calendar.ics", config.AppConfig.Server.Port)
		resp, err := httpclient.DefaultClient.Get(url)
		if err != nil {
			logger.Logger.Errorf("Failed to fetch bridge-lifts calendar ICS: %v", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return

	case "vessels-ics":
		// Fetch vessels calendar ICS feed
		url := fmt.Sprintf("http://localhost:%d/vessels/calendar.ics", config.AppConfig.Server.Port)
		resp, err := httpclient.DefaultClient.Get(url)
		if err != nil {
			logger.Logger.Errorf("Failed to fetch vessels calendar ICS: %v", err)
			os.Exit(1)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Println(string(body))
		return

	case "bridge-lifts":
		lifts, err := svc.GetBridgeLifts()
		if err != nil {
			logger.Logger.Errorf("Failed to scrape bridge lifts: %v", err)
			os.Exit(1)
		}
		printJSON(lifts)

	case "vessels":
		vesselList, err := svc.GetVessels("inport")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape vessels in port: %v", err)
			os.Exit(1)
		}
		printJSON(vesselList)

	case "arrivals":
		arrivalList, err := svc.GetVessels("arrivals")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape vessel arrivals: %v", err)
			os.Exit(1)
		}
		printJSON(arrivalList)

	case "departures":
		departureList, err := svc.GetVessels("departures")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape vessel departures: %v", err)
			os.Exit(1)
		}
		printJSON(departureList)

	case "forecast":
		forecastList, err := svc.GetVessels("forecast")
		if err != nil {
			logger.Logger.Errorf("Failed to scrape vessel forecasts: %v", err)
			os.Exit(1)
		}
		printJSON(forecastList)

	default:
		logger.Logger.Errorf("Unknown command: %s. Usage: Use 'bridge-lifts', 'vessels', 'arrivals', 'departures', 'forecast', 'ics', 'bridge-ics', or 'vessels-ics'", os.Args[1])
		os.Exit(1)
	}
}

func printJSON(data interface{}) {
	output, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(output))
}
