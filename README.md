# go-dhook

A Go library for sending messages to Discord webhooks.

![GitHub Release](https://img.shields.io/github/v/release/ErikKalkoken/go-dhook)
[![CI/CD](https://github.com/ErikKalkoken/go-dhook/actions/workflows/go.yml/badge.svg)](https://github.com/ErikKalkoken/go-dhook/actions/workflows/go.yml)
[![codecov](https://codecov.io/gh/ErikKalkoken/go-dhook/graph/badge.svg?token=4bqOmx0RKh)](https://codecov.io/gh/ErikKalkoken/go-dhook)
![GitHub License](https://img.shields.io/github/license/ErikKalkoken/go-dhook)
[![Go Reference](https://pkg.go.dev/badge/github.com/ErikKalkoken/go-dhook.svg)](https://pkg.go.dev/github.com/ErikKalkoken/go-dhook)

## Description

go-dhook is Go library for sending messages to Discord webhooks. It's key features are:

- Automatically respects all Discord rate limits
- Automatically blocks further message sending when 429 is received
- Safe to use concurrently
- Full support of Discord embeds
- Detects invalid messages (e.g. a field exceeding it's character limit)
- Unit tested

You can add this library to your current Go module with this command:

```sh
go get github.com/ErikKalkoken/go-dhook
```

For the API documentation and examples please see [Go Reference](https://pkg.go.dev/github.com/ErikKalkoken/go-dhook).
