package utils

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

type RetryableHTTPRequestImpl struct {
	client *http.Client
	config RetryConfig
}

func NewRetryableHTTPRequest(config RetryConfig) *RetryableHTTPRequestImpl {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &RetryableHTTPRequestImpl{
		client: client,
		config: config,
	}
}

func (s *RetryableHTTPRequestImpl) RetryableHTTPRequest(ctx context.Context, url, method string, body io.Reader, headers []http.Header) (*http.Response, error) {

	var lastErr error

	for attempt := 0; attempt <= s.config.MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, body)

		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
		for _, header := range headers {
			req.Header.Set(header.Get("Key"), header.Get("Value"))
		}

		resp, err := s.client.Do(req)
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
		if attempt < s.config.MaxRetries {
			delay := time.Second * 2

			fmt.Printf("Retrying in %v... (attempt %d/%d)\n", delay, attempt+1, s.config.MaxRetries)

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
