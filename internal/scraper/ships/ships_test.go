package ships

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Takenobou/thamestracker/config"
	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/stretchr/testify/assert"
)

// Mock API response covering all ship categories
const mockAPIResponse = `{
	"inport": [
		{
			"location_name": "WOODS QUAY",
			"vessel_name": "SILVER STURGEON",
			"visit": "S7670",
			"last_rep_dt": "2025-01-25 20:33:47.150"
		}
	],
	"arrivals": [
		{
			"location_from": "MAPTM",
			"location_to": "LONDON GATEWAY1",
			"vessel_name": "SAN NICOLAS MAERSK",
			"visit": "S7795",
			"last_rep_dt": "2025-03-13 14:22:09.300"
		}
	],
	"departures": [
		{
			"location_from": "TILBURY DOCK",
			"location_to": "SESOE",
			"vessel_name": "FRISIAN SPRING",
			"visit": "F1785",
			"first_rep_dt": "2025-03-13 15:39:03.690"
		}
	],
	"forecast": [
		{
			"location_from": "NLVLI",
			"location_to": "FORDS JETTY",
			"vessel_name": "ADELINE",
			"visit": "A9999",
			"etad_dt": "2025-03-14 14:15:00.000"
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

	// Backup and override API URL
	originalURL := config.AppConfig.URLs.PortOfLondon
	config.AppConfig.URLs.PortOfLondon = server.URL
	defer func() { config.AppConfig.URLs.PortOfLondon = originalURL }() // Restore after test

	// Define test cases for different ship types including "all"
	tests := []struct {
		name          string
		shipType      string
		expectedLen   int
		expectedShips []models.Ship
	}{
		{
			name:        "Inport",
			shipType:    "inport",
			expectedLen: 1,
			expectedShips: []models.Ship{
				{
					Time:         "20:33",
					Date:         "25/01/2025",
					LocationName: "WOODS QUAY",
					Name:         "SILVER STURGEON",
					VoyageNo:     "S7670",
					Type:         "inport",
				},
			},
		},
		{
			name:        "Arrivals",
			shipType:    "arrivals",
			expectedLen: 1,
			expectedShips: []models.Ship{
				{
					Time:         "14:22",
					Date:         "13/03/2025",
					LocationFrom: "MAPTM",
					LocationTo:   "LONDON GATEWAY1",
					Name:         "SAN NICOLAS MAERSK",
					VoyageNo:     "S7795",
					Type:         "arrivals",
				},
			},
		},
		{
			name:        "Departures",
			shipType:    "departures",
			expectedLen: 1,
			expectedShips: []models.Ship{
				{
					Time:         "15:39",
					Date:         "13/03/2025",
					LocationFrom: "TILBURY DOCK",
					LocationTo:   "SESOE",
					Name:         "FRISIAN SPRING",
					VoyageNo:     "F1785",
					Type:         "departures",
				},
			},
		},
		{
			name:        "Forecast",
			shipType:    "forecast",
			expectedLen: 1,
			expectedShips: []models.Ship{
				{
					Time:         "14:15",
					Date:         "14/03/2025",
					LocationFrom: "NLVLI",
					LocationTo:   "FORDS JETTY",
					Name:         "ADELINE",
					VoyageNo:     "A9999",
					Type:         "forecast",
				},
			},
		},
		{
			name:        "All",
			shipType:    "all",
			expectedLen: 4,
			expectedShips: []models.Ship{
				{
					Time:         "20:33",
					Date:         "25/01/2025",
					LocationName: "WOODS QUAY",
					Name:         "SILVER STURGEON",
					VoyageNo:     "S7670",
					Type:         "inport",
				},
				{
					Time:         "14:22",
					Date:         "13/03/2025",
					LocationFrom: "MAPTM",
					LocationTo:   "LONDON GATEWAY1",
					Name:         "SAN NICOLAS MAERSK",
					VoyageNo:     "S7795",
					Type:         "arrivals",
				},
				{
					Time:         "15:39",
					Date:         "13/03/2025",
					LocationFrom: "TILBURY DOCK",
					LocationTo:   "SESOE",
					Name:         "FRISIAN SPRING",
					VoyageNo:     "F1785",
					Type:         "departures",
				},
				{
					Time:         "14:15",
					Date:         "14/03/2025",
					LocationFrom: "NLVLI",
					LocationTo:   "FORDS JETTY",
					Name:         "ADELINE",
					VoyageNo:     "A9999",
					Type:         "forecast",
				},
			},
		},
	}

	// Run tests for each ship type
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ships, err := ScrapeShips(tc.shipType)
			assert.NoError(t, err)
			assert.Len(t, ships, tc.expectedLen, "unexpected number of ships for type %s", tc.shipType)

			// Validate each field for every expected ship
			for i, expected := range tc.expectedShips {
				assert.Equal(t, expected.Time, ships[i].Time, "Time mismatch for %s", expected.Name)
				assert.Equal(t, expected.Date, ships[i].Date, "Date mismatch for %s", expected.Name)
				assert.Equal(t, expected.LocationFrom, ships[i].LocationFrom, "LocationFrom mismatch for %s", expected.Name)
				assert.Equal(t, expected.LocationTo, ships[i].LocationTo, "LocationTo mismatch for %s", expected.Name)
				assert.Equal(t, expected.LocationName, ships[i].LocationName, "LocationName mismatch for %s", expected.Name)
				assert.Equal(t, expected.Name, ships[i].Name, "Name mismatch")
				assert.Equal(t, expected.VoyageNo, ships[i].VoyageNo, "VoyageNo mismatch for %s", expected.Name)
				assert.Equal(t, expected.Type, ships[i].Type, "Type mismatch for %s", expected.Name)
			}
		})
	}
}
