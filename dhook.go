/*
Package dhook provides types and functions for sending messages to Discord webhooks.

# Example

The following shows how to use the library for sending a message to a Discord Webhook.

	package main

	import (
		"net/http"

		"github.com/ErikKalkoken/go-dhook"
	)

	func main() {
		c := dhook.NewClient(http.DefaultClient)
		wh := dhook.NewWebhook(c, WEBHOOK_URL) // !! Please replace with a valid URL
		err := wh.Execute(dhook.Message{Content: "Hello"})
		if err != nil {
			panic(err)
		}
	}
*/
package dhook

import (
	"net/http"
	"time"
)

const (
	globalRateLimitPeriod   = 1 * time.Second
	globalRateLimitRequests = 50
)

// Client represents a shared client used by all webhooks to access the Discord API.
//
// The shared client enabled dealing with the global rate limit and ensures a shared http client is used.
type Client struct {
	httpClient    *http.Client
	limiterGlobal *limiter

	rl rateLimited
}

// NewClient returns a new client for webhook. All webhooks share the provided HTTP client.
func NewClient(httpClient *http.Client) *Client {
	s := &Client{
		httpClient:    httpClient,
		limiterGlobal: newLimiter(globalRateLimitPeriod, globalRateLimitRequests, "global"),
	}
	return s
}
