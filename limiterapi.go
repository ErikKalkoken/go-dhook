package dhook

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"
)

// limiterAPI implements a limiter from the Discord API rate limit
// as communicated by "X-RateLimit-" response headers.
type limiterAPI struct {
	rl     rateLimitInfo
	logger Logger
}

// wait will wait until a free slot is available if necessary
func (l *limiterAPI) wait(ctx context.Context) error {
	l.logger.Debug("API rate limit", "info", l.rl)
	if !l.rl.limitExceeded(time.Now()) {
		return nil
	}
	retryAfter := roundUpDuration(time.Until(l.rl.resetAt), time.Second)
	if err := checkDeadline(ctx, retryAfter); err != nil {
		return err
	}
	l.logger.Info("API rate limit exhausted. Waiting for reset", "retryAfter", retryAfter)
	return sleepContext(ctx, retryAfter)
}

// consume counts a request.
func (l *limiterAPI) consume() {
	if l.rl.remaining > 0 {
		l.rl.remaining--
	}
}

// updateFromHeader updates the limiter from a header.
func (l *limiterAPI) updateFromHeader(h http.Header) error {
	l.consume()
	rl2, err := newRateLimitInfo(h)
	if err != nil {
		return err
	}
	if !rl2.isSet() {
		return nil
	}
	if rl2.bucket == l.rl.bucket && rl2.resetAt.Equal(l.rl.resetAt) {
		return nil
	}
	l.rl = rl2
	return nil
}

// rateLimitInfo represents the rate limit information as returned from the Discord API
type rateLimitInfo struct {
	limit      int
	remaining  int
	resetAt    time.Time
	resetAfter float64
	bucket     string
	timestamp  time.Time
}

// newRateLimitInfo returns a new rateLimitInfo from a header.
// Will return an empty rateLimitInfo when the rate limit headers are missing, incomplete.
// will return an error when the rate limit headers are invalid.
func newRateLimitInfo(h http.Header) (rateLimitInfo, error) {
	var r rateLimitInfo
	var err error
	limit := h.Get("X-RateLimit-Limit")
	if limit == "" {
		return r, nil
	}
	remaining := h.Get("X-RateLimit-Remaining")
	if remaining == "" {
		return r, nil
	}
	reset := h.Get("X-RateLimit-Reset")
	if reset == "" {
		return r, nil
	}
	resetAfter := h.Get("X-RateLimit-Reset-After")
	if resetAfter == "" {
		return r, nil
	}
	bucket := h.Get("X-RateLimit-Bucket")
	if bucket == "" {
		return r, nil
	}
	wrapErr := func(err error) error {
		return fmt.Errorf("newRateLimitInfo invalid header %+v : %w", h, err)
	}
	r.limit, err = strconv.Atoi(limit)
	if err != nil {
		return r, wrapErr(err)
	}
	r.remaining, err = strconv.Atoi(remaining)
	if err != nil {
		return r, wrapErr(err)
	}
	resetEpoch, err := strconv.ParseFloat(reset, 64)
	if err != nil {
		return r, wrapErr(err)
	}
	r.resetAt = time.UnixMilli(int64(math.Round(resetEpoch * 1000))).UTC()
	r.resetAfter, err = strconv.ParseFloat(resetAfter, 64)
	if err != nil {
		return r, wrapErr(err)
	}
	r.bucket = bucket
	r.timestamp = time.Now().UTC()
	return r, nil
}

func (rl rateLimitInfo) String() string {
	return fmt.Sprintf(
		"limit:%d remaining:%d reset:%s resetAfter:%f",
		rl.limit,
		rl.remaining,
		rl.resetAt, time.Until(rl.resetAt).Seconds(),
	)
}

func (rl rateLimitInfo) isSet() bool {
	return !rl.timestamp.IsZero()
}

func (rl rateLimitInfo) limitExceeded(now time.Time) bool {
	if !rl.isSet() {
		return false
	}
	if rl.remaining > 0 {
		return false
	}
	if rl.resetAt.Before(now) {
		return false
	}
	return true
}
