package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config holds the application configuration
type Config struct {
	// ArgoCD connection settings
	ArgocdURL      string `mapstructure:"argocd_url"`
	ArgocdUsername string `mapstructure:"argocd_username"`
	ArgocdPassword string `mapstructure:"argocd_password"`
	ArgocdInsecure bool   `mapstructure:"argocd_insecure"` // Skip TLS verification

	// Search scope
	Projects []string          `mapstructure:"projects"`  // List of projects to check, or ["*"] for all
	AppNames []string          `mapstructure:"app_names"` // List of app names to check, or ["*"] for all
	Labels   map[string]string `mapstructure:"labels"`    // Label filters

	// Notification settings
	NotificationChannel string `mapstructure:"notification_channel"` // "telegram", "email", or empty

	// Telegram settings
	TelegramWebhook string `mapstructure:"telegram_webhook"`
	TelegramChatID  string `mapstructure:"telegram_chat_id"`

	// Email settings
	EmailSmtpHost     string   `mapstructure:"email_smtp_host"`
	EmailSmtpPort     int      `mapstructure:"email_smtp_port"`
	EmailSmtpUsername string   `mapstructure:"email_smtp_username"`
	EmailSmtpPassword string   `mapstructure:"email_smtp_password"`
	EmailFrom         string   `mapstructure:"email_from"`
	EmailTo           []string `mapstructure:"email_to"`
	EmailUseTLS       bool     `mapstructure:"email_use_tls"`

	// General settings
	Verbose     bool   `mapstructure:"verbose"`
	SourceName  string `mapstructure:"source_name"` // Name of the source to check in multi-source applications
	Concurrency int    `mapstructure:"concurrency"` // Number of concurrent workers for checking applications

	// Repository authentication
	RepositoryAuth []RepositoryAuth `mapstructure:"repository_auth"`
}

// RepositoryAuth holds authentication for a specific repository or registry
type RepositoryAuth struct {
	URL      string `mapstructure:"url"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// Load loads configuration from various sources
func Load() (*Config, error) {
	// Set default values
	viper.SetDefault("verbose", false)
	viper.SetDefault("source_name", "chart-repo")
	viper.SetDefault("concurrency", 10) // Default to 10 concurrent workers
	viper.SetDefault("projects", []string{"*"})
	viper.SetDefault("app_names", []string{"*"})
	viper.SetDefault("argocd_insecure", false)
	viper.SetDefault("email_smtp_port", 587)
	viper.SetDefault("email_use_tls", true)

	// Set config file name and paths
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/argazer")
	viper.AddConfigPath("$HOME/.argazer")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, continue with defaults and env vars
	}

	// Set up environment variable prefix and replacer
	// AutomaticEnv() automatically binds all config fields to environment variables
	// with the AG_ prefix (e.g., AG_ARGOCD_URL maps to argocd_url)
	viper.SetEnvPrefix("AG")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Handle labels from environment variable BEFORE unmarshal
	// Format: AG_LABELS=key1=value1,key2=value2
	// Check if labels is set as a string (from env var) and convert it to a map
	if viper.IsSet("labels") {
		// Try to get it as a string first (from env var)
		if labelsStr, ok := viper.Get("labels").(string); ok && labelsStr != "" {
			// Parse the string and set it back as a map
			labelsMap := parseLabelsFromString(labelsStr)
			viper.Set("labels", labelsMap)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if cfg.ArgocdURL == "" {
		return nil, fmt.Errorf("argocd_url is required")
	}
	if cfg.ArgocdUsername == "" {
		return nil, fmt.Errorf("argocd_username is required")
	}
	if cfg.ArgocdPassword == "" {
		return nil, fmt.Errorf("argocd_password is required")
	}

	// Validate notification channel settings
	if cfg.NotificationChannel == "telegram" {
		if cfg.TelegramWebhook == "" {
			return nil, fmt.Errorf("telegram_webhook is required when notification_channel is 'telegram'")
		}
		if cfg.TelegramChatID == "" {
			return nil, fmt.Errorf("telegram_chat_id is required when notification_channel is 'telegram'")
		}
	} else if cfg.NotificationChannel == "email" {
		if cfg.EmailSmtpHost == "" {
			return nil, fmt.Errorf("email_smtp_host is required when notification_channel is 'email'")
		}
		if cfg.EmailFrom == "" {
			return nil, fmt.Errorf("email_from is required when notification_channel is 'email'")
		}
		if len(cfg.EmailTo) == 0 {
			return nil, fmt.Errorf("email_to is required when notification_channel is 'email'")
		}
	}

	return &cfg, nil
}

// parseLabelsFromString parses a comma-separated key=value string into a map
// Example: "key1=value1,key2=value2" -> map[string]string{"key1": "value1", "key2": "value2"}
func parseLabelsFromString(labelsStr string) map[string]string {
	labels := make(map[string]string)
	if labelsStr == "" {
		return labels
	}

	pairs := strings.Split(labelsStr, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" {
				labels[key] = value
			}
		}
	}

	return labels
}
