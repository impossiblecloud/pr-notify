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
}
