# go-dhook

A Go library for sending messages to Discord webhooks.

![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/go-dhook)
[![CI/CD](https://github.com/ErikKalkoken/go-dhook/actions/workflows/go.yml/badge.svg)](https://github.com/ErikKalkoken/go-dhook/actions/workflows/go.yml)
![GitHub License](https://img.shields.io/github/license/ErikKalkoken/go-dhook)
[![Go Reference](https://pkg.go.dev/badge/github.com/ErikKalkoken/go-dhook.svg)](https://pkg.go.dev/github.com/ErikKalkoken/go-dhook)

## Description

go-dhook is Go library for sending messages to Discord webhooks. It's key features are:

- Automatically respects all known Discord rate limits
- Automatically blocks further message sending when 429 is received
- Safe to use concurrently
- Full support of Discord embeds
- Detects invalid messages (e.g. a field exceeding it's character limit)

## Installation

You can add this library to your current Go module with this command:

```sh
go get github.com/ErikKalkoken/go-dhook
```

## Example

The following is an example on how to send a simple message with the library.

```go
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
```
