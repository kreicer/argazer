package notification

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEmailNotifier(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"smtp.example.com",
		587,
		"username",
		"password",
		"sender@example.com",
		[]string{"recipient@example.com"},
		true,
		logger,
	)

	require.NotNil(t, notifier)
	assert.Equal(t, "smtp.example.com", notifier.smtpHost)
	assert.Equal(t, 587, notifier.smtpPort)
	assert.Equal(t, "username", notifier.smtpUsername)
	assert.Equal(t, "password", notifier.smtpPassword)
	assert.Equal(t, "sender@example.com", notifier.from)
	assert.Equal(t, []string{"recipient@example.com"}, notifier.to)
	assert.True(t, notifier.useTLS)
	assert.NotNil(t, notifier.logger)
}

func TestEmailNotifier_Send_InvalidSMTP(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"invalid-smtp-server-that-does-not-exist.example.com",
		587,
		"",
		"",
		"sender@example.com",
		[]string{"recipient@example.com"},
		false,
		logger,
	)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test Subject", "Test message")
	// Should fail because the SMTP server doesn't exist
	require.Error(t, err)
}

func TestEmailNotifier_Send_WithTLS_InvalidSMTP(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"invalid-smtp-server-that-does-not-exist.example.com",
		587,
		"username",
		"password",
		"sender@example.com",
		[]string{"recipient@example.com"},
		true,
		logger,
	)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test Subject", "Test message")
	// Should fail because the SMTP server doesn't exist
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to connect to SMTP server")
}

func TestEmailNotifier_Send_MultipleRecipients(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"invalid.example.com",
		587,
		"",
		"",
		"sender@example.com",
		[]string{"recipient1@example.com", "recipient2@example.com", "recipient3@example.com"},
		false,
		logger,
	)

	// Just test that the notifier is created correctly with multiple recipients
	assert.Equal(t, 3, len(notifier.to))
	assert.Equal(t, "recipient1@example.com", notifier.to[0])
	assert.Equal(t, "recipient2@example.com", notifier.to[1])
	assert.Equal(t, "recipient3@example.com", notifier.to[2])
}

func TestEmailNotifier_Send_NoAuth(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewEmailNotifier(
		"invalid.example.com",
		587,
		"", // No username
		"", // No password
		"sender@example.com",
		[]string{"recipient@example.com"},
		false,
		logger,
	)

	require.NotNil(t, notifier)
	assert.Equal(t, "", notifier.smtpUsername)
	assert.Equal(t, "", notifier.smtpPassword)
}
