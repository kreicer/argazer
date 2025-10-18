package helm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Checker checks Helm repositories for new chart versions
type Checker struct {
	httpClient *http.Client
	logger     *logrus.Entry
}

// NewChecker creates a new Helm checker
func NewChecker(logger *logrus.Entry) (*Checker, error) {
	return &Checker{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}, nil
}

// GetLatestVersion gets the latest version of a Helm chart from a repository
func (c *Checker) GetLatestVersion(ctx context.Context, repoURL, chartName string) (string, error) {
	// Check if this is an OCI repository
	if strings.HasPrefix(repoURL, "oci://") {
		c.logger.WithFields(logrus.Fields{
			"repo":  repoURL,
			"chart": chartName,
		}).Info("Skipping OCI repository (not supported for version checking)")
		return "", fmt.Errorf("OCI repositories are not supported for automated version checking")
	}

	// Ensure repoURL has a scheme (protocol)
	if !strings.HasPrefix(repoURL, "http://") && !strings.HasPrefix(repoURL, "https://") {
		repoURL = "https://" + repoURL
		c.logger.WithField("repo", repoURL).Debug("Added https:// prefix to repository URL")
	}

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
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("User-Agent", "watcher/1.0")
	req.Header.Set("Accept", "application/x-yaml, application/yaml, text/yaml")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("repository does not provide index.yaml (status %d) - likely an OCI/container registry", resp.StatusCode)
	}

	// Check content type - if it's HTML, this is likely not a Helm repo
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/html") {
		return "", fmt.Errorf("repository returned HTML instead of YAML - likely an OCI/container registry, not a traditional Helm repository")
	}

	// Parse the index
	index, err := c.parseIndex(resp.Body)
	if err != nil {
		// Check if error is due to HTML response (common for OCI repos)
		if strings.Contains(err.Error(), "<!DOCTY") || strings.Contains(err.Error(), "<html") {
			return "", fmt.Errorf("repository is an OCI/container registry, not a traditional Helm repository")
		}
		return "", fmt.Errorf("failed to parse index: %w", err)
	}

	// Find the chart
	chart, exists := index.Entries[chartName]
	if !exists {
		return "", fmt.Errorf("chart %s not found in repository", chartName)
	}

	if len(chart) == 0 {
		return "", fmt.Errorf("no versions found for chart %s", chartName)
	}

	// Sort versions and get the latest
	versions := make([]string, len(chart))
	for i, entry := range chart {
		versions[i] = entry.Version
	}

	latestVersion, err := c.getLatestVersion(versions)
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

// getLatestVersion determines the latest version from a list of versions
func (c *Checker) getLatestVersion(versions []string) (string, error) {
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions provided")
	}

	// Sort versions in descending order
	sort.Slice(versions, func(i, j int) bool {
		return c.compareVersions(versions[i], versions[j]) > 0
	})

	return versions[0], nil
}

// compareVersions compares two semantic versions using proper semver logic
// Returns: 1 if v1 > v2, 0 if v1 == v2, -1 if v1 < v2
func (c *Checker) compareVersions(v1, v2 string) int {
	// Parse versions using semver library
	version1, err1 := semver.NewVersion(v1)
	version2, err2 := semver.NewVersion(v2)

	// If either version fails to parse, fall back to string comparison
	if err1 != nil || err2 != nil {
		c.logger.WithFields(logrus.Fields{
			"v1":       v1,
			"v2":       v2,
			"v1_error": err1,
			"v2_error": err2,
		}).Debug("Failed to parse versions as semver, falling back to string comparison")

		if v1 == v2 {
			return 0
		}
		if v1 > v2 {
			return 1
		}
		return -1
	}

	// Use proper semantic version comparison
	return version1.Compare(version2)
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
