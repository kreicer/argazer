package notification

import "context"

// Notifier is a generic interface for sending notifications
type Notifier interface {
	Send(ctx context.Context, subject, message string) error
}
