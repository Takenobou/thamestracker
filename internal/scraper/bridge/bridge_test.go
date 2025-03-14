package bridge

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Takenobou/thamestracker/internal/models"

	"github.com/gocolly/colly"
	"github.com/stretchr/testify/assert"
)

// Mock HTML similar to the Tower Bridge site
const sampleBridgeHTML = `
<html>
	<body>
		<table>
			<tbody>
				<tr><td>Sat</td><td>05 Apr 2025</td><td>17:45</td><td>Paddle Steamer Dixie Queen</td><td>Up river</td></tr>
				<tr><td>Sat</td><td>05 Apr 2025</td><td>18:45</td><td>Paddle Steamer Dixie Queen</td><td>Down river</td></tr>
			</tbody>
		</table>
	</body>
</html>
`

func TestScrapeBridgeLifts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(sampleBridgeHTML))
	}))
	defer server.Close()

	c := colly.NewCollector()
	var lifts []models.BridgeLift

	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		lift := models.BridgeLift{
			Date:      e.ChildText("td:nth-child(2)"),
			Time:      e.ChildText("td:nth-child(3)"),
			Vessel:    e.ChildText("td:nth-child(4)"),
			Direction: e.ChildText("td:nth-child(5)"),
		}
		lifts = append(lifts, lift)
	})

	err := c.Visit(server.URL)
	assert.NoError(t, err)

	// Validate scraped data
	assert.Len(t, lifts, 2)
	assert.Equal(t, "05 Apr 2025", lifts[0].Date)
	assert.Equal(t, "17:45", lifts[0].Time)
	assert.Equal(t, "Paddle Steamer Dixie Queen", lifts[0].Vessel)
	assert.Equal(t, "Up river", lifts[0].Direction)
	assert.Equal(t, "Down river", lifts[1].Direction)
}
