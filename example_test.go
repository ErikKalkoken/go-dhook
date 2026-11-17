package dhook_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/ErikKalkoken/go-dhook"
)

// newFakeWebhook starts a local server that imitates a Discord webhook,
// so the examples can run as tests. It returns the webhook URL and a function to stop the server.
// In real code, use the URL of your Discord webhook instead.
func newFakeWebhook() (string, func()) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		if r.URL.Query().Get("wait") != "" {
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"id":"123","content":"Hello, World!"}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	return srv.URL, srv.Close
}

// This example shows how to send a simple message to a Discord webhook.
func Example_simple() {
	url, stop := newFakeWebhook()
	defer stop()

	c := dhook.NewClient()
	wh := c.NewWebhook(url)
	_, err := wh.Execute(dhook.Message{Content: "Hello, World!"}, nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("Message sent")
	// Output: Message sent
}

// This example shows how to send a complex message with a Discord embed.
func Example_complex() {
	url, stop := newFakeWebhook()
	defer stop()

	c := dhook.NewClient()
	wh := c.NewWebhook(url)
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
	fmt.Println("Message sent")
	// Output: Message sent
}

// This example shows how to use execute options when sending a message.
// With Wait enabled, Discord returns the created message.
func Example_options() {
	url, stop := newFakeWebhook()
	defer stop()

	c := dhook.NewClient()
	wh := c.NewWebhook(url)
	b, err := wh.Execute(dhook.Message{Content: "Hello, World!"}, &dhook.WebhookExecuteOptions{
		Wait: true,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(string(b))
	// Output: {"id":"123","content":"Hello, World!"}
}
