package service_test

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/Takenobou/thamestracker/internal/helpers/logger"
	"github.com/Takenobou/thamestracker/internal/models"
	service "github.com/Takenobou/thamestracker/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	logger.InitLogger()
	os.Exit(m.Run())
}

// --- Fakes for dependency injection ---
type fakeCache struct {
	store   map[string]interface{}
	failSet bool
	failGet bool
}

func newFakeCache() *fakeCache {
	return &fakeCache{store: make(map[string]interface{})}
}
func (f *fakeCache) Set(key string, value interface{}, ttl time.Duration) error {
	if f.failSet {
		return errors.New("fail set")
	}
	f.store[key] = value
	return nil
}
func (f *fakeCache) Get(key string, dest interface{}) error {
	if f.failGet {
		return errors.New("fail get")
	}
	v, ok := f.store[key]
	if !ok {
		return errors.New("cache miss")
	}
	// shallow copy for test
	switch d := dest.(type) {
	case *[]models.Event:
		*d = v.([]models.Event)
	}
	return nil
}

type fakeBridgeScraper struct {
	result []models.Event
	err    error
	called *bool
}

func (f *fakeBridgeScraper) ScrapeBridgeLifts() ([]models.Event, error) {
	if f.called != nil {
		*f.called = true
	}
	return f.result, f.err
}

type fakeVesselScraper struct {
	result map[string][]models.Event
	err    error
}

func (f *fakeVesselScraper) ScrapeVessels(vesselType string) ([]models.Event, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.result[vesselType], nil
}

func TestGetBridgeLifts_CacheHit(t *testing.T) {
	cache := newFakeCache()
	cache.Set("bridge_lifts", []models.Event{{VesselName: "Cached"}}, 0)
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{})
	res, err := svc.GetBridgeLifts()
	assert.NoError(t, err)
	assert.Equal(t, "Cached", res[0].VesselName)
}

func TestGetBridgeLifts_CacheMissAndScraper(t *testing.T) {
	cache := newFakeCache()
	called := false
	svc := service.NewService(nil, cache, &fakeBridgeScraper{
		result: []models.Event{{VesselName: "Scraped"}},
		called: &called,
	}, &fakeVesselScraper{})
	res, err := svc.GetBridgeLifts()
	assert.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "Scraped", res[0].VesselName)
}

func TestGetBridgeLifts_ScraperError(t *testing.T) {
	cache := newFakeCache()
	svc := service.NewService(nil, cache, &fakeBridgeScraper{err: errors.New("fail")}, &fakeVesselScraper{})
	_, err := svc.GetBridgeLifts()
	assert.Error(t, err)
}

func TestGetBridgeLifts_CacheSetError(t *testing.T) {
	cache := newFakeCache()
	cache.failSet = true
	svc := service.NewService(nil, cache, &fakeBridgeScraper{result: []models.Event{{VesselName: "Scraped"}}}, &fakeVesselScraper{})
	_, err := svc.GetBridgeLifts()
	assert.Error(t, err)
}

func TestGetVessels_AllTypes(t *testing.T) {
	cache := newFakeCache()
	results := map[string][]models.Event{
		"inport":     {{VesselName: "inport"}},
		"arrivals":   {{VesselName: "arrivals"}},
		"departures": {{VesselName: "departures"}},
		"forecast":   {{VesselName: "forecast"}},
		"all":        {{VesselName: "all"}},
	}
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{result: results})
	for _, typ := range []string{"inport", "arrivals", "departures", "forecast", "all"} {
		res, err := svc.GetVessels(typ)
		assert.NoError(t, err)
		assert.Equal(t, typ, res[0].VesselName)
	}
}

func TestGetVessels_InvalidType(t *testing.T) {
	cache := newFakeCache()
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{})
	_, err := svc.GetVessels("badtype")
	assert.Error(t, err)
}

func TestGetVessels_CacheHit(t *testing.T) {
	cache := newFakeCache()
	cache.Set("v3_vessels_inport", []models.Event{{VesselName: "Cached"}}, 0)
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{})
	res, err := svc.GetVessels("inport")
	assert.NoError(t, err)
	assert.Equal(t, "Cached", res[0].VesselName)
}

func TestGetVessels_ScraperError(t *testing.T) {
	cache := newFakeCache()
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{err: errors.New("fail")})
	_, err := svc.GetVessels("inport")
	assert.Error(t, err)
}

func TestGetVessels_CacheSetError(t *testing.T) {
	cache := newFakeCache()
	cache.failSet = true
	results := map[string][]models.Event{"inport": {{VesselName: "Scraped"}}}
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{result: results})
	_, err := svc.GetVessels("inport")
	assert.Error(t, err)
}

func TestGetFilteredVessels_NoLocation(t *testing.T) {
	cache := newFakeCache()
	results := map[string][]models.Event{"inport": {{VesselName: "A"}}}
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{result: results})
	res, err := svc.GetFilteredVessels("inport", "")
	assert.NoError(t, err)
	assert.Equal(t, "A", res[0].VesselName)
}

func TestGetFilteredVessels_CacheHit(t *testing.T) {
	cache := newFakeCache()
	cache.Set("v3_vessels_inport_location_L1", []models.Event{{VesselName: "Cached", Location: "L1"}}, 0)
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{})
	res, err := svc.GetFilteredVessels("inport", "L1")
	assert.NoError(t, err)
	assert.Equal(t, "Cached", res[0].VesselName)
}

func TestGetFilteredVessels_CacheMissAndFilter(t *testing.T) {
	cache := newFakeCache()
	results := map[string][]models.Event{"inport": {{VesselName: "A", Location: "L1"}, {VesselName: "B", Location: "L2"}}}
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{result: results})
	res, err := svc.GetFilteredVessels("inport", "L1")
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assert.Equal(t, "L1", res[0].Location)
}

func TestListLocations(t *testing.T) {
	cache := newFakeCache()
	results := map[string][]models.Event{"all": {
		{VesselName: "A", Category: "inport", Location: "Port1"},
		{VesselName: "B", Category: "arrivals", To: "Port1"},
		{VesselName: "C", Category: "departures", From: "Port2"},
		{VesselName: "D", Category: "forecast", To: "Port2"},
	}}
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{result: results})
	locs, err := svc.ListLocations()
	assert.NoError(t, err)
	assert.Len(t, locs, 2)
	var names []string
	for _, l := range locs {
		names = append(names, l.Name)
	}
	assert.Contains(t, names, "Port1")
	assert.Contains(t, names, "Port2")
}

func TestListLocations_Error(t *testing.T) {
	cache := newFakeCache()
	svc := service.NewService(nil, cache, &fakeBridgeScraper{}, &fakeVesselScraper{err: errors.New("fail")})
	_, err := svc.ListLocations()
	assert.Error(t, err)
}
