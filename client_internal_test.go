package dhook

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		c := NewClient()
		assert.Equal(t, http.DefaultClient, c.httpClient)
	})
	t.Run("custom HTTP configuration", func(t *testing.T) {
		hc := &http.Client{}
		c := NewClient(WithHTTPClient(hc))
		assert.Equal(t, hc, c.httpClient)
	})
}
