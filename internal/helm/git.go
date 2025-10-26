package helm

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// GitClient handles operations with Git repositories containing Helm charts
type GitClient struct {
	username string
	password string
	logger   *logrus.Entry
}

// NewGitClient creates a new Git client
func NewGitClient(username, password string, logger *logrus.Entry) *GitClient {
	return &GitClient{
		username: username,
		password: password,
		logger:   logger,
	}
}

// ChartMetadata represents the structure of Chart.yaml
type ChartMetadata struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Description string `yaml:"description"`
	APIVersion  string `yaml:"apiVersion"`
}

// isGitURL determines if a URL is a Git repository
func isGitURL(repoURL string) bool {
	// Git URLs typically:
	// - Contain .git
	// - Start with git@
	// - Are GitHub/GitLab/Bitbucket URLs without /helm suffix
	// - Don't have http/https prefix for OCI registries (those are handled separately)

	lower := strings.ToLower(repoURL)

	// Explicit Git URLs
	if strings.HasSuffix(lower, ".git") || strings.HasPrefix(lower, "git@") {
		return true
	}

	// Common Git hosting platforms (if they don't look like OCI registries)
	gitPlatforms := []string{"github.com", "gitlab.com", "bitbucket.org", "gitea"}
	for _, platform := range gitPlatforms {
		if strings.Contains(lower, platform) {
			// Not an OCI URL (no oci:// prefix and has http/https)
			if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
				return true
			}
		}
	}

	return false
}

// GetLatestVersion fetches the latest semantic version from Git repository
// It looks at Git tags for version information
func (g *GitClient) GetLatestVersion(ctx context.Context, repoURL, chartPath string) (string, error) {
	g.logger.WithFields(logrus.Fields{
		"repo":       repoURL,
		"chart_path": chartPath,
	}).Debug("Fetching latest version from Git repository")

	// Create temporary directory for cloning
	tmpDir, err := os.MkdirTemp("", "argazer-git-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone options
	cloneOpts := &git.CloneOptions{
		URL:      repoURL,
		Progress: nil, // Silent clone
		Tags:     git.AllTags,
	}

	// Add authentication if provided
	if g.username != "" && g.password != "" {
		cloneOpts.Auth = &http.BasicAuth{
			Username: g.username,
			Password: g.password,
		}
	}

	// Clone the repository
	repo, err := git.PlainCloneContext(ctx, tmpDir, false, cloneOpts)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get all tags
	tags, err := repo.Tags()
	if err != nil {
		return "", fmt.Errorf("failed to list tags: %w", err)
	}

	var versions []*semver.Version
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()

		// Try to parse as semantic version
		// Remove common prefixes (v, release-, etc.)
		versionStr := strings.TrimPrefix(tagName, "v")
		versionStr = strings.TrimPrefix(versionStr, "release-")
		versionStr = strings.TrimPrefix(versionStr, "chart-")

		// If chartPath is specified, look for tags like "chartname-v1.2.3"
		if chartPath != "" {
			chartName := filepath.Base(chartPath)
			prefix := chartName + "-"
			if strings.HasPrefix(tagName, prefix) {
				versionStr = strings.TrimPrefix(tagName, prefix)
				versionStr = strings.TrimPrefix(versionStr, "v")
			}
		}

		v, err := semver.NewVersion(versionStr)
		if err != nil {
			// Not a valid semver tag, skip it
			g.logger.WithFields(logrus.Fields{
				"tag":   tagName,
				"error": err,
			}).Debug("Skipping non-semver tag")
			return nil
		}

		versions = append(versions, v)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("error processing tags: %w", err)
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no valid semantic version tags found in repository")
	}

	// Find the latest version
	var latest *semver.Version
	for _, v := range versions {
		if latest == nil || v.GreaterThan(latest) {
			latest = v
		}
	}

	g.logger.WithFields(logrus.Fields{
		"repo":           repoURL,
		"latest_version": latest.String(),
		"total_versions": len(versions),
	}).Debug("Found latest version from Git tags")

	return latest.String(), nil
}

// GetChartVersion fetches the chart version from Chart.yaml in the repository
// This is useful when tags don't follow semver or when you want the chart version directly
func (g *GitClient) GetChartVersion(ctx context.Context, repoURL, chartPath string) (string, error) {
	g.logger.WithFields(logrus.Fields{
		"repo":       repoURL,
		"chart_path": chartPath,
	}).Debug("Fetching chart version from Chart.yaml")

	// Create temporary directory for cloning
	tmpDir, err := os.MkdirTemp("", "argazer-git-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone options (shallow clone for faster operation)
	cloneOpts := &git.CloneOptions{
		URL:      repoURL,
		Progress: nil,
		Depth:    1, // Shallow clone
	}

	// Add authentication if provided
	if g.username != "" && g.password != "" {
		cloneOpts.Auth = &http.BasicAuth{
			Username: g.username,
			Password: g.password,
		}
	}

	// Clone the repository
	_, err = git.PlainCloneContext(ctx, tmpDir, false, cloneOpts)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	// Construct path to Chart.yaml
	chartYAMLPath := filepath.Join(tmpDir, chartPath, "Chart.yaml")

	// Read Chart.yaml
	data, err := os.ReadFile(chartYAMLPath)
	if err != nil {
		return "", fmt.Errorf("failed to read Chart.yaml: %w", err)
	}

	// Parse Chart.yaml
	var chart ChartMetadata
	if err := yaml.Unmarshal(data, &chart); err != nil {
		return "", fmt.Errorf("failed to parse Chart.yaml: %w", err)
	}

	if chart.Version == "" {
		return "", fmt.Errorf("no version found in Chart.yaml")
	}

	g.logger.WithFields(logrus.Fields{
		"repo":    repoURL,
		"chart":   chart.Name,
		"version": chart.Version,
	}).Debug("Found version from Chart.yaml")

	return chart.Version, nil
}

// GetAllVersions fetches all semantic versions from Git tags
func (g *GitClient) GetAllVersions(ctx context.Context, repoURL, chartPath string) ([]string, error) {
	g.logger.WithFields(logrus.Fields{
		"repo":       repoURL,
		"chart_path": chartPath,
	}).Debug("Fetching all versions from Git repository")

	// Create temporary directory for cloning
	tmpDir, err := os.MkdirTemp("", "argazer-git-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Clone options
	cloneOpts := &git.CloneOptions{
		URL:      repoURL,
		Progress: nil,
		Tags:     git.AllTags,
	}

	// Add authentication if provided
	if g.username != "" && g.password != "" {
		cloneOpts.Auth = &http.BasicAuth{
			Username: g.username,
			Password: g.password,
		}
	}

	// Clone the repository
	repo, err := git.PlainCloneContext(ctx, tmpDir, false, cloneOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get all tags
	tags, err := repo.Tags()
	if err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	var versions []string
	err = tags.ForEach(func(ref *plumbing.Reference) error {
		tagName := ref.Name().Short()

		// Try to parse as semantic version
		versionStr := strings.TrimPrefix(tagName, "v")
		versionStr = strings.TrimPrefix(versionStr, "release-")
		versionStr = strings.TrimPrefix(versionStr, "chart-")

		// If chartPath is specified, look for tags like "chartname-v1.2.3"
		if chartPath != "" {
			chartName := filepath.Base(chartPath)
			prefix := chartName + "-"
			if strings.HasPrefix(tagName, prefix) {
				versionStr = strings.TrimPrefix(tagName, prefix)
				versionStr = strings.TrimPrefix(versionStr, "v")
			}
		}

		_, err := semver.NewVersion(versionStr)
		if err != nil {
			// Not a valid semver tag, skip it
			return nil
		}

		versions = append(versions, versionStr)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error processing tags: %w", err)
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no valid semantic version tags found in repository")
	}

	g.logger.WithFields(logrus.Fields{
		"repo":           repoURL,
		"total_versions": len(versions),
	}).Debug("Found versions from Git tags")

	return versions, nil
}
