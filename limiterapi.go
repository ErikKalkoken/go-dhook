package dhook

import (
	"fmt"
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
// and report whether it has waited.
func (l *limiterAPI) wait() bool {
	l.logger.Debug("API rate limit", "info", l.rl)
	if !l.rl.limitExceeded(time.Now()) {
		return false
	}
	retryAfter := roundUpDuration(time.Until(l.rl.resetAt), time.Second)
	l.logger.Info("API rate limit exhausted. Waiting for reset", "retryAfter", retryAfter)
	time.Sleep(retryAfter)
	return true
}

// updateFromHeader updates the limiter from a header.
func (l *limiterAPI) updateFromHeader(h http.Header) error {
	if l.rl.remaining > 0 {
		l.rl.remaining--
	}
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
	r.limit, err = strconv.Atoi(limit)
	if err != nil {
		return r, err
	}
	r.remaining, err = strconv.Atoi(remaining)
	if err != nil {
		return r, err
	}
	resetEpoch, err := strconv.Atoi(reset)
	if err != nil {
		return r, err
	}
	r.resetAt = time.Unix(int64(resetEpoch), 0).UTC()
	r.resetAfter, err = strconv.ParseFloat(resetAfter, 64)
	if err != nil {
		return r, err
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

func (l rateLimitInfo) isSet() bool {
	return !l.timestamp.IsZero()
}

func (l rateLimitInfo) limitExceeded(now time.Time) bool {
	if !l.isSet() {
		return false
	}
	if l.remaining > 0 {
		return false
	}
	if l.resetAt.Before(now) {
		return false
	}
	return true
}
