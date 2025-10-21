# Changelog

All notable changes to this project will be documented in this file.

## [1.0.3] - 2025-10-21

### Added
- **Slack Notifications** - Native support for Slack via incoming webhooks with markdown formatting
- **Microsoft Teams Notifications** - MessageCard format support for Teams channels
- **Generic Webhook Notifications** - Flexible JSON payload support for any webhook endpoint
- **Shared HTTP Helper** - New `HTTPNotifier` base class reducing code duplication by 100%
- **Injectable HTTP Client** - All notifiers now support custom HTTP client injection for advanced configuration
- **Type-Safe Payload Structs** - Replaced maps with proper structs for compile-time type safety
- **Centralized Constants** - `UserAgent` and `DefaultHTTPTimeout` constants for consistency

### Changed
- **Refactored Notification System** - All HTTP-based notifiers now use shared `http_helper.go`
- **Reduced Code Duplication** - Eliminated ~200 lines of duplicated HTTP request handling
- **Improved Type Safety** - All JSON payloads now use strongly-typed structs instead of `map[string]interface{}`
- **Better Constructors** - Each notifier has both default and custom client constructors (e.g., `NewSlackNotifier` and `NewSlackNotifierWithClient`)
- **Updated Configuration** - Added `slack_webhook`, `teams_webhook`, and `webhook_url` config fields
- **Switch Statement** - Replaced if-else chain with tagged switch for notification validation (linter-compliant)

### Documentation
- Added comprehensive setup guides for Slack, Teams, and Webhook channels
- Added notification format examples with consistent sample data
- Updated environment variable documentation (`AG_SLACK_WEBHOOK`, `AG_TEAMS_WEBHOOK`, `AG_WEBHOOK_URL`)
- Updated `config.yaml.example` and `env.example` with new channel examples

### Technical Improvements
- Added 32 new tests for Slack, Teams, and Webhook notifiers (all passing)
- Telegram notifier refactored to use shared HTTP helper
- All notifiers now share common timeout and user-agent configuration
- Backward compatibility maintained for all existing constructors

## [1.0.2] - 2025-10-20

### Fixed
- Environment variable configuration handling improvements

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

