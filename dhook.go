/*
Package dhook provides types and functions for sending messages to Discord webhooks.

The dhook package was specifically designed for sending a high volume of messages to Discord webhooks.
A main challenge when sending many messages within a short time are to conform with Discord's rate limits.

Dhook will automatically respect all rate limits by waiting until a slot is free before sending a new message.

There are three different rate limits and this package will automatically respect them all:
- Global rate limit (static)
- Per-route rate limit (dynamic)
- Webhook rate limit (static)

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

	[Discord's rate limits]: https://discord.com/developers/docs/topics/rate-limits
*/
package dhook
