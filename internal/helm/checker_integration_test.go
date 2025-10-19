package helm

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"argazer/internal/auth"

	"github.com/sirupsen/logrus"
)

// TestCheckerGetLatestVersion_Success tests successful retrieval of latest version
func TestCheckerGetLatestVersion_Success(t *testing.T) {
	// Create a test server that returns a valid Helm index
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/index.yaml" {
			t.Errorf("Expected request to /index.yaml, got %s", r.URL.Path)
		}
		// Return a valid Helm index.yaml
		indexYAML := `apiVersion: v1
entries:
  nginx:
    - name: nginx
      version: 1.21.0
      description: NGINX chart
    - name: nginx
      version: 1.20.0
      description: NGINX chart
    - name: nginx
      version: 1.19.5
      description: NGINX chart
`
		w.Header().Set("Content-Type", "application/x-yaml")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, indexYAML)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	authProvider, _ := auth.NewProvider(nil, logger)
	checker, err := NewChecker(authProvider, logger)
	if err != nil {
		t.Fatalf("Failed to create checker: %v", err)
	}

	ctx := context.Background()
	version, err := checker.GetLatestVersion(ctx, server.URL, "nginx")
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}

	expected := "1.21.0"
	if version != expected {
		t.Errorf("Expected version %s, got %s", expected, version)
	}
}

// TestCheckerGetLatestVersion_ChartNotFound tests handling of chart not found
func TestCheckerGetLatestVersion_ChartNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		indexYAML := `apiVersion: v1
entries:
  redis:
    - name: redis
      version: 7.0.0
`
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, indexYAML)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	authProvider, _ := auth.NewProvider(nil, logger)
	checker, _ := NewChecker(authProvider, logger)

	ctx := context.Background()
	_, err := checker.GetLatestVersion(ctx, server.URL, "nginx")
	if err == nil {
		t.Fatal("Expected error for chart not found, got nil")
	}

	// Check if it's the right error type
	if !isErrorType(err, ErrChartNotFound) {
		t.Errorf("Expected ErrChartNotFound, got: %v", err)
	}
}

// TestCheckerGetLatestVersion_ServerError tests handling of server errors
func TestCheckerGetLatestVersion_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Internal Server Error")
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	authProvider, _ := auth.NewProvider(nil, logger)
	checker, _ := NewChecker(authProvider, logger)

	ctx := context.Background()
	_, err := checker.GetLatestVersion(ctx, server.URL, "nginx")
	if err == nil {
		t.Fatal("Expected error for server error, got nil")
	}
}

// TestCheckerGetLatestVersion_InvalidYAML tests handling of invalid YAML
func TestCheckerGetLatestVersion_InvalidYAML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "this is not valid YAML: {[}]}")
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	authProvider, _ := auth.NewProvider(nil, logger)
	checker, _ := NewChecker(authProvider, logger)

	ctx := context.Background()
	_, err := checker.GetLatestVersion(ctx, server.URL, "nginx")
	if err == nil {
		t.Fatal("Expected error for invalid YAML, got nil")
	}
}

// TestCheckerGetLatestVersion_HTMLResponse tests handling of HTML instead of YAML
func TestCheckerGetLatestVersion_HTMLResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "<html><body>This is HTML</body></html>")
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	authProvider, _ := auth.NewProvider(nil, logger)
	checker, _ := NewChecker(authProvider, logger)

	ctx := context.Background()
	_, err := checker.GetLatestVersion(ctx, server.URL, "nginx")
	if err == nil {
		t.Fatal("Expected error for HTML response, got nil")
	}
}

// TestCheckerGetLatestVersion_EmptyVersionList tests handling of chart with no versions
func TestCheckerGetLatestVersion_EmptyVersionList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		indexYAML := `apiVersion: v1
entries:
  nginx: []
`
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, indexYAML)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	authProvider, _ := auth.NewProvider(nil, logger)
	checker, _ := NewChecker(authProvider, logger)

	ctx := context.Background()
	_, err := checker.GetLatestVersion(ctx, server.URL, "nginx")
	if err == nil {
		t.Fatal("Expected error for empty version list, got nil")
	}

	if !isErrorType(err, ErrChartNotFound) {
		t.Errorf("Expected ErrChartNotFound, got: %v", err)
	}
}

// TestCheckerGetLatestVersion_WithAuthentication tests that auth credentials are applied
func TestCheckerGetLatestVersion_WithAuthentication(t *testing.T) {
	receivedAuth := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Authorization header
		auth := r.Header.Get("Authorization")
		if auth != "" {
			receivedAuth = true
		}

		indexYAML := `apiVersion: v1
entries:
  nginx:
    - name: nginx
      version: 1.0.0
`
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, indexYAML)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	// Create auth provider with credentials for our test server
	configAuth := []auth.ConfigAuth{
		{
			URL:      server.URL,
			Username: "testuser",
			Password: "testpass",
		},
	}
	authProvider, _ := auth.NewProvider(configAuth, logger)
	checker, _ := NewChecker(authProvider, logger)

	ctx := context.Background()
	_, err := checker.GetLatestVersion(ctx, server.URL, "nginx")
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}

	if !receivedAuth {
		t.Error("Expected Authorization header to be sent, but it wasn't")
	}
}

// TestCheckerGetLatestVersion_InvalidVersions tests handling of non-semver versions
func TestCheckerGetLatestVersion_InvalidVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		indexYAML := `apiVersion: v1
entries:
  nginx:
    - name: nginx
      version: latest
    - name: nginx
      version: dev
    - name: nginx
      version: 1.2.3
    - name: nginx
      version: invalid-version
`
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, indexYAML)
	}))
	defer server.Close()

	logger := logrus.NewEntry(logrus.New())
	authProvider, _ := auth.NewProvider(nil, logger)
	checker, _ := NewChecker(authProvider, logger)

	ctx := context.Background()
	version, err := checker.GetLatestVersion(ctx, server.URL, "nginx")
	if err != nil {
		t.Fatalf("GetLatestVersion failed: %v", err)
	}

	// Should return the only valid semver version
	expected := "1.2.3"
	if version != expected {
		t.Errorf("Expected version %s, got %s", expected, version)
	}
}

// isErrorType checks if an error wraps a specific error type
func isErrorType(err, target error) bool {
	if err == nil {
		return false
	}
	// Simple string contains check for wrapped errors
	return fmt.Sprintf("%v", err) != fmt.Sprintf("%v", target) &&
		(err == target || fmt.Sprintf("%v", err)[:len(fmt.Sprintf("%v", target))] == fmt.Sprintf("%v", target))
}
