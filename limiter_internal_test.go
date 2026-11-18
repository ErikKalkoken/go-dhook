package dhook

import (
	"context"
	"log/slog"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimiter(t *testing.T) {
	t.Run("should allow first 10 calls without delay, but delay the 11st call to successive period", func(t *testing.T) {
		log := make([]time.Time, 11)
		l := newLimiter(10, 100*time.Millisecond, "", slog.Default())
		start := time.Now()
		for i := range 11 {
			err := l.wait(t.Context())
			require.NoError(t, err)
			log[i] = time.Now()
		}
		assert.WithinDuration(t, start, log[9], 1*time.Millisecond)
		assert.WithinDuration(t, start.Add(100*time.Millisecond), log[10], 10*time.Millisecond)
	})
	t.Run("should work concurrently", func(t *testing.T) {
		log := make([]time.Time, 11)
		l := newLimiter(10, 100*time.Millisecond, "", slog.Default())
		start := time.Now()
		var wg sync.WaitGroup
		for i := range 11 {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := l.wait(t.Context())
				assert.NoError(t, err)
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
	t.Run("should return early when context is already canceled", func(t *testing.T) {
		l := newLimiter(2, time.Second, "", slog.Default())
		entries := slices.Clone(l.entries)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		err := l.wait(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, entries, l.entries)
		assert.Equal(t, 0, l.index)
	})
	t.Run("should return error when context is canceled while waiting", func(t *testing.T) {
		l := newLimiter(2, time.Second, "", slog.Default())
		require.NoError(t, l.wait(t.Context()))
		require.NoError(t, l.wait(t.Context()))
		entries := slices.Clone(l.entries)
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()
		start := time.Now()
		err := l.wait(ctx)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Less(t, time.Since(start), 500*time.Millisecond)
		assert.Equal(t, entries, l.entries)
		assert.Equal(t, 0, l.index)
	})
	t.Run("should return error when context is canceled while waiting for the lock", func(t *testing.T) {
		l := newLimiter(2, time.Second, "", slog.Default())
		entries := slices.Clone(l.entries)
		ctx, cancel := context.WithCancel(t.Context())
		l.mu.Lock() // let wait block on the lock
		errC := make(chan error)
		go func() {
			errC <- l.wait(ctx)
		}()
		time.Sleep(20 * time.Millisecond) // goroutine passed the pre-check and is blocked on the lock
		cancel()
		l.mu.Unlock()
		assert.ErrorIs(t, <-errC, context.Canceled)
		assert.Equal(t, entries, l.entries)
		assert.Equal(t, 0, l.index)
	})
	t.Run("should return error when context deadline is exceeded while waiting", func(t *testing.T) {
		l := newLimiter(1, time.Second, "", slog.Default())
		err := l.wait(t.Context())
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		start := time.Now()
		err = l.wait(ctx)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Less(t, time.Since(start), 500*time.Millisecond)
	})
}

func TestSleepContext(t *testing.T) {
	t.Run("should sleep for the given duration", func(t *testing.T) {
		start := time.Now()
		err := sleepContext(t.Context(), 50*time.Millisecond)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, time.Since(start), 50*time.Millisecond)
	})
	t.Run("should return early when context is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()
		start := time.Now()
		err := sleepContext(ctx, time.Second)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Less(t, time.Since(start), 500*time.Millisecond)
	})
	t.Run("should return immediately when context is already canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		start := time.Now()
		err := sleepContext(ctx, time.Second)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Less(t, time.Since(start), 100*time.Millisecond)
	})
}

func TestCheckDeadline(t *testing.T) {
	t.Run("should return error when deadline is before end of wait", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		err := checkDeadline(ctx, 5*time.Second)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
	t.Run("should return nil when deadline is after end of wait", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		err := checkDeadline(ctx, 50*time.Millisecond)
		assert.NoError(t, err)
	})
	t.Run("should return nil when there is no deadline", func(t *testing.T) {
		err := checkDeadline(context.Background(), time.Hour)
		assert.NoError(t, err)
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
