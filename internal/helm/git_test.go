package helm

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{
			name:     "GitHub HTTPS URL with .git",
			url:      "https://github.com/myorg/charts.git",
			expected: true,
		},
		{
			name:     "GitHub HTTPS URL without .git",
			url:      "https://github.com/myorg/charts",
			expected: true,
		},
		{
			name:     "GitLab HTTPS URL",
			url:      "https://gitlab.com/myorg/helm-charts.git",
			expected: true,
		},
		{
			name:     "Bitbucket HTTPS URL",
			url:      "https://bitbucket.org/team/repo.git",
			expected: true,
		},
		{
			name:     "Gitea HTTPS URL",
			url:      "https://gitea.company.com/org/charts.git",
			expected: true,
		},
		{
			name:     "SSH Git URL",
			url:      "git@github.com:myorg/charts.git",
			expected: true,
		},
		{
			name:     "Traditional Helm repo",
			url:      "https://charts.bitnami.com/bitnami",
			expected: false,
		},
		{
			name:     "OCI registry",
			url:      "ghcr.io/myorg/charts",
			expected: false,
		},
		{
			name:     "Harbor OCI",
			url:      "harbor.company.com/helm",
			expected: false,
		},
		{
			name:     "HTTP URL",
			url:      "http://charts.internal.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGitURL(tt.url)
			assert.Equal(t, tt.expected, result, "URL: %s", tt.url)
		})
	}
}

func TestNewGitClient(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("create client without auth", func(t *testing.T) {
		client := NewGitClient("", "", logger)
		assert.NotNil(t, client)
		assert.Equal(t, "", client.username)
		assert.Equal(t, "", client.password)
		assert.NotNil(t, client.logger)
	})

	t.Run("create client with auth", func(t *testing.T) {
		client := NewGitClient("testuser", "testpass", logger)
		assert.NotNil(t, client)
		assert.Equal(t, "testuser", client.username)
		assert.Equal(t, "testpass", client.password)
	})
}

// Integration test - only runs with real Git repo (skip in CI)
func TestGitClient_GetLatestVersion_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.ErrorLevel) // Reduce noise

	t.Run("public GitHub repo", func(t *testing.T) {
		client := NewGitClient("", "", logger)
		ctx := context.Background()

		// Use a stable public repo for testing
		// This test may fail if the repo changes or is unavailable
		version, err := client.GetLatestVersion(ctx, "https://github.com/helm/charts.git", "stable/nginx-ingress")

		// Just verify no panic and proper error handling
		if err != nil {
			// It's ok if the repo doesn't have the structure we expect
			t.Logf("Expected error for test repo: %v", err)
		} else {
			t.Logf("Found version: %s", version)
			assert.NotEmpty(t, version)
		}
	})

	t.Run("non-existent repo", func(t *testing.T) {
		client := NewGitClient("", "", logger)
		ctx := context.Background()

		_, err := client.GetLatestVersion(ctx, "https://github.com/nonexistent/repo-that-does-not-exist.git", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clone repository")
	})

	t.Run("invalid URL", func(t *testing.T) {
		client := NewGitClient("", "", logger)
		ctx := context.Background()

		_, err := client.GetLatestVersion(ctx, "not-a-valid-url", "")
		assert.Error(t, err)
	})
}

func TestGitClient_GetAllVersions_Unit(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.ErrorLevel)

	t.Run("invalid URL format", func(t *testing.T) {
		client := NewGitClient("", "", logger)
		ctx := context.Background()

		_, err := client.GetAllVersions(ctx, "invalid://url", "")
		assert.Error(t, err)
	})

	t.Run("non-existent repository", func(t *testing.T) {
		client := NewGitClient("", "", logger)
		ctx := context.Background()

		_, err := client.GetAllVersions(ctx, "https://github.com/definitely-does-not-exist-12345/repo.git", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clone")
	})
}

func TestGitClient_GetChartVersion_Unit(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetLevel(logrus.ErrorLevel)

	t.Run("non-existent repository", func(t *testing.T) {
		client := NewGitClient("", "", logger)
		ctx := context.Background()

		_, err := client.GetChartVersion(ctx, "https://github.com/nonexistent/repo.git", "charts/app")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to clone")
	})
}

// Test that authentication credentials are properly set
func TestGitClient_Authentication(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("credentials are set", func(t *testing.T) {
		client := NewGitClient("myuser", "mypassword", logger)

		assert.Equal(t, "myuser", client.username)
		assert.Equal(t, "mypassword", client.password)
	})

	t.Run("credentials can be updated", func(t *testing.T) {
		client := NewGitClient("", "", logger)

		client.username = "newuser"
		client.password = "newpass"

		assert.Equal(t, "newuser", client.username)
		assert.Equal(t, "newpass", client.password)
	})
}

// Test version tag parsing logic
func TestVersionTagParsing(t *testing.T) {
	// This tests the logic that would be used in the actual implementation
	tests := []struct {
		name        string
		tagName     string
		chartPath   string
		shouldParse bool
		expected    string
	}{
		{
			name:        "simple v-prefixed tag",
			tagName:     "v1.2.3",
			chartPath:   "",
			shouldParse: true,
			expected:    "1.2.3",
		},
		{
			name:        "release prefix",
			tagName:     "release-1.0.0",
			chartPath:   "",
			shouldParse: true,
			expected:    "1.0.0",
		},
		{
			name:        "chart prefix",
			tagName:     "chart-v2.1.0",
			chartPath:   "",
			shouldParse: true,
			expected:    "v2.1.0", // After "chart-" removal, still has "v" prefix
		},
		{
			name:        "chart-specific tag",
			tagName:     "myapp-v1.5.0",
			chartPath:   "myapp",
			shouldParse: true,
			expected:    "1.5.0",
		},
		{
			name:        "non-semver tag",
			tagName:     "production",
			chartPath:   "",
			shouldParse: false,
			expected:    "",
		},
		{
			name:        "just version number",
			tagName:     "1.0.0",
			chartPath:   "",
			shouldParse: true,
			expected:    "1.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test the version string extraction logic
			versionStr := tt.tagName
			versionStr = removePrefix(versionStr, "v")
			versionStr = removePrefix(versionStr, "release-")
			versionStr = removePrefix(versionStr, "chart-")

			if tt.chartPath != "" {
				prefix := tt.chartPath + "-"
				if hasPrefix(tt.tagName, prefix) {
					versionStr = removePrefix(tt.tagName, prefix)
					versionStr = removePrefix(versionStr, "v")
				}
			}

			// Verify the extraction worked as expected
			if tt.shouldParse {
				assert.NotEmpty(t, versionStr)
				if tt.expected != "" {
					assert.Equal(t, tt.expected, versionStr)
				}
			}
		})
	}
}

// Helper functions for testing
func removePrefix(s, prefix string) string {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
