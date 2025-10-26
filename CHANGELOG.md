# Changelog

All notable changes to this project will be documented in this file.

## [1.1.0] - 2025-10-26

### Added
- **Git Repository Support** - Full support for Git-based Helm chart repositories
  - Automatic detection of Git URLs (github.com, gitlab.com, bitbucket.org, etc.)
  - HTTPS authentication support for private repositories
  - Version detection from Git tags (semver)
  - Support for monorepo chart structures
  - Full version constraint support (major/minor/patch)
- **Interactive Configure Command** - New `argazer configure` command for easy setup
  - Step-by-step interactive wizard
  - Configure ArgoCD connection with validation
  - Select and configure notification channels
  - Set version constraints and output preferences
  - Test notification before saving configuration
  - Saves configuration to config.yaml
  - User-friendly prompts with contextual help
- **Retry Mechanism for Notifications** - HTTP-based notifiers now automatically retry failed requests
  - 3 retry attempts with exponential backoff (1s, 2s, 4s)
  - Retries on network errors, 5xx server errors, and 429 rate limits
  - Includes jitter to prevent thundering herd
  - Respects context cancellation
- **Graceful Shutdown** - Clean application shutdown on SIGINT/SIGTERM signals
  - Context cancellation propagated to all operations
  - In-flight operations complete gracefully
  - Proper resource cleanup

### Changed
- **Repository Type Support** - Checker now supports three repository types: Traditional Helm, OCI, and Git
- **Enhanced Authentication** - Authentication provider now used for Git repositories
- **Improved Repository Detection** - Better automatic detection of repository types
- **Fixed Config File Flag** - `--config` flag now properly loads specified configuration files
- **Better Error Handling** - Error messages now properly serialize in JSON output
- **Improved Output Functions** - Output renderers refactored for better testability and maintainability

### Code Quality Improvements
Following a comprehensive code review, several internal improvements were made:
- Refactored config loading into focused helper functions
- Improved type safety in configure command (using structs vs maps)
- Moved notification formatting to dedicated package
- Enhanced output functions with `io.Writer` parameter
- Better code organization and separation of concerns
- Improved test coverage and reliability

### Technical Details
- New dependencies: `go-git/go-git/v5` for Git operations, `AlecAivazis/survey/v2` for interactive prompts
- New files: `internal/helm/git.go`, `cmd/configure.go`, `internal/notification/formatter.go`
- Added signal handling with `os/signal` and `syscall` packages
- HTTP clients now have explicit timeouts and retry logic
- All tests passing with improved coverage

## [1.0.4] - 2025-10-24

### Added
- **Multiple Output Formats** - Support for table, JSON, and markdown output formats
  - New `--output-format` / `-o` flag with options: `table`, `json`, `markdown`
  - New `AG_OUTPUT_FORMAT` environment variable
  - New `output_format` config option (default: `table`)
  - JSON format provides structured output for programmatic processing
  - Markdown format ideal for reports and documentation
  - Table format preserves original human-readable output (default)
- **Version Constraint Strategy** - Control which version updates to check: `major` (all), `minor` (same major), or `patch` (same major.minor)
  - New `--version-constraint` flag
  - New `AG_VERSION_CONSTRAINT` environment variable
  - New `version_constraint` config option (default: `major`)
- **OCI Constraint Support** - Version constraints now fully supported for OCI registries (Harbor, GHCR, ACR, etc.)
- **Constraint Awareness in Output** - Console and notifications show constraint info and updates outside constraint
- **Comprehensive Constraint Tests** - Added 11 comprehensive test cases covering all constraint scenarios

### Changed
- **Enhanced ApplicationCheckResult** - Added JSON tags for proper serialization
- **Output Architecture** - Refactored output system with separate formatters for each format type
- **Refactored Helm Checker** - Extracted `getChartVersionsFromRepo` helper to eliminate code duplication
- **Refactored OCI Checker** - Extracted `getTagsFromOCI` helper to eliminate code duplication
- **Reduced Code Duplication** - Eliminated ~160 lines of duplicated fetching/parsing logic
- **Improved Maintainability** - Constraint logic centralized in `findLatestSemverWithConstraint`

### Code Quality Improvements
- **Refactored Output Logic** - Extracted common result processing into `processResults()` function
  - Eliminated ~80 lines of duplicated categorization code across output formatters
  - Renamed output functions to `renderTable()`, `renderJSON()`, `renderMarkdown()` for clarity
  - Introduced `categorizedResults` struct for clean data flow
  - Single source of truth for result processing logic
- **Flexible Log Formatting** - Added configurable log format option
  - New `--log-format` / `-l` flag with options: `json` (default), `text`
  - New `AG_LOG_FORMAT` environment variable
  - New `log_format` config option
  - Text format ideal for development/debugging with human-readable output
  - JSON format for production with structured logging
- **Constants for Configuration Values** - Defined constants in `config` package for type safety
  - `OutputFormatTable`, `OutputFormatJSON`, `OutputFormatMarkdown`
  - `VersionConstraintMajor`, `VersionConstraintMinor`, `VersionConstraintPatch`
  - `LogFormatJSON`, `LogFormatText`
  - Reduces typos and improves IDE autocomplete
- **Simplified Concurrency Logic** - Removed unnecessary goroutine from `checkApplicationsConcurrently()`
  - More linear and easier to follow control flow
  - Eliminated extra goroutine for channel closing
- **Explicit Non-Helm App Handling** - Added clear documentation for app skipping logic
  - Added function comments explaining empty `AppName` contract
  - Improved code clarity with explicit skip comments
- **Notification Message Length Constant** - Extracted `maxNotificationMessageLength` as package constant
  - Previously hardcoded value now clearly documented
  - Easy to adjust for different notification services
- **Context Propagation** - Fixed unused context parameter in `initializeClients()`
  - Improved future-proofing for cancellable operations
  - Removed linter warning

### Documentation
- Updated README with output format examples and usage
- Updated `config.yaml.example` with `output_format` and `log_format` documentation
- Added examples for all three output formats
- Added comprehensive "Version Constraint Strategy" section to README
- Added usage examples for all constraint modes
- Updated `config.yaml.example` with constraint examples
- Updated `env.example` with `AG_VERSION_CONSTRAINT`
- Added version constraint use cases and examples
- Added inline comments for future I/O error handling considerations

### Technical Improvements
- Fixed viper flag-to-config mapping using RegisterAlias for proper dash-to-underscore conversion
- Added proper `context.Background()` usage in tests (replaced nil contexts)
- All tests passing with new output format parameter
- All 110+ tests passing with new constraint logic
- Zero linter errors
- Clean separation of concerns between fetching and filtering logic
- Backward compatible - defaults to `major` (all versions)
- Improved code maintainability and readability
- Better separation of concerns
- Reduced code duplication by ~160 lines total

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

