package recipe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/x/constant"
	"github.com/instill-ai/x/minio"

	resourcex "github.com/instill-ai/x/resource"
)

// SystemVariables contain information about a pipeline trigger.
// TODO jvallesm: we should remove the __ prefix from the fields as it's an
// outdated convention.
type SystemVariables struct {
	PipelineTriggerID  string    `json:"__PIPELINE_TRIGGER_ID"`
	PipelineID         string    `json:"__PIPELINE_ID"`
	PipelineUID        uuid.UUID `json:"__PIPELINE_UID"`
	PipelineReleaseID  string    `json:"__PIPELINE_RELEASE_ID"`
	PipelineReleaseUID uuid.UUID `json:"__PIPELINE_RELEASE_UID"`

	// PipelineOwner represents the namespace that owns the pipeline. This is typically
	// the namespace where the pipeline was created and is stored.
	PipelineOwner resource.Namespace `json:"__PIPELINE_OWNER"`
	// PipelineUserUID is the unique identifier of the authenticated user who is
	// executing the pipeline. This is used for access control and audit logging.
	PipelineUserUID uuid.UUID `json:"__PIPELINE_USER_UID"`
	// PipelineRequesterID is the ID of the entity (user or organization)
	// that initiated the pipeline execution. This may differ from PipelineUserUID
	// when the pipeline is triggered by on behalf of an organization.
	// TODO: we should use resource.Namespace for PipelineRequester
	PipelineRequesterID string `json:"__PIPELINE_REQUESTER_ID"`
	// PipelineRequesterUID is the unique identifier of the entity (user or organization)
	// that initiated the pipeline execution. This may differ from PipelineUserUID
	// when the pipeline is triggered by on behalf of an organization.
	PipelineRequesterUID uuid.UUID `json:"__PIPELINE_REQUESTER_UID"`

	// ExpiryRule defines the tag and object expiration for the blob storage
	// associated to the pipeline run data (e.g. recipe, input, output).
	ExpiryRule minio.ExpiryRule `json:"__EXPIRY_RULE"`

	HeaderAuthorization string `json:"__PIPELINE_HEADER_AUTHORIZATION"`
	ModelBackend        string `json:"__MODEL_BACKEND"`
	MgmtBackend         string `json:"__MGMT_BACKEND"`
	ArtifactBackend     string `json:"__ARTIFACT_BACKEND"`
	AgentBackend        string `json:"__AGENT_BACKEND"`

	// OriginalHeader contains the original context header from the request.
	OriginalHeader map[string]string `json:"__ORIGINAL_HEADER"`
}

// TODO: GenerateSystemVariables will be refactored for better code structure.
// Planned refactor: 2024-11

// GenerateSystemVariables fills SystemVariable fields with information from
// the context and instance configuration.
func GenerateSystemVariables(ctx context.Context, sysVar SystemVariables) (map[string]any, error) {
	if sysVar.PipelineUserUID.IsNil() {
		sysVar.PipelineUserUID = uuid.FromStringOrNil(resourcex.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	}
	if sysVar.HeaderAuthorization == "" {
		sysVar.HeaderAuthorization = resourcex.GetRequestSingleHeader(ctx, "Authorization")
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
	if sysVar.AgentBackend == "" {
		sysVar.AgentBackend = fmt.Sprintf("%s:%d", config.Config.AgentBackend.Host, config.Config.AgentBackend.PublicPort)
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
