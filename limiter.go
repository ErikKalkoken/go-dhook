package dhook

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// limiter represents a rate limiter implementing the sliding log algorithm.
// This type is safe for concurrent use by multiple goroutines.
type limiter struct {
	logger Logger        // logger for rate limit messages
	max    int           // maximum number of events in any window of length period
	name   string        // name of the limiter for log messages
	period time.Duration // length of the sliding window

	mu      sync.Mutex  // protects entries and index
	entries []time.Time // ring of the times of the last max events
	index   int         // position of the oldest entry, which is replaced next
}

// newLimiter returns a new limiter object.
func newLimiter(max int, period time.Duration, name string, logger Logger) *limiter {
	l := limiter{
		index:  0,
		logger: logger,
		max:    max,
		name:   name,
		period: period,
	}
	l.entries = make([]time.Time, max)
	before := time.Now().Add(-2 * period)
	for i := range max {
		l.entries[i] = before
	}
	return &l
}

// wait will register a new event.
// In case the current tick is exhausted it will block until the tick is reset.
// The wait duration will be rounded up to the next rate tick (e.g. 100ms if the rate is 10/sec)
func (l *limiter) wait(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	// Check again, because ctx may have been canceled while waiting for the lock.
	if err := ctx.Err(); err != nil {
		return err
	}
	last := l.entries[l.index]
	next := last.Add(l.period)
	if now := time.Now(); now.Before(next) {
		d := roundUpDuration(next.Sub(now), l.period/time.Duration(l.max))
		if err := checkDeadline(ctx, d); err != nil {
			return err
		}
		l.logger.Info("Rate limit exhausted. Waiting for reset", "retryAfter", d, "name", l.name)
		if err := sleepContext(ctx, d); err != nil {
			return err
		}
	}
	l.entries[l.index] = time.Now()
	l.index = l.index + 1
	if l.index == l.max {
		l.index = 0
	}
	return nil
}

// checkDeadline returns an error when waiting d would exceed the ctx deadline.
func checkDeadline(ctx context.Context, d time.Duration) error {
	if dl, ok := ctx.Deadline(); ok && time.Until(dl) < d {
		return fmt.Errorf("rate limit wait of %s exceeds context deadline: %w", d, context.DeadlineExceeded)
	}
	return nil
}

// sleepContext pauses for duration d or until ctx is done, whichever comes first.
func sleepContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func roundUpDuration(d time.Duration, m time.Duration) time.Duration {
	x := d.Round(m)
	if x < d {
		return x + m
	}
	return x
}
