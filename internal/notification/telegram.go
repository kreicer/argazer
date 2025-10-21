package notification

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// telegramPayload represents the JSON payload for Telegram webhooks
type telegramPayload struct {
	ChatID string `json:"chat_id"`
	Text   string `json:"text"`
}

// TelegramNotifier handles sending notifications via Telegram
type TelegramNotifier struct {
	*HTTPNotifier
	chatID string
}

// NewTelegramNotifier creates a new Telegram notifier
func NewTelegramNotifier(webhookURL, chatID string, logger *logrus.Entry) *TelegramNotifier {
	return NewTelegramNotifierWithClient(webhookURL, chatID, nil, logger)
}

// NewTelegramNotifierWithClient creates a new Telegram notifier with a custom HTTP client
func NewTelegramNotifierWithClient(webhookURL, chatID string, httpClient *http.Client, logger *logrus.Entry) *TelegramNotifier {
	return &TelegramNotifier{
		HTTPNotifier: NewHTTPNotifier(webhookURL, httpClient, logger),
		chatID:       chatID,
	}
}

// Send sends a notification via Telegram (implements Notifier interface)
func (n *TelegramNotifier) Send(ctx context.Context, subject, message string) error {
	// Combine subject and message for Telegram
	fullMessage := message
	if subject != "" {
		fullMessage = fmt.Sprintf("%s\n\n%s", subject, message)
	}

	payload := telegramPayload{
		ChatID: n.chatID,
		Text:   fullMessage,
	}

	n.logger.WithField("chat_id", n.chatID).Debug("Sending Telegram notification")

	if err := n.SendJSON(ctx, payload); err != nil {
		return err
	}

	n.logger.WithField("chat_id", n.chatID).Info("Successfully sent Telegram notification")
	return nil
}
