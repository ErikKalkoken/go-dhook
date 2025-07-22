package dhook

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		c := NewClient()
		assert.Equal(t, http.DefaultClient, c.HTTPClient)
		assert.Equal(t, httpTimeoutDefault, c.HTTPTimeout)
	})
}
