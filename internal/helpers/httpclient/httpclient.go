package httpclient

import (
	"fmt"
	"net/http"
	"time"

	"github.com/sony/gobreaker"
)

type Client interface {
	Get(url string) (*http.Response, error)
}

// ClientFunc adapter for using ordinary functions as Client.
type ClientFunc func(url string) (*http.Response, error)

// Get calls f(url).
func (f ClientFunc) Get(url string) (*http.Response, error) {
	return f(url)
}

var DefaultClient Client = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
	},
}

// NewBreakerClient wraps an existing Client with a circuit breaker.
func NewBreakerClient(inner Client, maxFailures int, coolOffSeconds int) Client {
	cb := gobreaker.NewCircuitBreaker(gobreaker.Settings{
		Name:    "http_get_breaker",
		Timeout: time.Duration(coolOffSeconds) * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= uint32(maxFailures)
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			// no-op or log state changes
		},
	})
	return &cbClient{client: inner, breaker: cb}
}

// cbClient implements Client by wrapping Get calls in a circuit breaker.
type cbClient struct {
	client  Client
	breaker *gobreaker.CircuitBreaker
}

// Get invokes the inner Client.Get within the circuit breaker. Returns an error if the breaker is open.
func (c *cbClient) Get(url string) (*http.Response, error) {
	result, err := c.breaker.Execute(func() (interface{}, error) {
		resp, err := c.client.Get(url)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= http.StatusInternalServerError {
			resp.Body.Close()
			return nil, fmt.Errorf("server error %d", resp.StatusCode)
		}
		return resp, nil
	})
	if err != nil {
		return nil, err
	}
	return result.(*http.Response), nil
}
