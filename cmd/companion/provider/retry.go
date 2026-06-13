package provider

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// MaxRetries is the number of times SendWithRetry re-attempts the connection +
// header phase after the initial try (so up to MaxRetries+1 total attempts).
const MaxRetries = 10

const maxBackoff = 15 * time.Second

// RetryInfo describes a backoff about to happen.
type RetryInfo struct {
	Attempt int
	Max     int
	Delay   time.Duration
	Err     error
}

// RetryNotify is a callback invoked before each backoff sleep.
type RetryNotify func(RetryInfo)

type retryNotifyKey struct{}

// WithRetryNotify attaches a callback that SendWithRetry invokes before each
// backoff sleep, so the agent can surface a transient "retrying (n/m)" status.
func WithRetryNotify(ctx context.Context, fn RetryNotify) context.Context {
	if fn == nil {
		return ctx
	}
	return context.WithValue(ctx, retryNotifyKey{}, fn)
}

func retryNotifyFromContext(ctx context.Context) RetryNotify {
	fn, _ := ctx.Value(retryNotifyKey{}).(RetryNotify)
	return fn
}

// RetryableStatus reports whether a backoff can plausibly recover from status s:
// 408 (request timeout), 429 (rate limit) and 5xx. Other 4xx are caller/config
// problems retrying can't fix.
func RetryableStatus(s int) bool {
	return s == http.StatusRequestTimeout || s == http.StatusTooManyRequests || (s >= 500 && s <= 599)
}

func transientErr(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	return true
}

// IsConnReset reports whether err is a connection-level drop (peer reset,
// truncated body, closed socket) as opposed to a protocol or caller error.
func IsConnReset(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) ||
		errors.Is(err, net.ErrClosed) ||
		errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNABORTED) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr)
}

func backoffDelay(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > maxBackoff {
			return maxBackoff
		}
		return retryAfter
	}
	d := time.Duration(1<<(attempt-1)) * 500 * time.Millisecond
	if d > maxBackoff {
		d = maxBackoff
	}
	return d + time.Duration(rand.Intn(250))*time.Millisecond
}

func parseRetryAfter(resp *http.Response) time.Duration {
	v := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	return 0
}

// SendWithRetry POSTs a streaming request built by newReq and returns the OK
// response. It retries the connection+header phase up to MaxRetries times on
// transient network errors and retryable statuses with capped exponential
// backoff + jitter, honoring Retry-After.
func SendWithRetry(ctx context.Context, httpClient *http.Client, provName, keyEnv string, newReq func(context.Context) (*http.Request, error)) (*http.Response, error) {
	notify := retryNotifyFromContext(ctx)
	var lastErr error
	var retryAfter time.Duration

	for attempt := 0; attempt <= MaxRetries; attempt++ {
		if attempt > 0 {
			delay := backoffDelay(attempt, retryAfter)
			if notify != nil {
				notify(RetryInfo{Attempt: attempt, Max: MaxRetries, Delay: delay, Err: lastErr})
			}
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}
		retryAfter = 0

		req, err := newReq(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s: build request: %w", provName, err)
		}
		resp, err := httpClient.Do(req)
		if err != nil {
			if !transientErr(err) {
				return nil, fmt.Errorf("%s: request failed: %w", provName, err)
			}
			lastErr = fmt.Errorf("%s: request failed: %w", provName, err)
			continue
		}
		if resp.StatusCode == http.StatusOK {
			return resp, nil
		}

		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		retryAfter = parseRetryAfter(resp)
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil, &AuthError{Provider: provName, KeyEnv: keyEnv, Status: resp.StatusCode}
		}
		apiErr := &APIError{Provider: provName, Status: resp.StatusCode, Body: strings.TrimSpace(string(msg))}
		if !RetryableStatus(resp.StatusCode) {
			return nil, apiErr
		}
		lastErr = apiErr
	}
	return nil, lastErr
}
