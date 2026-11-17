package dhook

import (
	"log/slog"
	"net/http"
	"sync"
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

func TestWebhookKey(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{"discord.com", "https://discord.com/api/webhooks/123/token", "123"},
		{"discordapp.com", "https://discordapp.com/api/webhooks/123/token", "123"},
		{"API version", "https://discord.com/api/v10/webhooks/123/token", "123"},
		{"trailing slash", "https://discord.com/api/webhooks/123/token/", "123"},
		{"thread", "https://discord.com/api/webhooks/123/token?thread_id=1", "123"},
		{"other token", "https://discord.com/api/webhooks/123/other", "123"},
		{"other URL", "http://127.0.0.1:8080/hook", "http://127.0.0.1:8080/hook"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, webhookKey(tc.url))
		})
	}
}

func TestClient_NewWebhook_sharedState(t *testing.T) {
	t.Run("should share state for same URL", func(t *testing.T) {
		c := NewClient()
		assert.Same(t, c.NewWebhook("url").state, c.NewWebhook("url").state)
	})
	t.Run("should share state for different URLs of same Discord webhook", func(t *testing.T) {
		c := NewClient()
		wh1 := c.NewWebhook("https://discord.com/api/webhooks/123/token")
		wh2 := c.NewWebhook("https://discordapp.com/api/v10/webhooks/123/token/?thread_id=1")
		assert.Same(t, wh1.state, wh2.state)
		assert.NotEqual(t, wh1.url, wh2.url)
	})
	t.Run("should not share state between different Discord webhooks", func(t *testing.T) {
		c := NewClient()
		wh1 := c.NewWebhook("https://discord.com/api/webhooks/123/token")
		wh2 := c.NewWebhook("https://discord.com/api/webhooks/456/token")
		assert.NotSame(t, wh1.state, wh2.state)
	})
	t.Run("should not share state between clients", func(t *testing.T) {
		c1 := NewClient()
		c2 := NewClient()
		assert.NotSame(t, c1.NewWebhook("url").state, c2.NewWebhook("url").state)
	})
	t.Run("should share state when called concurrently", func(t *testing.T) {
		c := NewClient()
		const n = 50
		got := make([]*webhookState, n)
		var wg sync.WaitGroup
		for i := range n {
			wg.Add(1)
			go func() {
				defer wg.Done()
				got[i] = c.NewWebhook("url").state
			}()
		}
		wg.Wait()
		for _, st := range got {
			assert.Same(t, got[0], st)
		}
	})
}
