package helm

import "errors"

// Common errors that can be checked with errors.Is()
var (
	// ErrChartNotFound indicates that the requested chart was not found in the repository
	ErrChartNotFound = errors.New("chart not found in repository")

	// ErrAuthenticationFailed indicates that authentication to the repository failed
	ErrAuthenticationFailed = errors.New("authentication failed")

	// ErrNoValidVersions indicates that no valid semantic versions were found
	ErrNoValidVersions = errors.New("no valid semantic versions found")

	// ErrRepositoryUnavailable indicates that the repository could not be reached
	ErrRepositoryUnavailable = errors.New("repository unavailable")
)
