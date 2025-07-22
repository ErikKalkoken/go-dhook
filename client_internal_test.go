package dhook

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		c := NewClient()
		assert.Equal(t, http.DefaultClient, c.httpClient)
		assert.Zero(t, c.httpTimeout)
	})
	t.Run("custom HTTP configuration", func(t *testing.T) {
		hc := &http.Client{}
		c := NewClient(WithHTTPClient(hc))
		assert.Equal(t, hc, c.httpClient)
	})
	t.Run("custom timeout configuration", func(t *testing.T) {
		c := NewClient(WithHTTPTimeout(3 * time.Second))
		assert.Equal(t, c.httpTimeout, 3*time.Second)
	})
}
