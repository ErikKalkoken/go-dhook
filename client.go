package dhook

import (
	"net/http"
	"time"
)

const (
	globalRateLimitPeriod   = 1 * time.Second
	globalRateLimitRequests = 50
	httpTimeoutDefault      = 30 * time.Second
)

// Client represents a shared client used by all webhooks to access the Discord API.
//
// The shared client enabled dealing with the global rate limit and ensures a shared http client is used.
type Client struct {
	httpClient    *http.Client
	httpTimeout   time.Duration
	limiterGlobal *limiter // Discord's global rate limit

	rl rateLimited
}

// NewClient returns a new [Client]
//
// A client can be configured with option functions (e.g. [WithHTTPClient]).
//
// When no options are provided it returns a default client.
// The default client uses [http.DefaultClient] as HTTP client and a timeout of 30 seconds.
func NewClient(options ...func(*Client)) *Client {
	c := &Client{
		limiterGlobal: newLimiter(globalRateLimitPeriod, globalRateLimitRequests, "global"),
		httpClient:    http.DefaultClient,
		httpTimeout:   httpTimeoutDefault,
	}
	for _, o := range options {
		o(c)
	}
	return c
}

// WithHTTPClient sets a custom HTTP client to be used by all webhooks.
func WithHTTPClient(httpClient *http.Client) func(*Client) {
	return func(s *Client) {
		s.httpClient = httpClient
	}
}

// WithHTTPTimeout sets a timeout to be used by all HTTP requests.
func WithHTTPTimeout(timeout time.Duration) func(*Client) {
	if timeout <= 0 {
		panic("timeout must have a positive value")
	}
	return func(s *Client) {
		s.httpTimeout = timeout
	}
}

// NewWebhook returns a new webhook for a client.
func (c *Client) NewWebhook(url string) *Webhook {
	if c.limiterGlobal == nil {
		panic("can not use uninitialized Client")
	}
	wh := &Webhook{
		client:         c,
		url:            url,
		limiterWebhook: newLimiter(webhookRateLimitPeriod, webhookRateLimitRequests, "webhook"),
	}
	return wh
}
