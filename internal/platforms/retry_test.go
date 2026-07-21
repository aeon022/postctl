package platforms

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

func fastRetryConfig() RetryConfig {
	return RetryConfig{MaxAttempts: 3, BaseDelay: 1 * time.Millisecond, MaxDelay: 5 * time.Millisecond}
}

func TestWithRetry_SucceedsFirstTry(t *testing.T) {
	calls := 0
	result, err := WithRetry(context.Background(), fastRetryConfig(), "test", func() (string, error) {
		calls++
		return "ok-id", nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != "ok-id" {
		t.Errorf("expected result %q, got %q", "ok-id", result)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestWithRetry_RetriesTransientThenSucceeds(t *testing.T) {
	calls := 0
	result, err := WithRetry(context.Background(), fastRetryConfig(), "test", func() (string, error) {
		calls++
		if calls < 2 {
			return "", fmt.Errorf("post failed (status 503): server error")
		}
		return "recovered-id", nil
	})
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if result != "recovered-id" {
		t.Errorf("expected result %q, got %q", "recovered-id", result)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
}

func TestWithRetry_StopsAfterMaxAttempts(t *testing.T) {
	calls := 0
	_, err := WithRetry(context.Background(), fastRetryConfig(), "test", func() (string, error) {
		calls++
		return "", fmt.Errorf("post failed (status 500): still failing")
	})
	if err == nil {
		t.Fatal("expected error after exhausting retries, got nil")
	}
	if calls != 3 {
		t.Errorf("expected exactly MaxAttempts=3 calls, got %d", calls)
	}
}

func TestWithRetry_DoesNotRetryPermanentError(t *testing.T) {
	calls := 0
	_, err := WithRetry(context.Background(), fastRetryConfig(), "test", func() (string, error) {
		calls++
		return "", fmt.Errorf("post failed (status 401): invalid token")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call for a non-retryable 401, got %d", calls)
	}
}

func TestWithRetry_DoesNotRetryNonHTTPPermanentError(t *testing.T) {
	calls := 0
	_, err := WithRetry(context.Background(), fastRetryConfig(), "test", func() (string, error) {
		calls++
		return "", errors.New("not authenticated with twitter")
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if calls != 1 {
		t.Errorf("expected 1 call for an unclassified error (treated as permanent), got %d", calls)
	}
}

func TestWithRetry_RetriesNetworkError(t *testing.T) {
	calls := 0
	netErr := &net.DNSError{Err: "no such host", Name: "api.example.com", IsTimeout: true}
	_, err := WithRetry(context.Background(), fastRetryConfig(), "test", func() (string, error) {
		calls++
		if calls < 3 {
			return "", fmt.Errorf("http request failed: %w", netErr)
		}
		return "ok-id", nil
	})
	if err != nil {
		t.Fatalf("expected eventual success, got: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestWithRetry_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	cfg := RetryConfig{MaxAttempts: 5, BaseDelay: 20 * time.Millisecond, MaxDelay: 100 * time.Millisecond}

	go func() {
		time.Sleep(5 * time.Millisecond)
		cancel()
	}()

	_, err := WithRetry(ctx, cfg, "test", func() (string, error) {
		calls++
		return "", fmt.Errorf("post failed (status 503): server error")
	})
	if err == nil {
		t.Fatal("expected error after context cancellation, got nil")
	}
	if calls >= cfg.MaxAttempts {
		t.Errorf("expected cancellation to cut retries short, got all %d attempts", calls)
	}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"429", fmt.Errorf("rate limited (status 429): slow down"), true},
		{"500", fmt.Errorf("server error (status 500): oops"), true},
		{"503", fmt.Errorf("unavailable (status 503): try later"), true},
		{"400", fmt.Errorf("bad request (status 400): invalid body"), false},
		{"401", fmt.Errorf("unauthorized (status 401): bad token"), false},
		{"404", fmt.Errorf("not found (status 404): no such post"), false},
		{"plain error", errors.New("no tweets to post"), false},
		{"context canceled", context.Canceled, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isRetryable(tc.err); got != tc.want {
				t.Errorf("isRetryable(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
