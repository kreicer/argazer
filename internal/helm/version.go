package helm

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
)

// findLatestSemver determines the latest semantic version from a list of version strings.
// It filters out any strings that cannot be parsed as valid semantic versions,
// ensuring only valid versions are compared.
func findLatestSemver(versions []string, logger *logrus.Entry) (string, error) {
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions provided")
	}

	// Parse all versions and keep track of their original strings
	type versionPair struct {
		original string
		parsed   *semver.Version
	}

	var validVersions []versionPair

	for _, v := range versions {
		parsed, err := semver.NewVersion(v)
		if err != nil {
			// Log warning for unparseable versions and skip them
			logger.WithFields(logrus.Fields{
				"version": v,
				"error":   err.Error(),
			}).Debug("Skipping invalid semantic version")
			continue
		}
		validVersions = append(validVersions, versionPair{
			original: v,
			parsed:   parsed,
		})
	}

	if len(validVersions) == 0 {
		return "", ErrNoValidVersions
	}

	// Sort valid versions in descending order
	sort.Slice(validVersions, func(i, j int) bool {
		return validVersions[i].parsed.Compare(validVersions[j].parsed) > 0
	})

	// Return the original string representation of the highest version
	return validVersions[0].original, nil
}
