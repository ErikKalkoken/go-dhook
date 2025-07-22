package dhook_test

import (
	"testing"
	"time"

	"github.com/ErikKalkoken/go-dhook"
	"github.com/stretchr/testify/assert"
)

func TestWithTimeout(t *testing.T) {
	assert.Panics(t, func() {
		dhook.WithHTTPTimeout(0)
	})
	assert.Panics(t, func() {
		dhook.WithHTTPTimeout(-1 * time.Second)
	})
}

func TestClient_NewWebhook(t *testing.T) {
	c := &dhook.Client{}
	assert.Panics(t, func() {
		c.NewWebhook("abc")
	})

}
