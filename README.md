# go-dhook

A Go library for sending high volumes of messages to Discord webhooks while respecting rate limits.

![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/go-dhook)
[![CI/CD](https://github.com/ErikKalkoken/go-dhook/actions/workflows/go.yml/badge.svg)](https://github.com/ErikKalkoken/go-dhook/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/ErikKalkoken/go-dhook/graph/badge.svg?token=4bqOmx0RKh)](https://codecov.io/gh/ErikKalkoken/go-dhook)
![GitHub License](https://img.shields.io/github/license/ErikKalkoken/go-dhook)
[![Go Reference](https://pkg.go.dev/badge/github.com/ErikKalkoken/go-dhook.svg)](https://pkg.go.dev/github.com/ErikKalkoken/go-dhook)

## Features

- Respects Discord rate limits, including the undocumented webhook limit
- Blocks requests after a 429 response to prevent rate limit escalation
- Message validation
- Built-in logging
- Configurable HTTP client, timeout, logger and rate limits
- Messages with full embed support
- Named colors
- No dependencies (except for tests)

> [!NOTE]
> This library is pre-1.0. The API may still change between minor versions.

## Installation

go-dhook requires Go 1.24 or later.

You can add this library to your Go module with the following command:

```sh
go get github.com/ErikKalkoken/go-dhook
```

## Example

Below is an example of how to send a basic message to a webhook.

> [!TIP]
> Please don't forget to replace the webhook URL in the example with a real URL from your Discord server before trying it out.
> See [Intro to Webhooks](https://support.discord.com/hc/en-us/articles/228383668-Intro-to-Webhooks) for how to create one.

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

Create **one** `Client` and reuse it for the lifetime of your application. Otherwise rate limits cannot be tracked correctly. Clients and webhooks are safe for concurrent use.

## Rate limiting

go-dhook respects three rate limits that apply to webhooks:

1. **Global**: the global limit from the [Discord API documentation](https://discord.com/developers/docs/topics/rate-limits#global-rate-limit)
2. **Per-route**: the dynamic limit from the `X-RateLimit-*` response headers
3. **Webhook**: an undocumented limit of 30 requests per 60 seconds

All three are active by default. `Execute` blocks until a request can be sent within them.
Use `ExecuteContext` to limit how long to wait, e.g. with a timeout or cancellation.

If Discord still responds with 429, subsequent requests fail immediately with a
`TooManyRequestsError` until the `Retry-After` time has passed. This prevents
escalating the rate limit. A global 429, or a 429 without Discord's rate limit
headers (e.g. from Cloudflare), blocks all webhooks of the client.

## Documentation

For the API documentation and more examples please see [Go Reference](https://pkg.go.dev/github.com/ErikKalkoken/go-dhook).
