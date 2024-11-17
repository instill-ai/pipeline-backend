package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/recipe"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/x/errmsg"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
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

	// IdentifyEvent is used to identify the event type and return the identifiers
	identifierResult, err := s.component.IdentifyEvent(ctx, params.WebhookType, &base.RawEvent{
		Header:  params.Headers,
		Message: jsonMessage,
	})
	if err != nil {
		return DispatchPipelineWebhookEventResult{}, err
	}

	// If SkipTrigger is true, it means the event could be skipped and the response is returned
	if identifierResult.SkipTrigger {
		resp, err := identifierResult.Response.ToStructValue()
		if err != nil {
			return DispatchPipelineWebhookEventResult{}, err
		}
		return DispatchPipelineWebhookEventResult{
			Response: resp.GetStructValue(),
		}, nil
	}

	for _, identifier := range identifierResult.Identifiers {
		// ListPipelineRunOns is used to list the pipeline run ons for the given identifier
		runOns, err := s.repository.ListPipelineRunOns(ctx, repository.ListPipelineRunOnsParams{
			ComponentType: params.WebhookType,
			Identifier:    identifier,
		})
		if err != nil {
			return DispatchPipelineWebhookEventResult{}, err
		}

		for _, runOn := range runOns.PipelineRunOns {

			pipelineTriggerID, _ := uuid.NewV4()

			loadPipelineResult, err := s.loadPipeline(ctx, loadPipelineParams{
				PipelineUID: runOn.PipelineUID,
				ReleaseUID:  runOn.ReleaseUID,
			})
			if err != nil {
				return DispatchPipelineWebhookEventResult{}, err
			}

			var event *datamodel.Event
			for eventID, v := range loadPipelineResult.recipe.On {
				if eventID == runOn.EventID {
					event = v
					break
				}
			}

			marshaler := data.NewMarshaler()
			cfg, err := marshaler.Marshal(event.Config)
			if err != nil {
				return DispatchPipelineWebhookEventResult{}, err
			}
			setup, err := marshaler.Marshal(event.Setup)
			if err != nil {
				return DispatchPipelineWebhookEventResult{}, err
			}

			// If the setup is a reference string, it means the setup is a connection
			if connRef, ok := setup.(format.ReferenceString); ok {
				connID, err := recipe.ConnectionIDFromReference(connRef.String())
				if err != nil {
					return DispatchPipelineWebhookEventResult{}, err
				}

				conn, err := s.repository.GetNamespaceConnectionByID(ctx, loadPipelineResult.ns.NsUID, connID)
				if err != nil {
					if errors.Is(err, errdomain.ErrNotFound) {
						err = errmsg.AddMessage(err, fmt.Sprintf("Connection %s doesn't exist.", connID))
					}
					return DispatchPipelineWebhookEventResult{}, err
				}

				var s map[string]any
				if err := json.Unmarshal(conn.Setup, &s); err != nil {
					return DispatchPipelineWebhookEventResult{}, err
				}

				setupVal, err := data.NewValue(s)
				if err != nil {
					return DispatchPipelineWebhookEventResult{}, err
				}
				setup = setupVal
			}

			// ParseEvent is used to parse the event and return the parsed message
			parsedEvent, err := s.component.ParseEvent(ctx, params.WebhookType, &base.RawEvent{
				EventSettings: base.EventSettings{
					Setup:  setup,
					Config: cfg,
				},
				Header:  params.Headers,
				Message: jsonMessage,
			})
			if err != nil {
				return DispatchPipelineWebhookEventResult{}, err
			}

			err = s.initEventWorkflow(ctx, initEventWorkflowParams{
				recipe:             loadPipelineResult.recipe,
				pipelineUID:        runOn.PipelineUID,
				releaseUID:         runOn.ReleaseUID,
				eventID:            runOn.EventID,
				parsedMessage:      parsedEvent.ParsedMessage,
				pipelineTriggerUID: pipelineTriggerID,
			})
			if err != nil {
				return DispatchPipelineWebhookEventResult{}, err
			}
			pipelineRun := s.logPipelineRunStart(ctx, logPipelineRunStartParams{
				pipelineTriggerID: pipelineTriggerID.String(),
				pipelineUID:       runOn.PipelineUID,
				pipelineReleaseID: defaultPipelineReleaseID,
				requesterUID:      loadPipelineResult.ns.NsUID,
				userUID:           loadPipelineResult.ns.NsUID,
			})
			defer func() {
				if err != nil {
					s.logPipelineRunError(ctx, pipelineTriggerID.String(), err, pipelineRun.StartedTime)
				}
			}()

			if runOn.ReleaseUID == uuid.Nil {
				_, err = s.triggerAsyncPipeline(ctx, triggerParams{
					ns:                loadPipelineResult.ns,
					pipelineID:        loadPipelineResult.pipeline.ID,
					pipelineUID:       loadPipelineResult.pipeline.UID,
					userUID:           loadPipelineResult.ns.NsUID,
					requesterUID:      loadPipelineResult.ns.NsUID,
					pipelineTriggerID: pipelineTriggerID.String(),
				})
				if err != nil {
					return DispatchPipelineWebhookEventResult{}, err
				}
			}
		}
	}

	return DispatchPipelineWebhookEventResult{}, nil
}

type loadPipelineParams struct {
	PipelineUID uuid.UUID
	ReleaseUID  uuid.UUID
}

type loadPipelineResult struct {
	ns       resource.Namespace
	pipeline *datamodel.Pipeline
	release  *datamodel.PipelineRelease
	recipe   *datamodel.Recipe
}

func (s *service) loadPipeline(ctx context.Context, params loadPipelineParams) (loadPipelineResult, error) {
	dbPipeline, err := s.repository.GetPipelineByUID(ctx, params.PipelineUID, false, false)
	if err != nil {
		return loadPipelineResult{}, err
	}
	recipe := dbPipeline.Recipe
	var dbRelease *datamodel.PipelineRelease

	if params.ReleaseUID != uuid.Nil {
		dbRelease, err = s.repository.GetPipelineReleaseByUIDAdmin(ctx, params.ReleaseUID, false)
		if err != nil {
			return loadPipelineResult{}, err
		}
		recipe = dbRelease.Recipe
	}

	ns, err := s.GetNamespaceByUID(ctx, dbPipeline.OwnerUID())
	if err != nil {
		return loadPipelineResult{}, err
	}
	return loadPipelineResult{
		ns:       ns,
		pipeline: dbPipeline,
		release:  dbRelease,
		recipe:   recipe,
	}, nil
}

type initEventWorkflowParams struct {
	recipe             *datamodel.Recipe
	pipelineUID        uuid.UUID
	releaseUID         uuid.UUID
	eventID            string
	parsedMessage      format.Value
	pipelineTriggerUID uuid.UUID
}

func (s *service) initEventWorkflow(ctx context.Context, params initEventWorkflowParams) error {

	wfm, err := s.memory.NewWorkflowMemory(ctx, params.pipelineTriggerUID.String(), nil, 1)
	if err != nil {
		return err
	}

	variable := data.Map{}

	for key, v := range params.recipe.Variable {
		for _, l := range v.Listen {
			l := l[2 : len(l)-1]
			s := strings.Split(l, ".")
			if s[0] != "on" {
				return fmt.Errorf("cannot listen to data outside of `on`")
			}

			p, err := path.NewPath(strings.Join(s[2:], "."))
			if err != nil {
				return err
			}

			if s[1] == params.eventID {
				messageValue, err := params.parsedMessage.Get(p)
				if err != nil {
					return err
				}
				variable[key] = messageValue
			}
		}
	}

	err = wfm.Set(ctx, 0, constant.SegVariable, variable)
	if err != nil {
		return err
	}

	return nil
}
