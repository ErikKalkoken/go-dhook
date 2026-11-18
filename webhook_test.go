package dhook_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/iotest"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ErikKalkoken/go-dhook"
)

func TestWebhook_Execute(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	url := "https://www.example.com/hook"
	t.Run("can post a message", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url, httpmock.NewStringResponder(204, ""))
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		if assert.NoError(t, err) {
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
		}
	})
	t.Run("should add wait to URL with query parameters", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url+"?thread_id=1&wait=true", httpmock.NewStringResponder(200, "{}"))
		c := dhook.NewClient()
		wh := c.NewWebhook(url + "?thread_id=1")
		_, err := wh.Execute(dhook.Message{Content: "content"}, &dhook.WebhookExecuteOptions{Wait: true})
		if assert.NoError(t, err) {
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
		}
	})
	t.Run("should replace existing wait parameter", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url+"?wait=true", httpmock.NewStringResponder(200, "{}"))
		c := dhook.NewClient()
		wh := c.NewWebhook(url + "?wait=false")
		_, err := wh.Execute(dhook.Message{Content: "content"}, &dhook.WebhookExecuteOptions{Wait: true})
		if assert.NoError(t, err) {
			assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+url+"?wait=true"])
		}
	})
	t.Run("should apply webhook rate limit across URLs of same Discord webhook", func(t *testing.T) {
		httpmock.Reset()
		url1 := "https://discord.com/api/webhooks/123/token"
		url2 := "https://discordapp.com/api/v10/webhooks/123/token"
		httpmock.RegisterResponder("POST", url1, httpmock.NewStringResponder(204, ""))
		httpmock.RegisterResponder("POST", url2, httpmock.NewStringResponder(204, ""))
		const period = 200 * time.Millisecond
		c := dhook.NewClient(dhook.WithWebhookRateLimit(1, period))
		start := time.Now()
		_, err := c.NewWebhook(url1).Execute(dhook.Message{Content: "first"}, nil)
		assert.NoError(t, err)
		_, err = c.NewWebhook(url2).Execute(dhook.Message{Content: "second"}, nil)
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, time.Since(start), period)
		info := httpmock.GetCallCountInfo()
		assert.Equal(t, 1, info["POST "+url1])
		assert.Equal(t, 1, info["POST "+url2])
	})
	t.Run("should return http 400 as HTTPError", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			url,
			httpmock.NewStringResponder(400, ""),
		)
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		httpErr, _ := err.(dhook.HTTPError)
		assert.Equal(t, 400, httpErr.Status)
	})
	t.Run("should return http 429 as TooManyRequestsError", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			url,
			httpmock.NewJsonResponderOrPanic(429,
				map[string]any{
					"message":     "You are being rate limited.",
					"retry_after": 2.57,
					"global":      true,
				}).HeaderSet(http.Header{
				"Retry-After":        []string{"3"},
				"X-RateLimit-Global": []string{"true"},
				"X-RateLimit-Scope":  []string{"global"},
			}),
		)
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		err2, _ := err.(dhook.TooManyRequestsError)
		assert.Equal(t, 3*time.Second, err2.RetryAfter)
		assert.True(t, err2.Global)
	})
	t.Run("should return http 429 with user scope as non-global TooManyRequestsError", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			url,
			httpmock.NewStringResponder(429, "").HeaderSet(http.Header{
				"Retry-After":       []string{"3"},
				"X-RateLimit-Scope": []string{"user"},
			}),
		)
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		var err2 dhook.TooManyRequestsError
		if assert.ErrorAs(t, err, &err2) {
			assert.Equal(t, 3*time.Second, err2.RetryAfter)
			assert.False(t, err2.Global)
		}
	})
	t.Run("should treat http 429 without Discord headers as global", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			url,
			httpmock.NewStringResponder(429, "<html>Cloudflare</html>").HeaderSet(http.Header{
				"Retry-After": []string{"3"},
			}),
		)
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		var err2 dhook.TooManyRequestsError
		if assert.ErrorAs(t, err, &err2) {
			assert.True(t, err2.Global)
		}
		wh2 := c.NewWebhook("https://www.example.com/other-hook")
		_, err = wh2.Execute(dhook.Message{Content: "content"}, nil)
		if assert.ErrorAs(t, err, &err2) {
			assert.True(t, err2.Global)
		}
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
	})
	t.Run("should record http 429 without reading the body", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url,
			func(req *http.Request) (*http.Response, error) {
				h := http.Header{}
				h.Set("Retry-After", "3")
				h.Set("X-RateLimit-Scope", "user")
				return &http.Response{
					StatusCode: 429,
					Status:     "429 Too Many Requests",
					Header:     h,
					Body:       io.NopCloser(iotest.ErrReader(errors.New("body read failed"))),
				}, nil
			},
		)
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		var err2 dhook.TooManyRequestsError
		require.ErrorAs(t, err, &err2)
		_, err = wh.Execute(dhook.Message{Content: "content"}, nil)
		if assert.ErrorAs(t, err, &err2) {
			assert.False(t, err2.Global)
		}
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
	})
	t.Run("should return http 429 as TooManyRequestsError and use default retry duration", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			url,
			httpmock.NewStringResponder(429, "").HeaderSet(http.Header{"Retry-After": []string{"invalid"}}),
		)
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		httpErr, _ := err.(dhook.TooManyRequestsError)
		assert.Equal(t, 60*time.Second, httpErr.RetryAfter)
	})
	t.Run("should return http 429 as TooManyRequestsError and use default retry duration 2", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			url,
			httpmock.NewStringResponder(429, ""),
		)
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		httpErr, _ := err.(dhook.TooManyRequestsError)
		assert.Equal(t, 60*time.Second, httpErr.RetryAfter)
	})
	t.Run("should not timeout when not configured", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url,
			func(req *http.Request) (*http.Response, error) {
				time.Sleep(250 * time.Millisecond)
				return httpmock.NewStringResponse(204, ""), nil
			},
		)
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		if assert.NoError(t, err) {
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
		}
	})
	t.Run("should timeout when configured", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url,
			func(req *http.Request) (*http.Response, error) {
				time.Sleep(250 * time.Millisecond)
				return httpmock.NewStringResponse(204, ""), nil
			},
		)
		c := dhook.NewClient(dhook.WithHTTPTimeout(100 * time.Millisecond))
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	})
	t.Run("should return error when webhook was not initialized", func(t *testing.T) {
		wh := dhook.Webhook{}
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		assert.ErrorIs(t, err, dhook.ErrInvalidConfiguration)
	})
	t.Run("should return error when message is invalid", func(t *testing.T) {
		c := dhook.NewClient(dhook.WithHTTPTimeout(100 * time.Millisecond))
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{}, nil)
		assert.ErrorIs(t, err, dhook.ErrInvalidMessage)
	})
	t.Run("message with wait option returns response body", func(t *testing.T) {
		httpmock.Reset()
		url2 := url + "?wait=true"
		httpmock.RegisterResponder("POST", url2, httpmock.NewStringResponder(200, "message"))
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		b, err := wh.Execute(dhook.Message{Content: "content"}, &dhook.WebhookExecuteOptions{
			Wait: true,
		})
		if assert.NoError(t, err) {
			info := httpmock.GetCallCountInfo()
			assert.Equal(t, 1, info["POST "+url2])
		}
		assert.Equal(t, "message", string(b))
	})
}

func TestWebhook_ExecuteContext(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	url := "https://www.example.com/hook"
	t.Run("should return error when context is canceled", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url, httpmock.NewStringResponder(204, ""))
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		_, err := wh.ExecuteContext(ctx, dhook.Message{Content: "content"}, nil)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 0, httpmock.GetTotalCallCount())
	})
	t.Run("should return error when context deadline is exceeded while waiting for rate limit", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url, httpmock.NewStringResponder(204, ""))
		c := dhook.NewClient(dhook.WithWebhookRateLimit(1, time.Second))
		wh := c.NewWebhook(url)
		_, err := wh.ExecuteContext(t.Context(), dhook.Message{Content: "first"}, nil)
		require.NoError(t, err)
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		start := time.Now()
		_, err = wh.ExecuteContext(ctx, dhook.Message{Content: "second"}, nil)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Less(t, time.Since(start), 500*time.Millisecond)
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
	})
	// newLastSlotResponse returns a response reporting one remaining slot in the API rate limit.
	newLastSlotResponse := func() *http.Response {
		resp := httpmock.NewStringResponse(204, "")
		resp.Header.Set("X-RateLimit-Limit", "5")
		resp.Header.Set("X-RateLimit-Remaining", "1")
		resp.Header.Set("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(5*time.Second).Unix(), 10))
		resp.Header.Set("X-RateLimit-Reset-After", "5")
		resp.Header.Set("X-RateLimit-Bucket", "abcd1234")
		return resp
	}
	t.Run("should count a request timed out while in flight against the API rate limit", func(t *testing.T) {
		httpmock.Reset()
		calls := 0
		httpmock.RegisterResponder("POST", url, func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 2 {
				<-req.Context().Done() // simulate a request in flight until ctx ends
				return nil, req.Context().Err()
			}
			return newLastSlotResponse(), nil
		})
		c := dhook.NewClient(dhook.WithHTTPTimeout(50 * time.Millisecond))
		wh := c.NewWebhook(url)
		_, err := wh.ExecuteContext(t.Context(), dhook.Message{Content: "first"}, nil)
		require.NoError(t, err)
		_, err = wh.ExecuteContext(t.Context(), dhook.Message{Content: "second"}, nil)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		// The last slot was used by the canceled request, so the next call must wait for the reset.
		ctx2, cancel2 := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel2()
		_, err = wh.ExecuteContext(ctx2, dhook.Message{Content: "third"}, nil)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Equal(t, 2, httpmock.GetTotalCallCount())
	})
	t.Run("should complete a request in flight when ctx is canceled", func(t *testing.T) {
		httpmock.Reset()
		ctx, cancel := context.WithCancel(t.Context())
		httpmock.RegisterResponder("POST", url, func(req *http.Request) (*http.Response, error) {
			cancel() // cancel while the request is in flight
			time.Sleep(20 * time.Millisecond)
			if err := req.Context().Err(); err != nil {
				return nil, err
			}
			return httpmock.NewStringResponse(204, ""), nil
		})
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.ExecuteContext(ctx, dhook.Message{Content: "content"}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
	})
	t.Run("should not count a request failing with a connection error against the API rate limit", func(t *testing.T) {
		httpmock.Reset()
		calls := 0
		httpmock.RegisterResponder("POST", url, func(req *http.Request) (*http.Response, error) {
			calls++
			if calls == 2 {
				return nil, errors.New("connection refused")
			}
			return newLastSlotResponse(), nil
		})
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.ExecuteContext(t.Context(), dhook.Message{Content: "first"}, nil)
		require.NoError(t, err)
		_, err = wh.ExecuteContext(t.Context(), dhook.Message{Content: "second"}, nil)
		require.Error(t, err)
		// The last slot is still free, so the next call is sent without waiting.
		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()
		_, err = wh.ExecuteContext(ctx, dhook.Message{Content: "third"}, nil)
		assert.NoError(t, err)
		assert.Equal(t, 3, httpmock.GetTotalCallCount())
	})
}

func TestTooManyRequestsError_Error(t *testing.T) {
	t.Run("return normal error text", func(t *testing.T) {
		err := dhook.TooManyRequestsError{}
		assert.Equal(t, "rate limit exceeded", err.Error())
	})
	t.Run("return global error text", func(t *testing.T) {
		err := dhook.TooManyRequestsError{Global: true}
		assert.Equal(t, "global rate limit exceeded", err.Error())
	})
}

func TestHTTPError_Error(t *testing.T) {
	err := dhook.HTTPError{
		Message: "message",
	}
	assert.Equal(t, "message", err.Error())
}

// recordingLogger records all log lines.
type recordingLogger struct {
	mu    sync.Mutex
	lines []string
}

func (l *recordingLogger) log(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.lines = append(l.lines, fmt.Sprint(append([]any{msg}, args...)...))
}

func (l *recordingLogger) Debug(msg string, args ...any) { l.log(msg, args...) }
func (l *recordingLogger) Info(msg string, args ...any)  { l.log(msg, args...) }
func (l *recordingLogger) Warn(msg string, args ...any)  { l.log(msg, args...) }
func (l *recordingLogger) Error(msg string, args ...any) { l.log(msg, args...) }

func TestWebhook_Execute_redactsToken(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	const token = "secret-token"
	url := "https://discord.com/api/webhooks/123/" + token
	t.Run("should not log webhook token", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder("POST", url, httpmock.NewStringResponder(204, ""))
		l := &recordingLogger{}
		c := dhook.NewClient(dhook.WithLogger(l))
		_, err := c.NewWebhook(url).Execute(dhook.Message{Content: "content"}, nil)
		if assert.NoError(t, err) {
			assert.NotEmpty(t, l.lines)
			for _, line := range l.lines {
				assert.NotContains(t, line, token)
			}
		}
	})
	t.Run("should not return webhook token in errors", func(t *testing.T) {
		httpmock.Reset()
		errBoom := errors.New("boom")
		httpmock.RegisterResponder("POST", url, httpmock.NewErrorResponder(errBoom))
		c := dhook.NewClient()
		_, err := c.NewWebhook(url).Execute(dhook.Message{Content: "content"}, nil)
		if assert.Error(t, err) {
			assert.False(t, strings.Contains(err.Error(), token), err.Error())
			assert.ErrorIs(t, err, errBoom)
		}
	})
}
