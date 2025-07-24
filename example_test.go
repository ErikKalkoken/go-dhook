package dhook_test

import (
	"fmt"
	"time"

	"github.com/ErikKalkoken/go-dhook"
)

// This example shows how to send a simple message to a Discord webhook.
func Example_simple() {
	c := dhook.NewClient()
	wh := c.NewWebhook("YOUR-WEBHOOK-URL")
	_, err := wh.Execute(dhook.Message{Content: "Hello, World!"}, nil)
	if err != nil {
		panic(err)
	}
}

// This example shows how to send a complex message with a Discord embed.
func Example_complex() {
	c := dhook.NewClient()
	wh := c.NewWebhook("YOUR-WEBHOOK-URL")
	_, err := wh.Execute(dhook.Message{
		Content: "Content",
		Embeds: []dhook.Embed{{
			Author: dhook.Author{
				Name:    "Bruce Wayne",
				IconURL: "https://picsum.photos/64",
				URL:     "https://www.google.com",
			},
			Color: dhook.ColorOrange,
			Fields: []dhook.Field{
				{
					Name:  "First",
					Value: "42",
				},
				{
					Name:  "Second",
					Value: "99",
				},
			},
			Footer: dhook.Footer{
				Text:    "Footer",
				IconURL: "https://picsum.photos/64",
			},
			Description: "Description",
			Image:       dhook.Image{URL: "https://picsum.photos/200/300"},
			Timestamp:   time.Now(),
			Title:       "Title",
			URL:         "https://www.google.com",
		}},
	}, nil)
	if err != nil {
		panic(err)
	}
}

// This example shows how to use execute options when sending a message.
func Example_options() {
	c := dhook.NewClient()
	wh := c.NewWebhook("YOUR-WEBHOOK-URL")
	b, err := wh.Execute(dhook.Message{Content: "Hello, World!"}, &dhook.WebhookExecuteOptions{
		Wait: true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
}
