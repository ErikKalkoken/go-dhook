/*
Package dhook provides a client for sending messages to Discord webhooks.

The client was specifically designed to allow sending a high volume of messages
without being rate limited by the Discord API (i.e. 429 response).

The client achieved this by always respecting the following three rate limits
when a request is sent to Discord:
  - Global rate limit: The global rate limit as specified in the official API documentation
  - Per-route rate limit: A dynamic rate limit taken given in the response header
  - Webhook rate limit: An undocumented rate limit specific to webhooks

Should the client still become rate limited it will block further requests to Discord
for the time the rate limit is in effect to prevent further escalation.

# Example

The following shows how to use the library for sending a message to a Discord Webhook.

	package main

	import (
		"net/http"

		"github.com/ErikKalkoken/go-dhook"
	)

	func main() {
		c := dhook.NewClient()
		wh := c.NewWebhook(c, WEBHOOK_URL) // !! Please replace with a valid URL
		err := wh.Execute(dhook.Message{Content: "Hello"})
		if err != nil {
			panic(err)
		}
	}
*/
package dhook

import (
	"log/slog"
	"net/http"
	"time"
)

const (
	globalRateLimitPeriodDefault    = 1 * time.Second
	globalRateLimitRequestsDefault  = 50
	httpTimeoutDefault              = 30 * time.Second
	webhookRateLimitPeriodDefault   = 60 * time.Second
	webhookRateLimitRequestsDefault = 30
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
	globalRateLimitPeriod    time.Duration
	globalRateLimitRequests  int
	httpClient               *http.Client
	httpTimeout              time.Duration
	limiterGlobal            *limiter
	logger                   Logger
	rl                       rateLimited
	webhookRateLimitPeriod   time.Duration
	webhookRateLimitRequests int
}

// NewClient returns a new [Client] with defaults.
// The default client uses [http.DefaultClient] as HTTP client,
// a HTTP timeout of 30 seconds and [slog.Default] as logger.
//
// The client can be optionally configured through options,
// for example with [WithHTTPClient].
func NewClient(options ...func(*Client)) *Client {
	client := &Client{
		globalRateLimitPeriod:    globalRateLimitPeriodDefault,
		globalRateLimitRequests:  globalRateLimitRequestsDefault,
		httpClient:               http.DefaultClient,
		httpTimeout:              httpTimeoutDefault,
		logger:                   slog.Default(),
		webhookRateLimitPeriod:   webhookRateLimitPeriodDefault,
		webhookRateLimitRequests: webhookRateLimitRequestsDefault,
	}
	for _, o := range options {
		o(client)
	}
	client.limiterGlobal = newLimiter(
		client.globalRateLimitRequests,
		client.globalRateLimitPeriod,
		"global",
		client.logger,
	)
	return client
}

// WithHTTPClient sets a custom HTTP client for a client.
func WithHTTPClient(httpClient *http.Client) func(*Client) {
	if httpClient == nil {
		panic("must provide an HTTP client")
	}
	return func(s *Client) {
		s.httpClient = httpClient
	}
}

// WithHTTPClient sets a custom HTTP client for a client.
func WithHTTPTimeout(timeout time.Duration) func(*Client) {
	if timeout <= 0 {
		panic("timeout must be positive")
	}
	return func(s *Client) {
		s.httpTimeout = timeout
	}
}

// WithLogger sets a custom logger for a client.
func WithLogger(logger Logger) func(*Client) {
	if logger == nil {
		panic("must provide a logger")
	}
	return func(s *Client) {
		s.logger = logger
	}
}

// WithGlobalRateLimit sets a custom global rate limit for a client.
// The rate limit is given by the maximum number of allowed requests per period.
func WithGlobalRateLimit(requests int, period time.Duration) func(*Client) {
	if period <= 0 {
		panic("invalid period")
	}
	if requests <= 0 {
		panic("invalid requests")
	}
	return func(s *Client) {
		s.globalRateLimitRequests = requests
		s.globalRateLimitPeriod = period
	}
}

// WithWebhookRateLimit sets a custom webhook rate limit for a client.
// The rate limit is given by the maximum number of allowed requests per period.
func WithWebhookRateLimit(requests int, period time.Duration) func(*Client) {
	if period <= 0 {
		panic("invalid period")
	}
	if requests <= 0 {
		panic("invalid requests")
	}
	return func(s *Client) {
		s.webhookRateLimitRequests = requests
		s.webhookRateLimitPeriod = period
	}
}

// NewWebhook returns a new webhook for a client.
func (c *Client) NewWebhook(url string) *Webhook {
	if c.limiterGlobal == nil {
		panic("can not use uninitialized Client")
	}
	wh := &Webhook{
		client: c,
		url:    url,
		limiterWebhook: newLimiter(
			c.webhookRateLimitRequests,
			c.webhookRateLimitPeriod,
			"webhook",
			c.logger,
		),
	}
	wh.limiterAPI.logger = c.logger
	return wh
}
