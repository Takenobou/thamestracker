package test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sampleBridgeHTML))
	}))
	defer server.Close()

	c := colly.NewCollector()
	var events []models.Event

	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		timeStr := e.ChildAttr("td:nth-child(3) time", "datetime")
		timestamp, _ := time.Parse(time.RFC3339, timeStr)
		vessel := e.ChildText("td:nth-child(4)")
		direction := e.ChildText("td:nth-child(5)")
		event := models.Event{
			Timestamp:  timestamp,
			VesselName: vessel,
			Category:   "bridge",
			Direction:  direction,
			Location:   "Tower Bridge Road, London",
		}
		events = append(events, event)
	})

	err := c.Visit(server.URL)
	assert.NoError(t, err)

	assert.Len(t, events, 2, "expected 2 bridge lift events")
	assert.Equal(t, "Paddle Steamer Dixie Queen", events[0].VesselName)
	assert.Equal(t, "Up river", events[0].Direction)
	assert.Equal(t, "Down river", events[1].Direction)
	assert.Equal(t, "bridge", events[0].Category)
	assert.Equal(t, "Tower Bridge Road, London", events[0].Location)
	assert.Equal(t, 17, events[0].Timestamp.Hour())
	assert.Equal(t, 45, events[0].Timestamp.Minute())
}
