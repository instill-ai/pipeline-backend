//go:generate compogen readme ./config ./README.mdx --extraContents bottom=.compogen/bottom.mdx
package github

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"

	_ "embed"

	"github.com/google/go-github/v62/github"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/x/errmsg"
)

const (
	taskCreateIssue         = "TASK_CREATE_ISSUE"
	taskCreateReviewComment = "TASK_CREATE_REVIEW_COMMENT"
	taskCreateWebhook       = "TASK_CREATE_WEBHOOK"
	taskGetCommit           = "TASK_GET_COMMIT"
	taskGetIssue            = "TASK_GET_ISSUE"
	taskGetPullRequest      = "TASK_GET_PULL_REQUEST"
	taskListIssues          = "TASK_LIST_ISSUES"
	taskListPullRequests    = "TASK_LIST_PULL_REQUESTS"
	taskListReviewComments  = "TASK_LIST_REVIEW_COMMENTS"
)

var (
	//go:embed config/definition.json
	definitionJSON []byte
	//go:embed config/setup.json
	setupJSON []byte
	//go:embed config/tasks.json
	tasksJSON []byte
	//go:embed config/events.json
	eventsJSON []byte

	once sync.Once
	comp *component
)

type component struct {
	base.Component
	base.OAuthConnector
}

type execution struct {
	base.ComponentExecution
	execute func(context.Context, *base.Job) error
	client  Client
}

// Init initializes a Component that interacts with GitHub.
func Init(bc base.Component) *component {
	once.Do(func() {
		comp = &component{Component: bc}
		err := comp.LoadDefinition(definitionJSON, setupJSON, tasksJSON, eventsJSON, nil)
		if err != nil {
			panic(err)
		}
	})
	return comp
}

// CreateExecution initializes a component executor that can be used in a
// pipeline run.
func (c *component) CreateExecution(x base.ComponentExecution) (base.IExecution, error) {

	ctx := context.Background()
	client := newClient(ctx, x.Setup)

	e := &execution{
		ComponentExecution: x,
		client:             client,
	}

	switch x.Task {
	case taskCreateIssue:
		e.execute = e.client.createIssue
	case taskCreateReviewComment:
		e.execute = e.client.createReviewComment
	case taskCreateWebhook:
		e.execute = e.client.createWebhook
	case taskGetCommit:
		e.execute = e.client.getCommit
	case taskGetIssue:
		e.execute = e.client.getIssue
	case taskGetPullRequest:
		e.execute = e.client.getPullRequest
	case taskListIssues:
		e.execute = e.client.listIssues
	case taskListPullRequests:
		e.execute = e.client.listPullRequests
	case taskListReviewComments:
		e.execute = e.client.listReviewComments
	default:
		return nil, errmsg.AddMessage(
			fmt.Errorf("not supported task: %s", x.Task),
			fmt.Sprintf("%s task is not supported.", x.Task),
		)
	}

	return e, nil
}

func (e *execution) Execute(ctx context.Context, jobs []*base.Job) error {
	var input any
	switch e.Task {
	case taskCreateIssue:
		input = &createIssueInput{}
	case taskCreateReviewComment:
		input = &createReviewCommentInput{}
	case taskCreateWebhook:
		input = &createWebHookInput{}
	case taskGetCommit:
		input = &getCommitInput{}
	case taskGetIssue:
		input = &getIssueInput{}
	case taskGetPullRequest:
		input = &getPullRequestInput{}
	case taskListIssues:
		input = &listIssuesInput{}
	case taskListPullRequests:
		input = &listPullRequestsInput{}
	case taskListReviewComments:
		input = &listReviewCommentsInput{}
	default:
		return fmt.Errorf("unsupported task: %s", e.Task)
	}

	for _, job := range jobs {
		if err := job.Input.ReadData(ctx, input); err != nil {
			return fmt.Errorf("reading input data: %w", err)
		}
		if _, err := e.FillInDefaultValues(input); err != nil {
			job.Error.Error(ctx, err)
			continue
		}
	}
	return base.ConcurrentExecutor(ctx, jobs, e.execute)
}

func (c *component) IdentifyEvent(ctx context.Context, rawEvent *base.RawEvent) (identifierResult *base.IdentifierResult, err error) {

	// TODO: validate signature
	if len(rawEvent.Header["x-github-event"]) > 0 && rawEvent.Header["x-github-event"][0] == "ping" {
		return &base.IdentifierResult{
			SkipTrigger: true,
			Response:    data.Map{},
		}, nil
	}
	if len(rawEvent.Header["x-github-hook-id"]) > 0 {
		hookID := rawEvent.Header["x-github-hook-id"][0]
		hookIDInt, err := strconv.Atoi(hookID)
		if err != nil {
			return nil, err
		}
		return &base.IdentifierResult{
			Identifiers: []base.Identifier{{"hook-id": hookIDInt}},
		}, nil
	}
	return nil, nil
}

func (c *component) ParseEvent(ctx context.Context, rawEvent *base.RawEvent) (parsedEvent *base.ParsedEvent, err error) {

	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
	rawGithubEvent := rawGithubEvent{}
	err = unmarshaler.Unmarshal(ctx, rawEvent.Message, &rawGithubEvent)
	if err != nil {
		return nil, err
	}

	event := rawEvent.Header["x-github-event"][0]

	switch event + "." + rawGithubEvent.Action {
	case "star.created":
		return c.handleStarCreated(ctx, rawEvent)
	default:
		return nil, fmt.Errorf("not supported event: %s.%s", event, rawGithubEvent.Action)
	}
}

func (c *component) RegisterEvent(ctx context.Context, settings *base.RegisterEventSettings) ([]base.Identifier, error) {
	// TODO: Handle errors from CreateHook and EditHook properly
	// TODO: Support additional GitHub webhook events beyond 'star'
	// TODO: Add validation for repository format and webhook configuration
	setup, err := settings.Setup.ToStructValue()
	if err != nil {
		return nil, err
	}
	client := newClient(ctx, setup.GetStructValue())

	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
	cfg := githubEventStarCreatedConfig{}
	err = unmarshaler.Unmarshal(ctx, settings.Config, &cfg)
	if err != nil {
		return nil, err
	}
	namespace, repo, ok := strings.Cut(cfg.Repository, "/")
	if !ok {
		return nil, fmt.Errorf("invalid repository format: %s", cfg.Repository)
	}

	// TODO: avoid directly reading from config
	host := config.Config.Server.InstillCoreHost

	hooks := []*github.Hook{}
	page := 1
	for {
		pageHooks, _, err := client.Repositories.ListHooks(ctx, namespace, repo, &github.ListOptions{Page: page, PerPage: 100})
		if err != nil {
			break
		}
		if len(pageHooks) == 0 {
			break
		}
		hooks = append(hooks, pageHooks...)
		page++
	}

	hookID := int64(0)
	existingHook := false
	for _, hook := range hooks {
		parsedURL, err := url.Parse(hook.GetConfig().GetURL())
		if err != nil {
			return nil, err
		}
		query, err := url.ParseQuery(parsedURL.RawQuery)
		if err != nil {
			return nil, err
		}
		if parsedURL.Path == "/v1beta/pipeline-webhooks/github" && query.Get("uid") == settings.RegistrationUID.String() {
			hookID = hook.GetID()
			existingHook = true
			break
		}
	}

	if !existingHook {
		insecureSSL := github.String("1")
		if strings.HasPrefix(host, "https://") {
			insecureSSL = github.String("0")
		}
		u := fmt.Sprintf("%s/v1beta/pipeline-webhooks/github?uid=%s", host, settings.RegistrationUID.String())

		hook, _, err := client.Repositories.CreateHook(ctx, namespace, repo, &github.Hook{
			Config: &github.HookConfig{
				URL:         github.String(u),
				ContentType: github.String("json"),
				InsecureSSL: insecureSSL,
				Secret:      github.String(settings.RegistrationUID.String()),
			},
			Events: []string{"star"},
			Active: github.Bool(true),
		})
		if err != nil {
			// TODO: Handle error
			return nil, nil
		}
		hookID = hook.GetID()
	} else {
		_, _, err := client.Repositories.EditHook(ctx, namespace, repo, hookID, &github.Hook{
			Active: github.Bool(true),
		})
		if err != nil {
			// TODO: Handle error
			return nil, nil
		}
	}

	return []base.Identifier{{"hook-id": hookID}}, nil
}

func (c *component) UnregisterEvent(ctx context.Context, settings *base.UnregisterEventSettings, identifier []base.Identifier) error {
	setup, err := settings.Setup.ToStructValue()
	if err != nil {
		return err
	}
	client := newClient(ctx, setup.GetStructValue())

	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)
	cfg := githubEventStarCreatedConfig{}
	err = unmarshaler.Unmarshal(ctx, settings.Config, &cfg)
	if err != nil {
		return err
	}
	namespace, repo, ok := strings.Cut(cfg.Repository, "/")
	if !ok {
		return fmt.Errorf("invalid repository format: %s", cfg.Repository)
	}

	for _, id := range identifier {
		if hookID, ok := id["hook-id"]; ok {
			// Note: Only repository administrators can delete webhooks, so we temporarily disable it instead
			_, _, _ = client.Repositories.EditHook(ctx, namespace, repo, int64(hookID.(float64)), &github.Hook{
				Active: github.Bool(false),
			})
		}
	}
	return nil
}

// SupportsOAuth checks whether the component is configured to support OAuth.
func (c *component) SupportsOAuth() bool {
	return c.OAuthConnector.SupportsOAuth()
}
