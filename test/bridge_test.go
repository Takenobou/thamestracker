package test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Takenobou/thamestracker/internal/models"
	"github.com/gocolly/colly"
	"github.com/stretchr/testify/assert"
)

const sampleBridgeHTML = `
<html>
	<body>
		<table>
			<tbody>
				<tr>
					<td>Sat</td>
					<td><time datetime="2025-04-05T00:00:00Z">05 Apr 2025</time></td>
					<td><time datetime="2025-04-05T17:45:00Z">17:45</time></td>
					<td>Paddle Steamer Dixie Queen</td>
					<td>Up river</td>
				</tr>
				<tr>
					<td>Sat</td>
					<td><time datetime="2025-04-05T00:00:00Z">05 Apr 2025</time></td>
					<td><time datetime="2025-04-05T18:45:00Z">18:45</time></td>
					<td>Paddle Steamer Dixie Queen</td>
					<td>Down river</td>
				</tr>
			</tbody>
		</table>
	</body>
</html>
`

func TestScrapeBridgeLifts(t *testing.T) {
	// Create a mock HTTP server that serves the sample HTML.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleBridgeHTML))
	}))
	defer server.Close()

	// Create a new Colly collector.
	c := colly.NewCollector()
	var lifts []models.BridgeLift

	// The scraper expects to extract date/time from <time> elements.
	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		lift := models.BridgeLift{
			Date:      e.ChildAttr("td:nth-child(2) time", "datetime"),
			Time:      e.ChildAttr("td:nth-child(3) time", "datetime"),
			Vessel:    e.ChildText("td:nth-child(4)"),
			Direction: e.ChildText("td:nth-child(5)"),
		}

		// The scraper then slices the datetime strings.
		if lift.Date != "" {
			lift.Date = lift.Date[:10] // e.g., "2025-04-05"
		}
		if lift.Time != "" {
			lift.Time = lift.Time[11:16] // e.g., "17:45" from "2025-04-05T17:45:00Z"
		}
		lifts = append(lifts, lift)
	})

	err := c.Visit(server.URL)
	assert.NoError(t, err)

	// Validate that two lift events were scraped.
	assert.Len(t, lifts, 2, "expected 2 bridge lift events")

	// For first row, expected values:
	assert.Equal(t, "2025-04-05", lifts[0].Date)
	assert.Equal(t, "17:45", lifts[0].Time)
	assert.Equal(t, "Paddle Steamer Dixie Queen", lifts[0].Vessel)
	assert.Equal(t, "Up river", lifts[0].Direction)

	// For second row:
	assert.Equal(t, "2025-04-05", lifts[1].Date)
	assert.Equal(t, "18:45", lifts[1].Time)
	assert.Equal(t, "Down river", lifts[1].Direction)
}
