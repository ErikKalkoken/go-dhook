package dhooks_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"

	"github.com/ErikKalkoken/go-dhooks"
)

func TestWebhook(t *testing.T) {
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	url := "https://www.example.com/hook"
	t.Run("can post a message", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			url,
			httpmock.NewStringResponder(204, ""),
		)
		c := dhooks.NewClient(http.DefaultClient)
		wh := dhooks.NewWebhook(c, url)
		err := wh.Execute(dhooks.Message{Content: "content"})
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
		c := dhooks.NewClient(http.DefaultClient)
		wh := dhooks.NewWebhook(c, url)
		err := wh.Execute(dhooks.Message{Content: "content"})
		httpErr, _ := err.(dhooks.HTTPError)
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
		c := dhooks.NewClient(http.DefaultClient)
		wh := dhooks.NewWebhook(c, url)
		err := wh.Execute(dhooks.Message{Content: "content"})
		err2, _ := err.(dhooks.TooManyRequestsError)
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
		c := dhooks.NewClient(http.DefaultClient)
		wh := dhooks.NewWebhook(c, url)
		err := wh.Execute(dhooks.Message{Content: "content"})
		httpErr, _ := err.(dhooks.TooManyRequestsError)
		assert.Equal(t, 60*time.Second, httpErr.RetryAfter)
	})
	t.Run("should return http 429 as TooManyRequestsError and use default retry duration 2", func(t *testing.T) {
		httpmock.Reset()
		httpmock.RegisterResponder(
			"POST",
			url,
			httpmock.NewStringResponder(429, ""),
		)
		c := dhooks.NewClient(http.DefaultClient)
		wh := dhooks.NewWebhook(c, url)
		err := wh.Execute(dhooks.Message{Content: "content"})
		httpErr, _ := err.(dhooks.TooManyRequestsError)
		assert.Equal(t, 60*time.Second, httpErr.RetryAfter)
	})
}
