package dhook

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebhook(t *testing.T) {
	t.Run("should abort when rateLimitExceeded and not yet reset", func(t *testing.T) {
		c := NewClient()
		wh := c.NewWebhook("url")
		wh.state.rl.set(60 * time.Second)
		_, err := wh.Execute(Message{Content: "content"}, nil)
		err2, _ := err.(TooManyRequestsError)
		assert.False(t, err2.Global)
	})
	t.Run("should abort when rateLimitExceeded and not yet reset", func(t *testing.T) {
		c := NewClient()
		c.rl.set(60 * time.Second)
		wh := c.NewWebhook("url")
		_, err := wh.Execute(Message{Content: "content"}, nil)
		err2, _ := err.(TooManyRequestsError)
		assert.True(t, err2.Global)
	})
	t.Run("should abort when global rate limit occurs while waiting", func(t *testing.T) {
		c := NewClient()
		wh := c.NewWebhook("url")
		require.NoError(t, wh.state.lock(t.Context())) // let Execute wait for the lock
		errC := make(chan error)
		go func() {
			_, err := wh.Execute(Message{Content: "content"}, nil)
			errC <- err
		}()
		time.Sleep(50 * time.Millisecond)
		c.rl.set(60 * time.Second)
		wh.state.unlock()
		var err2 TooManyRequestsError
		if assert.ErrorAs(t, <-errC, &err2) {
			assert.True(t, err2.Global)
		}
	})
	t.Run("should return error when context deadline is exceeded while waiting for the lock", func(t *testing.T) {
		c := NewClient()
		wh := c.NewWebhook("url")
		require.NoError(t, wh.state.lock(t.Context())) // simulate another request in progress
		defer wh.state.unlock()
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, err := wh.ExecuteContext(ctx, Message{Content: "content"}, nil)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Less(t, time.Since(start), 500*time.Millisecond)
	})
}

func TestParseTooManyRequests_RetryAfter(t *testing.T) {
	cases := []struct {
		name    string
		header  string
		want    time.Duration
		wantErr bool
	}{
		{"valid", "3", 3 * time.Second, false},
		{"huge", "99999999999", retryAfterMax, false},
		{"out of range", "99999999999999999999", retryAfterMax, false},
		{"negative out of range", "-99999999999999999999", retryAfterTooManyRequestDefault, true},
		{"zero", "0", retryAfterTooManyRequestDefault, true},
		{"negative", "-5", retryAfterTooManyRequestDefault, true},
		{"invalid", "invalid", retryAfterTooManyRequestDefault, true},
		{"missing", "", retryAfterTooManyRequestDefault, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.header != "" {
				h.Set("Retry-After", tc.header)
			}
			got, _, err := parseTooManyRequests(h)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseTooManyRequests_RetryAfterDate(t *testing.T) {
	ts := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	past := time.Now().Add(-time.Hour)
	httpDate := func(t time.Time) string {
		return t.UTC().Format(http.TimeFormat)
	}
	cases := []struct {
		name       string
		date       string
		retryAfter string
		want       time.Duration
		wantErr    bool
	}{
		{"with Date header", httpDate(ts), httpDate(ts.Add(10 * time.Second)), 10 * time.Second, false},
		{"clock skew", httpDate(past), httpDate(past.Add(10 * time.Second)), 10 * time.Second, false},
		{"asctime format", httpDate(ts), ts.Add(10 * time.Second).Format(time.ANSIC), 10 * time.Second, false},
		{"far future", httpDate(ts), httpDate(ts.Add(48 * time.Hour)), retryAfterMax, false},
		{"before Date", httpDate(ts), httpDate(ts.Add(-10 * time.Second)), retryAfterTooManyRequestDefault, true},
		{"equal to Date", httpDate(ts), httpDate(ts), retryAfterTooManyRequestDefault, true},
		{"no Date header and past", "", httpDate(time.Now().Add(-time.Minute)), retryAfterTooManyRequestDefault, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			h.Set("Retry-After", tc.retryAfter)
			if tc.date != "" {
				h.Set("Date", tc.date)
			}
			got, _, err := parseTooManyRequests(h)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
	// Without a valid Date header the local clock is used,
	// which can make the result up to one second shorter due to HTTP dates having seconds resolution.
	for _, date := range []string{"", "invalid"} {
		t.Run(fmt.Sprintf("fallback to local clock with Date %q", date), func(t *testing.T) {
			h := http.Header{}
			h.Set("Retry-After", httpDate(time.Now().Add(10*time.Second)))
			if date != "" {
				h.Set("Date", date)
			}
			got, _, err := parseTooManyRequests(h)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, got, 9*time.Second)
			assert.LessOrEqual(t, got, 10*time.Second)
		})
	}
}

func TestParseTooManyRequests_Global(t *testing.T) {
	cases := []struct {
		name   string
		header map[string]string
		want   bool
	}{
		{"scope user", map[string]string{"X-RateLimit-Scope": "user"}, false},
		{"scope shared", map[string]string{"X-RateLimit-Scope": "shared"}, false},
		{"scope global", map[string]string{"X-RateLimit-Scope": "global"}, true},
		{"global true", map[string]string{"X-RateLimit-Global": "true"}, true},
		{"global false", map[string]string{"X-RateLimit-Global": "false"}, false},
		{"bucket only", map[string]string{"X-RateLimit-Bucket": "abcd1234"}, false},
		{"no Discord header", map[string]string{}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			h.Set("Retry-After", "3")
			for k, v := range tc.header {
				h.Set(k, v)
			}
			_, got, err := parseTooManyRequests(h)
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRedactURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want string
	}{
		{"plain", "https://discord.com/api/webhooks/123/token", "https://discord.com/api/webhooks/123/REDACTED"},
		{"API version", "https://discord.com/api/v10/webhooks/123/token", "https://discord.com/api/v10/webhooks/123/REDACTED"},
		{"trailing slash", "https://discord.com/api/webhooks/123/token/", "https://discord.com/api/webhooks/123/REDACTED/"},
		{"query", "https://discord.com/api/webhooks/123/token?thread_id=1&wait=true", "https://discord.com/api/webhooks/123/REDACTED?thread_id=1&wait=true"},
		{"other URL", "http://127.0.0.1:8080/hook", "http://127.0.0.1:8080/hook"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, redactURL(tc.url))
		})
	}
}
