# go-dhook

A Go library for sending messages to Discord webhooks.

![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/go-dhook)
[![CI/CD](https://github.com/ErikKalkoken/go-dhook/actions/workflows/go.yml/badge.svg)](https://github.com/ErikKalkoken/go-dhook/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/ErikKalkoken/go-dhook/graph/badge.svg?token=4bqOmx0RKh)](https://codecov.io/gh/ErikKalkoken/go-dhook)
![GitHub License](https://img.shields.io/github/license/ErikKalkoken/go-dhook)
[![Go Reference](https://pkg.go.dev/badge/github.com/ErikKalkoken/go-dhook.svg)](https://pkg.go.dev/github.com/ErikKalkoken/go-dhook)

## Description

go-dhook is a Go library for sending messages to Discord webhooks. It was specifically designed to allow sending a high volume of messages without being rate limited by the Discord API (i.e. 429 "Too many Request" response).

Key features:

- Automatically respects Discord rate limits
- Prevents rate limit escalation when rate limited
- Message validation
- Build-in logging
- Configurable client
- Basic messages with complete embed spec
- Named colors
- Unit tested
- No dependencies (except for tests)

## Example

Below is an example on how to send a basic message to a webhook.

> [!TIP]
> Please don't forget to replace the webhook URL in the example with a real URL from your Discord server before trying it out.

```go
    package main

    import (
        "github.com/ErikKalkoken/go-dhook"
    )

    func main() {
        c := dhook.NewClient()
        wh := c.NewWebhook("YOUR-WEBHOOK-URL")
        _, err := wh.Execute(dhook.Message{Content: "Hello, World!"}, nil)
        if err != nil {
            panic(err)
        }
    }
```

## Installation

You can add this library to your Go module with the following command:

```sh
go get github.com/ErikKalkoken/go-dhook
```

## Documentation

For the API documentation and more examples please see [Go Reference](https://pkg.go.dev/github.com/ErikKalkoken/go-dhook).
