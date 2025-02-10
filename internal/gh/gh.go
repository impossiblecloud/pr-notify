package gh

import (
	"context"
	"os"
	"strconv"

	"github.com/golang/glog"
	"github.com/google/go-github/v69/github"
	"github.com/impossiblecloud/pr-notify/internal/cfg"
	"github.com/jferrl/go-githubauth"
	"golang.org/x/oauth2"
)

type Github struct {
	Client *github.Client
}

func labelsMatched(prLabels []*github.Label, filterLabels []string) bool {
	matched := 0

	for _, prLabel := range prLabels {
		for _, filterLabel := range filterLabels {
			if *prLabel.Name == filterLabel {
				matched++
			}
		}
	}

	return matched == len(filterLabels)
}

// Init initializes GH client
func (g *Github) Init() error {
	appID, _ := strconv.ParseInt(os.Getenv("GITHUB_APP_ID"), 10, 64)
	installationID, _ := strconv.ParseInt(os.Getenv("GITHUB_INSTALLATION_ID"), 10, 64)
	privateKey := []byte(os.Getenv("GITHUB_APP_PRIVATE_KEY"))

	appTokenSource, err := githubauth.NewApplicationTokenSource(appID, privateKey)
	if err != nil {
		return err
	}

	installationTokenSource := githubauth.NewInstallationTokenSource(installationID, appTokenSource)

	// oauth2.NewClient create a new http.Client that adds an Authorization header with the token.
	// Transport src use oauth2.ReuseTokenSource to reuse the token.
	// The token will be reused until it expires.
	// The token will be refreshed if it's expired.
	httpClient := oauth2.NewClient(context.Background(), installationTokenSource)

	g.Client = github.NewClient(httpClient)
	return nil
}

// GetPullRequests returns pull requests for a given PR notification object
func (g *Github) GetPullRequests(prn cfg.PrNotification) ([]*github.PullRequest, error) {
	var result []*github.PullRequest

	opts := github.PullRequestListOptions{
		State: "open",
	}

	glog.V(8).Infof("Getting pull requests from %s/%s", prn.Owner, prn.Repo)
	prs, _, err := g.Client.PullRequests.List(context.Background(), prn.Owner, prn.Repo, &opts)
	if err != nil {
		return nil, err
	}

	// If no labels filter and draft PRs are included, return all PRs
	if prn.Labels == nil && prn.IncludeDrafts {
		return prs, nil
	}

	// Otherwise filter based on config
	for _, pr := range prs {
		if !prn.IncludeDrafts && *pr.Draft {
			continue
		}
		if labelsMatched(pr.Labels, prn.Labels) {
			result = append(result, pr)
		}
	}

	return result, nil
}
