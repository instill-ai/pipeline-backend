package recipe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
)

// SystemVariables contain information about a pipeline trigger.
type SystemVariables struct {
	PipelineTriggerID  string    `json:"__PIPELINE_TRIGGER_ID"`
	PipelineID         string    `json:"__PIPELINE_ID"`
	PipelineUID        uuid.UUID `json:"__PIPELINE_UID"`
	PipelineReleaseID  string    `json:"__PIPELINE_RELEASE_ID"`
	PipelineReleaseUID uuid.UUID `json:"__PIPELINE_RELEASE_UID"`
	ExpiryRuleTag      string    `json:"__EXPIRY_RULE_TAG"`

	// PipelineNamespace is the namespace of the requester of the pipeline.
	PipelineNamespace resource.Namespace `json:"__PIPELINE_NAMESPACE"`

	// PipelineUserUID is the authenticated user executing a pipeline.
	PipelineUserUID uuid.UUID `json:"__PIPELINE_USER_UID"`
	// PipelineRequesterUID is the entity requesting the pipeline execution.
	PipelineRequesterUID uuid.UUID `json:"__PIPELINE_REQUESTER_UID"`

	HeaderAuthorization string `json:"__PIPELINE_HEADER_AUTHORIZATION"`
	ModelBackend        string `json:"__MODEL_BACKEND"`
	MgmtBackend         string `json:"__MGMT_BACKEND"`
	ArtifactBackend     string `json:"__ARTIFACT_BACKEND"`
	AppBackend          string `json:"__APP_BACKEND"`
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
	if sysVar.AppBackend == "" {
		sysVar.AppBackend = fmt.Sprintf("%s:%d", config.Config.AppBackend.Host, config.Config.AppBackend.PublicPort)
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
