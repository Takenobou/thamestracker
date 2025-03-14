package scraper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Takenobou/thamestracker/config"
	"github.com/stretchr/testify/assert"
)

// Mock API response
const mockAPIResponse = `{
	"inport": [
		{
			"location_name": "WOODS QUAY",
			"vessel_name": "SILVER STURGEON",
			"visit": "S7670",
			"last_rep_dt": "2025-01-25 20:33:47.150"
		}
	]
}`

func TestScrapeShips(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(mockAPIResponse))
	}))
	defer server.Close()

	// Temporarily override the API URL in the config
	originalURL := config.AppConfig.URLs.PortOfLondon
	config.AppConfig.URLs.PortOfLondon = server.URL
	defer func() { config.AppConfig.URLs.PortOfLondon = originalURL }() // Restore after test

	// Call the function (now using the mock URL)
	ships, err := ScrapeShips()
	assert.NoError(t, err)
	assert.Len(t, ships, 1)

	// Validate the parsed ship data
	assert.Equal(t, "20:33", ships[0].Time)
	assert.Equal(t, "25/01/2025", ships[0].Date)
	assert.Equal(t, "WOODS QUAY", ships[0].Location)
	assert.Equal(t, "SILVER STURGEON", ships[0].Name)
	assert.Equal(t, "S7670", ships[0].VoyageNo)
}
