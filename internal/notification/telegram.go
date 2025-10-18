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

// TelegramNotifier handles sending notifications via Telegram
type TelegramNotifier struct {
	webhookURL string
	chatID     string
	httpClient *http.Client
	logger     *logrus.Entry
}

// NewTelegramNotifier creates a new Telegram notifier
func NewTelegramNotifier(webhookURL, chatID string, logger *logrus.Entry) *TelegramNotifier {
	return &TelegramNotifier{
		webhookURL: webhookURL,
		chatID:     chatID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// Send sends a notification via Telegram (implements Notifier interface)
func (n *TelegramNotifier) Send(ctx context.Context, subject, message string) error {
	// Combine subject and message for Telegram
	fullMessage := message
	if subject != "" {
		fullMessage = fmt.Sprintf("*%s*\n\n%s", subject, message)
	}
	return n.sendMessage(ctx, fullMessage)
}

// sendMessage sends a message to the configured Telegram chat
func (n *TelegramNotifier) sendMessage(ctx context.Context, message string) error {
	// Prepare the payload
	payload := map[string]string{
		"chat_id":    n.chatID,
		"text":       message,
		"parse_mode": "Markdown",
	}

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
	req.Header.Set("User-Agent", "argazer/1.0")

	n.logger.WithFields(logrus.Fields{
		"chat_id": n.chatID,
	}).Debug("Sending Telegram notification")

	// Send request
	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send message: status %d", resp.StatusCode)
	}

	n.logger.WithField("chat_id", n.chatID).Info("Successfully sent Telegram notification")
	return nil
}
