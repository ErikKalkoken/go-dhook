package dhook

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
		wh.state.mu.Lock() // let Execute wait for the lock
		errC := make(chan error)
		go func() {
			_, err := wh.Execute(Message{Content: "content"}, nil)
			errC <- err
		}()
		time.Sleep(50 * time.Millisecond)
		c.rl.set(60 * time.Second)
		wh.state.mu.Unlock()
		var err2 TooManyRequestsError
		if assert.ErrorAs(t, <-errC, &err2) {
			assert.True(t, err2.Global)
		}
	})
}

func TestParseRetryAfter(t *testing.T) {
	cases := []struct {
		name    string
		header  string
		body    float64
		want    time.Duration
		wantErr bool
	}{
		{"header", "3", 2.57, 3 * time.Second, false},
		{"header without body", "3", 0, 3 * time.Second, false},
		{"header smaller than body", "0", 1.5, 2 * time.Second, false},
		{"header larger than body", "3", 1.2, 3 * time.Second, false},
		{"negative header", "-5", 0, retryAfterTooManyRequestDefault, true},
		{"zero header without body", "0", 0, retryAfterTooManyRequestDefault, true},
		{"negative header with body", "-5", 1.2, 2 * time.Second, false},
		{"huge header", "99999999999", 0, retryAfterMax, false},
		{"out of range header with body", "99999999999999999999", 1.2, 2 * time.Second, false},
		{"huge body", "", 1e10, retryAfterMax, false},
		{"invalid header with body", "invalid", 1.2, 2 * time.Second, false},
		{"missing header with body", "", 0.5, 1 * time.Second, false},
		{"missing header with whole body", "", 2, 2 * time.Second, false},
		{"invalid header without body", "invalid", 0, retryAfterTooManyRequestDefault, true},
		{"missing header without body", "", 0, retryAfterTooManyRequestDefault, true},
		{"missing header with negative body", "", -1, retryAfterTooManyRequestDefault, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.Header{}
			if tc.header != "" {
				h.Set("Retry-After", tc.header)
			}
			got, err := parseRetryAfter(h, tooManyRequestsResponse{RetryAfter: tc.body})
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
		})
	}
}
