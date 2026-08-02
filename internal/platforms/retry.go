package platforms

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"
)

// RetryConfig controls how WithRetry retries a transient-failing operation.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultRetryConfig is postctl's standard retry policy for platform Post calls:
// up to 3 attempts with exponential backoff (roughly 1s, 2s) capped at 30s, plus jitter.
var DefaultRetryConfig = RetryConfig{
	MaxAttempts: 3,
	BaseDelay:   1 * time.Second,
	MaxDelay:    30 * time.Second,
}

// WithRetry runs op, retrying with exponential backoff + jitter while the returned
// error is classified as transient (network errors, timeouts, HTTP 429/5xx) and
// attempts remain. It stops immediately on context cancellation or a non-retryable
// error (e.g. a 4xx auth/validation failure), so a bad request fails fast instead
// of retrying something that will never succeed.
//
// This is the single, centralized retry path for all platforms — individual
// Platform implementations should not roll their own generic retry loops for
// network/rate-limit errors. A platform may still retry its own narrow,
// API-specific transient conditions internally (e.g. Threads waiting for
// container indexing) since that is a different failure mode than "the request
// failed, try again".
func WithRetry(ctx context.Context, cfg RetryConfig, platformName string, op func() (string, error)) (string, error) {
	var lastErr error
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		result, err := op()
		if err == nil {
			return result, nil
		}
		lastErr = err

		if !isRetryable(err) || attempt == cfg.MaxAttempts {
			break
		}

		delay := backoffDelay(cfg, attempt)
		Log("[RETRY] %s: attempt %d/%d failed (%v), retrying in %s...", platformName, attempt, cfg.MaxAttempts, err, delay.Round(time.Millisecond))

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(delay):
		}
	}
	return "", lastErr
}

// backoffDelay returns an exponential delay for the given attempt (1-indexed),
// capped at cfg.MaxDelay, with up to 50% random jitter added to avoid thundering
// herds when several posts fail around the same time.
func backoffDelay(cfg RetryConfig, attempt int) time.Duration {
	d := time.Duration(float64(cfg.BaseDelay) * math.Pow(2, float64(attempt-1)))
	if d > cfg.MaxDelay {
		d = cfg.MaxDelay
	}
	if d <= 0 {
		return 0
	}
	jitter := time.Duration(rand.Int63n(int64(d)/2 + 1))
	return d + jitter
}

// isRetryable decides whether an error from a platform call is worth retrying.
// Network-level failures (timeouts, connection resets, DNS lookup errors) and
// HTTP 429/5xx responses are treated as transient; everything else — 4xx auth or
// validation errors, decode errors, "not authenticated", etc. — is permanent and
// retrying it would just waste time and hit rate limits harder.
func isRetryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	if code, ok := parseHTTPStatus(err.Error()); ok {
		return code == http.StatusTooManyRequests || code >= 500
	}

	return false
}

// parseHTTPStatus extracts an HTTP status code from platform error messages,
// which consistently follow the "... (status 429): ..." / "...status 503: ..."
// convention used across every platform implementation in this package.
func parseHTTPStatus(msg string) (int, bool) {
	idx := strings.Index(msg, "status ")
	if idx == -1 {
		return 0, false
	}
	rest := msg[idx+len("status "):]
	var code int
	if _, err := fmt.Sscanf(rest, "%d", &code); err != nil {
		return 0, false
	}
	return code, true
}
