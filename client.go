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

// WithHTTPTimeout sets a custom HTTP client for a client.
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
