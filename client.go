package dhook

import (
	"log/slog"
	"net/http"
	"time"
)

const (
	globalRateLimitPeriod   = 1 * time.Second
	globalRateLimitRequests = 50
	httpTimeoutDefault      = 30 * time.Second
)

// Logger represents an interface for implementing a logger similar to slog.
type Logger interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

// Client represents a shared client used by all webhooks to access the Discord API.
//
// The shared client enabled dealing with the global rate limit and ensures a shared http client is used.
type Client struct {
	// The HTTP client used by all webhooks. Will use [http.DefaultClient] when not set.
	HTTPClient *http.Client

	// The default timeout for all HTTP requests. The default is 30 seconds.
	HTTPTimeout time.Duration

	// The logger used for all logging. Will use [slog.Default] when not set.
	Logger Logger

	limiterGlobal *limiter
	rl            rateLimited
}

// NewClient returns a new [Client] with defaults.
//
// For custom configuration the fields can be set on the returned [Client] object.
func NewClient() *Client {
	c := &Client{
		limiterGlobal: newLimiter(globalRateLimitPeriod, globalRateLimitRequests, "global", slog.Default()),
		HTTPClient:    http.DefaultClient,
		HTTPTimeout:   httpTimeoutDefault,
		Logger:        slog.Default(),
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
		limiterWebhook: newLimiter(webhookRateLimitPeriod, webhookRateLimitRequests, "webhook", c.Logger),
	}
	wh.limiterAPI.logger = c.Logger
	return wh
}
