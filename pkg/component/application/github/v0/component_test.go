package github

import (
	"context"
	"fmt"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
)

const (
	token = "testkey"
)

var MockGithubClient = &Client{
	Repositories:  &MockRepositoriesService{},
	PullRequests:  &MockPullRequestsService{},
	Issues:        &MockIssuesService{},
	Users:         &MockUsersService{},
	Organizations: &MockOrganizationsService{},
}

type TaskCase[inType any, outType any] struct {
	_type      string
	name       string
	input      inType
	wantOutput outType
	wantErr    string
}

func taskTesting[inType any, outType any](testCases []TaskCase[inType, outType], execution *execution, t *testing.T) {
	c := qt.New(t)
	for _, tc := range testCases {
		c.Run(tc._type+`-`+tc.name, func(c *qt.C) {
			ir, ow, eh, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				if ptr, ok := input.(*inType); ok {
					*ptr = tc.input
					return nil
				}
				return fmt.Errorf("unsupported input type: %T", input)
			})
			ow.WriteDataMock.Optional().Set(func(ctx context.Context, output any) error {
				c.Assert(output, qt.DeepEquals, tc.wantOutput)
				return nil
			})
			eh.ErrorMock.Optional().Set(func(ctx context.Context, err error) {
				c.Assert(err, qt.ErrorMatches, tc.wantErr)
			})
			err := execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
		})
	}
}

func TestComponent_WithOAuthConfig(t *testing.T) {
	c := qt.New(t)

	bc := base.Component{}

	test := func(name string, conf map[string]any, check qt.Checker) {
		c.Run(name, func(c *qt.C) {
			cmp := Init(bc)
			cmp.WithOAuthConfig(conf)
			c.Check(cmp.SupportsOAuth(), check)
		})
	}

	newConf := func(clientID, clientSecret string) map[string]any {
		conf := map[string]any{}
		if clientID != "" {
			conf["oauthclientid"] = clientID
		}

		if clientSecret != "" {
			conf["oauthclientsecret"] = clientSecret
		}

		return conf
	}

	test("ok - with OAuth details", newConf("foo", "bar"), qt.IsTrue)
	test("ok - without OAuth secret", newConf("foo", ""), qt.IsFalse)
	test("ok - without OAuth ID", newConf("", "bar"), qt.IsFalse)
	test("ok - without OAuth ID", newConf("", ""), qt.IsFalse)
}
