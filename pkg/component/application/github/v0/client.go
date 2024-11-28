package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/go-github/v62/github"
	"golang.org/x/oauth2"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/x/errmsg"
)

// Client is a struct that contains the github client and the repositories service
type Client struct {
	*github.Client
	Repositories  RepositoriesService
	PullRequests  PullRequestsService
	Issues        IssuesService
	Users         UsersService
	Organizations OrganizationsService
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
	return Client{
		Client:        client,
		Repositories:  client.Repositories,
		PullRequests:  client.PullRequests,
		Issues:        client.Issues,
		Users:         client.Users,
		Organizations: client.Organizations,
	}
}

func parseTargetRepo(info RepoInfoInterface) (string, string, error) {
	owner, ownerErr := info.getOwner()
	repository, repoErr := info.getRepository()
	if err := errors.Join(ownerErr, repoErr); err != nil {
		return "", "", err
	}

	return owner, repository, nil
}

func getToken(setup *structpb.Struct) string {
	return setup.GetFields()["token"].GetStringValue()
}

// addErrMsgToClientError extracts the GitHub response information from an
// error. If this information is present, an end-user message is added to the
// error.
func addErrMsgToClientError(err error) error {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) {
		if ghErr.Response != nil {
			msg := fmt.Sprintf("GitHub responded with %d %v.", ghErr.Response.StatusCode, ghErr.Message)
			switch ghErr.Response.StatusCode {
			case http.StatusNotFound:
				msg += " Check that the repository exists and you have permissions to access it."
			case http.StatusUnprocessableEntity:
				for _, e := range ghErr.Errors {
					if e.Message != "" {
						msg = fmt.Sprintf("%s %s.", msg, e.Message)
					}
				}
			}
			return errmsg.AddMessage(err, msg)
		}
	}
	return err
}
