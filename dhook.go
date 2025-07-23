/*
Package dhook provides a client for sending messages to Discord webhooks.

The client was specifically designed to allow sending a high volume of messages
without being rate limited by the Discord API (i.e. 429 response).

The client achieved this by always respecting the following three rate limits
when a request is sent to Discord:
  - Global rate limit: The global rate limit as specified in the official API documentation
  - Per-route rate limit: A dynamic rate limit taken given in the response header
  - Webhook rate limit: An undocumented rate limit specific to webhooks

Should the client still become rate limited it will block further requests to Discord
for the time the rate limit is in effect to prevent further escalation.

# Example: Basic

The following is a basic example on how to send a message to a Discord Webhook.

	package main

	import (
		"net/http"

		"github.com/ErikKalkoken/go-dhook"
	)

	func main() {
		c := dhook.NewClient()
		wh := c.NewWebhook(WEBHOOK_URL) // Please replace with a valid URL
		err := wh.Execute(dhook.Message{Content: "Hello"})
		if err != nil {
			panic(err)
		}
	}

# Example: Sending a message with embed

The following is a basic example on how to send a message to a Discord Webhook.

	package main

	import (
		"net/http"

		"github.com/ErikKalkoken/go-dhook"
	)

	func main() {
		c := dhook.NewClient()
		wh := c.NewWebhook(WEBHOOK_URL) // Please replace with a valid URL
		err := wh.Execute(dhook.Message{Content: "Hello"})
		if err != nil {
			panic(err)
		}
	}
*/
package dhook
