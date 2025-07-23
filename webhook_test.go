package dhook_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

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
					"retry_after": 64.57,
					"global":      true,
				}).HeaderSet(http.Header{"Retry-After": []string{"3"}}),
		)
		c := dhook.NewClient()
		wh := c.NewWebhook(url)
		_, err := wh.Execute(dhook.Message{Content: "content"}, nil)
		err2, _ := err.(dhook.TooManyRequestsError)
		assert.Equal(t, 3*time.Second, err2.RetryAfter)
		assert.True(t, err2.Global)
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
		url2 := url + "?wait=1"
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
