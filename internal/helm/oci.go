package helm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"argazer/internal/auth"

	"github.com/sirupsen/logrus"
)

// OCIChecker checks OCI-based Helm repositories for new chart versions
type OCIChecker struct {
	httpClient   *http.Client
	authProvider *auth.Provider
	logger       *logrus.Entry
}

// NewOCIChecker creates a new OCI checker
func NewOCIChecker(authProvider *auth.Provider, logger *logrus.Entry) *OCIChecker {
	return &OCIChecker{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		authProvider: authProvider,
		logger:       logger,
	}
}

// TagsResponse represents the response from Docker Registry API v2 tags list endpoint
type TagsResponse struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// GetLatestVersion gets the latest version of a Helm chart from an OCI registry
func (o *OCIChecker) GetLatestVersion(ctx context.Context, repoURL, chartName string) (string, error) {
	o.logger.WithFields(logrus.Fields{
		"repo":  repoURL,
		"chart": chartName,
	}).Debug("Checking OCI registry for latest version")

	// Parse OCI registry URL and build repository path
	registry, repoPath := parseOCIURL(repoURL)

	// Build full repository path: repoPath/chartName
	var fullRepoPath string
	if repoPath != "" {
		fullRepoPath = fmt.Sprintf("%s/%s", repoPath, chartName)
	} else {
		fullRepoPath = chartName
	}

	o.logger.WithFields(logrus.Fields{
		"registry":       registry,
		"repo_path":      repoPath,
		"full_repo_path": fullRepoPath,
	}).Debug("Parsed OCI URL")

	// Build Docker Registry API v2 endpoint
	tagsURL := fmt.Sprintf("https://%s/v2/%s/tags/list", registry, fullRepoPath)

	o.logger.WithField("url", tagsURL).Debug("Fetching tags from OCI registry")

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", tagsURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "argazer/1.0")
	req.Header.Set("Accept", "application/json")

	// Add authentication if available
	creds := o.authProvider.GetCredentials(registry)
	if creds != nil {
		req.SetBasicAuth(creds.Username, creds.Password)
		o.logger.WithFields(logrus.Fields{
			"source":   creds.Source,
			"username": creds.Username,
			"registry": registry,
		}).Debug("Using authentication for OCI registry")
	} else {
		o.logger.WithField("registry", registry).Debug("No credentials found, trying anonymous access")
	}

	// Make request
	resp, err := o.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch tags from OCI registry: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			o.logger.WithError(err).Warn("Failed to close response body")
		}
	}()

	// Check response status
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		if creds != nil {
			return "", fmt.Errorf("OCI registry authentication failed (status %d). Check credentials for %s", resp.StatusCode, registry)
		}
		return "", fmt.Errorf("OCI registry requires authentication (status %d). No credentials found for %s. Use 'docker login %s' or 'helm registry login %s' or set AG_AUTH_* environment variables", resp.StatusCode, registry, registry, registry)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OCI registry returned status %d", resp.StatusCode)
	}

	// Parse response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var tagsResp TagsResponse
	if err := json.Unmarshal(body, &tagsResp); err != nil {
		return "", fmt.Errorf("failed to parse tags response: %w", err)
	}

	if len(tagsResp.Tags) == 0 {
		return "", fmt.Errorf("no tags found for chart %s in OCI registry", chartName)
	}

	o.logger.WithFields(logrus.Fields{
		"chart":      chartName,
		"tags_count": len(tagsResp.Tags),
		"tags":       tagsResp.Tags,
	}).Debug("Retrieved tags from OCI registry")

	// Filter and find latest semver version
	latestVersion, err := o.findLatestSemver(tagsResp.Tags)
	if err != nil {
		return "", fmt.Errorf("failed to determine latest version: %w", err)
	}

	o.logger.WithFields(logrus.Fields{
		"chart":          chartName,
		"latest_version": latestVersion,
	}).Debug("Found latest version in OCI registry")

	return latestVersion, nil
}

// parseOCIURL parses an OCI registry URL into registry and repository path
// Examples:
//   - "ghcr.io/myorg/charts" -> registry: "ghcr.io", repoPath: "myorg/charts"
//   - "harbor.company.com/helm" -> registry: "harbor.company.com", repoPath: "helm"
//   - "registry.example.com" -> registry: "registry.example.com", repoPath: ""
func parseOCIURL(repoURL string) (registry string, repoPath string) {
	// Remove any trailing slashes
	repoURL = strings.TrimSuffix(repoURL, "/")

	// Split by first slash
	parts := strings.SplitN(repoURL, "/", 2)
	registry = parts[0]

	if len(parts) > 1 {
		repoPath = parts[1]
	}

	return registry, repoPath
}

// findLatestSemver finds the latest semantic version from a list of tags
// It filters out non-semver tags (like "latest", "dev", etc.) and returns the highest version
func (o *OCIChecker) findLatestSemver(tags []string) (string, error) {
	// Filter out non-semver tags and common non-version tags
	var versions []string
	excludedTags := map[string]bool{
		"latest": true,
		"dev":    true,
		"main":   true,
		"master": true,
		"stable": true,
	}

	for _, tag := range tags {
		// Skip excluded tags
		if excludedTags[tag] {
			continue
		}

		// Try to identify semver-like tags
		// Accept tags that start with a digit or 'v' followed by a digit
		if len(tag) > 0 && (isDigit(tag[0]) || (tag[0] == 'v' && len(tag) > 1 && isDigit(tag[1]))) {
			versions = append(versions, tag)
		}
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no semantic version tags found")
	}

	// Use the existing Checker's version comparison logic
	// Create a temporary checker to reuse the comparison logic
	checker := &Checker{logger: o.logger}
	latestVersion, err := checker.getLatestVersion(versions)
	if err != nil {
		return "", err
	}

	return latestVersion, nil
}

// isDigit checks if a byte is a digit
func isDigit(b byte) bool {
	return b >= '0' && b <= '9'
}
