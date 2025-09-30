package gh

import (
	"testing"
	"time"

	"github.com/google/go-github/v69/github"
	"github.com/impossiblecloud/pr-notify/internal/cfg"
)

func TestMatchesConditions_OlderThanSeconds(t *testing.T) {
	g := &Github{}

	// PR created 2 hours ago
	pr := &github.PullRequest{
		CreatedAt: &github.Timestamp{Time: time.Now().Add(-2 * time.Hour)},
	}

	// Condition: PR older than 1 hour
	prn := cfg.PrNotification{
		Conditions: cfg.PrConditions{
			OlderThanSeconds: 3600, // 1 hour
		},
	}

	if !g.MatchesConditions(pr, prn) {
		t.Error("Expected MatchesConditions to return true for PR older than OlderThanSeconds")
	}
}

func TestMatchesConditions_NotOlderThanSeconds(t *testing.T) {
	g := &Github{}

	// PR created 30 minutes ago
	pr := &github.PullRequest{
		CreatedAt: &github.Timestamp{Time: time.Now().Add(-30 * time.Minute)},
	}

	// Condition: PR older than 1 hour
	prn := cfg.PrNotification{
		Conditions: cfg.PrConditions{
			OlderThanSeconds: 3600, // 1 hour
		},
	}

	if g.MatchesConditions(pr, prn) {
		t.Error("Expected MatchesConditions to return false for PR not older than OlderThanSeconds")
	}
}

func TestMatchesConditions_ZeroOlderThanSeconds(t *testing.T) {
	g := &Github{}

	pr := &github.PullRequest{
		CreatedAt: &github.Timestamp{Time: time.Now()},
	}

	// Condition: OlderThanSeconds is zero
	prn := cfg.PrNotification{
		Conditions: cfg.PrConditions{
			OlderThanSeconds: 0,
		},
	}

	if !g.MatchesConditions(pr, prn) {
		t.Error("Expected MatchesConditions to return true when OlderThanSeconds is zero")
	}
}
