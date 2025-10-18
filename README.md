![Claude Assisted](https://img.shields.io/badge/Made%20with-Claude-8A2BE2?logo=anthropic)

# Argazer

**Argazer** (a wordplay on "Argo" and "gazer") is a lightweight tool that monitors your ArgoCD applications for Helm chart updates. It connects to ArgoCD via API, scans your applications, and notifies you when newer versions are available.

## Features

- **Single-run execution** - Runs once on launch, perfect for CI/CD or cron jobs
- **Flexible filtering** - Filter by projects, application names, and labels
- **Multiple notification channels** - Telegram, Email, or console-only output
- **Secure ArgoCD connection** - Username/password authentication with optional TLS verification
- **Environment variable support** - All settings configurable via AG_* environment variables
- **Structured JSON logging** - All logs output in JSON format for easy parsing and integration
- **Graceful error handling** - Skips unsupported repositories (OCI, container registries) with clear messages
- **Multi-source support** - Handles ArgoCD applications with multiple Helm sources

## Installation

### From Source

```bash
git clone <repository-url>
cd argazer
go build -o argazer .
```

### Using Docker

```bash
# Build the image
docker build -t argazer:latest .

# Or use pre-built image (if available)
docker pull ghcr.io/your-org/argazer:latest
```

## Configuration

Argazer can be configured via:
1. Configuration file (config.yaml)
2. Command-line flags
3. Environment variables (prefixed with `AG_`)

### Configuration File

Create a `config.yaml` file:

```yaml
# ArgoCD Connection
argocd_url: "argocd.example.com"  # Just hostname, no https:// prefix
argocd_username: "admin"
argocd_password: "your-password"
argocd_insecure: false  # Set to true to skip TLS verification

# Search Scope
projects:
  - "*"  # All projects, or specify: ["project1", "project2"]
app_names:
  - "*"  # All apps, or specify: ["app1", "app2"]
labels:  # Optional: filter by labels
  type: "operator"
  environment: "production"

# Notification Channel ("telegram", "email", or empty for console-only)
notification_channel: "telegram"

# Telegram Settings
telegram_webhook: "https://api.telegram.org/botTOKEN/sendMessage"
telegram_chat_id: "123456789"

# Email Settings
email_smtp_host: "smtp.gmail.com"
email_smtp_port: 587
email_smtp_username: "your-email@gmail.com"
email_smtp_password: "your-app-password"
email_from: "argazer@example.com"
email_to:
  - "devops@example.com"
email_use_tls: true

# General
verbose: false
source_name: "chart-repo"  # For multi-source apps, specify which source to check
```

### Environment Variables

All configuration options can be set via environment variables with the `AG_` prefix:

```bash
# ArgoCD Connection
export AG_ARGOCD_URL="argocd.example.com"
export AG_ARGOCD_USERNAME="admin"
export AG_ARGOCD_PASSWORD="your-password"
export AG_ARGOCD_INSECURE="false"

# Search Scope
export AG_PROJECTS="project1,project2"  # or "*" for all
export AG_APP_NAMES="app1,app2"         # or "*" for all
export AG_LABELS="type=operator,environment=production"  # Format: key1=value1,key2=value2

# Notification
export AG_NOTIFICATION_CHANNEL="telegram"  # "telegram", "email", or empty

# Telegram
export AG_TELEGRAM_WEBHOOK="https://api.telegram.org/botTOKEN/sendMessage"
export AG_TELEGRAM_CHAT_ID="123456789"

# Email
export AG_EMAIL_SMTP_HOST="smtp.gmail.com"
export AG_EMAIL_SMTP_PORT="587"
export AG_EMAIL_SMTP_USERNAME="your-email@gmail.com"
export AG_EMAIL_SMTP_PASSWORD="your-app-password"
export AG_EMAIL_FROM="argazer@example.com"
export AG_EMAIL_TO="devops@example.com,team@example.com"
export AG_EMAIL_USE_TLS="true"

# General
export AG_VERBOSE="false"
export AG_SOURCE_NAME="chart-repo"
```

## Usage

### Basic Usage

```bash
# Run with config file
./argazer --config config.yaml

# Run with environment variables
AG_ARGOCD_URL="argocd.example.com" \
AG_ARGOCD_USERNAME="admin" \
AG_ARGOCD_PASSWORD="password" \
./argazer

# Run with flags
./argazer \
  --argocd-url="argocd.example.com" \
  --argocd-username="admin" \
  --argocd-password="password" \
  --projects="production" \
  --notification-channel="telegram"
```

### Filtering Examples

```bash
# Check all applications in all projects
./argazer --projects="*" --app-names="*"

# Check specific projects
./argazer --projects="production,staging"

# Check specific applications
./argazer --app-names="frontend,backend"

# Filter by labels (using environment variable)
AG_LABELS="type=operator,environment=production" ./argazer

# Combine filters
./argazer --projects="production" --app-names="frontend,backend"

# Using config file with label filters (see config.yaml example)
./argazer --config config.yaml
```

### Notification Examples

```bash
# Console output only (no notifications)
./argazer --notification-channel=""

# Send Telegram notifications
./argazer --notification-channel="telegram"

# Send Email notifications
./argazer --notification-channel="email"
```

### Cron Job Example

Add to your crontab to run every hour:

```cron
0 * * * * /path/to/argazer --config /path/to/config.yaml
```

### Docker Usage

```bash
# Using config file (mount to /app/config.yaml)
docker run --rm \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  argazer:latest

# Using environment variables only
docker run --rm \
  -e AG_ARGOCD_URL="argocd.example.com" \
  -e AG_ARGOCD_USERNAME="admin" \
  -e AG_ARGOCD_PASSWORD="password" \
  -e AG_NOTIFICATION_CHANNEL="telegram" \
  -e AG_TELEGRAM_WEBHOOK="https://api.telegram.org/bot.../sendMessage" \
  -e AG_TELEGRAM_CHAT_ID="123456789" \
  argazer:latest

# Override default config path
docker run --rm \
  -v $(pwd)/custom-config.yaml:/config/argazer.yaml:ro \
  argazer:latest --config /config/argazer.yaml

# With verbose logging
docker run --rm \
  -v $(pwd)/config.yaml:/app/config.yaml:ro \
  argazer:latest --verbose

# Using docker-compose (see example below)
docker-compose up
```

#### Docker Compose Example

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  argazer:
    image: argazer:latest
    build: .
    environment:
      - AG_ARGOCD_URL=argocd.example.com
      - AG_ARGOCD_USERNAME=admin
      - AG_ARGOCD_PASSWORD=${ARGOCD_PASSWORD}  # Use .env file
      - AG_PROJECTS=production,staging
      - AG_LABELS=type=operator,environment=production  # Optional: filter by labels
      - AG_NOTIFICATION_CHANNEL=telegram
      - AG_TELEGRAM_WEBHOOK=${TELEGRAM_WEBHOOK}
      - AG_TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID}
    # Or mount config file instead
    # volumes:
    #   - ./config.yaml:/app/config.yaml:ro
    restart: "no"  # Run once and exit
```

Then run:

```bash
# Create .env file with sensitive data
cat > .env << EOF
ARGOCD_PASSWORD=your-password
TELEGRAM_WEBHOOK=https://api.telegram.org/bot.../sendMessage
TELEGRAM_CHAT_ID=123456789
EOF

# Run
docker-compose up
```

## Output Example

### Console Output

```
================================================================================
ARGAZER SCAN RESULTS
================================================================================

Total applications checked: 25

Up to date: 20
Updates available: 3
Skipped: 2

--------------------------------------------------------------------------------
APPLICATIONS WITH UPDATES AVAILABLE:
--------------------------------------------------------------------------------

Application: frontend
  Project: production
  Chart: nginx
  Current Version: 1.20.0
  Latest Version: 1.21.0
  Repository: https://charts.bitnami.com/bitnami

Application: backend
  Project: production
  Chart: postgresql
  Current Version: 11.9.13
  Latest Version: 11.10.0
  Repository: https://charts.bitnami.com/bitnami

--------------------------------------------------------------------------------
APPLICATIONS SKIPPED (Unable to check):
--------------------------------------------------------------------------------

Application: internal-app
  Project: platform
  Chart: custom-chart
  Repository: cr.example.com/helm
  Reason: repository returned HTML instead of YAML - likely an OCI/container registry, not a traditional Helm repository

================================================================================
```

### JSON Logs

All operational logs are output in structured JSON format:

```json
{"level":"info","msg":"Starting Argazer","argocd_url":"argo.example.com","projects":["production"],"app_names":["*"],"labels":{"type":"operator"},"time":"2025-10-15T12:00:00+00:00"}
{"level":"info","msg":"Creating ArgoCD API client","server":"argo.example.com","username":"admin","time":"2025-10-15T12:00:01+00:00"}
{"level":"info","msg":"Found applications","count":25,"time":"2025-10-15T12:00:02+00:00"}
{"level":"info","msg":"Processing application","app_name":"frontend","project":"production","time":"2025-10-15T12:00:02+00:00"}
{"level":"info","msg":"Found Helm-based application","app_name":"frontend","chart_name":"nginx","chart_version":"1.20.0","time":"2025-10-15T12:00:02+00:00"}
{"level":"warning","msg":"Update available!","app_name":"frontend","current_version":"1.20.0","latest_version":"1.21.0","time":"2025-10-15T12:00:03+00:00"}
{"level":"info","msg":"Argazer completed","total_checked":25,"time":"2025-10-15T12:00:15+00:00"}
```

## Notification Formats

### Telegram

Argazer sends formatted Markdown messages to Telegram with visual separators:

```
*Argazer Alert*

Found 2 application(s) with Helm chart updates:

----------------------------------------
*Application:* `frontend`
*Project:* `production`
*Chart:* `nginx`
*Current:* `1.20.0`
*Latest:* `1.21.0`
*Repo:* `https://charts.bitnami.com/bitnami`
----------------------------------------

----------------------------------------
*Application:* `backend`
*Project:* `production`
*Chart:* `postgresql`
*Current:* `11.9.13`
*Latest:* `11.10.0`
*Repo:* `https://charts.bitnami.com/bitnami`
----------------------------------------
```

### Email

Plain text email with clear formatting and visual separators:

```
Subject: Argazer Alert: 2 Helm Chart Update(s) Available

Argazer found 2 application(s) with Helm chart updates:

------------------------------------------------------------
Application: frontend
Project: production
Chart: nginx
Current Version: 1.20.0
Latest Version: 1.21.0
Repository: https://charts.bitnami.com/bitnami
------------------------------------------------------------

------------------------------------------------------------
Application: backend
Project: production
Chart: postgresql
Current Version: 11.9.13
Latest Version: 11.10.0
Repository: https://charts.bitnami.com/bitnami
------------------------------------------------------------
```

## Development

### Prerequisites

- Go 1.21 or higher
- Access to an ArgoCD instance

### Building

```bash
go build -o argazer .
```


## CI/CD Integration

### GitHub Actions

To run Argazer on a schedule (e.g., check for updates every 6 hours):

```yaml
name: Check Helm Updates
on:
  schedule:
    - cron: '0 */6 * * *'  # Every 6 hours
  workflow_dispatch:  # Allow manual trigger

jobs:
  check:
    runs-on: ubuntu-latest
    steps:
      - name: Run Argazer
        uses: docker://ghcr.io/<owner>/<repo>:<version>
        env:
          AG_ARGOCD_URL: ${{ secrets.ARGOCD_URL }}
          AG_ARGOCD_USERNAME: ${{ secrets.ARGOCD_USERNAME }}
          AG_ARGOCD_PASSWORD: ${{ secrets.ARGOCD_PASSWORD }}
          AG_NOTIFICATION_CHANNEL: telegram
          AG_TELEGRAM_WEBHOOK: ${{ secrets.TELEGRAM_WEBHOOK }}
          AG_TELEGRAM_CHAT_ID: ${{ secrets.TELEGRAM_CHAT_ID }}
          AG_PROJECTS: production
          AG_LABELS: type=operator
```

### GitLab CI

```yaml
argazer-check:
  stage: check
  image: ghcr.io/your-org/argazer:latest
  script:
    - argazer
  variables:
    AG_ARGOCD_URL: ${ARGOCD_URL}
    AG_ARGOCD_USERNAME: ${ARGOCD_USERNAME}
    AG_ARGOCD_PASSWORD: ${ARGOCD_PASSWORD}
    AG_NOTIFICATION_CHANNEL: telegram
    AG_TELEGRAM_WEBHOOK: ${TELEGRAM_WEBHOOK}
    AG_TELEGRAM_CHAT_ID: ${TELEGRAM_CHAT_ID}
  only:
    - schedules
```

## Troubleshooting

### Connection Issues

If you're having trouble connecting to ArgoCD:

1. Verify the URL format is correct (use `argocd.example.com`, not `https://argocd.example.com`)
2. Check username/password credentials
3. Try with `--argocd-insecure=true` for self-signed certificates
4. Use `--verbose` flag for detailed JSON logging

### No Applications Found

If Argazer reports 0 applications:

1. Check your project and app name filters
2. Verify label selectors if using them (format: `AG_LABELS=key1=value1,key2=value2`)
3. Ensure your user has permission to list applications
4. Try with `--projects="*" --app-names="*"` and without label filters to check all apps
5. Verify labels exist on your ArgoCD applications (check in ArgoCD UI)

### Email Not Sending

For email notification issues:

1. Verify SMTP settings are correct
2. Check if you need an "app password" for Gmail
3. Ensure firewall allows SMTP traffic
4. Try with `email_use_tls: false` if needed

### Applications Skipped

If applications are being skipped with "OCI/container registry" messages:

1. This is expected behavior - OCI repositories don't support traditional Helm index.yaml files
2. Argazer can only check traditional Helm repositories (HTTP/HTTPS with index.yaml)
3. For OCI repositories, version checking must be done differently (not currently supported)
4. These skipped applications won't affect the update notifications for other apps

### Multi-Source Applications

For applications with multiple Helm sources:

1. Set `source_name` in config to match the specific source name you want to check
2. Default is `"chart-repo"` - change if your sources use different names
3. If no matching source name is found, Argazer will use the first Helm source it finds
4. Use `--verbose` to see which sources are being checked

## License

GPL-3.0 license

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
