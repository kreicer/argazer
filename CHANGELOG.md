# Changelog

All notable changes to this project will be documented in this file.

## [1.0.1] - 2025-10-19

### Added
- **OCI Registry Support** - Full support for OCI-based Helm chart repositories (Harbor, GHCR, ACR, etc.)
- **Authentication for Private Repositories** - Support for authenticated access to private Helm and OCI registries
  - Environment variable authentication: `AG_AUTH_URL_*`, `AG_AUTH_USER_*`, `AG_AUTH_PASS_*`
  - Config file authentication via `repository_auth` section
- **Configurable Concurrency** - Control the number of concurrent workers via `--concurrency` flag, `AG_CONCURRENCY` env var, or config file (default: 10)
- **Comprehensive Test Suite** - Added 17 integration tests using `httptest` for Helm and OCI checkers
- **Custom Error Types** - Programmatic error handling with `ErrChartNotFound`, `ErrAuthenticationFailed`, `ErrNoValidVersions`

### Changed
- **Improved Version Comparison** - Non-semantic versions are now logged and excluded rather than using unreliable string comparison
- **Refactored Version Logic** - Extracted shared `findLatestSemver()` utility function for better code reuse
- **Smarter OCI Tag Filtering** - Uses proper semver library to parse and validate all tags
- **Compact Notification Messages** - Telegram/Email messages are now more concise and readable
- **Message Splitting** - Long notifications automatically split into multiple messages (respects Telegram's 4096 character limit)
- **Simplified Configuration** - Removed redundant environment variable binding code

### Fixed
- **Plain Text Notifications** - Removed Markdown parse mode from Telegram to prevent formatting issues with special characters
- **Localhost Registry Support** - OCI checker now correctly uses `http://` for localhost/testing registries
- **Authentication Error Messages** - Clearer error messages for authentication failures with actionable guidance

### Documentation
- Added comprehensive authentication guide with security warnings
- Added ArgoCD RBAC setup section with examples
- Updated notification format examples
- Added concurrency configuration documentation

## [1.0.0] - 2025-10-15

### Initial Release
- ArgoCD application monitoring via API
- Traditional Helm repository support
- Telegram and Email notifications
- Project and application filtering
- Label-based filtering
- Multi-source application support
- Structured JSON logging

