package notification

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// slackPayload represents the JSON payload for Slack webhooks
type slackPayload struct {
	Text string `json:"text"`
}

// SlackNotifier handles sending notifications via Slack
type SlackNotifier struct {
	*HTTPNotifier
}

// NewSlackNotifier creates a new Slack notifier with an optional HTTP client
func NewSlackNotifier(webhookURL string, logger *logrus.Entry) *SlackNotifier {
	return NewSlackNotifierWithClient(webhookURL, nil, logger)
}

// NewSlackNotifierWithClient creates a new Slack notifier with a custom HTTP client
func NewSlackNotifierWithClient(webhookURL string, httpClient *http.Client, logger *logrus.Entry) *SlackNotifier {
	return &SlackNotifier{
		HTTPNotifier: NewHTTPNotifier(webhookURL, httpClient, logger),
	}
}

// Send sends a notification via Slack (implements Notifier interface)
func (n *SlackNotifier) Send(ctx context.Context, subject, message string) error {
	// Combine subject and message for Slack with markdown formatting
	fullMessage := message
	if subject != "" {
		fullMessage = fmt.Sprintf("*%s*\n\n%s", subject, message)
	}

	payload := slackPayload{
		Text: fullMessage,
	}

	if err := n.SendJSON(ctx, payload); err != nil {
		return err
	}

	n.logger.Info("Successfully sent Slack notification")
	return nil
}
