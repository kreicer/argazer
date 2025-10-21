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

// SendJSON sends a JSON payload to the webhook URL
func (n *HTTPNotifier) SendJSON(ctx context.Context, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", n.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", UserAgent)

	n.logger.Debug("Sending HTTP notification")

	// Send request
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			n.logger.WithError(err).Warn("Failed to close response body")
		}
	}()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("failed to send message: status %d", resp.StatusCode)
	}

	return nil
}
