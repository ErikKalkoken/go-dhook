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
	// The HTTP client used by all webhooks. Will use [http.DefaultClient] when not set.
	HTTPClient *http.Client

	// The default timeout for all HTTP requests. The default is 30 seconds.
	HTTPTimeout time.Duration

	limiterGlobal *limiter
	rl            rateLimited
}

// NewClient returns a new [Client] with defaults.
//
// For custom configuration the fields can be set on the returned [Client] object.
func NewClient() *Client {
	c := &Client{
		limiterGlobal: newLimiter(globalRateLimitPeriod, globalRateLimitRequests, "global"),
		HTTPClient:    http.DefaultClient,
		HTTPTimeout:   httpTimeoutDefault,
	}
	return c
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
