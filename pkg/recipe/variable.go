package recipe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
)

// SystemVariables contain information about a pipeline trigger.
type SystemVariables struct {
	PipelineTriggerID  string                 `json:"__PIPELINE_TRIGGER_ID"`
	PipelineID         string                 `json:"__PIPELINE_ID"`
	PipelineUID        uuid.UUID              `json:"__PIPELINE_UID"`
	PipelineReleaseID  string                 `json:"__PIPELINE_RELEASE_ID"`
	PipelineReleaseUID uuid.UUID              `json:"__PIPELINE_RELEASE_UID"`
	PipelineRecipe     *datamodel.Recipe      `json:"__PIPELINE_RECIPE"`
	PipelineOwnerType  resource.NamespaceType `json:"__PIPELINE_OWNER_TYPE"`
	PipelineOwnerUID   uuid.UUID              `json:"__PIPELINE_OWNER_UID"`
	PipelineRunSource  datamodel.RunSource    `json:"__PIPELINE_RUN_SOURCE"`

	// PipelineUserUID is the authenticated user executing a pipeline.
	PipelineUserUID uuid.UUID `json:"__PIPELINE_USER_UID"`
	// PipelineRequesterUID is the entity requesting the pipeline execution.
	PipelineRequesterUID uuid.UUID `json:"__PIPELINE_REQUESTER_UID"`

	HeaderAuthorization string `json:"__PIPELINE_HEADER_AUTHORIZATION"`
	ModelBackend        string `json:"__MODEL_BACKEND"`
	MgmtBackend         string `json:"__MGMT_BACKEND"`
	ArtifactBackend     string `json:"__ARTIFACT_BACKEND"`
}

// GenerateSystemVariables fills SystemVariable fields with information from
// the context and instance configuration.
func GenerateSystemVariables(ctx context.Context, sysVar SystemVariables) (map[string]any, error) {
	if sysVar.PipelineUserUID.IsNil() {
		sysVar.PipelineUserUID = uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	}
	if sysVar.PipelineRequesterUID.IsNil() {
		sysVar.PipelineRequesterUID = uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderRequesterUIDKey))
		if sysVar.PipelineRequesterUID.IsNil() {
			sysVar.PipelineRequesterUID = sysVar.PipelineUserUID
		}
	}
	if sysVar.HeaderAuthorization == "" {
		sysVar.HeaderAuthorization = resource.GetRequestSingleHeader(ctx, "Authorization")
	}
	if sysVar.ModelBackend == "" {
		sysVar.ModelBackend = fmt.Sprintf("%s:%d", config.Config.ModelBackend.Host, config.Config.ModelBackend.PublicPort)
	}
	if sysVar.MgmtBackend == "" {
		sysVar.MgmtBackend = fmt.Sprintf("%s:%d", config.Config.MgmtBackend.Host, config.Config.MgmtBackend.PublicPort)
	}
	if sysVar.ArtifactBackend == "" {
		sysVar.ArtifactBackend = fmt.Sprintf("%s:%d", config.Config.ArtifactBackend.Host, config.Config.ArtifactBackend.PublicPort)
	}

	b, err := json.Marshal(sysVar)
	if err != nil {
		return nil, err
	}
	vars := map[string]any{}
	err = json.Unmarshal(b, &vars)
	if err != nil {
		return nil, err
	}

	return vars, nil
}
