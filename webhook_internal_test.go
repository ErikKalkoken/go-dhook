package dhook

import (
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
