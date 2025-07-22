package dhook_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/go-dhook"
	"github.com/stretchr/testify/assert"
)

func TestWithHTTPTimeout(t *testing.T) {
	assert.Panics(t, func() {
		dhook.WithHTTPTimeout(0)
	})
	assert.Panics(t, func() {
		dhook.WithHTTPTimeout(-1 * time.Second)
	})
}

func TestWithHTTPClient(t *testing.T) {
	assert.Panics(t, func() {
		dhook.WithHTTPClient(nil)
	})
}

func TestWithLogger(t *testing.T) {
	assert.Panics(t, func() {
		dhook.WithLogger(nil)
	})
}

func TestWithGlobalRateLimit(t *testing.T) {
	assert.Panics(t, func() {
		dhook.WithGlobalRateLimit(0, 0)
	})
	assert.Panics(t, func() {
		dhook.WithGlobalRateLimit(-10, time.Second)
	})
	assert.Panics(t, func() {
		dhook.WithGlobalRateLimit(10, 0)
	})
	assert.Panics(t, func() {
		dhook.WithGlobalRateLimit(10, -time.Second)
	})
}

func TestWithWebhookRateLimit(t *testing.T) {
	assert.Panics(t, func() {
		dhook.WithWebhookRateLimit(0, 0)
	})
	assert.Panics(t, func() {
		dhook.WithWebhookRateLimit(-10, time.Second)
	})
	assert.Panics(t, func() {
		dhook.WithWebhookRateLimit(10, 0)
	})
	assert.Panics(t, func() {
		dhook.WithWebhookRateLimit(10, -time.Second)
	})
}
func TestClient_NewWebhook(t *testing.T) {
	c := &dhook.Client{}
	assert.Panics(t, func() {
		c.NewWebhook("abc")
	})
}
