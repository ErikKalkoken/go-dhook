package dhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	retryAfterTooManyRequestDefault = 60 * time.Second
	webhookRateLimitPeriod          = 60 * time.Second
	webhookRateLimitRequests        = 30
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
// HTTP status codes of 400 or above are returns as [HTTPError],
// except for 429s, which are returned as [TooManyRequestsError].
func (wh *Webhook) Execute(m Message) error {
	slog.Debug("message", "detail", fmt.Sprintf("%+v", m))
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
	slog.Debug("request", "url", wh.url, "body", string(dat))
	resp, err := wh.client.httpClient.Post(wh.url, "application/json", bytes.NewBuffer(dat))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if err := wh.limiterAPI.updateFromHeader(resp.Header); err != nil {
		slog.Error("Failed to update API limiter from header", "error", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	slog.Debug("response", "url", wh.url, "status", resp.Status, "headers", resp.Header, "body", string(body))
	if resp.StatusCode >= http.StatusBadRequest {
		slog.Warn("response", "url", wh.url, "status", resp.Status)
	} else {
		slog.Info("response", "url", wh.url, "status", resp.Status)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		var m tooManyRequestsResponse
		if err := json.Unmarshal(body, &m); err != nil {
			slog.Warn("Failed to parse 429 response body", "error", err)
		}
		retryAfter := retryAfterTooManyRequestDefault
		s := resp.Header.Get("Retry-After")
		if s != "" {
			x, err := strconv.Atoi(s)
			if err != nil {
				slog.Warn("Failed to parse retry after. Assuming default", "error", err)
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
