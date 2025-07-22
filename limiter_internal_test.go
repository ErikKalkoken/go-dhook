package dhook

import (
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLimiter(t *testing.T) {
	t.Run("should allow first 10 calls without delay, but delay the 11st call to successive period", func(t *testing.T) {
		log := make([]time.Time, 11)
		l := newLimiter(100*time.Millisecond, 10, "")
		start := time.Now()
		for i := 0; i < 11; i++ {
			l.wait()
			log[i] = time.Now()
		}
		assert.WithinDuration(t, start, log[9], 1*time.Millisecond)
		assert.WithinDuration(t, start.Add(100*time.Millisecond), log[10], 10*time.Millisecond)
	})
	t.Run("should work concurrently", func(t *testing.T) {
		log := make([]time.Time, 11)
		l := newLimiter(100*time.Millisecond, 10, "")
		start := time.Now()
		var wg sync.WaitGroup
		for i := 0; i < 11; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				l.wait()
				log[i] = time.Now()
			}()
		}
		wg.Wait()
		slices.SortFunc(log, func(a, b time.Time) int {
			return a.Compare(b)
		})
		assert.WithinDuration(t, start, log[9], 1*time.Millisecond)
		assert.WithinDuration(t, start.Add(100*time.Millisecond), log[10], 10*time.Millisecond)
	})
}

func TestRoundUpDuration(t *testing.T) {
	t.Run("should round up small fraction", func(t *testing.T) {
		x := roundUpDuration(1*time.Second+100*time.Millisecond, time.Second)
		assert.Equal(t, 2*time.Second, x)
	})
	t.Run("should round up large fraction", func(t *testing.T) {
		x := roundUpDuration(1*time.Second+900*time.Millisecond, time.Second)
		assert.Equal(t, 2*time.Second, x)
	})
	t.Run("should not round when no fraction", func(t *testing.T) {
		x := roundUpDuration(1*time.Second, time.Second)
		assert.Equal(t, 1*time.Second, x)
	})
}
