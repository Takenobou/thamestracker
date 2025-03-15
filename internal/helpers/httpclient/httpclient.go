package httpclient

import (
	"net/http"
	"time"
)

// Client defines a simple HTTP client interface.
type Client interface {
	Get(url string) (*http.Response, error)
}

// DefaultClient is the production HTTP client.
var DefaultClient Client = &http.Client{
	Timeout: 10 * time.Second,
}
