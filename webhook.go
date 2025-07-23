package dhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	retryAfterTooManyRequestDefault = 60 * time.Second
)

// TooManyRequestsError represents a HTTP status code 429 error.
type TooManyRequestsError struct {
	RetryAfter time.Duration
	Global     bool
}

func (e TooManyRequestsError) Error() string {
	if e.Global {
		return "global rate limit exceeded"
	}
	return "rate limit exceeded"
}

// HTTPError represents a HTTP error, e.g. 400 Bad Request
type HTTPError struct {
	Status  int
	Message string
}

func (e HTTPError) Error() string {
	return e.Message
}

// Error representing an invalid configuration, e.g. a negative HTTP timeout.
var ErrInvalidConfiguration = errors.New("invalid configuration")

// Webhook represents a Discord webhook.
// Webhooks are safe for concurrent use by multiple goroutines.
type Webhook struct {
	client *Client
	url    string

	mu             sync.Mutex
	rl             rateLimited
	limiterAPI     limiterAPI
	limiterWebhook *limiter
}

type WebhookExecuteOptions struct {
	// Waits for server confirmation of message send before response
	// and returns the created message body.
	Wait bool
}

// Execute posts a message to the configured webhook and optionally returns the message created by Discord.
//
// Execute respects Discord's rate limits and will wait until there is a free slot to post the message if necessary.
// Options can be provided through opt. Opt can be left nil to provide no options.
// Execute will only return the message created by Discord when the Wait option is enabled.
//
// Returned key errors:
//   - [HTTPError]: Discord returned HTTP status codes of 400 or above (except 429)
//   - [TooManyRequestsError]: Discord returned status HTTP status code 429
//   - [context.DeadlineExceeded]: Timeout is exceeded during the HTTP request to Discord
func (wh *Webhook) Execute(m Message, opt *WebhookExecuteOptions) ([]byte, error) {
	if wh.client == nil {
		return nil, fmt.Errorf("Webhook not initialized: %w", ErrInvalidConfiguration)
	}
	wh.client.logger.Debug("message", "detail", fmt.Sprintf("%+v", m))
	dat, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	if isActive, retryAfter := wh.client.rl.getOrReset(); isActive {
		return nil, TooManyRequestsError{RetryAfter: retryAfter, Global: true}
	}
	wh.mu.Lock()
	defer wh.mu.Unlock()
	if isActive, retryAfter := wh.rl.getOrReset(); isActive {
		return nil, TooManyRequestsError{RetryAfter: retryAfter}
	}
	wh.client.limiterGlobal.wait()
	wh.limiterAPI.wait()
	wh.limiterWebhook.wait()

	url := wh.url
	if opt != nil && opt.Wait {
		url += "?wait=1"
	}
	ctx, cancel := context.WithTimeout(context.Background(), wh.client.httpTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(dat))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	wh.client.logger.Debug("request", "url", url, "body", string(dat))
	resp, err := wh.client.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if err := wh.limiterAPI.updateFromHeader(resp.Header); err != nil {
		wh.client.logger.Error("Failed to update API limiter from header", "error", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	wh.client.logger.Debug("response", "url", url, "status", resp.Status, "headers", resp.Header, "body", string(body))
	if resp.StatusCode >= http.StatusBadRequest {
		wh.client.logger.Warn("response", "url", url, "status", resp.Status)
	} else {
		wh.client.logger.Info("response", "url", url, "status", resp.Status)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		var m tooManyRequestsResponse
		if err := json.Unmarshal(body, &m); err != nil {
			wh.client.logger.Warn("Failed to parse 429 response body", "error", err)
		}
		retryAfter := retryAfterTooManyRequestDefault
		s := resp.Header.Get("Retry-After")
		if s != "" {
			x, err := strconv.Atoi(s)
			if err != nil {
				wh.client.logger.Warn("Failed to parse retry after. Assuming default", "error", err)
			} else {
				retryAfter = time.Duration(x) * time.Second
			}
		}
		wh.rl.set(retryAfter)
		if m.Global {
			wh.client.rl.set(retryAfter)
		}
		return body, TooManyRequestsError{
			RetryAfter: retryAfter, // Value from header is more reliable
			Global:     m.Global,
		}
	}
	if resp.StatusCode >= 400 {
		err := HTTPError{
			Status:  resp.StatusCode,
			Message: resp.Status,
		}
		return nil, err
	}
	return body, nil
}

type tooManyRequestsResponse struct {
	Message    string  `json:"message,omitempty"`
	RetryAfter float64 `json:"retry_after,omitempty"`
	Global     bool    `json:"global,omitempty"`
}
