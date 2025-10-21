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

func TestNewTeamsNotifier(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	notifier := NewTeamsNotifier("https://outlook.office.com/webhook/TEST", logger)

	require.NotNil(t, notifier)
	assert.Equal(t, "https://outlook.office.com/webhook/TEST", notifier.webhookURL)
	assert.NotNil(t, notifier.httpClient)
	assert.NotNil(t, notifier.logger)
}

func TestTeamsNotifier_Send_Success(t *testing.T) {
	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "argazer/1.0", r.Header.Get("User-Agent"))

		// Verify the payload structure
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)
		assert.Equal(t, "MessageCard", payload["@type"])
		assert.Equal(t, "https://schema.org/extensions", payload["@context"])

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("1"))
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTeamsNotifier(server.URL, logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test Subject", "Test message")
	require.NoError(t, err)
}

func TestTeamsNotifier_Send_WithSubjectAndMessage(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTeamsNotifier(server.URL, logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "Subject", "Message")
	require.NoError(t, err)
	assert.Equal(t, "Subject", receivedPayload["title"])
	assert.Equal(t, "Subject", receivedPayload["summary"])
	assert.Equal(t, "Message", receivedPayload["text"])
	assert.Equal(t, "0078D7", receivedPayload["themeColor"])
}

func TestTeamsNotifier_Send_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTeamsNotifier(server.URL, logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "Test", "Message")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 400")
}

func TestTeamsNotifier_Send_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTeamsNotifier(server.URL, logger)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := notifier.Send(ctx, "Test", "Message")
	require.Error(t, err)
}

func TestTeamsNotifier_Send_EmptySubject(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	notifier := NewTeamsNotifier(server.URL, logger)

	ctx := context.Background()
	err := notifier.Send(ctx, "", "Message only")
	require.NoError(t, err)
	assert.Equal(t, "", receivedPayload["title"])
	assert.Equal(t, "Message only", receivedPayload["text"])
}
