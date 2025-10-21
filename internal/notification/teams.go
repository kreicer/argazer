package notification

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"
)

// teamsMessageCard represents the JSON payload for Microsoft Teams webhooks
type teamsMessageCard struct {
	Type       string `json:"@type"`
	Context    string `json:"@context"`
	Summary    string `json:"summary"`
	ThemeColor string `json:"themeColor"`
	Title      string `json:"title"`
	Text       string `json:"text"`
}

// TeamsNotifier handles sending notifications via Microsoft Teams
type TeamsNotifier struct {
	*HTTPNotifier
}

// NewTeamsNotifier creates a new Microsoft Teams notifier
func NewTeamsNotifier(webhookURL string, logger *logrus.Entry) *TeamsNotifier {
	return NewTeamsNotifierWithClient(webhookURL, nil, logger)
}

// NewTeamsNotifierWithClient creates a new Microsoft Teams notifier with a custom HTTP client
func NewTeamsNotifierWithClient(webhookURL string, httpClient *http.Client, logger *logrus.Entry) *TeamsNotifier {
	return &TeamsNotifier{
		HTTPNotifier: NewHTTPNotifier(webhookURL, httpClient, logger),
	}
}

// Send sends a notification via Microsoft Teams (implements Notifier interface)
func (n *TeamsNotifier) Send(ctx context.Context, subject, message string) error {
	// Prepare the payload using MessageCard format for better compatibility
	payload := teamsMessageCard{
		Type:       "MessageCard",
		Context:    "https://schema.org/extensions",
		Summary:    subject,
		ThemeColor: "0078D7",
		Title:      subject,
		Text:       message,
	}

	if err := n.SendJSON(ctx, payload); err != nil {
		return err
	}

	n.logger.Info("Successfully sent Microsoft Teams notification")
	return nil
}
