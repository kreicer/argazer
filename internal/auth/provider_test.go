package auth

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeURL(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())
	p := &Provider{logger: logger}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "https with path",
			input:    "https://charts.example.com/helm",
			expected: "charts.example.com",
		},
		{
			name:     "http with path",
			input:    "http://charts.example.com/helm",
			expected: "charts.example.com",
		},
		{
			name:     "oci with path",
			input:    "oci://ghcr.io/myorg/charts",
			expected: "ghcr.io",
		},
		{
			name:     "registry without protocol",
			input:    "registry.example.com/helm",
			expected: "registry.example.com",
		},
		{
			name:     "registry with port",
			input:    "registry.example.com:5000/helm",
			expected: "registry.example.com",
		},
		{
			name:     "simple hostname",
			input:    "ghcr.io",
			expected: "ghcr.io",
		},
		{
			name:     "localhost with port",
			input:    "localhost:5000",
			expected: "localhost",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.normalizeURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewProvider(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	t.Run("empty config", func(t *testing.T) {
		p, err := NewProvider([]ConfigAuth{}, logger)
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, 0, len(p.credentials))
	})

	t.Run("with config auth", func(t *testing.T) {
		configAuth := []ConfigAuth{
			{
				URL:      "https://charts.example.com",
				Username: "user1",
				Password: "pass1",
			},
			{
				URL:      "oci://ghcr.io/myorg",
				Username: "user2",
				Password: "pass2",
			},
		}

		p, err := NewProvider(configAuth, logger)
		require.NoError(t, err)
		require.NotNil(t, p)
		assert.Equal(t, 2, len(p.credentials))

		// Check normalized credentials
		creds1 := p.credentials["charts.example.com"]
		assert.Equal(t, "user1", creds1.Username)
		assert.Equal(t, "pass1", creds1.Password)
		assert.Equal(t, "config", creds1.Source)

		creds2 := p.credentials["ghcr.io"]
		assert.Equal(t, "user2", creds2.Username)
		assert.Equal(t, "pass2", creds2.Password)
		assert.Equal(t, "config", creds2.Source)
	})

	t.Run("incomplete config auth", func(t *testing.T) {
		configAuth := []ConfigAuth{
			{
				URL:      "",
				Username: "user1",
				Password: "pass1",
			},
			{
				URL:      "https://charts.example.com",
				Username: "",
				Password: "pass1",
			},
			{
				URL:      "https://valid.example.com",
				Username: "user",
				Password: "pass",
			},
		}

		p, err := NewProvider(configAuth, logger)
		require.NoError(t, err)
		require.NotNil(t, p)
		// Only the valid one should be loaded
		assert.Equal(t, 1, len(p.credentials))
	})
}

func TestGetCredentials(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	configAuth := []ConfigAuth{
		{
			URL:      "https://charts.example.com",
			Username: "user1",
			Password: "pass1",
		},
		{
			URL:      "ghcr.io",
			Username: "user2",
			Password: "pass2",
		},
	}

	p, err := NewProvider(configAuth, logger)
	require.NoError(t, err)

	tests := []struct {
		name           string
		repoURL        string
		expectedCreds  bool
		expectedUser   string
		expectedSource string
	}{
		{
			name:           "exact match with https",
			repoURL:        "https://charts.example.com/myrepo",
			expectedCreds:  true,
			expectedUser:   "user1",
			expectedSource: "config",
		},
		{
			name:           "match without protocol",
			repoURL:        "charts.example.com/myrepo",
			expectedCreds:  true,
			expectedUser:   "user1",
			expectedSource: "config",
		},
		{
			name:           "oci registry match",
			repoURL:        "oci://ghcr.io/myorg/charts",
			expectedCreds:  true,
			expectedUser:   "user2",
			expectedSource: "config",
		},
		{
			name:          "no match",
			repoURL:       "https://unknown.example.com",
			expectedCreds: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := p.GetCredentials(tt.repoURL)
			if tt.expectedCreds {
				require.NotNil(t, creds)
				assert.Equal(t, tt.expectedUser, creds.Username)
				assert.Equal(t, tt.expectedSource, creds.Source)
			} else {
				assert.Nil(t, creds)
			}
		})
	}
}

func TestLoadEnvAuth(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	// Set up environment variables
	os.Setenv("AG_AUTH_URL_1", "registry.example.com")
	os.Setenv("AG_AUTH_USER_1", "envuser1")
	os.Setenv("AG_AUTH_PASS_1", "envpass1")

	os.Setenv("AG_AUTH_URL_HARBOR", "harbor.example.com")
	os.Setenv("AG_AUTH_USER_HARBOR", "harboruser")
	os.Setenv("AG_AUTH_PASS_HARBOR", "harborpass")

	// Incomplete auth group (should be skipped)
	os.Setenv("AG_AUTH_URL_INCOMPLETE", "incomplete.example.com")
	os.Setenv("AG_AUTH_USER_INCOMPLETE", "user")
	// Missing AG_AUTH_PASS_INCOMPLETE

	defer func() {
		os.Unsetenv("AG_AUTH_URL_1")
		os.Unsetenv("AG_AUTH_USER_1")
		os.Unsetenv("AG_AUTH_PASS_1")
		os.Unsetenv("AG_AUTH_URL_HARBOR")
		os.Unsetenv("AG_AUTH_USER_HARBOR")
		os.Unsetenv("AG_AUTH_PASS_HARBOR")
		os.Unsetenv("AG_AUTH_URL_INCOMPLETE")
		os.Unsetenv("AG_AUTH_USER_INCOMPLETE")
	}()

	p, err := NewProvider([]ConfigAuth{}, logger)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Should have 2 credentials (the incomplete one is skipped)
	assert.Equal(t, 2, len(p.credentials))

	// Check credentials for registry.example.com
	creds1 := p.GetCredentials("registry.example.com")
	require.NotNil(t, creds1)
	assert.Equal(t, "envuser1", creds1.Username)
	assert.Equal(t, "envpass1", creds1.Password)
	assert.Equal(t, "env:1", creds1.Source)

	// Check credentials for harbor.example.com
	creds2 := p.GetCredentials("harbor.example.com")
	require.NotNil(t, creds2)
	assert.Equal(t, "harboruser", creds2.Username)
	assert.Equal(t, "harborpass", creds2.Password)
	assert.Equal(t, "env:HARBOR", creds2.Source)

	// Incomplete should not have credentials
	creds3 := p.GetCredentials("incomplete.example.com")
	assert.Nil(t, creds3)
}

func TestEnvOverridesConfig(t *testing.T) {
	logger := logrus.NewEntry(logrus.New())

	// First load config auth
	configAuth := []ConfigAuth{
		{
			URL:      "registry.example.com",
			Username: "configuser",
			Password: "configpass",
		},
	}

	// Set environment variable that should override
	os.Setenv("AG_AUTH_URL_1", "registry.example.com")
	os.Setenv("AG_AUTH_USER_1", "envuser")
	os.Setenv("AG_AUTH_PASS_1", "envpass")

	defer func() {
		os.Unsetenv("AG_AUTH_URL_1")
		os.Unsetenv("AG_AUTH_USER_1")
		os.Unsetenv("AG_AUTH_PASS_1")
	}()

	p, err := NewProvider(configAuth, logger)
	require.NoError(t, err)

	// Should use env credentials (they override config)
	creds := p.GetCredentials("registry.example.com")
	require.NotNil(t, creds)
	assert.Equal(t, "envuser", creds.Username)
	assert.Equal(t, "envpass", creds.Password)
	assert.Equal(t, "env:1", creds.Source)
}
