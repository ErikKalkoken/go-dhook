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

# Example: Sending a simple message

The following is a basic example on how to send a message to a Discord Webhook.

	package main

	import (

		"github.com/ErikKalkoken/go-dhook"

	)

	func main() {
		c := dhook.NewClient()
		wh := c.NewWebhook("WEBHOOK-URL")
		_, err := wh.Execute(dhook.Message{Content: "Hello, World!"}, nil)
		if err != nil {
			panic(err)
		}
	}

# Example: Sending a complex message

Here we show how to send a message with an embed.

	package main

	import (

		"time"

		"github.com/ErikKalkoken/go-dhook"

	)

	func main() {
		c := dhook.NewClient()
		wh := c.NewWebhook("WEBHOOK-URL")
		_, err := wh.Execute(dhook.Message{
			Content: "Content",
			Embeds: []dhook.Embed{{
				Author: dhook.EmbedAuthor{
					Name:    "Bruce Wayne",
					IconURL: "https://picsum.photos/64",
					URL:     "https://www.google.com",
				},
				Color: dhook.ColorOrange,
				Fields: []dhook.EmbedField{
					{
						Name:  "First",
						Value: "42",
					},
					{
						Name:  "Second",
						Value: "99",
					},
				},
				Footer: dhook.EmbedFooter{
					Text:    "Footer",
					IconURL: "https://picsum.photos/64",
				},
				Description: "Description",
				Image:       dhook.EmbedImage{URL: "https://picsum.photos/200/300"},
				Timestamp: time.Now(),
				Title:     "Title",
				URL:       "https://www.google.com",
			}},
		}, nil)
		if err != nil {
			panic(err)
		}
	}

# Example: Sending a message with execute options

This example shows how to use the execute options to wait for a response from Discord
and print the result.

	package main

	import (
		"fmt"

		"github.com/ErikKalkoken/go-dhook"
	)

	func main() {
		c := dhook.NewClient()
		wh := c.NewWebhook("WEBHOOK-URL")
		b, err := wh.Execute(dhook.Message{Content: "Hello, World!"}, &dhook.WebhookExecuteOptions{
			Wait: true,
		})
		if err != nil {
			panic(err)
		}
		fmt.Println(string(b))
	}
*/
package dhook
