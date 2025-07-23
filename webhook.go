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

// Execute posts a message to the configured webhook.
//
// Execute respects Discord's rate limits and will wait until there is a free slot to post the message if necessary.
//
// HTTP status codes of 400 or above are returned as [HTTPError],
// except for 429s, which are returned as [TooManyRequestsError].
//
// Returns [context.DeadlineExceeded] when the timeout is exceeded during the HTTP request to the Discord server.
func (wh *Webhook) Execute(m Message) error {
	if wh.client == nil {
		return fmt.Errorf("Webhook not inititalized: %w", ErrInvalidConfiguration)
	}
	wh.client.logger.Debug("message", "detail", fmt.Sprintf("%+v", m))
	dat, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if isActive, retryAfter := wh.client.rl.getOrReset(); isActive {
		return TooManyRequestsError{RetryAfter: retryAfter, Global: true}
	}
	wh.mu.Lock()
	defer wh.mu.Unlock()
	if isActive, retryAfter := wh.rl.getOrReset(); isActive {
		return TooManyRequestsError{RetryAfter: retryAfter}
	}
	wh.client.limiterGlobal.wait()
	wh.limiterAPI.wait()
	wh.limiterWebhook.wait()
	wh.client.logger.Debug("request", "url", wh.url, "body", string(dat))

	ctx, cancel := context.WithTimeout(context.Background(), wh.client.httpTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", wh.url, bytes.NewBuffer(dat))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := wh.client.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := wh.limiterAPI.updateFromHeader(resp.Header); err != nil {
		wh.client.logger.Error("Failed to update API limiter from header", "error", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	wh.client.logger.Debug("response", "url", wh.url, "status", resp.Status, "headers", resp.Header, "body", string(body))
	if resp.StatusCode >= http.StatusBadRequest {
		wh.client.logger.Warn("response", "url", wh.url, "status", resp.Status)
	} else {
		wh.client.logger.Info("response", "url", wh.url, "status", resp.Status)
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
		return TooManyRequestsError{
			RetryAfter: retryAfter, // Value from header is more reliable
			Global:     m.Global,
		}
	}
	if resp.StatusCode >= 400 {
		err := HTTPError{
			Status:  resp.StatusCode,
			Message: resp.Status,
		}
		return err
	}
	return nil
}

type tooManyRequestsResponse struct {
	Message    string  `json:"message,omitempty"`
	RetryAfter float64 `json:"retry_after,omitempty"`
	Global     bool    `json:"global,omitempty"`
}
