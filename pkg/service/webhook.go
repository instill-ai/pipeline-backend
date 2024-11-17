package service

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
)

type DispatchPipelineWebhookEventParams struct {
	WebhookType string
	Headers     map[string][]string
	Message     *structpb.Struct
}

type DispatchPipelineWebhookEventResult struct {
	Response *structpb.Struct
}

func (s *service) DispatchPipelineWebhookEvent(ctx context.Context, params DispatchPipelineWebhookEventParams) (DispatchPipelineWebhookEventResult, error) {

	var j any
	err := base.ConvertFromStructpb(params.Message, &j)
	if err != nil {
		return DispatchPipelineWebhookEventResult{}, err
	}
	jsonMessage, err := data.NewJSONValue(j)
	if err != nil {
		return DispatchPipelineWebhookEventResult{}, err
	}
	parsedEvent, err := s.component.ParseEvent(ctx, params.WebhookType, &base.RawEvent{
		Header:  params.Headers,
		Message: jsonMessage,
	})
	if err != nil {
		return DispatchPipelineWebhookEventResult{}, err
	}

	if parsedEvent.SkipTrigger {
		resp, err := parsedEvent.Response.ToStructValue()
		if err != nil {
			return DispatchPipelineWebhookEventResult{}, err
		}
		return DispatchPipelineWebhookEventResult{
			Response: resp.GetStructValue(),
		}, nil
	}

	for _, identifier := range parsedEvent.Identifiers {
		runOns, err := s.repository.ListPipelineRunOns(ctx, repository.ListPipelineRunOnsParams{
			ComponentType: params.WebhookType,
			Identifier:    identifier,
		})
		if err != nil {
			return DispatchPipelineWebhookEventResult{}, err
		}

		for _, runOn := range runOns.PipelineRunOns {

			pipelineTriggerID, _ := uuid.NewV4()

			result, err := s.initEventWorkflow(ctx, initEventWorkflowParams{
				PipelineUID:        runOn.PipelineUID,
				ReleaseUID:         runOn.ReleaseUID,
				EventID:            runOn.EventID,
				ParsedMessage:      parsedEvent.ParsedMessage,
				PipelineTriggerUID: pipelineTriggerID,
			})
			if err != nil {
				return DispatchPipelineWebhookEventResult{}, err
			}
			pipelineRun := s.logPipelineRunStart(ctx, logPipelineRunStartParams{
				pipelineTriggerID: pipelineTriggerID.String(),
				pipelineUID:       runOn.PipelineUID,
				pipelineReleaseID: defaultPipelineReleaseID,
				requesterUID:      result.ns.NsUID,
				userUID:           result.ns.NsUID,
			})
			defer func() {
				if err != nil {
					s.logPipelineRunError(ctx, pipelineTriggerID.String(), err, pipelineRun.StartedTime)
				}
			}()

			if runOn.ReleaseUID == uuid.Nil {
				_, err = s.triggerAsyncPipeline(ctx, triggerParams{
					ns:                result.ns,
					pipelineID:        result.pipeline.ID,
					pipelineUID:       result.pipeline.UID,
					userUID:           result.ns.NsUID,
					requesterUID:      result.ns.NsUID,
					pipelineTriggerID: pipelineTriggerID.String(),
				})
				if err != nil {
					return DispatchPipelineWebhookEventResult{}, err
				}
			}
		}
	}

	if parsedEvent.Response != nil {
		respStruct, err := parsedEvent.Response.ToStructValue()
		if err != nil {
			return DispatchPipelineWebhookEventResult{}, err
		}
		return DispatchPipelineWebhookEventResult{
			Response: respStruct.GetStructValue(),
		}, nil
	}
	return DispatchPipelineWebhookEventResult{}, nil
}

type initEventWorkflowParams struct {
	PipelineUID        uuid.UUID
	ReleaseUID         uuid.UUID
	EventID            string
	ParsedMessage      format.Value
	PipelineTriggerUID uuid.UUID
}

type initEventWorkflowResult struct {
	ns       resource.Namespace
	pipeline *datamodel.Pipeline
	release  *datamodel.PipelineRelease
}

func (s *service) initEventWorkflow(ctx context.Context, params initEventWorkflowParams) (initEventWorkflowResult, error) {

	dbPipeline, err := s.repository.GetPipelineByUID(ctx, params.PipelineUID, false, false)
	if err != nil {
		return initEventWorkflowResult{}, err
	}
	recipe := dbPipeline.Recipe
	var dbRelease *datamodel.PipelineRelease

	if params.ReleaseUID != uuid.Nil {
		dbRelease, err = s.repository.GetPipelineReleaseByUIDAdmin(ctx, params.ReleaseUID, false)
		if err != nil {
			return initEventWorkflowResult{}, err
		}
		recipe = dbRelease.Recipe
	}

	wfm, err := s.memory.NewWorkflowMemory(ctx, params.PipelineTriggerUID.String(), nil, 1)
	if err != nil {
		return initEventWorkflowResult{}, err
	}

	variable := data.Map{}

	for key, v := range recipe.Variable {
		for _, l := range v.Listen {
			l := l[2 : len(l)-1]
			s := strings.Split(l, ".")
			if s[0] != "on" {
				return initEventWorkflowResult{}, fmt.Errorf("cannot listen to data outside of `on`")
			}

			p, err := path.NewPath(strings.Join(s[2:], "."))
			if err != nil {
				return initEventWorkflowResult{}, err
			}

			if s[1] == params.EventID {
				messageValue, err := params.ParsedMessage.Get(p)
				if err != nil {
					return initEventWorkflowResult{}, err
				}
				variable[key] = messageValue
			}
		}
	}
	err = wfm.Set(ctx, 0, constant.SegVariable, variable)
	if err != nil {
		return initEventWorkflowResult{}, err
	}

	ns, err := s.GetNamespaceByUID(ctx, dbPipeline.OwnerUID())
	if err != nil {
		return initEventWorkflowResult{}, err
	}
	return initEventWorkflowResult{
		ns:       ns,
		pipeline: dbPipeline,
		release:  dbRelease,
	}, nil
}
