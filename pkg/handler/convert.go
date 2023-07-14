package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// PBToDBPipeline converts protobuf data model to db data model
func PBToDBPipeline(ctx context.Context, owner string, pbPipeline *pipelinePB.Pipeline) *datamodel.Pipeline {
	logger, _ := logger.GetZapLogger(ctx)

	return &datamodel.Pipeline{
		Owner: owner,
		ID:    pbPipeline.GetId(),
		State: datamodel.PipelineState(pbPipeline.GetState()),

		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbPipeline.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbPipeline.GetUid())
				if err != nil {
					logger.Error(err.Error())
				}
				return id
			}(),

			CreateTime: func() time.Time {
				if pbPipeline.GetCreateTime() != nil {
					return pbPipeline.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbPipeline.GetUpdateTime() != nil {
					return pbPipeline.GetUpdateTime().AsTime()
				}
				return time.Time{}
			}(),
		},

		Description: sql.NullString{
			String: pbPipeline.GetDescription(),
			Valid:  true,
		},

		Recipe: func() *datamodel.Recipe {
			if pbPipeline.GetRecipe() != nil {
				b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipeline.GetRecipe())
				if err != nil {
					logger.Error(err.Error())
				}

				recipe := datamodel.Recipe{}
				if err := json.Unmarshal(b, &recipe); err != nil {
					logger.Error(err.Error())
				}
				return &recipe
			}
			return nil
		}(),
	}
}

// DBToPBPipeline converts db data model to protobuf data model
func DBToPBPipeline(ctx context.Context, dbPipeline *datamodel.Pipeline) *pipelinePB.Pipeline {
	logger, _ := logger.GetZapLogger(ctx)

	pbPipeline := pipelinePB.Pipeline{
		Name:        fmt.Sprintf("pipelines/%s", dbPipeline.ID),
		Uid:         dbPipeline.BaseDynamic.UID.String(),
		Id:          dbPipeline.ID,
		State:       pipelinePB.Pipeline_State(dbPipeline.State),
		CreateTime:  timestamppb.New(dbPipeline.CreateTime),
		UpdateTime:  timestamppb.New(dbPipeline.UpdateTime),
		Description: &dbPipeline.Description.String,

		Recipe: func() *pipelinePB.Recipe {
			if dbPipeline.Recipe != nil {
				b, err := json.Marshal(dbPipeline.Recipe)
				if err != nil {
					logger.Error(err.Error())
				}
				pbRecipe := pipelinePB.Recipe{}

				err = json.Unmarshal(b, &pbRecipe)
				if err != nil {
					logger.Error(err.Error())
				}
				for i := range pbRecipe.Components {
					pbRecipe.Components[i].Type = pbRecipe.Components[i].ResourceDetail.GetFields()["connector_type"].GetStringValue()
				}
				return &pbRecipe
			}
			return nil
		}(),
	}

	if strings.HasPrefix(dbPipeline.Owner, "users/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_User{User: dbPipeline.Owner}
	} else if strings.HasPrefix(dbPipeline.Owner, "orgs/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_Org{Org: dbPipeline.Owner}
	}

	return &pbPipeline
}
