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

func TestNewWebhookNotifier(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewWebhookNotifier("https://webhook.example.com/notify", logger)

	require.NotNil(t, notifier)
	assert.Equal(t, "https://webhook.example.com/notify", notifier.webhookURL)
	assert.NotNil(t, notifier.httpClient)
	assert.NotNil(t, notifier.logger)
}

func TestWebhookNotifier_Send_Success(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "argazer/1.0", r.Header.Get("User-Agent"))

		// Verify the payload structure
		var payload map[string]string
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Contains(t, payload, "subject")
		assert.Contains(t, payload, "message")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewWebhookNotifier(server.URL, logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test Subject", "Test message")
	require.NoError(t, err)
}

func TestWebhookNotifier_Send_WithSubjectAndMessage(t *testing.T) {
	var receivedPayload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewWebhookNotifier(server.URL, logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "Subject", "Message")
	require.NoError(t, err)
	assert.Equal(t, "Subject", receivedPayload["subject"])
	assert.Equal(t, "Message", receivedPayload["message"])
}

func TestWebhookNotifier_Send_EmptySubject(t *testing.T) {
	var receivedPayload map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewWebhookNotifier(server.URL, logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "", "Message only")
	require.NoError(t, err)
	assert.Equal(t, "", receivedPayload["subject"])
	assert.Equal(t, "Message only", receivedPayload["message"])
}

func TestWebhookNotifier_Send_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewWebhookNotifier(server.URL, logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test", "Message")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
}

func TestWebhookNotifier_Send_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewWebhookNotifier(server.URL, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := notifier.Send(ctx, "Test", "Message")
	require.Error(t, err)
}

func TestWebhookNotifier_Send_AcceptsAllSuccess(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		shouldFail bool
	}{
		{"200 OK", http.StatusOK, false},
		{"201 Created", http.StatusCreated, false},
		{"202 Accepted", http.StatusAccepted, false},
		{"204 No Content", http.StatusNoContent, false},
		{"400 Bad Request", http.StatusBadRequest, true},
		{"404 Not Found", http.StatusNotFound, true},
		{"500 Internal Error", http.StatusInternalServerError, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			logger := logrus.NewEntry(logrus.New())
			notifier := NewWebhookNotifier(server.URL, logger)

			ctx := context.Background()
			err := notifier.Send(ctx, "Test", "Message")

			if tt.shouldFail {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
