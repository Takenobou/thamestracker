package httpclient

import (
	"net/http"
	"time"
)

type Client interface {
	Get(url string) (*http.Response, error)
}

var DefaultClient Client = &http.Client{
	Timeout: 10 * time.Second,
}
