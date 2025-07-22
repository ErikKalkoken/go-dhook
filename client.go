package dhook

import (
	"net/http"
	"time"
)

const (
	globalRateLimitPeriod   = 1 * time.Second
	globalRateLimitRequests = 50
)

// Client represents a shared client used by all webhooks to access the Discord API.
//
// The shared client enabled dealing with the global rate limit and ensures a shared http client is used.
type Client struct {
	httpClient    *http.Client
	limiterGlobal *limiter // Discord's global rate limit

	rl rateLimited
}

// NewClient returns a new [Client].
// A client can be configured with option functions (e.g. [WithHTTPClient]).
func NewClient(options ...func(*Client)) *Client {
	client := &Client{
		limiterGlobal: newLimiter(globalRateLimitPeriod, globalRateLimitRequests, "global"),
		httpClient:    http.DefaultClient,
	}
	for _, o := range options {
		o(client)
	}
	return client
}

// WithHTTPClient configures a [Client] with a custom HTTP client.
func WithHTTPClient(c *http.Client) func(*Client) {
	return func(s *Client) {
		s.httpClient = c
	}
}
