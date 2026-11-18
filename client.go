/*
Package dhook provides a client for sending messages to Discord webhooks.

The client was specifically designed to allow sending a high volume of messages
without being rate limited by the Discord API (i.e. 429 response).
This is achieved by respecting all known rate limits when a request is sent to Discord:
  - Global rate limit: The global rate limit as specified in the official API documentation
  - Per-route rate limit: A dynamic rate limit given in the response headers
  - Webhook rate limit: An undocumented rate limit specific to webhooks

Should the client still get rate limited it will block further requests to Discord
for the time the rate limit is in effect to prevent further escalation.
*/
package dhook

import (
	"log/slog"
	"net/http"
	"regexp"
	"sync"
	"time"
)

const (
	globalRateLimitPeriodDefault    = 1 * time.Second
	globalRateLimitRequestsDefault  = 50
	httpTimeoutDefault              = 30 * time.Second
	webhookRateLimitPeriodDefault   = 60 * time.Second
	webhookRateLimitRequestsDefault = 30
)

var webhookIDRE = regexp.MustCompile(`/webhooks/(\d+)/`)

// Logger represents an interface for implementing a logger similar to slog.
type Logger interface {
	Debug(msg string, args ...any)
	Error(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
}

type (
	// Client represents a shared client used by all webhooks to access the Discord API.
	// This enables sharing the HTTP client and the global rate limit among all webhooks.
	Client struct {
		globalRateLimitPeriod    time.Duration
		globalRateLimitRequests  int
		httpClient               *http.Client
		httpTimeout              time.Duration
		limiterGlobal            *limiter
		logger                   Logger
		rl                       rateLimited
		webhookRateLimitPeriod   time.Duration
		webhookRateLimitRequests int

		mu     sync.Mutex               // protects states
		states map[string]*webhookState // rate limit state by webhook key
	}

	// ClientOption represents an option for configuring a [Client].
	ClientOption func(*Client)
)

// WithHTTPClient sets a custom HTTP client for a client.
func WithHTTPClient(httpClient *http.Client) ClientOption {
	if httpClient == nil {
		panic("must provide an HTTP client")
	}
	return func(s *Client) {
		s.httpClient = httpClient
	}
}

// WithHTTPTimeout sets a custom timeout for each request to the Discord API.
// The default is 30 seconds.
func WithHTTPTimeout(timeout time.Duration) ClientOption {
	if timeout <= 0 {
		panic("timeout must be positive")
	}
	return func(s *Client) {
		s.httpTimeout = timeout
	}
}

// WithLogger sets a custom logger for a client.
func WithLogger(logger Logger) ClientOption {
	if logger == nil {
		panic("must provide a logger")
	}
	return func(s *Client) {
		s.logger = logger
	}
}

// WithGlobalRateLimit sets a custom global rate limit for a client.
// The rate limit is given by the maximum number of allowed requests per period.
func WithGlobalRateLimit(requests int, period time.Duration) ClientOption {
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
func WithWebhookRateLimit(requests int, period time.Duration) ClientOption {
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

// NewClient returns a new [Client] with defaults.
// The default client uses [http.DefaultClient] as HTTP client,
// a HTTP timeout of 30 seconds and [slog.Default] as logger.
//
// The client can be optionally configured through options,
// for example with [WithHTTPClient].
func NewClient(opts ...ClientOption) *Client {
	client := &Client{
		globalRateLimitPeriod:    globalRateLimitPeriodDefault,
		globalRateLimitRequests:  globalRateLimitRequestsDefault,
		httpClient:               http.DefaultClient,
		httpTimeout:              httpTimeoutDefault,
		logger:                   slog.Default(),
		webhookRateLimitPeriod:   webhookRateLimitPeriodDefault,
		webhookRateLimitRequests: webhookRateLimitRequestsDefault,
		states:                   make(map[string]*webhookState),
	}
	for _, opt := range opts {
		opt(client)
	}
	client.limiterGlobal = newLimiter(
		client.globalRateLimitRequests,
		client.globalRateLimitPeriod,
		"global",
		client.logger,
	)
	return client
}

// NewWebhook returns a webhook for url.
// Webhooks for the same Discord webhook share their rate limits.
// Rate limit state is kept for the lifetime of the client.
func (c *Client) NewWebhook(url string) *Webhook {
	if c.limiterGlobal == nil {
		panic("can not use uninitialized Client")
	}
	return &Webhook{client: c, url: url, state: c.webhookState(url)}
}

// webhookState returns the shared rate limit state for the Discord webhook of url.
func (c *Client) webhookState(url string) *webhookState {
	key := webhookKey(url)
	c.mu.Lock()
	defer c.mu.Unlock()
	if st, ok := c.states[key]; ok {
		return st
	}
	st := &webhookState{
		sem: make(chan struct{}, 1),
		limiterWebhook: newLimiter(
			c.webhookRateLimitRequests,
			c.webhookRateLimitPeriod,
			"webhook",
			c.logger,
		),
	}
	st.limiterAPI.logger = c.logger
	c.states[key] = st
	return st
}

// webhookKey returns the ID of the Discord webhook in url,
// or url itself if it is not a Discord webhook URL.
func webhookKey(url string) string {
	if m := webhookIDRE.FindStringSubmatch(url); m != nil {
		return m[1]
	}
	return url
}
