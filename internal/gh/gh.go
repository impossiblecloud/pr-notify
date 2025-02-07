package gh

import "github.com/google/go-github/v69/github"

type Github struct {
	Client *github.Client
}

func (g *Github) Init() error {
	g.Client = github.NewClient(nil).WithAuthToken("... your access token ...")
	return nil
}
