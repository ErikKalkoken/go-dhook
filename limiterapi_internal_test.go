package dhook

import (
	"fmt"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimitInfo_New(t *testing.T) {
	t.Run("should extract rate limit from header", func(t *testing.T) {
		header := http.Header{}
		header.Set("X-RateLimit-Limit", "5")
		header.Set("X-RateLimit-Remaining", "1")
		header.Set("X-RateLimit-Reset", "1470173023")
		header.Set("X-RateLimit-Reset-After", "1.2")
		header.Set("X-RateLimit-Bucket", "abcd1234")
		rl, err := newRateLimitInfo(header)
		if assert.NoError(t, err) {
			assert.Equal(t, 5, rl.limit)
			assert.Equal(t, 1, rl.remaining)
			assert.Equal(t, time.Date(2016, 8, 2, 21, 23, 43, 0, time.UTC), rl.resetAt)
			assert.Equal(t, 1.2, rl.resetAfter)
			assert.Equal(t, "abcd1234", rl.bucket)
		}
	})
	t.Run("should return empty rate limit if header is incomplete", func(t *testing.T) {
		header := http.Header{}
		rl, err := newRateLimitInfo(header)
		if assert.NoError(t, err) {
			assert.True(t, rl.resetAt.IsZero())
		}
	})
}

func TestRateLimitInfo_NewErrorValidation(t *testing.T) {
	cases := []struct {
		limit, remaining, reset, resetAfter, bucket string
		isValid                                     bool
	}{
		{limit: "5", remaining: "1", reset: "1470173023", resetAfter: "1.2", bucket: "abcd1234", isValid: true},
		{limit: "x", remaining: "1", reset: "1470173023", resetAfter: "1.2", bucket: "abcd1234", isValid: false},
		{limit: "5", remaining: "x", reset: "1470173023", resetAfter: "1.2", bucket: "abcd1234", isValid: false},
		{limit: "5", remaining: "1", reset: "x", resetAfter: "1.2", bucket: "abcd1234", isValid: false},
		{limit: "5", remaining: "1", reset: "1470173023", resetAfter: "x", bucket: "abcd1234", isValid: false},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case %d", i+1), func(t *testing.T) {
			header := http.Header{}
			header.Set("X-RateLimit-Limit", tc.limit)
			header.Set("X-RateLimit-Remaining", tc.remaining)
			header.Set("X-RateLimit-Reset", tc.reset)
			header.Set("X-RateLimit-Reset-After", tc.resetAfter)
			header.Set("X-RateLimit-Bucket", tc.bucket)
			_, err := newRateLimitInfo(header)
			if tc.isValid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestRateLimitInfo_ReturnEmptyWhenIncomplete(t *testing.T) {
	cases := []struct {
		limit, remaining, reset, resetAfter, bucket string
		isEmpty                                     bool
	}{
		{limit: "5", remaining: "1", reset: "1470173023", resetAfter: "1.2", bucket: "abcd1234", isEmpty: false},
		{limit: "", remaining: "1", reset: "1470173023", resetAfter: "1.2", bucket: "abcd1234", isEmpty: true},
		{limit: "5", remaining: "", reset: "1470173023", resetAfter: "1.2", bucket: "abcd1234", isEmpty: true},
		{limit: "5", remaining: "1", reset: "", resetAfter: "1.2", bucket: "abcd1234", isEmpty: true},
		{limit: "5", remaining: "1", reset: "1470173023", resetAfter: "", bucket: "abcd1234", isEmpty: true},
		{limit: "5", remaining: "1", reset: "1470173023", resetAfter: "1.2", bucket: "", isEmpty: true},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case %d", i+1), func(t *testing.T) {
			header := http.Header{}
			header.Set("X-RateLimit-Limit", tc.limit)
			header.Set("X-RateLimit-Remaining", tc.remaining)
			header.Set("X-RateLimit-Reset", tc.reset)
			header.Set("X-RateLimit-Reset-After", tc.resetAfter)
			header.Set("X-RateLimit-Bucket", tc.bucket)
			got, err := newRateLimitInfo(header)
			if assert.NoError(t, err) {
				if tc.isEmpty {
					assert.Empty(t, got)
				} else {
					assert.NotEmpty(t, got)
				}
			}
		})
	}
}

func TestRateLimitInfo_String(t *testing.T) {
	x := rateLimitInfo{timestamp: time.Now(), remaining: 0, resetAt: time.Now().Add(5 * time.Second)}
	assert.NotZero(t, fmt.Sprint(x))
}

func TestRateLimitInfo_LimitExceeded(t *testing.T) {
	now := time.Now().UTC()
	cases := []struct {
		rl   rateLimitInfo
		want bool
	}{
		{rateLimitInfo{}, false},
		{rateLimitInfo{timestamp: now, remaining: 1}, false},
		{rateLimitInfo{timestamp: now, remaining: 0, resetAt: now.Add(-5 * time.Second)}, false},
		{rateLimitInfo{timestamp: now, remaining: 0, resetAt: now.Add(5 * time.Second)}, true},
	}
	for i, tc := range cases {
		t.Run(fmt.Sprintf("case %d", i+1), func(t *testing.T) {
			assert.Equal(t, tc.want, tc.rl.limitExceeded(now))
		})
	}
}

func TestRateLimitInfo_IsSet(t *testing.T) {
	t.Run("return true when set", func(t *testing.T) {
		x := rateLimitInfo{timestamp: time.Now()}
		assert.True(t, x.isSet())
	})
	t.Run("return false when not set", func(t *testing.T) {
		x := rateLimitInfo{}
		assert.False(t, x.isSet())
	})
}

func TestLimiterAPI_UpdateFromHeader2(t *testing.T) {
	cases := []struct {
		name, limit, remaining, reset, resetAfter, bucket string
		current                                           rateLimitInfo
		want                                              int
	}{
		{
			name:       "should decrease remaining if header is about same period and bucket",
			current:    rateLimitInfo{remaining: 2, resetAt: time.Unix(1470173023, 0).UTC(), bucket: "abcd1234"},
			limit:      "5",
			remaining:  "3",
			reset:      "1470173023",
			resetAfter: "1.2",
			bucket:     "abcd1234",
			want:       1,
		},
		{
			name:       "should update when header is about new period and same bucket",
			current:    rateLimitInfo{remaining: 2, resetAt: time.Unix(1470173022, 0).UTC(), bucket: "abcd1234"},
			limit:      "5",
			remaining:  "4",
			reset:      "1470173023",
			resetAfter: "1.2",
			bucket:     "abcd1234",
			want:       4,
		},
		{
			name:       "should update when header is about same period and different bucket",
			current:    rateLimitInfo{remaining: 2, resetAt: time.Unix(1470173022, 0).UTC(), bucket: "abcd1234"},
			limit:      "5",
			remaining:  "4",
			reset:      "1470173022",
			resetAfter: "1.2",
			bucket:     "abcd9234",
			want:       4,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := limiterAPI{rl: tc.current}
			header := http.Header{}
			header.Set("X-RateLimit-Limit", tc.limit)
			header.Set("X-RateLimit-Remaining", tc.remaining)
			header.Set("X-RateLimit-Reset", tc.reset)
			header.Set("X-RateLimit-Reset-After", tc.resetAfter)
			header.Set("X-RateLimit-Bucket", tc.bucket)
			err := l.updateFromHeader(header)
			if assert.NoError(t, err) {
				assert.Equal(t, tc.want, l.rl.remaining)
			}
		})
	}
}

func TestLimiterAPI_UpdateFromHeader_Error(t *testing.T) {
	l := limiterAPI{rl: rateLimitInfo{remaining: 2, resetAt: time.Unix(1470173023, 0).UTC(), bucket: "abcd1234"}}
	header := http.Header{}
	header.Set("X-RateLimit-Limit", "x")
	header.Set("X-RateLimit-Remaining", "3")
	header.Set("X-RateLimit-Reset", "1470173023")
	header.Set("X-RateLimit-Reset-After", "1.2")
	header.Set("X-RateLimit-Bucket", "abcd1234")
	err := l.updateFromHeader(header)
	assert.Error(t, err)
}

func TestLimiterAPI_Wait(t *testing.T) {
	t.Run("should not wait if limit not exceeded", func(t *testing.T) {
		l := limiterAPI{rl: rateLimitInfo{timestamp: time.Now(), remaining: 1}}
		l.logger = slog.Default()
		got := l.wait()
		assert.False(t, got)
	})
	t.Run("should wait if limit is exceeded", func(t *testing.T) {
		l := limiterAPI{rl: rateLimitInfo{timestamp: time.Now(), remaining: 0, resetAt: time.Now().Add(200 * time.Millisecond)}}
		l.logger = slog.Default()
		got := l.wait()
		assert.True(t, got)
	})
}

func TestLimiterAPI_String(t *testing.T) {
	l := limiterAPI{rl: rateLimitInfo{timestamp: time.Now(), remaining: 0, resetAt: time.Now().Add(200 * time.Millisecond)}}
	assert.NotEqual(t, "", fmt.Sprint(l))
}
