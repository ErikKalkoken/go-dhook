package dhook

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimited(t *testing.T) {
	t.Run("should set", func(t *testing.T) {
		var rl rateLimited
		rl.set(5 * time.Minute)
		ok, d := rl.getOrReset()
		assert.True(t, ok)
		now := time.Now()
		assert.WithinDuration(t, now.Add(5*time.Minute), now.Add(d), 1*time.Second)
	})
	t.Run("should return false when expired", func(t *testing.T) {
		var rl rateLimited
		rl.set(-1 * time.Second)
		ok, _ := rl.getOrReset()
		assert.False(t, ok)
	})
	t.Run("should report zero-value as not active", func(t *testing.T) {
		var rl rateLimited
		ok, _ := rl.getOrReset()
		assert.False(t, ok)
	})
}
