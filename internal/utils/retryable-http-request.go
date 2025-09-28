package utils

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"time"
)

type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

func RetryableHTTPRequest(ctx context.Context, url string) (*http.Response, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	var lastErr error

	for attempt := 0; attempt <= 3; attempt++ {
		// Create request with context
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			fmt.Printf("Request attempt %d failed: %v\n", attempt+1, err)
		} else if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			// Success
			return resp, nil
		} else if resp.StatusCode == http.StatusTooManyRequests {
			// 429 Too Many Requests - retry with backoff
			resp.Body.Close()
			lastErr = fmt.Errorf("rate limited: %d %s", resp.StatusCode, resp.Status)
			fmt.Printf("Request attempt %d failed with status %d (rate limited)\n", attempt+1, resp.StatusCode)
		} else {
			// All other errors (4xx, 5xx) - don't retry
			resp.Body.Close()
			return nil, fmt.Errorf("request failed %d: %s", resp.StatusCode, resp.Status)
		}

		// max attempts for now = 3
		if attempt < 3 {
			delay := min(time.Duration(math.Pow(2, float64(attempt)))*1*time.Second, 30*time.Second)

			fmt.Printf("Retrying in %v... (attempt %d/%d)\n", delay, attempt+1, 3)

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				// Continue to next attempt
			}
		}
	}

	return nil, fmt.Errorf("all retry attempts failed, last error: %w", lastErr)
}
