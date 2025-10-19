package auth

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Credentials holds authentication credentials for a registry or repository
type Credentials struct {
	Username string
	Password string
	Source   string // "config", "env", etc.
}

// Provider manages authentication for various registries and repositories
type Provider struct {
	credentials map[string]Credentials
	logger      *logrus.Entry
}

// ConfigAuth represents authentication from config file
type ConfigAuth struct {
	URL      string
	Username string
	Password string
}

// NewProvider creates a new authentication provider
func NewProvider(configAuth []ConfigAuth, logger *logrus.Entry) (*Provider, error) {
	p := &Provider{
		credentials: make(map[string]Credentials),
		logger:      logger,
	}

	// Load credentials from config file
	p.loadConfigAuth(configAuth)

	// Load credentials from environment variables (overrides config)
	p.loadEnvAuth()

	// Log summary
	logger.WithField("auths", len(p.credentials)).Debug("Loaded authentication credentials")

	return p, nil
}

// GetCredentials returns credentials for a given registry or repository URL
func (p *Provider) GetCredentials(repoURL string) *Credentials {
	// Normalize URL for matching
	normalized := p.normalizeURL(repoURL)

	p.logger.WithFields(logrus.Fields{
		"repo_url":   repoURL,
		"normalized": normalized,
	}).Debug("Looking up credentials")

	// Check credentials map
	if creds, ok := p.credentials[normalized]; ok {
		p.logger.WithField("source", creds.Source).Debug("Found credentials")
		return &creds
	}

	p.logger.Debug("No credentials found, will try anonymous access")
	return nil
}

// normalizeURL normalizes a URL for credential matching
// Examples:
//   - "https://charts.example.com" -> "charts.example.com"
//   - "registry.example.com/helm" -> "registry.example.com"
//   - "ghcr.io/myorg/charts" -> "ghcr.io"
func (p *Provider) normalizeURL(repoURL string) string {
	// Remove protocol if present
	repoURL = strings.TrimPrefix(repoURL, "https://")
	repoURL = strings.TrimPrefix(repoURL, "http://")
	repoURL = strings.TrimPrefix(repoURL, "oci://")

	// Extract hostname/registry part (before first slash or use whole string)
	parts := strings.SplitN(repoURL, "/", 2)
	hostname := parts[0]

	// Remove port if present
	hostname = strings.Split(hostname, ":")[0]

	return hostname
}

// loadConfigAuth loads authentication from config file
func (p *Provider) loadConfigAuth(configAuths []ConfigAuth) {
	for _, auth := range configAuths {
		if auth.URL == "" || auth.Username == "" || auth.Password == "" {
			p.logger.WithField("url", auth.URL).Warn("Incomplete auth configuration, skipping")
			continue
		}

		// Normalize the URL and store credentials
		normalized := p.normalizeURL(auth.URL)
		p.credentials[normalized] = Credentials{
			Username: auth.Username,
			Password: auth.Password,
			Source:   "config",
		}

		p.logger.WithFields(logrus.Fields{
			"url":        auth.URL,
			"normalized": normalized,
			"username":   auth.Username,
		}).Debug("Loaded credentials from config file")
	}
}

// loadEnvAuth loads authentication from environment variables
// Format: AG_AUTH_URL_<id>=registry, AG_AUTH_USER_<id>=user, AG_AUTH_PASS_<id>=pass
func (p *Provider) loadEnvAuth() {
	// Find all AG_AUTH_URL_* variables
	authGroups := make(map[string]map[string]string)

	for _, env := range os.Environ() {
		pair := strings.SplitN(env, "=", 2)
		if len(pair) != 2 {
			continue
		}

		key := pair[0]
		value := pair[1]

		// Check if it's an AG_AUTH_* variable
		if !strings.HasPrefix(key, "AG_AUTH_") {
			continue
		}

		// Extract the type and ID
		// AG_AUTH_URL_1 -> type=URL, id=1
		// AG_AUTH_USER_HARBOR -> type=USER, id=HARBOR
		parts := strings.SplitN(key, "_", 4) // ["AG", "AUTH", "TYPE", "ID"]
		if len(parts) != 4 {
			continue
		}

		authType := parts[2] // URL, USER, or PASS
		authID := parts[3]   // The identifier (1, 2, HARBOR, etc.)

		if authGroups[authID] == nil {
			authGroups[authID] = make(map[string]string)
		}

		authGroups[authID][authType] = value
	}

	// Process each auth group
	for id, group := range authGroups {
		url, hasURL := group["URL"]
		user, hasUser := group["USER"]
		pass, hasPass := group["PASS"]

		if !hasURL || !hasUser || !hasPass {
			p.logger.WithFields(logrus.Fields{
				"id":       id,
				"has_url":  hasURL,
				"has_user": hasUser,
				"has_pass": hasPass,
			}).Warn("Incomplete auth group in environment variables")
			continue
		}

		// Normalize the URL and store credentials (env vars override config)
		normalized := p.normalizeURL(url)
		p.credentials[normalized] = Credentials{
			Username: user,
			Password: pass,
			Source:   fmt.Sprintf("env:%s", id),
		}

		p.logger.WithFields(logrus.Fields{
			"id":         id,
			"url":        url,
			"normalized": normalized,
		}).Debug("Loaded credentials from environment variables")
	}
}
