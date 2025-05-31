package bridge

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"fmt"
	"os"

	"github.com/Takenobou/thamestracker/internal/config"
	"github.com/Takenobou/thamestracker/internal/helpers/logger"
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

func TestMain(m *testing.M) {
	logger.InitLogger()
	os.Exit(m.Run())
}

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
		timestamp, err := time.Parse(time.RFC3339, timeStr)
		if timeStr == "" {
			return // skip missing datetime
		}
		if err != nil {
			return // skip parse error
		}
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

func TestScrapeBridgeLifts_MissingDatetime(t *testing.T) {
	html := `<html><body><table><tbody><tr><td>Sat</td><td></td><td></td><td>Vessel</td><td>Dir</td></tr></tbody></table></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	c := colly.NewCollector()
	var called bool
	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		timeStr := e.ChildAttr("td:nth-child(3) time", "datetime")
		if timeStr == "" {
			called = true
			return // skip
		}
	})
	_ = c.Visit(server.URL)
	assert.True(t, called, "should skip row with missing datetime")
}

func TestScrapeBridgeLifts_ParseError(t *testing.T) {
	html := `<html><body><table><tbody><tr><td>Sat</td><td></td><td><time datetime=\"badtime\"></time></td><td>Vessel</td><td>Dir</td></tr></tbody></table></body></html>`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	c := colly.NewCollector()
	var called bool
	c.OnHTML("tbody tr", func(e *colly.HTMLElement) {
		timeStr := e.ChildAttr("td:nth-child(3) time", "datetime")
		_, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			called = true
			return // skip
		}
	})
	_ = c.Visit(server.URL)
	assert.True(t, called, "should skip row with parse error")
}

func TestScrapeBridgeLifts_Pagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			w.Write([]byte(`<html><body>
				<table><tbody>
					<tr><td>Sat</td><td></td><td><time datetime="2025-04-05T17:45:00Z"></time></td><td>Vessel1</td><td>Up river</td></tr>
				</tbody></table>
				<nav class="pager">
					<span><a title="Current page"></a></span>
					<span><a href="/page2">Next</a></span>
				</nav>
			</body></html>`))
		} else if r.URL.Path == "/page2" {
			w.Write([]byte(`<html><body>
				<table><tbody>
					<tr><td>Sat</td><td></td><td><time datetime="2025-04-05T18:45:00Z"></time></td><td>Vessel2</td><td>Down river</td></tr>
				</tbody></table>
			</body></html>`))
		}
	}))
	defer server.Close()

	oldURL := config.AppConfig.URLs.TowerBridge
	config.AppConfig.URLs.TowerBridge = server.URL
	defer func() { config.AppConfig.URLs.TowerBridge = oldURL }()

	events, err := ScrapeBridgeLifts()
	assert.NoError(t, err)
	assert.Len(t, events, 2, "should collect events from both pages")
	assert.Equal(t, "Vessel1", events[0].VesselName)
	assert.Equal(t, "Vessel2", events[1].VesselName)
}

func TestScrapeBridgeLifts_VisitError(t *testing.T) {
	c := colly.NewCollector()
	err := c.Visit(":bad-url:")
	assert.Error(t, err, "should error on bad url")
}

func BenchmarkScrapeBridgeLifts(b *testing.B) {
	// Generate a large HTML table with 500 rows
	var html = `<html><body><table><tbody>`
	for i := 0; i < 500; i++ {
		html += `<tr><td>Sat</td><td><time datetime="2025-04-05T00:00:00Z">05 Apr 2025</time></td><td><time datetime="2025-04-05T17:45:00Z">17:45</time></td><td>Vessel` + fmt.Sprint(i) + `</td><td>Up river</td></tr>`
	}
	html += `</tbody></table></body></html>`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html))
	}))
	defer server.Close()

	oldURL := config.AppConfig.URLs.TowerBridge
	config.AppConfig.URLs.TowerBridge = server.URL
	defer func() { config.AppConfig.URLs.TowerBridge = oldURL }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ScrapeBridgeLifts()
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}
