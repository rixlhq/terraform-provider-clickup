package clickupclient

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// SetRetryPolicy configures the maximum retries and backoff limits for transient errors and rate limits.
func (c *Client) SetRetryPolicy(maxRetries int, minWait, maxWait time.Duration) {
	c.maxRetries = maxRetries
	c.minRetryWait = minWait
	c.maxRetryWait = maxWait
}

// isRetryableStatusCode reports whether an HTTP status code represents a rate-limit or transient gateway error.
func isRetryableStatusCode(code int) bool {
	return code == http.StatusTooManyRequests ||
		code == http.StatusBadGateway ||
		code == http.StatusServiceUnavailable ||
		code == http.StatusGatewayTimeout
}

// calculateBackoff computes sleep duration respecting Retry-After and X-Ratelimit-Reset headers.
func (c *Client) calculateBackoff(attempt int, headers http.Header) time.Duration {
	if headers != nil {
		if retryAfter := headers.Get("Retry-After"); retryAfter != "" {
			if secs, err := strconv.Atoi(retryAfter); err == nil && secs > 0 {
				dur := time.Duration(secs) * time.Second
				if dur > c.maxRetryWait {
					return c.maxRetryWait
				}
				return dur
			}
		}
		if resetVal := headers.Get("X-Ratelimit-Reset"); resetVal != "" {
			if epoch, err := strconv.ParseInt(resetVal, 10, 64); err == nil {
				resetTime := time.Unix(epoch, 0)
				wait := time.Until(resetTime)
				if wait > 0 && wait <= c.maxRetryWait {
					return wait
				}
			}
		}
	}

	backoff := c.minRetryWait * (1 << attempt)
	if backoff > c.maxRetryWait {
		return c.maxRetryWait
	}
	return backoff
}

// waitRetry sleeps for the calculated backoff duration or returns an error if the context is canceled.
func (c *Client) waitRetry(ctx context.Context, attempt int, headers http.Header) error {
	sleep := c.calculateBackoff(attempt, headers)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(sleep):
		return nil
	}
}
