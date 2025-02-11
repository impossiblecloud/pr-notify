package cfg

import (
	"os"

	"github.com/impossiblecloud/pr-notify/internal/metrics"
	"gopkg.in/yaml.v3"
)

// AppConfig is the main app runtime config
type AppConfig struct {
	Metrics         metrics.AppMetrics
	PrNotifications []PrNotification
}

// appConfigFile is the main app config file
type appConfigFile struct {
	PrNotifications []PrNotification `yaml:"github_pr_notifications"`
}

// SlackNotification struct describes slack notification config
type SlackNotification struct {
	ChannelID string `yaml:"channel_id"`
	Header    string `yaml:"message_header"`
}

// Notification struct describes desired notification routes
type Notification struct {
	Slack SlackNotification `yaml:"slack"`
}

// PrNotification is a struct for a single GH repo PRs notifications
type PrNotification struct {
	Owner         string       `yaml:"gh_owner"`
	Labels        []string     `yaml:"gh_pr_labels"`
	Repo          string       `yaml:"gh_repo"`
	Schedule      string       `yaml:"schedule"`
	IncludeDrafts bool         `yaml:"gh_pr_include_drafts"`
	Notifications Notification `yaml:"notify"`
}

// LoadConfig loads config file
func (config *AppConfig) LoadConfig(cf string) error {

	configFile := appConfigFile{}
	yamlFile, err := os.ReadFile(cf)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(yamlFile, &configFile)
	if err != nil {
		return err
	}

	config.PrNotifications = configFile.PrNotifications
	return nil
}
