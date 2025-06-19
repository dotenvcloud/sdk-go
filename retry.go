package dotenv

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxRetries           int
	MinWaitTime          time.Duration
	MaxWaitTime          time.Duration
	RetryableHTTPMethods []string
	RetryableStatusCodes []int
}

// DefaultRetryConfig provides sensible defaults
var DefaultRetryConfig = RetryConfig{
	MaxRetries:           3,
	MinWaitTime:          1 * time.Second,
	MaxWaitTime:          30 * time.Second,
	RetryableHTTPMethods: []string{"GET", "HEAD", "OPTIONS", "DELETE"},
	RetryableStatusCodes: []int{429, 500, 502, 503, 504},
}

// doWithRetry performs request with retry logic
func (c *Client) doWithRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	retryConfig := DefaultRetryConfig

	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		// Clone request for each attempt
		if attempt > 0 {
			req = req.Clone(ctx)
		}

		resp, err = c.httpClient.Do(req)

		// Check if we should retry
		if !shouldRetry(req, resp, err, retryConfig) {
			return resp, err
		}

		// Close the response body if we're going to retry
		if resp != nil {
			resp.Body.Close()
		}

		// Calculate wait time
		waitTime := calculateWaitTime(attempt, resp, retryConfig)

		// Wait before retry
		select {
		case <-time.After(waitTime):
			// Continue to next attempt
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	return resp, err
}

// shouldRetry determines if request should be retried
func shouldRetry(req *http.Request, resp *http.Response, err error, config RetryConfig) bool {
	// Don't retry if context is cancelled
	if req.Context().Err() != nil {
		return false
	}

	// Retry on network errors
	if err != nil {
		return true
	}

	// Check if method is retryable
	methodRetryable := false
	for _, method := range config.RetryableHTTPMethods {
		if req.Method == method {
			methodRetryable = true
			break
		}
	}

	if !methodRetryable {
		return false
	}

	// Check if status code is retryable
	for _, code := range config.RetryableStatusCodes {
		if resp.StatusCode == code {
			return true
		}
	}

	return false
}

// calculateWaitTime calculates exponential backoff with jitter
func calculateWaitTime(attempt int, resp *http.Response, config RetryConfig) time.Duration {
	// Check for Retry-After header
	if resp != nil && resp.StatusCode == 429 {
		if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
			// Try to parse as seconds
			if seconds, err := strconv.Atoi(retryAfter); err == nil {
				return time.Duration(seconds) * time.Second
			}
			// Try to parse as HTTP date
			if t, err := time.Parse(time.RFC1123, retryAfter); err == nil {
				return time.Until(t)
			}
		}
	}

	// Exponential backoff with jitter
	waitTime := config.MinWaitTime * time.Duration(math.Pow(2, float64(attempt)))

	// Add jitter (up to 25% of wait time)
	jitter := time.Duration(rand.Int63n(int64(waitTime / 4)))
	waitTime = waitTime + jitter

	// Cap at max wait time
	if waitTime > config.MaxWaitTime {
		waitTime = config.MaxWaitTime
	}

	return waitTime
}
