package notification

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"

	"github.com/sirupsen/logrus"
)

// EmailNotifier handles sending notifications via Email
type EmailNotifier struct {
	smtpHost     string
	smtpPort     int
	smtpUsername string
	smtpPassword string
	from         string
	to           []string
	useTLS       bool
	logger       *logrus.Entry
}

// NewEmailNotifier creates a new Email notifier
func NewEmailNotifier(smtpHost string, smtpPort int, smtpUsername, smtpPassword, from string, to []string, useTLS bool, logger *logrus.Entry) *EmailNotifier {
	return &EmailNotifier{
		smtpHost:     smtpHost,
		smtpPort:     smtpPort,
		smtpUsername: smtpUsername,
		smtpPassword: smtpPassword,
		from:         from,
		to:           to,
		useTLS:       useTLS,
		logger:       logger,
	}
}

// Send sends an email notification (implements Notifier interface)
func (e *EmailNotifier) Send(ctx context.Context, subject, message string) error {
	// Prepare email headers and body
	body := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		e.from,
		strings.Join(e.to, ", "),
		subject,
		message,
	)

	addr := fmt.Sprintf("%s:%d", e.smtpHost, e.smtpPort)

	e.logger.WithFields(logrus.Fields{
		"smtp_host": e.smtpHost,
		"smtp_port": e.smtpPort,
		"from":      e.from,
		"to":        e.to,
		"subject":   subject,
	}).Debug("Sending email notification")

	var auth smtp.Auth
	if e.smtpUsername != "" && e.smtpPassword != "" {
		auth = smtp.PlainAuth("", e.smtpUsername, e.smtpPassword, e.smtpHost)
	}

	// Send email with TLS if enabled
	if e.useTLS {
		return e.sendWithTLS(addr, auth, []byte(body))
	}

	// Send without TLS
	err := smtp.SendMail(addr, auth, e.from, e.to, []byte(body))
	if err == nil {
		e.logger.WithField("to", e.to).Info("Successfully sent email notification")
	}
	return err
}

// sendWithTLS sends email with TLS encryption
func (e *EmailNotifier) sendWithTLS(addr string, auth smtp.Auth, body []byte) error {
	// Connect to SMTP server
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	// Start TLS
	tlsConfig := &tls.Config{
		ServerName: e.smtpHost,
	}
	if err := client.StartTLS(tlsConfig); err != nil {
		return fmt.Errorf("failed to start TLS: %w", err)
	}

	// Authenticate
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("failed to authenticate: %w", err)
		}
	}

	// Set sender
	if err := client.Mail(e.from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set recipients
	for _, to := range e.to {
		if err := client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to set recipient %s: %w", to, err)
		}
	}

	// Send email body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer w.Close()

	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("failed to write email body: %w", err)
	}

	e.logger.WithField("to", e.to).Info("Successfully sent email notification")
	return nil
}
