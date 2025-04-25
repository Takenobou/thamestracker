package httpclient

import (
	"net/http"
	"time"
)

type Client interface {
	Get(url string) (*http.Response, error)
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
