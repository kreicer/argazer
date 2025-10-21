package notification

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTelegramNotifier(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewTelegramNotifier("https://api.telegram.org/bot123/sendMessage", "12345", logger)

	require.NotNil(t, notifier)
	assert.Equal(t, "https://api.telegram.org/bot123/sendMessage", notifier.webhookURL)
	assert.Equal(t, "12345", notifier.chatID)
	assert.NotNil(t, notifier.httpClient)
	assert.NotNil(t, notifier.logger)
}

func TestTelegramNotifier_Send_Success(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "argazer/1.0", r.Header.Get("User-Agent"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTelegramNotifier(server.URL, "12345", logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test Subject", "Test message")
	require.NoError(t, err)
}

func TestTelegramNotifier_Send_WithSubject(t *testing.T) {
	receivedMessage := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err == nil {
			receivedMessage = payload["text"]
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTelegramNotifier(server.URL, "12345", logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "Subject", "Message")
	require.NoError(t, err)
	assert.Contains(t, receivedMessage, "Subject")
	assert.Contains(t, receivedMessage, "Message")
}

func TestTelegramNotifier_Send_EmptySubject(t *testing.T) {
	receivedMessage := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err == nil {
			receivedMessage = payload["text"]
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTelegramNotifier(server.URL, "12345", logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "", "Message only")
	require.NoError(t, err)
	assert.Equal(t, "Message only", receivedMessage)
}

func TestTelegramNotifier_Send_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTelegramNotifier(server.URL, "12345", logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test", "Message")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
}

func TestTelegramNotifier_Send_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTelegramNotifier(server.URL, "12345", logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := notifier.Send(ctx, "Test", "Message")
	require.Error(t, err)
}
