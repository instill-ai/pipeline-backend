package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

// PBPipelineToDBPipeline converts protobuf data model to db data model
func PBPipelineToDBPipeline(owner string, pbPipeline *pipelinePB.Pipeline) *datamodel.Pipeline {
	logger, _ := logger.GetZapLogger()

	return &datamodel.Pipeline{
		Owner: owner,
		ID:    pbPipeline.GetId(),
		Mode:  datamodel.PipelineMode(pbPipeline.GetMode()),
		State: datamodel.PipelineState(pbPipeline.GetState()),

		BaseDynamic: datamodel.BaseDynamic{
			UID: func() uuid.UUID {
				if pbPipeline.GetUid() == "" {
					return uuid.UUID{}
				}
				id, err := uuid.FromString(pbPipeline.GetUid())
				if err != nil {
					logger.Fatal(err.Error())
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
			Valid:  len(pbPipeline.GetDescription()) > 0,
		},

		Recipe: func() *datamodel.Recipe {
			b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipeline.GetRecipe())
			if err != nil {
				logger.Fatal(err.Error())
			}

			recipe := datamodel.Recipe{}
			if err := json.Unmarshal(b, &recipe); err != nil {
				logger.Fatal(err.Error())
			}

			return &recipe
		}(),
	}
}

// DBPipelineToPBPipeline converts db data model to protobuf data model
func DBPipelineToPBPipeline(dbPipeline *datamodel.Pipeline) *pipelinePB.Pipeline {
	logger, _ := logger.GetZapLogger()

	pbPipeline := pipelinePB.Pipeline{
		Name:       fmt.Sprintf("pipelines/%s", dbPipeline.ID),
		Uid:        dbPipeline.BaseDynamic.UID.String(),
		Id:         dbPipeline.ID,
		Mode:       pipelinePB.Pipeline_Mode(dbPipeline.Mode),
		State:      pipelinePB.Pipeline_State(dbPipeline.State),
		CreateTime: timestamppb.New(dbPipeline.CreateTime),
		UpdateTime: timestamppb.New(dbPipeline.UpdateTime),

		Description: func() *string {
			if dbPipeline.Description.Valid {
				return &dbPipeline.Description.String
			}
			emptyStr := ""
			return &emptyStr
		}(),

		Recipe: func() *pipelinePB.Recipe {
			if dbPipeline.Recipe != nil {
				b, err := json.Marshal(dbPipeline.Recipe)
				if err != nil {
					logger.Fatal(err.Error())
				}
				pbRecipe := pipelinePB.Recipe{}
				err = json.Unmarshal(b, &pbRecipe)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return &pbRecipe
			}
			return nil
		}(),
	}

	if strings.HasPrefix(dbPipeline.Owner, "users/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_User{User: dbPipeline.Owner}
	} else if strings.HasPrefix(dbPipeline.Owner, "organizations/") {
		pbPipeline.Owner = &pipelinePB.Pipeline_Org{Org: dbPipeline.Owner}
	}

	return &pbPipeline
}
