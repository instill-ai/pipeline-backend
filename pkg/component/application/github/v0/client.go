package github

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/x/errmsg"
)

type RepoInfoInterface interface {
	getOwner() (string, error)
	getRepository() (string, error)
}

type RepoInfo struct {
	Owner      string `json:"owner"`
	Repository string `json:"repository"`
}

func (info RepoInfo) getOwner() (string, error) {
	if info.Owner == "" {
		return "", errmsg.AddMessage(
			fmt.Errorf("owner not provided"),
			"owner not provided",
		)
	}
	return info.Owner, nil
}
func (info RepoInfo) getRepository() (string, error) {
	if info.Repository == "" {
		return "", errmsg.AddMessage(
			fmt.Errorf("repository not provided"),
			"repository not provided",
		)
	}
	return info.Repository, nil
}

type Client struct {
	*github.Client
	Repositories RepositoriesService
	PullRequests PullRequestService
	Issues       IssuesService
}

func newClient(ctx context.Context, setup *structpb.Struct) Client {
	token := getToken(setup)

	var oauth2Client *http.Client
	if token != "" {
		tokenSource := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		oauth2Client = oauth2.NewClient(ctx, tokenSource)
	}
	client := github.NewClient(oauth2Client)
	githubClient := Client{
		Client:       client,
		Repositories: client.Repositories,
		PullRequests: client.PullRequests,
		Issues:       client.Issues,
	}
	return githubClient
}

func parseTargetRepo(info RepoInfoInterface) (string, string, error) {
	owner, ownerErr := info.getOwner()
	repository, RepoErr := info.getRepository()
	if ownerErr != nil && RepoErr != nil {
		return "", "", errmsg.AddMessage(
			fmt.Errorf("owner and repository not provided"),
			"owner and repository not provided",
		)
	}
	if ownerErr != nil {
		return "", "", ownerErr
	}
	if RepoErr != nil {
		return "", "", RepoErr
	}
	return owner, repository, nil
}

func getToken(setup *structpb.Struct) string {
	return setup.GetFields()["token"].GetStringValue()
}
