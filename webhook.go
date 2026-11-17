package dhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"sync"
	"time"
)

const (
	retryAfterTooManyRequestDefault = 60 * time.Second
	retryAfterMax                   = 24 * time.Hour
)

var webhookTokenRE = regexp.MustCompile(`(/webhooks/\d+/)[^/?#]+`)

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

// ErrInvalidConfiguration represents an invalid configuration, e.g. a negative HTTP timeout.
var ErrInvalidConfiguration = errors.New("invalid configuration")

// Webhook represents a Discord webhook.
// Webhooks are safe for concurrent use by multiple goroutines.
type Webhook struct {
	client *Client
	url    string
	state  *webhookState
}

// webhookState holds the rate limit state shared by all webhooks for the same Discord webhook.
type webhookState struct {
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
// Options can be provided through opt or opt can be nil for executing without options.
// Execute will only return a response from Discord (e.g. the message created) when the Wait option is enabled.
//
// Execute will automatically comply with Discord's rate limits by waiting
// until there is a free slot to post the message if necessary.
//
// Execute will check that a message is not empty, but not do a full validation.
// A full validation can be performed with [Message.Validate].
//
// Common errors returned:
//   - [HTTPError]: Discord returned HTTP status codes of 400 or above (except 429)
//   - [TooManyRequestsError]: Discord returned status HTTP status code 429
//   - [context.DeadlineExceeded]: Timeout is exceeded during the HTTP request to Discord
func (wh *Webhook) Execute(message Message, opt *WebhookExecuteOptions) ([]byte, error) {
	if wh.client == nil {
		return nil, fmt.Errorf("Webhook not initialized: %w", ErrInvalidConfiguration)
	}
	wh.client.logger.Debug("message", "detail", fmt.Sprintf("%+v", message))
	if message.Content == "" && len(message.Embeds) == 0 {
		return nil, fmt.Errorf("message must have Content or Embed: %w", ErrInvalidMessage)
	}
	dat, err := json.Marshal(message)
	if err != nil {
		return nil, err
	}
	if isActive, retryAfter := wh.client.rl.getOrReset(); isActive {
		return nil, TooManyRequestsError{RetryAfter: retryAfter, Global: true}
	}
	wh.state.mu.Lock()
	defer wh.state.mu.Unlock()
	if isActive, retryAfter := wh.state.rl.getOrReset(); isActive {
		return nil, TooManyRequestsError{RetryAfter: retryAfter}
	}
	wh.client.limiterGlobal.wait()
	wh.state.limiterAPI.wait()
	wh.state.limiterWebhook.wait()
	// Check again, because a global 429 may have occurred while waiting.
	if isActive, retryAfter := wh.client.rl.getOrReset(); isActive {
		return nil, TooManyRequestsError{RetryAfter: retryAfter, Global: true}
	}

	ctx, cancel := context.WithTimeout(context.Background(), wh.client.httpTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", wh.url, bytes.NewBuffer(dat))
	if err != nil {
		return nil, redactError(err)
	}
	if opt != nil && opt.Wait {
		q := req.URL.Query()
		q.Set("wait", "true")
		req.URL.RawQuery = q.Encode()
	}
	logURL := redactURL(req.URL.String())
	req.Header.Set("Content-Type", "application/json")
	wh.client.logger.Debug("request", "url", logURL, "body", string(dat))
	resp, err := wh.client.httpClient.Do(req)
	if err != nil {
		return nil, redactError(err)
	}
	defer resp.Body.Close()
	if err := wh.state.limiterAPI.updateFromHeader(resp.Header); err != nil {
		wh.client.logger.Error("Failed to update API limiter from header", "error", err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	wh.client.logger.Debug("response", "url", logURL, "status", resp.Status, "headers", resp.Header, "body", string(body))
	if resp.StatusCode >= http.StatusBadRequest {
		wh.client.logger.Warn("response", "url", logURL, "status", resp.Status)
	} else {
		wh.client.logger.Info("response", "url", logURL, "status", resp.Status)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		var m tooManyRequestsResponse
		if err := json.Unmarshal(body, &m); err != nil {
			wh.client.logger.Warn("Failed to parse 429 response body", "error", err)
		}
		retryAfter, err := parseRetryAfter(resp.Header, m)
		if err != nil {
			wh.client.logger.Warn("Failed to parse retry after. Assuming default", "error", err)
		}
		wh.state.rl.set(retryAfter)
		if m.Global {
			wh.client.rl.set(retryAfter)
		}
		return body, TooManyRequestsError{
			RetryAfter: retryAfter,
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

// redactURL returns url with the webhook token replaced.
func redactURL(url string) string {
	return webhookTokenRE.ReplaceAllString(url, "${1}REDACTED")
}

// redactError removes the webhook token from the URL in err.
func redactError(err error) error {
	var ue *url.Error
	if errors.As(err, &ue) {
		ue.URL = redactURL(ue.URL)
	}
	return err
}

// parseRetryAfter returns how long to wait after a 429 response.
func parseRetryAfter(h http.Header, m tooManyRequestsResponse) (time.Duration, error) {
	secs := math.Ceil(m.RetryAfter)
	x, err := strconv.Atoi(h.Get("Retry-After"))
	if err != nil || x <= 0 {
		if secs <= 0 {
			return retryAfterTooManyRequestDefault, errors.New("no valid retry after in response")
		}
		x = 0
	}
	secs = min(max(secs, float64(x)), retryAfterMax.Seconds())
	return time.Duration(secs) * time.Second, nil
}
