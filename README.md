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
- Full support of Discord embeds
- Named colors
- Unit tested

You can add this library to your current Go module with this command:

```sh
go get github.com/ErikKalkoken/go-dhook
```

For the API documentation and examples please see [Go Reference](https://pkg.go.dev/github.com/ErikKalkoken/go-dhook).
