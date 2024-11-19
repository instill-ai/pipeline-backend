package github

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

func (c *component) handleStarCreated(ctx context.Context, rawEvent *base.RawEvent) (parsedEvent *base.ParsedEvent, err error) {
	unmarshaler := data.NewUnmarshaler(c.BinaryFetcher)

	r := rawGithubStarCreated{}
	err = unmarshaler.Unmarshal(ctx, rawEvent.Message, &r)
	if err != nil {
		return nil, err
	}

	githubEvent := githubStarCreated{
		Action:     r.Action,
		StarredAt:  r.StarredAt,
		Repository: convertRawRepository(r.Repository),
		Sender:     convertRawUser(r.Sender),
	}
	marshaler := data.NewMarshaler()
	m, err := marshaler.Marshal(githubEvent)
	if err != nil {
		return nil, err
	}

	return &base.ParsedEvent{
		ParsedMessage: m,
		Response:      data.Map{},
	}, nil
}
