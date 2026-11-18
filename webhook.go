package dhook

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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
// It is held for the whole execution of a request.
type webhookState struct {
	sem            chan struct{} // capacity 1; works as context aware mutex
	rl             rateLimited
	limiterAPI     limiterAPI
	limiterWebhook *limiter
}

// lock acquires the state or returns ctx.Err() when ctx is done first.
func (st *webhookState) lock(ctx context.Context) error {
	select {
	case st.sem <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// unlock releases the state.
func (st *webhookState) unlock() {
	<-st.sem
}

type WebhookExecuteOptions struct {
	// Waits for server confirmation of message send before response
	// and returns the created message body.
	Wait bool
}

// Execute posts a message to the configured webhook similar to ExecuteContext,
// but is not context aware.
func (wh *Webhook) Execute(message Message, opt *WebhookExecuteOptions) ([]byte, error) {
	return wh.ExecuteContext(context.Background(), message, opt)
}

// ExecuteContext posts a message to the webhook, waiting for rate limits if needed.
// The response is only returned when opt.Wait is enabled. opt can be nil.
//
// ctx cancels only the waiting. A sent request is completed, bounded by the HTTP timeout.
//
// The message is only checked for being not empty. Use [Message.Validate] for a full check.
//
// Common errors returned:
//   - [HTTPError]: Discord returned status 400 or above (except 429)
//   - [TooManyRequestsError]: Discord returned status 429
//   - [context.Canceled]: ctx was canceled before sending
//   - [context.DeadlineExceeded]: ctx deadline exceeded before sending, or HTTP timeout exceeded
func (wh *Webhook) ExecuteContext(ctx context.Context, message Message, opt *WebhookExecuteOptions) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
	if err := wh.state.lock(ctx); err != nil {
		return nil, err
	}
	defer wh.state.unlock()
	if isActive, retryAfter := wh.state.rl.getOrReset(); isActive {
		return nil, TooManyRequestsError{RetryAfter: retryAfter}
	}
	if err := wh.client.limiterGlobal.wait(ctx); err != nil {
		return nil, err
	}
	if err := wh.state.limiterAPI.wait(ctx); err != nil {
		return nil, err
	}
	if err := wh.state.limiterWebhook.wait(ctx); err != nil {
		return nil, err
	}
	// Check again, because a global 429 may have occurred while waiting.
	if isActive, retryAfter := wh.client.rl.getOrReset(); isActive {
		return nil, TooManyRequestsError{RetryAfter: retryAfter, Global: true}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), wh.client.httpTimeout)
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
		if ctx.Err() != nil {
			wh.state.limiterAPI.consume() // request may have been canceled after reaching Discord
		}
		return nil, redactError(err)
	}
	defer resp.Body.Close()
	if err := wh.state.limiterAPI.updateFromHeader(resp.Header); err != nil {
		wh.client.logger.Error("Failed to update API limiter from header", "error", err)
	}
	if resp.StatusCode == http.StatusTooManyRequests {
		// Return early to ensure the rate limit is recorded even when the body can not be read.
		wh.client.logger.Warn("response", "url", logURL, "status", resp.Status, "headers", resp.Header)
		retryAfter, global, err := parseTooManyRequests(resp.Header)
		if err != nil {
			wh.client.logger.Warn("Failed to parse retry after. Assuming default", "error", err)
		}
		wh.state.rl.set(retryAfter)
		if global {
			wh.client.rl.set(retryAfter)
		}
		return nil, TooManyRequestsError{RetryAfter: retryAfter, Global: global}
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
	if resp.StatusCode >= 400 {
		err := HTTPError{
			Status:  resp.StatusCode,
			Message: resp.Status,
		}
		return nil, err
	}
	return body, nil
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

// parseTooManyRequests returns how long to wait after a 429 response
// and whether the rate limit is global.
// A 429 without Discord rate limit headers (e.g. from Cloudflare) is treated as global.
// Retry-After can be given in seconds or as an HTTP date.
func parseTooManyRequests(h http.Header) (time.Duration, bool, error) {
	scope := h.Get("X-RateLimit-Scope")
	isGlobal := h.Get("X-RateLimit-Global")
	hasDiscordHeader := scope != "" || isGlobal != "" || h.Get("X-RateLimit-Bucket") != ""
	global := isGlobal == "true" || scope == "global" || !hasDiscordHeader
	v := h.Get("Retry-After")
	if secs, err := strconv.Atoi(v); (err == nil || errors.Is(err, strconv.ErrRange)) && secs > 0 {
		secs = min(secs, int(retryAfterMax.Seconds()))
		return time.Duration(secs) * time.Second, global, nil
	}
	if t, err := http.ParseTime(v); err == nil {
		now := time.Now()
		if sd, err := http.ParseTime(h.Get("Date")); err == nil {
			now = sd // server time; avoids clock skew
		}
		if d := roundUpDuration(t.Sub(now), time.Second); d > 0 {
			return min(d, retryAfterMax), global, nil
		}
	}
	return retryAfterTooManyRequestDefault, global, errors.New("no valid retry after in response")
}
