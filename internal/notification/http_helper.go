package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// UserAgent is the User-Agent header value used for all HTTP requests
	UserAgent = "argazer/1.0"
	// DefaultHTTPTimeout is the default timeout for HTTP clients
	DefaultHTTPTimeout = 30 * time.Second
	// DefaultMaxRetries is the default maximum number of retry attempts
	DefaultMaxRetries = 3
	// DefaultInitialRetryDelay is the initial delay before retrying
	DefaultInitialRetryDelay = 1 * time.Second
)

// HTTPNotifier provides common functionality for HTTP-based notifiers
type HTTPNotifier struct {
	webhookURL string
	httpClient *http.Client
	logger     *logrus.Entry
}

// NewHTTPNotifier creates a new HTTP notifier with the given webhook URL and optional HTTP client
func NewHTTPNotifier(webhookURL string, httpClient *http.Client, logger *logrus.Entry) *HTTPNotifier {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: DefaultHTTPTimeout,
		}
	}

	return &HTTPNotifier{
		webhookURL: webhookURL,
		httpClient: httpClient,
		logger:     logger,
	}
}

// SendJSON sends a JSON payload to the webhook URL with retry logic
func (n *HTTPNotifier) SendJSON(ctx context.Context, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Retry logic with exponential backoff
	var lastErr error
	for attempt := 0; attempt < DefaultMaxRetries; attempt++ {
		if attempt > 0 {
			// Calculate exponential backoff with jitter: delay = base * 2^(attempt-1) + jitter
			delay := DefaultInitialRetryDelay * time.Duration(1<<uint(attempt-1))
			// Add up to 20% jitter to prevent thundering herd
			jitter := time.Duration(float64(delay) * 0.2 * (0.5 + (float64(time.Now().UnixNano()%100) / 100.0)))
			delay += jitter

			n.logger.WithFields(logrus.Fields{
				"attempt": attempt + 1,
				"delay":   delay,
			}).Debug("Retrying HTTP notification after delay")

			select {
			case <-time.After(delay):
				// Continue with retry
			case <-ctx.Done():
				return fmt.Errorf("context cancelled during retry: %w", ctx.Err())
			}
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, "POST", n.webhookURL, bytes.NewBuffer(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", UserAgent)

		if attempt == 0 {
			n.logger.Debug("Sending HTTP notification")
		}

		// Send request
		resp, err := n.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			n.logger.WithError(lastErr).WithField("attempt", attempt+1).Warn("HTTP request failed, will retry")
			continue // Retry on network errors
		}

		// Close response body in defer
		func() {
			if err := resp.Body.Close(); err != nil {
				n.logger.WithError(err).Warn("Failed to close response body")
			}
		}()

		// Check response status
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			// Success!
			if attempt > 0 {
				n.logger.WithField("attempts", attempt+1).Info("HTTP notification succeeded after retry")
			}
			return nil
		}

		// Check if error is retryable (5xx server errors or 429 rate limit)
		if resp.StatusCode >= 500 || resp.StatusCode == 429 {
			lastErr = fmt.Errorf("server returned retryable status %d", resp.StatusCode)
			n.logger.WithFields(logrus.Fields{
				"status":  resp.StatusCode,
				"attempt": attempt + 1,
			}).Warn("Server error, will retry")
			continue // Retry on server errors
		}

		// Non-retryable error (4xx client errors except 429)
		return fmt.Errorf("failed to send message: status %d", resp.StatusCode)
	}

	// All retries exhausted
	return fmt.Errorf("failed after %d attempts: %w", DefaultMaxRetries, lastErr)
}
