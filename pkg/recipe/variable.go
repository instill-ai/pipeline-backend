package recipe

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
)

type SystemVariables struct {
	PipelineID          string                 `json:"__PIPELINE_ID"`
	PipelineUID         uuid.UUID              `json:"__PIPELINE_UID"`
	PipelineReleaseID   string                 `json:"__PIPELINE_RELEASE_ID"`
	PipelineReleaseUID  uuid.UUID              `json:"__PIPELINE_RELEASE_UID"`
	PipelineRecipe      *datamodel.Recipe      `json:"__PIPELINE_RECIPE"`
	PipelineOwnerType   resource.NamespaceType `json:"__PIPELINE_OWNER_TYPE"`
	PipelineOwnerUID    uuid.UUID              `json:"__PIPELINE_OWNER_UID"`
	PipelineUserUID     uuid.UUID              `json:"__PIPELINE_USER_UID"`
	HeaderAuthorization string                 `json:"__PIPELINE_HEADER_AUTHORIZATION"`
	ModelBackend        string                 `json:"__MODEL_BACKEND"`
	MgmtBackend         string                 `json:"__MGMT_BACKEND"`
}

// System variables are available to all component
func GenerateSystemVariables(ctx context.Context, sysVar SystemVariables) (map[string]any, error) {

	if sysVar.PipelineUserUID == uuid.Nil {
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
