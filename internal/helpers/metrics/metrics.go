package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ScrapeCounter counts external API scrapes by api name.
	ScrapeCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "thamestracker_external_api_scrapes_total",
			Help: "Total number of external API scrapes.",
		},
		[]string{"api"},
	)
	// ScrapeDuration tracks duration of external API scrapes by api name.
	ScrapeDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "thamestracker_external_api_duration_seconds",
			Help:    "Duration of external API scrapes in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"api"},
	)
	// CacheHits counts cache hits.
	CacheHits = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "thamestracker_cache_hits_total",
			Help: "Total number of cache hits.",
		},
	)
	// CacheMisses counts cache misses.
	CacheMisses = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "thamestracker_cache_misses_total",
			Help: "Total number of cache misses.",
		},
	)
	// LocationsRequests counts GET /locations calls.
	LocationsRequests = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "thamestracker_locations_requests_total",
			Help: "Total number of /locations API requests.",
		},
	)
	// LocationsRequestDuration tracks duration of GET /locations.
	LocationsRequestDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "thamestracker_locations_request_duration_seconds",
			Help:    "Duration of /locations request handling in seconds.",
			Buckets: prometheus.DefBuckets,
		},
	)
	// RedisErrorsTotal counts Redis errors.
	RedisErrorsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "thamestracker_redis_errors_total",
			Help: "Total number of Redis errors.",
		},
	)
	// FilteredEventsTotal counts events filtered out by unique logic, labeled by category.
	FilteredEventsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "thamestracker_filtered_events_total",
			Help: "Total number of events filtered out by unique logic, labeled by category.",
		},
		[]string{"category"},
	)
)

func init() {
	prometheus.MustRegister(ScrapeCounter, ScrapeDuration, CacheHits, CacheMisses,
		LocationsRequests, LocationsRequestDuration, RedisErrorsTotal, FilteredEventsTotal)
}
