package helm

import (
	"fmt"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
)

// VersionConstraintResult holds the result of version constraint filtering
type VersionConstraintResult struct {
	LatestVersion              string // Latest version within constraint
	LatestVersionAll           string // Latest version without constraint
	HasUpdateOutsideConstraint bool   // True if newer versions exist outside constraint
}

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

// findLatestSemverWithConstraint finds the latest version respecting the given constraint
func findLatestSemverWithConstraint(versions []string, currentVersion, constraint string, logger *logrus.Entry) (*VersionConstraintResult, error) {
	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions provided")
	}

	// Parse current version
	current, err := semver.NewVersion(currentVersion)
	if err != nil {
		// If current version is invalid, fall back to no constraint
		logger.WithFields(logrus.Fields{
			"current_version": currentVersion,
			"error":           err.Error(),
		}).Warn("Current version is not valid semver, checking all versions")
		latest, err := findLatestSemver(versions, logger)
		if err != nil {
			return nil, err
		}
		return &VersionConstraintResult{
			LatestVersion:              latest,
			LatestVersionAll:           latest,
			HasUpdateOutsideConstraint: false,
		}, nil
	}

	// Parse all versions and filter by constraint
	type versionPair struct {
		original string
		parsed   *semver.Version
	}

	var allValidVersions []versionPair
	var constrainedVersions []versionPair

	for _, v := range versions {
		parsed, err := semver.NewVersion(v)
		if err != nil {
			logger.WithFields(logrus.Fields{
				"version": v,
				"error":   err.Error(),
			}).Debug("Skipping invalid semantic version")
			continue
		}

		allValidVersions = append(allValidVersions, versionPair{
			original: v,
			parsed:   parsed,
		})

		// Apply constraint filter
		matchesConstraint := false
		switch constraint {
		case "patch":
			// Same major and minor
			matchesConstraint = parsed.Major() == current.Major() && parsed.Minor() == current.Minor()
		case "minor":
			// Same major only
			matchesConstraint = parsed.Major() == current.Major()
		case "major", "":
			// All versions
			matchesConstraint = true
		}

		if matchesConstraint {
			constrainedVersions = append(constrainedVersions, versionPair{
				original: v,
				parsed:   parsed,
			})
		}
	}

	if len(allValidVersions) == 0 {
		return nil, ErrNoValidVersions
	}

	// Sort both lists
	sortVersions := func(versions []versionPair) {
		sort.Slice(versions, func(i, j int) bool {
			return versions[i].parsed.Compare(versions[j].parsed) > 0
		})
	}

	sortVersions(allValidVersions)
	latestAll := allValidVersions[0].original

	result := &VersionConstraintResult{
		LatestVersionAll:           latestAll,
		HasUpdateOutsideConstraint: false,
	}

	if len(constrainedVersions) == 0 {
		// No versions match constraint, return current as latest within constraint
		result.LatestVersion = currentVersion
		result.HasUpdateOutsideConstraint = latestAll != currentVersion
		return result, nil
	}

	sortVersions(constrainedVersions)
	latestConstrained := constrainedVersions[0].original

	// Only return a newer version if it's actually newer than current
	latestConstrainedVer, _ := semver.NewVersion(latestConstrained)
	if latestConstrainedVer.Compare(current) > 0 {
		result.LatestVersion = latestConstrained
	} else {
		// All constrained versions are older or equal to current
		result.LatestVersion = currentVersion
	}

	// Check if there are newer versions outside constraint
	if constraint != "major" && constraint != "" {
		latestAllVer, _ := semver.NewVersion(latestAll)
		currentVer := current
		result.HasUpdateOutsideConstraint = latestAllVer.Compare(currentVer) > 0 && latestAllVer.Compare(latestConstrainedVer) > 0
	}

	return result, nil
}
