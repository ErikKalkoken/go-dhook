package dhook

import (
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MyLogger struct{}

func (l *MyLogger) Debug(msg string, args ...any) {}
func (l *MyLogger) Error(msg string, args ...any) {}
func (l *MyLogger) Info(msg string, args ...any)  {}
func (l *MyLogger) Warn(msg string, args ...any)  {}

func TestClient(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		c := NewClient()
		assert.Equal(t, http.DefaultClient, c.httpClient)
		assert.Equal(t, httpTimeoutDefault, c.httpTimeout)
		assert.Equal(t, slog.Default(), c.logger)
		assert.Equal(t, globalRateLimitPeriodDefault, c.globalRateLimitPeriod)
		assert.Equal(t, globalRateLimitRequestsDefault, c.globalRateLimitRequests)
	})
	t.Run("custom HTTP timeout", func(t *testing.T) {
		c := NewClient(WithHTTPTimeout(1 * time.Second))
		assert.Equal(t, 1*time.Second, c.httpTimeout)
	})
	t.Run("custom HTTP client", func(t *testing.T) {
		hc := &http.Client{}
		c := NewClient(WithHTTPClient(hc))
		assert.Equal(t, hc, c.httpClient)
	})
	t.Run("custom logger", func(t *testing.T) {
		l := &MyLogger{}
		c := NewClient(WithLogger(l))
		assert.Equal(t, l, c.logger)
	})
	t.Run("custom global rate limit", func(t *testing.T) {
		c := NewClient(WithGlobalRateLimit(100, 10*time.Second))
		assert.Equal(t, 10*time.Second, c.globalRateLimitPeriod)
		assert.Equal(t, 100, c.globalRateLimitRequests)
	})
	t.Run("custom webhook rate limit", func(t *testing.T) {
		c := NewClient(WithWebhookRateLimit(100, 10*time.Second))
		assert.Equal(t, 10*time.Second, c.webhookRateLimitPeriod)
		assert.Equal(t, 100, c.webhookRateLimitRequests)
	})
}
