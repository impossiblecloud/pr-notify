package cfg

import (
	"testing"
)

func TestLoadConfig(t *testing.T) {
	config := AppConfig{}
	err := config.LoadConfig("../../fixtures/config.yaml")
	if err != nil {
		t.Errorf("Failed to load ./fixtures/config.yaml: %s", err.Error())
	}

	if len(config.PrNotifications) < 1 {
		t.Errorf("Length of github_pr_notifications is %d, but it should be not empty", len(config.PrNotifications))
	}

	if !config.PrNotifications[0].IncludeDrafts {
		t.Error("Expected gh_pr_include_drafts=true")
	}

	if config.PrNotifications[0].Notifications.Slack.ChannelID != "ABC123000" {
		t.Error("Did not find expected Slack channel ID in notification config")
	}

	if config.PrNotifications[0].Conditions.OlderThanSeconds != 3600 {
		t.Errorf("Expected gh_pr_conditions.older_than_seconds=3600")
	}
}

func TestSlackConfigLoad(t *testing.T) {
	config := AppConfig{}
	err := config.LoadConfig("../../fixtures/config.yaml")
	if err != nil {
		t.Errorf("Failed to load ./fixtures/config.yaml: %s", err.Error())
	}

	if config.SlackConfig.GithubCustomFieldPattern != "github.com" {
		t.Errorf("Expected github_custom_field_pattern to be 'github.com', but got %q", config.SlackConfig.GithubCustomFieldPattern)
	}

	if config.SlackConfig.SlackUsersPullIntervalSeconds != 600 {
		t.Errorf("Expected slack_users_pull_interval_seconds to be 600, but got %d", config.SlackConfig.SlackUsersPullIntervalSeconds)
	}

	if len(config.SlackConfig.GithubUsersChannels) != 1 {
		t.Errorf("Expected 1 GitHub users channel, but got %d", len(config.SlackConfig.GithubUsersChannels))
	}

	if config.SlackConfig.GithubUsersChannels[0].ID != "C01234567" {
		t.Errorf("Expected GitHub users channel ID to be 'C01234567', but got %q", config.SlackConfig.GithubUsersChannels[0].ID)
	}

	if len(config.SlackConfig.GithubUsersChannels[0].PeriodicNotifications.Message) == 0 {
		t.Errorf("Expected periodic_notifications.message to be set, but got empty")
	}

	if !config.SlackConfig.GithubUsersChannels[0].PeriodicNotifications.NotifyUsers {
		t.Errorf("Expected periodic_notifications.notify_users to be true, but got false")
	}
}

func TestGetSlackUID(t *testing.T) {
	config := AppConfig{}
	err := config.LoadConfig("../../fixtures/config.yaml")
	if err != nil {
		t.Errorf("Failed to load ./fixtures/config.yaml: %s", err.Error())
	}

	config.SlackToGithubUserMap = map[string]string{
		"bob":      "U12345678",
		"someuser": "U87654321",
	}

	slackUID, found := config.GetSlackUID("bob")
	if !found {
		t.Errorf("Expected to find Slack user ID for GitHub login 'bob', but did not")
	}
	if slackUID != "U12345678" {
		t.Errorf("Expected Slack user ID 'U12345678' for GitHub login 'bob', but got %q", slackUID)
	}

	_, found = config.GetSlackUID("nonexistent-gh-user")
	if found {
		t.Errorf("Did not expect to find Slack user ID for GitHub login 'nonexistent-gh-user', but found one")
	}
}

func TestPrConditions(t *testing.T) {
	config := AppConfig{}
	err := config.LoadConfig("../../fixtures/config.yaml")
	if err != nil {
		t.Errorf("Failed to load ./fixtures/config.yaml: %s", err.Error())
	}

	if len(config.PrNotifications) < 1 {
		t.Errorf("Length of github_pr_notifications is %d, but it should be not empty", len(config.PrNotifications))
	}

	conditions := config.PrNotifications[0].Conditions
	if conditions.OlderThanSeconds != 3600 {
		t.Errorf("Expected gh_pr_conditions.older_than_seconds=3600, but got %d", conditions.OlderThanSeconds)
	}

	expectedLabels := []string{"WIP", "do not merge"}
	if len(conditions.DoesNotHaveLabels) != len(expectedLabels) {
		t.Errorf("Expected gh_pr_conditions.does_not_have_labels length=%d, but got %d", len(expectedLabels), len(conditions.DoesNotHaveLabels))
	} else {
		for i, label := range expectedLabels {
			if conditions.DoesNotHaveLabels[i] != label {
				t.Errorf("Expected gh_pr_conditions.does_not_have_labels[%d]=%q, but got %q", i, label, conditions.DoesNotHaveLabels[i])
			}
		}
	}
}
