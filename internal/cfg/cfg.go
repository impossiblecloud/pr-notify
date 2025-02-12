package cfg

import (
	"os"

	"github.com/impossiblecloud/pr-notify/internal/metrics"
	"gopkg.in/yaml.v3"
)

const (
	defaultGithubCustomFieldPattern = "github.com"
	defaultSlackUsersPullInterval   = 3600
)

// AppConfig is the main app runtime config
type AppConfig struct {
	Metrics         metrics.AppMetrics
	PrNotifications []PrNotification
	SlackConfig     SlackConfig

	SlackToGithubUserMap map[string]string // Map of Slack user IDs to GitHub logins
}

// UserMapping is a struct for mapping GitHub users to other notification channels
type UserMapping struct {
	GithubLogin string `yaml:"github_login"`
	SlackUID    string `yaml:"slack_uid"`
}

// appConfigFile is the main app config file
type appConfigFile struct {
	PrNotifications []PrNotification `yaml:"github_pr_notifications"`
	PrUserMappings  []UserMapping    `yaml:"github_user_mappings"`
	SlackConfig     SlackConfig      `yaml:"slack_config"`
}

// SlackConfig is the configuration for Slack integration
type SlackConfig struct {
	GithubUsersChannels           []GHUsersChannel `yaml:"github_users_channels"`
	GithubCustomFieldPattern      string           `yaml:"github_custom_field_pattern"`
	SlackUsersPullIntervalSeconds int              `yaml:"slack_users_pull_interval_seconds"`
}

// GHUsersChannel struct describes a GitHub users channel mapping
type GHUsersChannel struct {
	ID                        string                    `yaml:"id"`
	PeriodicNotifications     PeriodicNotifications     `yaml:"periodic_notifications"`
	MessageReplyNotifications MessageReplyNotifications `yaml:"message_reply_notifications"`
}

// PeriodicNotifications struct describes the periodic notification settings for users without GH info
type PeriodicNotifications struct {
	NotifyUsers bool   `yaml:"notify_users"`
	Schedule    string `yaml:"schedule"`
	Message     string `yaml:"message"`
}

// MessageReplyNotifications struct describes the message reply notification settings
type MessageReplyNotifications struct {
	NotifyUsers   bool   `yaml:"notify_users"`
	MessageRegex  string `yaml:"message_regex"`
	Reply         string `yaml:"reply"`
	ReplyInThread bool   `yaml:"reply_in_thread"`
}

// SlackNotification struct describes slack notification config
type SlackNotification struct {
	ChannelID      string `yaml:"channel_id"`
	Header         string `yaml:"message_header"`
	NotifyAssignee bool   `yaml:"notify_assignee"`
}

// Notification struct describes desired notification routes
type Notification struct {
	Slack SlackNotification `yaml:"slack"`
}

// PrConditions struct describes additional conditions for PRs
type PrConditions struct {
	OlderThanSeconds int `yaml:"older_than_seconds"`
}

// PrNotification is a struct for a single GH repo PRs notifications
type PrNotification struct {
	Owner                  string       `yaml:"gh_owner"`
	Labels                 []string     `yaml:"gh_pr_labels"`
	Repo                   string       `yaml:"gh_repo"`
	Schedule               string       `yaml:"schedule"`
	IncludeDrafts          bool         `yaml:"gh_pr_include_drafts"`
	IgnoreApproved         bool         `yaml:"gh_pr_ignore_approved"`
	IgnoreChangesRequested bool         `yaml:"gh_pr_ignore_changes_requested"`
	Conditions             PrConditions `yaml:"gh_pr_conditions"`
	Notifications          Notification `yaml:"notify"`
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
	config.SlackConfig = configFile.SlackConfig

	if config.SlackConfig.GithubCustomFieldPattern == "" {
		config.SlackConfig.GithubCustomFieldPattern = defaultGithubCustomFieldPattern
	}
	if config.SlackConfig.SlackUsersPullIntervalSeconds <= 0 {
		config.SlackConfig.SlackUsersPullIntervalSeconds = defaultSlackUsersPullInterval
	}
	return nil
}

// GetSlackUID returns Slack user ID for a given GitHub login
func (config *AppConfig) GetSlackUID(githubLogin string) (string, bool) {
	if slackUID, exists := config.SlackToGithubUserMap[githubLogin]; exists {
		return slackUID, true
	}
	return "", false
}
