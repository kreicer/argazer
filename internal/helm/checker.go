package helm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"argazer/internal/auth"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Checker checks Helm repositories for new chart versions
type Checker struct {
	httpClient   *http.Client
	ociChecker   *OCIChecker
	authProvider *auth.Provider
	logger       *logrus.Entry
}

// NewChecker creates a new Helm checker
func NewChecker(authProvider *auth.Provider, logger *logrus.Entry) (*Checker, error) {
	return &Checker{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		ociChecker:   NewOCIChecker(authProvider, logger.WithField("type", "oci")),
		authProvider: authProvider,
		logger:       logger,
	}, nil
}

// GetLatestVersion gets the latest version of a Helm chart from a repository
func (c *Checker) GetLatestVersion(ctx context.Context, repoURL, chartName string) (string, error) {
	// Check if this is an OCI repository (no http/https prefix)
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		c.logger.WithFields(logrus.Fields{
			"repo":  repoURL,
			"chart": chartName,
		}).Info("Detected OCI repository, using OCI checker")
		return c.ociChecker.GetLatestVersion(ctx, repoURL, chartName)
	}
	return c.getLatestVersionFromRepo(ctx, repoURL, chartName, "", "")
}

// GetLatestVersionWithConstraint gets the latest version respecting the version constraint
func (c *Checker) GetLatestVersionWithConstraint(ctx context.Context, repoURL, chartName, currentVersion, constraint string) (*VersionConstraintResult, error) {
	// Check if this is an OCI repository (no http/https prefix)
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		c.logger.WithFields(logrus.Fields{
			"repo":  repoURL,
			"chart": chartName,
		}).Info("Detected OCI repository, using OCI checker")
		// Use OCI checker with constraint support
		return c.ociChecker.GetLatestVersionWithConstraint(ctx, repoURL, chartName, currentVersion, constraint)
	}

	return c.getLatestVersionFromRepoWithConstraint(ctx, repoURL, chartName, currentVersion, constraint)
}

// getChartVersionsFromRepo fetches and returns all available versions for a chart from a Helm repository
func (c *Checker) getChartVersionsFromRepo(ctx context.Context, repoURL, chartName string) ([]string, error) {
	// Construct the index URL
	indexURL := fmt.Sprintf("%s/index.yaml", repoURL)

	c.logger.WithFields(logrus.Fields{
		"repo":  repoURL,
		"chart": chartName,
		"url":   indexURL,
	}).Debug("Fetching Helm repository index")

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", indexURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "argazer/1.0")
	req.Header.Set("Accept", "application/x-yaml, application/yaml, text/yaml")

	// Add authentication if available
	if creds := c.authProvider.GetCredentials(repoURL); creds != nil {
		req.SetBasicAuth(creds.Username, creds.Password)
		c.logger.WithField("source", creds.Source).Debug("Using authentication for Helm repository")
	}

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch index: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			c.logger.WithError(err).Warn("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("repository does not provide index.yaml (status %d) - likely an OCI/container registry", resp.StatusCode)
	}

	// Check content type - if it's HTML, this is likely not a Helm repo
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return nil, fmt.Errorf("repository returned HTML instead of YAML - likely an OCI/container registry, not a traditional Helm repository")
	}

	// Parse the index
	index, err := c.parseIndex(resp.Body)
	if err != nil {
		// Check if error is due to HTML response (common for OCI repos)
		if strings.Contains(err.Error(), "<!DOCTY") || strings.Contains(err.Error(), "<html") {
			return nil, fmt.Errorf("repository is an OCI/container registry, not a traditional Helm repository")
		}
		return nil, fmt.Errorf("failed to parse index: %w", err)
	}

	// Find the chart
	chart, exists := index.Entries[chartName]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrChartNotFound, chartName)
	}

	if len(chart) == 0 {
		return nil, fmt.Errorf("%w: %s (no versions available)", ErrChartNotFound, chartName)
	}

	// Extract versions
	versions := make([]string, len(chart))
	for i, entry := range chart {
		versions[i] = entry.Version
	}

	return versions, nil
}

func (c *Checker) getLatestVersionFromRepo(ctx context.Context, repoURL, chartName, currentVersion, constraint string) (string, error) {
	// Fetch all versions
	versions, err := c.getChartVersionsFromRepo(ctx, repoURL, chartName)
	if err != nil {
		return "", err
	}

	// Use shared utility function for finding latest semantic version
	latestVersion, err := findLatestSemver(versions, c.logger)
	if err != nil {
		return "", fmt.Errorf("failed to determine latest version: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"repo":           repoURL,
		"chart":          chartName,
		"latest_version": latestVersion,
	}).Debug("Found latest version")

	return latestVersion, nil
}

// getLatestVersionFromRepoWithConstraint gets the latest version with constraint support
func (c *Checker) getLatestVersionFromRepoWithConstraint(ctx context.Context, repoURL, chartName, currentVersion, constraint string) (*VersionConstraintResult, error) {
	c.logger.WithFields(logrus.Fields{
		"repo":       repoURL,
		"chart":      chartName,
		"constraint": constraint,
	}).Debug("Fetching Helm repository index with constraint")

	// Fetch all versions using shared helper
	versions, err := c.getChartVersionsFromRepo(ctx, repoURL, chartName)
	if err != nil {
		return nil, err
	}

	// Apply constraint filtering
	result, err := findLatestSemverWithConstraint(versions, currentVersion, constraint, c.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to determine latest version: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"repo":                          repoURL,
		"chart":                         chartName,
		"current_version":               currentVersion,
		"latest_version":                result.LatestVersion,
		"latest_version_all":            result.LatestVersionAll,
		"constraint":                    constraint,
		"has_update_outside_constraint": result.HasUpdateOutsideConstraint,
	}).Debug("Found latest version with constraint")

	return result, nil
}

// parseIndex parses the Helm repository index YAML
func (c *Checker) parseIndex(body io.Reader) (*Index, error) {
	// Read the body
	data, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse YAML
	var index Index
	if err := yaml.Unmarshal(data, &index); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	return &index, nil
}

// Index represents a Helm repository index
type Index struct {
	APIVersion string             `yaml:"apiVersion"`
	Generated  time.Time          `yaml:"generated"`
	Entries    map[string][]Entry `yaml:"entries"`
}

// Entry represents a chart entry in the index
type Entry struct {
	Name        string    `yaml:"name"`
	Version     string    `yaml:"version"`
	Description string    `yaml:"description"`
	Created     time.Time `yaml:"created"`
	Digest      string    `yaml:"digest"`
	URLs        []string  `yaml:"urls"`
}
