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

	// PipelineOwner represents the namespace that owns the pipeline. This is typically
	// the namespace where the pipeline was created and is stored.
	PipelineOwner resource.Namespace `json:"__PIPELINE_OWNER"`
	// PipelineUserUID is the unique identifier of the authenticated user who is
	// executing the pipeline. This is used for access control and audit logging.
	PipelineUserUID uuid.UUID `json:"__PIPELINE_USER_UID"`
	// PipelineRequesterID is the ID of the entity (user or organization)
	// that initiated the pipeline execution. This may differ from PipelineUserUID
	// when the pipeline is triggered by on behalf of an organization.
	PipelineRequesterID string `json:"__PIPELINE_REQUESTER_ID"`
	// PipelineRequesterUID is the unique identifier of the entity (user or organization)
	// that initiated the pipeline execution. This may differ from PipelineUserUID
	// when the pipeline is triggered by on behalf of an organization.
	PipelineRequesterUID uuid.UUID `json:"__PIPELINE_REQUESTER_UID"`
	// TODO: we should use resource.Namespace for PipelineOwner and PipelineRequester

	HeaderAuthorization string `json:"__PIPELINE_HEADER_AUTHORIZATION"`
	ModelBackend        string `json:"__MODEL_BACKEND"`
	MgmtBackend         string `json:"__MGMT_BACKEND"`
	ArtifactBackend     string `json:"__ARTIFACT_BACKEND"`
	AppBackend          string `json:"__APP_BACKEND"`
}

// TODO: GenerateSystemVariables will be refactored for better code structure.
// Planned refactor: 2024-11

// GenerateSystemVariables fills SystemVariable fields with information from
// the context and instance configuration.
func GenerateSystemVariables(ctx context.Context, sysVar SystemVariables) (map[string]any, error) {
	if sysVar.PipelineUserUID.IsNil() {
		sysVar.PipelineUserUID = uuid.FromStringOrNil(resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
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
