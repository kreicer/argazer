package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"argazer/internal/argocd"
	"argazer/internal/auth"
	"argazer/internal/config"
	"argazer/internal/helm"
	"argazer/internal/notification"
)

var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "argazer",
		Short: "ArgoCD Application Gazer - Monitor Helm chart versions in ArgoCD applications",
		Long: `Argazer connects to ArgoCD via API and checks all applications for Helm chart updates.
It can filter by projects, application names, and labels, and send notifications via Telegram, Email, Slack, Microsoft Teams, or generic webhooks.`,
		RunE: run,
	}

	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("argazer version %s (commit: %s)\n", version, commit)
		},
	})

	// Add flags
	rootCmd.Flags().StringP("config", "c", "", "Configuration file path")
	rootCmd.Flags().String("argocd-url", "", "ArgoCD server URL")
	rootCmd.Flags().String("argocd-username", "", "ArgoCD username")
	rootCmd.Flags().String("argocd-password", "", "ArgoCD password")
	rootCmd.Flags().Bool("argocd-insecure", false, "Skip TLS verification")
	rootCmd.Flags().StringSlice("projects", []string{"*"}, "Projects to check (comma-separated, or '*' for all)")
	rootCmd.Flags().StringSlice("app-names", []string{"*"}, "Application names to check (comma-separated, or '*' for all)")
	rootCmd.Flags().String("notification-channel", "", "Notification channel: 'telegram', 'email', 'slack', 'teams', 'webhook', or empty for console only")
	rootCmd.Flags().Int("concurrency", 10, "Number of concurrent workers for checking applications")
	rootCmd.Flags().String("version-constraint", "major", "Version constraint: 'major' (all), 'minor' (same major), 'patch' (same major.minor)")
	rootCmd.Flags().BoolP("verbose", "v", false, "Enable verbose logging")

	// Bind flags to viper
	if err := viper.BindPFlags(rootCmd.Flags()); err != nil {
		logrus.WithError(err).Fatal("Failed to bind flags")
	}

	if err := rootCmd.Execute(); err != nil {
		logrus.Fatal(err)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Set up logging
	logger := setupLogging(cfg.Verbose)

	logger.WithFields(logrus.Fields{
		"argocd_url":   cfg.ArgocdURL,
		"projects":     cfg.Projects,
		"app_names":    cfg.AppNames,
		"labels":       cfg.Labels,
		"notification": cfg.NotificationChannel,
		"version":      version,
	}).Info("Starting Argazer")

	ctx := context.Background()

	// Initialize clients
	clients, err := initializeClients(ctx, cfg, logger)
	if err != nil {
		return err
	}

	// Fetch applications from ArgoCD
	apps, err := fetchApplications(ctx, clients.argocd, cfg, logger)
	if err != nil {
		return err
	}

	// Check applications for updates (with concurrency)
	results := checkApplicationsConcurrently(ctx, apps, clients.helm, cfg, logger)

	// Output results to console
	outputResults(results)

	// Send notifications if configured
	if clients.notifier != nil {
		if err := sendNotifications(ctx, clients.notifier, results, logger); err != nil {
			logger.WithError(err).Warn("Failed to send notifications")
		}
	}

	logger.WithField("total_checked", len(results)).Info("Argazer completed")

	return nil
}

// clients holds all initialized clients
type clients struct {
	argocd   *argocd.Client
	helm     *helm.Checker
	notifier notification.Notifier
}

// initializeClients creates all required clients (ArgoCD, Helm, Notifier)
func initializeClients(_ context.Context, cfg *config.Config, logger *logrus.Entry) (*clients, error) {
	c := &clients{}

	// Create authentication provider
	authLogger := logger.WithField("component", "auth")

	// Convert config auth to auth provider format
	var configAuth []auth.ConfigAuth
	for _, ra := range cfg.RepositoryAuth {
		configAuth = append(configAuth, auth.ConfigAuth{
			URL:      ra.URL,
			Username: ra.Username,
			Password: ra.Password,
		})
	}

	authProvider, err := auth.NewProvider(configAuth, authLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth provider: %w", err)
	}

	// Create ArgoCD API client
	argoLogger := logger.WithField("component", "argocd")
	argoClient, err := argocd.NewClient(cfg.ArgocdURL, cfg.ArgocdUsername, cfg.ArgocdPassword, cfg.ArgocdInsecure, argoLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create ArgoCD client: %w", err)
	}
	c.argocd = argoClient

	// Create helm checker
	helmLogger := logger.WithField("component", "helm")
	helmChecker, err := helm.NewChecker(authProvider, helmLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create helm checker: %w", err)
	}
	c.helm = helmChecker

	// Create notifier based on configuration
	if cfg.NotificationChannel != "" {
		notifierLogger := logger.WithField("component", "notifier")
		var notifier notification.Notifier

		switch cfg.NotificationChannel {
		case "telegram":
			notifier = notification.NewTelegramNotifier(cfg.TelegramWebhook, cfg.TelegramChatID, notifierLogger)
			logger.Info("Using Telegram notifications")
		case "email":
			notifier = notification.NewEmailNotifier(
				cfg.EmailSmtpHost,
				cfg.EmailSmtpPort,
				cfg.EmailSmtpUsername,
				cfg.EmailSmtpPassword,
				cfg.EmailFrom,
				cfg.EmailTo,
				cfg.EmailUseTLS,
				notifierLogger,
			)
			logger.Info("Using Email notifications")
		case "slack":
			notifier = notification.NewSlackNotifier(cfg.SlackWebhook, notifierLogger)
			logger.Info("Using Slack notifications")
		case "teams":
			notifier = notification.NewTeamsNotifier(cfg.TeamsWebhook, notifierLogger)
			logger.Info("Using Microsoft Teams notifications")
		case "webhook":
			notifier = notification.NewWebhookNotifier(cfg.WebhookURL, notifierLogger)
			logger.Info("Using generic webhook notifications")
		default:
			logger.Warnf("Unknown notification channel: %s", cfg.NotificationChannel)
		}

		c.notifier = notifier
	}

	return c, nil
}

// fetchApplications retrieves applications from ArgoCD based on filters
func fetchApplications(ctx context.Context, client *argocd.Client, cfg *config.Config, logger *logrus.Entry) ([]*v1alpha1.Application, error) {
	apps, err := client.ListApplications(ctx, argocd.FilterOptions{
		Projects: cfg.Projects,
		AppNames: cfg.AppNames,
		Labels:   cfg.Labels,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list applications: %w", err)
	}

	logger.WithField("count", len(apps)).Info("Found applications")
	return apps, nil
}

// ApplicationCheckResult holds the result of checking an application
type ApplicationCheckResult struct {
	AppName                    string
	Project                    string
	ChartName                  string
	CurrentVersion             string
	LatestVersion              string
	RepoURL                    string
	HasUpdate                  bool
	Error                      error
	ConstraintApplied          string // Version constraint used: "major", "minor", or "patch"
	HasUpdateOutsideConstraint bool   // True if updates exist outside the constraint
	LatestVersionAll           string // Latest version without constraint (if different)
}

// checkApplicationsConcurrently checks multiple applications in parallel using a worker pool
func checkApplicationsConcurrently(ctx context.Context, apps []*v1alpha1.Application, helmChecker *helm.Checker, cfg *config.Config, logger *logrus.Entry) []ApplicationCheckResult {
	numWorkers := cfg.Concurrency
	if numWorkers <= 0 {
		numWorkers = 10 // Fallback to default
	}

	logger.WithField("concurrency", numWorkers).Debug("Starting concurrent application checks")

	// Create channels for work distribution
	appChan := make(chan *v1alpha1.Application, len(apps))
	resultChan := make(chan ApplicationCheckResult, len(apps))

	// Start workers
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workerLogger := logger.WithField("worker_id", workerID)
			for app := range appChan {
				result := checkApplication(ctx, app, helmChecker, cfg, workerLogger)
				resultChan <- result
			}
		}(i)
	}

	// Send applications to workers
	for _, app := range apps {
		appChan <- app
	}
	close(appChan)

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	results := make([]ApplicationCheckResult, 0, len(apps))
	for result := range resultChan {
		results = append(results, result)
	}

	return results
}

// checkApplication checks a single application for Helm chart updates
func checkApplication(ctx context.Context, app *v1alpha1.Application, helmChecker *helm.Checker, cfg *config.Config, logger *logrus.Entry) ApplicationCheckResult {
	appLogger := logger.WithFields(logrus.Fields{
		"app_name": app.Name,
		"project":  app.Spec.Project,
	})

	appLogger.Info("Processing application")

	// Find Helm source
	helmSource := findHelmSource(app, cfg.SourceName, appLogger)
	if helmSource == nil {
		appLogger.Debug("Application does not use Helm charts, skipping")
		return ApplicationCheckResult{} // Return empty result, will be filtered out
	}

	result := ApplicationCheckResult{
		AppName:           app.Name,
		Project:           app.Spec.Project,
		ChartName:         helmSource.Chart,
		CurrentVersion:    helmSource.TargetRevision,
		RepoURL:           helmSource.RepoURL,
		ConstraintApplied: cfg.VersionConstraint,
	}

	appLogger = appLogger.WithFields(logrus.Fields{
		"chart_name":    helmSource.Chart,
		"chart_version": helmSource.TargetRevision,
		"repo_url":      helmSource.RepoURL,
		"constraint":    cfg.VersionConstraint,
	})

	appLogger.Info("Found Helm-based application")

	// Check for newer version with constraint
	constraintResult, err := helmChecker.GetLatestVersionWithConstraint(
		ctx,
		helmSource.RepoURL,
		helmSource.Chart,
		helmSource.TargetRevision,
		cfg.VersionConstraint,
	)
	if err != nil {
		appLogger.WithError(err).Error("Failed to check Helm version")
		result.Error = err
		return result
	}

	result.LatestVersion = constraintResult.LatestVersion
	result.LatestVersionAll = constraintResult.LatestVersionAll
	result.HasUpdateOutsideConstraint = constraintResult.HasUpdateOutsideConstraint

	if constraintResult.LatestVersion != helmSource.TargetRevision {
		appLogger.WithFields(logrus.Fields{
			"current_version":               helmSource.TargetRevision,
			"latest_version":                constraintResult.LatestVersion,
			"latest_version_all":            constraintResult.LatestVersionAll,
			"has_update_outside_constraint": constraintResult.HasUpdateOutsideConstraint,
		}).Warn("Update available!")
		result.HasUpdate = true
	} else {
		if constraintResult.HasUpdateOutsideConstraint {
			appLogger.WithFields(logrus.Fields{
				"current_version":    helmSource.TargetRevision,
				"latest_version_all": constraintResult.LatestVersionAll,
				"constraint":         cfg.VersionConstraint,
			}).Info("Application is up to date within constraint, but updates exist outside constraint")
		} else {
			appLogger.Info("Application is up to date")
		}
	}

	return result
}

// findHelmSource finds the Helm source in an ArgoCD application
func findHelmSource(app *v1alpha1.Application, sourceName string, logger *logrus.Entry) *v1alpha1.ApplicationSource {
	// Check if it's a single source application with Helm
	if app.Spec.Source != nil && app.Spec.Source.Chart != "" {
		return app.Spec.Source
	}

	// Check multi-source applications
	if app.Spec.Sources != nil {
		// If sourceName is specified, look for that specific source first
		if sourceName != "" {
			for i := range app.Spec.Sources {
				source := &app.Spec.Sources[i]
				// Match by name AND ensure it's a Helm chart
				if source.Name == sourceName && source.Chart != "" {
					logger.WithFields(logrus.Fields{
						"app":         app.Name,
						"source_name": source.Name,
						"chart":       source.Chart,
						"repo":        source.RepoURL,
					}).Debug("Found matching Helm source by name")
					return source
				}
			}
		}

		// Fallback: find any Helm source
		for i := range app.Spec.Sources {
			source := &app.Spec.Sources[i]
			if source.Chart != "" {
				logger.WithFields(logrus.Fields{
					"app":         app.Name,
					"source_name": source.Name,
					"chart":       source.Chart,
					"repo":        source.RepoURL,
				}).Debug("Found Helm source (fallback)")
				return source
			}
		}
	}

	return nil
}

// scanResults holds statistics about the scan
type scanResults struct {
	total    int
	upToDate int
	updates  int
	skipped  int
}

// outputResults displays the results to console
func outputResults(results []ApplicationCheckResult) {
	// Calculate stats in a single loop
	stats := scanResults{}
	var updatesAvailable []ApplicationCheckResult
	var upToDateWithConstraint []ApplicationCheckResult
	var errors []ApplicationCheckResult

	for _, result := range results {
		// Skip empty results (non-Helm apps)
		if result.AppName == "" {
			continue
		}

		stats.total++

		if result.Error != nil {
			stats.skipped++
			errors = append(errors, result)
		} else if result.HasUpdate {
			stats.updates++
			updatesAvailable = append(updatesAvailable, result)
		} else {
			stats.upToDate++
			// Track apps that are up to date but have updates outside constraint
			if result.HasUpdateOutsideConstraint {
				upToDateWithConstraint = append(upToDateWithConstraint, result)
			}
		}
	}

	// Display summary
	fmt.Println("\n" + strings.Repeat("=", 80))
	fmt.Println("ARGAZER SCAN RESULTS")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("\nTotal applications checked: %d\n\n", stats.total)
	fmt.Printf("Up to date: %d\n", stats.upToDate)
	fmt.Printf("Updates available: %d\n", stats.updates)
	fmt.Printf("Skipped: %d\n\n", stats.skipped)

	// Display updates
	if stats.updates > 0 {
		fmt.Println(strings.Repeat("-", 80))
		fmt.Println("APPLICATIONS WITH UPDATES AVAILABLE:")
		fmt.Println(strings.Repeat("-", 80))

		for _, result := range updatesAvailable {
			fmt.Printf("\nApplication: %s\n", result.AppName)
			fmt.Printf("  Project: %s\n", result.Project)
			fmt.Printf("  Chart: %s\n", result.ChartName)
			fmt.Printf("  Current Version: %s\n", result.CurrentVersion)
			fmt.Printf("  Latest Version: %s\n", result.LatestVersion)
			if result.ConstraintApplied != "major" && result.ConstraintApplied != "" {
				fmt.Printf("  Version Constraint: %s\n", result.ConstraintApplied)
			}
			if result.HasUpdateOutsideConstraint && result.LatestVersionAll != "" {
				fmt.Printf("  Note: Version %s available outside constraint\n", result.LatestVersionAll)
			}
			fmt.Printf("  Repository: %s\n", result.RepoURL)
		}
	}

	// Display apps that are up to date but have updates outside constraint
	if len(upToDateWithConstraint) > 0 {
		fmt.Println("\n" + strings.Repeat("-", 80))
		fmt.Println("UP TO DATE (with updates outside constraint):")
		fmt.Println(strings.Repeat("-", 80))

		for _, result := range upToDateWithConstraint {
			fmt.Printf("\nApplication: %s\n", result.AppName)
			fmt.Printf("  Project: %s\n", result.Project)
			fmt.Printf("  Chart: %s\n", result.ChartName)
			fmt.Printf("  Current Version: %s\n", result.CurrentVersion)
			fmt.Printf("  Status: Up to date within '%s' constraint\n", result.ConstraintApplied)
			if result.LatestVersionAll != "" {
				fmt.Printf("  Note: Version %s available outside constraint\n", result.LatestVersionAll)
			}
			fmt.Printf("  Repository: %s\n", result.RepoURL)
		}
	}

	// Display skipped applications
	if stats.skipped > 0 {
		fmt.Println("\n" + strings.Repeat("-", 80))
		fmt.Println("APPLICATIONS SKIPPED (Unable to check):")
		fmt.Println(strings.Repeat("-", 80))

		for _, result := range errors {
			fmt.Printf("\nApplication: %s\n", result.AppName)
			fmt.Printf("  Project: %s\n", result.Project)
			fmt.Printf("  Chart: %s\n", result.ChartName)
			fmt.Printf("  Repository: %s\n", result.RepoURL)
			fmt.Printf("  Reason: %s\n", result.Error.Error())
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 80) + "\n")
}

// sendNotifications sends notifications via the configured notifier
func sendNotifications(ctx context.Context, notifier notification.Notifier, results []ApplicationCheckResult, logger *logrus.Entry) error {
	// Check if there are updates in a single loop
	var updatesAvailable []ApplicationCheckResult
	for _, result := range results {
		if result.HasUpdate {
			updatesAvailable = append(updatesAvailable, result)
		}
	}

	if len(updatesAvailable) == 0 {
		logger.Info("No updates available, skipping notification")
		return nil
	}

	// Build notification messages (may be split if too long)
	messages := buildNotificationMessages(updatesAvailable)

	logger.WithField("message_count", len(messages)).Info("Sending notifications")

	// Send all messages
	for i, msg := range messages {
		subject := fmt.Sprintf("Argazer Notification: %d Helm Chart Update(s) Available", len(updatesAvailable))
		if len(messages) > 1 {
			subject = fmt.Sprintf("Argazer Notification [%d/%d]: %d Update(s)", i+1, len(messages), len(updatesAvailable))
		}

		if err := notifier.Send(ctx, subject, msg); err != nil {
			return fmt.Errorf("failed to send notification %d/%d: %w", i+1, len(messages), err)
		}
	}

	logger.Info("Successfully sent all notifications")
	return nil
}

// buildNotificationMessages builds notification message(s), splitting if needed for length limits
// Telegram has a 4096 character limit per message
func buildNotificationMessages(updates []ApplicationCheckResult) []string {
	const maxMessageLength = 3900 // Leave some room for headers and safety margin

	// Build individual app update strings
	var appMessages []string
	for _, result := range updates {
		var app strings.Builder
		// Compact format: app name as header with project
		app.WriteString(fmt.Sprintf("%s (%s)\n", result.AppName, result.Project))
		app.WriteString(fmt.Sprintf("  Chart: %s\n", result.ChartName))
		app.WriteString(fmt.Sprintf("  Version: %s -> %s\n", result.CurrentVersion, result.LatestVersion))
		// Show constraint if not "major" (default)
		if result.ConstraintApplied != "major" && result.ConstraintApplied != "" {
			app.WriteString(fmt.Sprintf("  Constraint: %s\n", result.ConstraintApplied))
		}
		// Show note if updates exist outside constraint
		if result.HasUpdateOutsideConstraint && result.LatestVersionAll != "" && result.LatestVersionAll != result.LatestVersion {
			app.WriteString(fmt.Sprintf("  Note: v%s available outside constraint\n", result.LatestVersionAll))
		}
		app.WriteString(fmt.Sprintf("  Repo: %s\n", result.RepoURL))
		app.WriteString("\n")
		appMessages = append(appMessages, app.String())
	}

	// Build header (empty for first message, apps only)
	header := ""

	// Check if we can fit everything in one message
	totalLength := len(header)
	for _, msg := range appMessages {
		totalLength += len(msg)
	}

	if totalLength <= maxMessageLength {
		// Everything fits in one message
		var message strings.Builder
		message.WriteString(header)
		for _, msg := range appMessages {
			message.WriteString(msg)
		}
		return []string{message.String()}
	}

	// Need to split into multiple messages
	var messages []string
	var currentMessage strings.Builder
	currentLength := 0

	// First message gets the header
	currentMessage.WriteString(header)
	currentLength = len(header)

	for _, appMsg := range appMessages {
		// Check if adding this app would exceed the limit
		if currentLength+len(appMsg) > maxMessageLength {
			// Save current message and start a new one
			messages = append(messages, currentMessage.String())
			currentMessage.Reset()
			currentLength = 0
		}

		currentMessage.WriteString(appMsg)
		currentLength += len(appMsg)
	}

	// Add the last message if it has content
	if currentLength > 0 {
		messages = append(messages, currentMessage.String())
	}

	return messages
}

// setupLogging configures the logging system
func setupLogging(verbose bool) *logrus.Entry {
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	// Use JSON logging format
	logrus.SetFormatter(&logrus.JSONFormatter{})

	// Return a base logger entry
	return logrus.WithField("service", "argazer")
}
