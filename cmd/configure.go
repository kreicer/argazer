package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"argazer/internal/config"
	"argazer/internal/notification"

	"github.com/AlecAivazis/survey/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// NewConfigureCmd creates the configure subcommand
func NewConfigureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "configure",
		Short: "Interactively configure Argazer settings",
		Long: `Configure Argazer in interactive mode.

This command will guide you through setting up:
- ArgoCD connection
- Notification channel
- Version constraints
- Output format

The configuration will be saved to config.yaml and the notification channel will be tested.`,
		RunE: runConfigure,
	}
}

// ConfigWizard holds the wizard configuration
type ConfigWizard struct {
	// ArgoCD
	ArgocdURL      string
	ArgocdUsername string
	ArgocdPassword string
	ArgocdInsecure bool

	// Filtering
	Projects []string
	AppNames []string

	// General
	VersionConstraint string
	OutputFormat      string
	LogFormat         string
	Concurrency       int

	// Notification
	NotificationChannel string

	// Telegram
	TelegramWebhook string
	TelegramChatID  string

	// Email
	EmailSMTPHost     string
	EmailSMTPPort     int
	EmailSMTPUsername string
	EmailSMTPPassword string
	EmailFrom         string
	EmailTo           []string
	EmailUseTLS       bool

	// Slack
	SlackWebhook string

	// Teams
	TeamsWebhook string

	// Webhook
	WebhookURL string
}

func runConfigure(cmd *cobra.Command, args []string) error {
	fmt.Println("\nArgazer Configuration Wizard")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	wizard := &ConfigWizard{}

	// Step 1: ArgoCD Configuration
	if err := configureArgoCD(wizard); err != nil {
		return err
	}

	// Step 2: Filtering
	if err := configureFiltering(wizard); err != nil {
		return err
	}

	// Step 3: General Settings
	if err := configureGeneral(wizard); err != nil {
		return err
	}

	// Step 4: Notification Channel
	if err := configureNotifications(wizard); err != nil {
		return err
	}

	// Step 5: Test Notification
	if wizard.NotificationChannel != "" {
		if err := testNotification(wizard); err != nil {
			fmt.Printf("\nWarning: Notification test failed: %v\n", err)
			fmt.Println("You can still save the configuration and fix it later.")

			var proceed bool
			prompt := &survey.Confirm{
				Message: "Do you want to save the configuration anyway?",
				Default: true,
			}
			if err := survey.AskOne(prompt, &proceed); err != nil {
				return err
			}
			if !proceed {
				return fmt.Errorf("configuration cancelled")
			}
		} else {
			fmt.Println("\nNotification test successful!")
		}
	}

	// Step 6: Save Configuration
	if err := saveConfiguration(wizard); err != nil {
		return err
	}

	fmt.Println("\nConfiguration saved successfully!")
	fmt.Println("\nYou can now run: argazer")
	fmt.Println()

	return nil
}

func configureArgoCD(wizard *ConfigWizard) error {
	fmt.Println("📡 ArgoCD Connection")
	fmt.Println(strings.Repeat("-", 60))

	questions := []*survey.Question{
		{
			Name: "argocdURL",
			Prompt: &survey.Input{
				Message: "ArgoCD Server URL (e.g., argocd.example.com):",
				Help:    "Just the hostname, without https:// prefix",
			},
			Validate: survey.Required,
		},
		{
			Name: "argocdUsername",
			Prompt: &survey.Input{
				Message: "ArgoCD Username:",
				Default: "admin",
			},
			Validate: survey.Required,
		},
		{
			Name: "argocdPassword",
			Prompt: &survey.Password{
				Message: "ArgoCD Password:",
			},
			Validate: survey.Required,
		},
		{
			Name: "argocdInsecure",
			Prompt: &survey.Confirm{
				Message: "Skip TLS verification (insecure)?",
				Default: false,
			},
		},
	}

	return survey.Ask(questions, wizard)
}

func configureFiltering(wizard *ConfigWizard) error {
	fmt.Println("\n🔍 Application Filtering")
	fmt.Println(strings.Repeat("-", 60))

	var projectsInput string
	prompt := &survey.Input{
		Message: "Projects to check (comma-separated, or * for all):",
		Default: "*",
		Help:    "Example: production,staging or *",
	}
	if err := survey.AskOne(prompt, &projectsInput); err != nil {
		return err
	}

	if projectsInput == "*" {
		wizard.Projects = []string{"*"}
	} else {
		wizard.Projects = strings.Split(projectsInput, ",")
		for i := range wizard.Projects {
			wizard.Projects[i] = strings.TrimSpace(wizard.Projects[i])
		}
	}

	var appNamesInput string
	prompt = &survey.Input{
		Message: "Application names to check (comma-separated, or * for all):",
		Default: "*",
		Help:    "Example: frontend,backend or *",
	}
	if err := survey.AskOne(prompt, &appNamesInput); err != nil {
		return err
	}

	if appNamesInput == "*" {
		wizard.AppNames = []string{"*"}
	} else {
		wizard.AppNames = strings.Split(appNamesInput, ",")
		for i := range wizard.AppNames {
			wizard.AppNames[i] = strings.TrimSpace(wizard.AppNames[i])
		}
	}

	return nil
}

func configureGeneral(wizard *ConfigWizard) error {
	fmt.Println("\nGeneral Settings")
	fmt.Println(strings.Repeat("-", 60))

	questions := []*survey.Question{
		{
			Name: "versionConstraint",
			Prompt: &survey.Select{
				Message: "Version constraint strategy:",
				Options: []string{"major", "minor", "patch"},
				Default: "major",
				Help:    "major: all updates, minor: same major, patch: same major.minor",
			},
		},
		{
			Name: "outputFormat",
			Prompt: &survey.Select{
				Message: "Default output format:",
				Options: []string{"table", "json", "markdown"},
				Default: "table",
			},
		},
		{
			Name: "logFormat",
			Prompt: &survey.Select{
				Message: "Log format:",
				Options: []string{"json", "text"},
				Default: "json",
				Help:    "json: for production, text: for development",
			},
		},
		{
			Name: "concurrency",
			Prompt: &survey.Input{
				Message: "Number of concurrent workers:",
				Default: "10",
			},
		},
	}

	return survey.Ask(questions, wizard)
}

func configureNotifications(wizard *ConfigWizard) error {
	fmt.Println("\n📬 Notification Channel")
	fmt.Println(strings.Repeat("-", 60))

	var channelOptions = []string{
		"None (console only)",
		"Telegram",
		"Email",
		"Slack",
		"Microsoft Teams",
		"Generic Webhook",
	}

	var selectedChannel string
	prompt := &survey.Select{
		Message: "Select notification channel:",
		Options: channelOptions,
		Default: "None (console only)",
	}
	if err := survey.AskOne(prompt, &selectedChannel); err != nil {
		return err
	}

	switch selectedChannel {
	case "Telegram":
		wizard.NotificationChannel = "telegram"
		return configureTelegram(wizard)
	case "Email":
		wizard.NotificationChannel = "email"
		return configureEmail(wizard)
	case "Slack":
		wizard.NotificationChannel = "slack"
		return configureSlack(wizard)
	case "Microsoft Teams":
		wizard.NotificationChannel = "teams"
		return configureTeams(wizard)
	case "Generic Webhook":
		wizard.NotificationChannel = "webhook"
		return configureWebhook(wizard)
	default:
		wizard.NotificationChannel = ""
	}

	return nil
}

func configureTelegram(wizard *ConfigWizard) error {
	questions := []*survey.Question{
		{
			Name: "telegramWebhook",
			Prompt: &survey.Input{
				Message: "Telegram Bot Webhook URL:",
				Help:    "Format: https://api.telegram.org/botTOKEN/sendMessage",
			},
			Validate: survey.Required,
		},
		{
			Name: "telegramChatID",
			Prompt: &survey.Input{
				Message: "Telegram Chat ID:",
				Help:    "Your chat ID or group chat ID",
			},
			Validate: survey.Required,
		},
	}

	return survey.Ask(questions, wizard)
}

func configureEmail(wizard *ConfigWizard) error {
	questions := []*survey.Question{
		{
			Name: "emailSMTPHost",
			Prompt: &survey.Input{
				Message: "SMTP Server Host:",
				Help:    "e.g., smtp.gmail.com",
			},
			Validate: survey.Required,
		},
		{
			Name: "emailSMTPPort",
			Prompt: &survey.Input{
				Message: "SMTP Server Port:",
				Default: "587",
			},
			Validate: survey.Required,
		},
		{
			Name: "emailSMTPUsername",
			Prompt: &survey.Input{
				Message: "SMTP Username (usually your email):",
			},
			Validate: survey.Required,
		},
		{
			Name: "emailSMTPPassword",
			Prompt: &survey.Password{
				Message: "SMTP Password:",
			},
			Validate: survey.Required,
		},
		{
			Name: "emailFrom",
			Prompt: &survey.Input{
				Message: "From Email Address:",
			},
			Validate: survey.Required,
		},
		{
			Name: "emailUseTLS",
			Prompt: &survey.Confirm{
				Message: "Use TLS?",
				Default: true,
			},
		},
	}

	if err := survey.Ask(questions, wizard); err != nil {
		return err
	}

	// Ask for recipient emails
	var emailToInput string
	prompt := &survey.Input{
		Message: "Recipient Email Addresses (comma-separated):",
		Help:    "Example: devops@example.com,admin@example.com",
	}
	if err := survey.AskOne(prompt, &emailToInput, survey.WithValidator(survey.Required)); err != nil {
		return err
	}

	wizard.EmailTo = strings.Split(emailToInput, ",")
	for i := range wizard.EmailTo {
		wizard.EmailTo[i] = strings.TrimSpace(wizard.EmailTo[i])
	}

	return nil
}

func configureSlack(wizard *ConfigWizard) error {
	question := &survey.Input{
		Message: "Slack Webhook URL:",
		Help:    "Format: https://hooks.slack.com/services/YOUR/WEBHOOK/URL",
	}

	return survey.AskOne(question, &wizard.SlackWebhook, survey.WithValidator(survey.Required))
}

func configureTeams(wizard *ConfigWizard) error {
	question := &survey.Input{
		Message: "Microsoft Teams Webhook URL:",
		Help:    "Format: https://outlook.office.com/webhook/YOUR/WEBHOOK/URL",
	}

	return survey.AskOne(question, &wizard.TeamsWebhook, survey.WithValidator(survey.Required))
}

func configureWebhook(wizard *ConfigWizard) error {
	question := &survey.Input{
		Message: "Webhook URL:",
		Help:    "Your custom webhook endpoint",
	}

	return survey.AskOne(question, &wizard.WebhookURL, survey.WithValidator(survey.Required))
}

func testNotification(wizard *ConfigWizard) error {
	fmt.Println("\nTesting notification channel...")

	logger := logrus.NewEntry(logrus.New())
	logger.Logger.SetOutput(os.Stderr)        // Send logs to stderr to keep output clean
	logger.Logger.SetLevel(logrus.ErrorLevel) // Only show errors

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var notifier notification.Notifier
	var err error

	switch wizard.NotificationChannel {
	case "telegram":
		notifier = notification.NewTelegramNotifier(
			wizard.TelegramWebhook,
			wizard.TelegramChatID,
			logger,
		)
	case "email":
		notifier = notification.NewEmailNotifier(
			wizard.EmailSMTPHost,
			wizard.EmailSMTPPort,
			wizard.EmailSMTPUsername,
			wizard.EmailSMTPPassword,
			wizard.EmailFrom,
			wizard.EmailTo,
			wizard.EmailUseTLS,
			logger,
		)
	case "slack":
		notifier = notification.NewSlackNotifier(wizard.SlackWebhook, logger)
	case "teams":
		notifier = notification.NewTeamsNotifier(wizard.TeamsWebhook, logger)
	case "webhook":
		notifier = notification.NewWebhookNotifier(wizard.WebhookURL, logger)
	default:
		return fmt.Errorf("unknown notification channel: %s", wizard.NotificationChannel)
	}

	testMessage := "Argazer configuration test\n\nThis is a test message from the configure command.\nIf you see this, your notification channel is working correctly!"

	err = notifier.Send(ctx, "Argazer Configuration Test", testMessage)
	if err != nil {
		return fmt.Errorf("failed to send test notification: %w", err)
	}

	return nil
}

func saveConfiguration(wizard *ConfigWizard) error {
	fmt.Println("\nSaving Configuration")
	fmt.Println(strings.Repeat("-", 60))

	// Build config structure using the Config struct for type safety
	cfg := &config.Config{
		ArgocdURL:           wizard.ArgocdURL,
		ArgocdUsername:      wizard.ArgocdUsername,
		ArgocdPassword:      wizard.ArgocdPassword,
		ArgocdInsecure:      wizard.ArgocdInsecure,
		Projects:            wizard.Projects,
		AppNames:            wizard.AppNames,
		VersionConstraint:   wizard.VersionConstraint,
		OutputFormat:        wizard.OutputFormat,
		LogFormat:           wizard.LogFormat,
		Concurrency:         wizard.Concurrency,
		NotificationChannel: wizard.NotificationChannel,
	}

	// Set notification-specific fields based on channel
	switch wizard.NotificationChannel {
	case "telegram":
		cfg.TelegramWebhook = wizard.TelegramWebhook
		cfg.TelegramChatID = wizard.TelegramChatID
	case "email":
		cfg.EmailSmtpHost = wizard.EmailSMTPHost
		cfg.EmailSmtpPort = wizard.EmailSMTPPort
		cfg.EmailSmtpUsername = wizard.EmailSMTPUsername
		cfg.EmailSmtpPassword = wizard.EmailSMTPPassword
		cfg.EmailFrom = wizard.EmailFrom
		cfg.EmailTo = wizard.EmailTo
		cfg.EmailUseTLS = wizard.EmailUseTLS
	case "slack":
		cfg.SlackWebhook = wizard.SlackWebhook
	case "teams":
		cfg.TeamsWebhook = wizard.TeamsWebhook
	case "webhook":
		cfg.WebhookURL = wizard.WebhookURL
	}

	// Determine config file path
	var configPath string
	prompt := &survey.Input{
		Message: "Config file path:",
		Default: "config.yaml",
		Help:    "Where to save the configuration file",
	}
	if err := survey.AskOne(prompt, &configPath); err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}

	// Marshal to YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	fmt.Printf("\nConfiguration saved to: %s\n", configPath)

	return nil
}
