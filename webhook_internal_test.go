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
		wh.rl.set(60 * time.Second)
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
}
