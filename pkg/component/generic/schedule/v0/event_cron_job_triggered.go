package schedule

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/data"
)

func (c *component) handleEventCronJobTriggered(ctx context.Context, rawEvent *base.RawEvent) (parsedEvent *base.ParsedEvent, err error) {

	return &base.ParsedEvent{
		ParsedMessage: rawEvent.Message,
		Response:      data.Map{},
	}, nil
}
