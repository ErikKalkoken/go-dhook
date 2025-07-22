package dhook_test

import (
	"testing"

	"github.com/ErikKalkoken/go-dhook"
	"github.com/stretchr/testify/assert"
)

func TestClient_NewWebhook(t *testing.T) {
	c := &dhook.Client{}
	assert.Panics(t, func() {
		c.NewWebhook("abc")
	})

}
