package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/Takenobou/thamestracker/internal/scraper"
)

func main() {
	lifts, err := scraper.ScrapeBridgeLifts()
	if err != nil {
		log.Fatalf("Failed to scrape bridge lifts: %v", err)
	}

	// Print results as JSON
	output, _ := json.MarshalIndent(lifts, "", "  ")
	fmt.Println(string(output))
}
