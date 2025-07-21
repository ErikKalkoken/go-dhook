# go-dhooks

A Go library for sending messages to Discord webhooks.

![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/go-dhooks)
[![CI/CD](https://github.com/ErikKalkoken/go-dhooks/actions/workflows/go.yml/badge.svg)](https://github.com/ErikKalkoken/go-dhooks/actions/workflows/go.yml)
![GitHub License](https://img.shields.io/github/license/ErikKalkoken/go-dhooks)
[![Go Reference](https://pkg.go.dev/badge/github.com/ErikKalkoken/go-dhooks.svg)](https://pkg.go.dev/github.com/ErikKalkoken/go-dhooks)

## Installation

You can add this library to your current Go project with the following command:

```sh
go get github.com/ErikKalkoken/go-dhooks
```

## Example

The following is an example on how to send a simple message with the library.

```go
package main

import (
	"net/http"

	"github.com/ErikKalkoken/go-dhooks"
)

func main() {
	c := dhooks.NewClient(http.DefaultClient)
	wh := dhooks.NewWebhook(c, WEBHOOK_URL)
	err := wh.Execute(dhooks.Message{Content: "Hello"})
	if err != nil {
		panic(err)
	}
}
```
