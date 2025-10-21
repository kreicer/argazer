package notification

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

// webhookPayload represents the JSON payload for generic webhooks
type webhookPayload struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// WebhookNotifier handles sending notifications via generic webhook
type WebhookNotifier struct {
	*HTTPNotifier
}

// NewWebhookNotifier creates a new generic webhook notifier
func NewWebhookNotifier(webhookURL string, logger *logrus.Entry) *WebhookNotifier {
	return NewWebhookNotifierWithClient(webhookURL, nil, logger)
}

// NewWebhookNotifierWithClient creates a new generic webhook notifier with a custom HTTP client
func NewWebhookNotifierWithClient(webhookURL string, httpClient *http.Client, logger *logrus.Entry) *WebhookNotifier {
	return &WebhookNotifier{
		HTTPNotifier: NewHTTPNotifier(webhookURL, httpClient, logger),
	}
}

// Send sends a notification via generic webhook (implements Notifier interface)
func (n *WebhookNotifier) Send(ctx context.Context, subject, message string) error {
	// Prepare a generic payload that can work with most webhook systems
	payload := webhookPayload{
		Subject: subject,
		Message: message,
	}

	if err := n.SendJSON(ctx, payload); err != nil {
		return err
	}

	n.logger.Info("Successfully sent webhook notification")
	return nil
}
