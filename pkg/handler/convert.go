package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

// PBPipelineToDBPipeline converts protobuf data model to db data model
func PBPipelineToDBPipeline(ownerID uuid.UUID, pbPipeline *pipelinePB.Pipeline) *datamodel.Pipeline {
	logger, _ := logger.GetZapLogger()

	return &datamodel.Pipeline{
		OwnerID: ownerID,
		Name:    pbPipeline.GetDisplayName(),
		Mode:    datamodel.PipelineMode(pbPipeline.GetMode()),
		Status:  datamodel.PipelineStatus(pbPipeline.GetStatus()),

		BaseDynamic: datamodel.BaseDynamic{
			ID: func() uuid.UUID {
				if pbPipeline.Id == "" {
					pbPipeline.Id = uuid.UUID{}.String()
				}
				id, err := uuid.FromString(pbPipeline.Id)
				if err != nil {
					logger.Fatal(err.Error())
				}
				return id
			}(),

			CreateTime: func() time.Time {
				if pbPipeline.UpdateTime != nil {
					return pbPipeline.GetCreateTime().AsTime()
				}
				return time.Time{}
			}(),

			UpdateTime: func() time.Time {
				if pbPipeline.UpdateTime != nil {
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
			b, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(pbPipeline.Recipe)
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

	return &pipelinePB.Pipeline{
		Name:        fmt.Sprintf("pipelines/%s", dbPipeline.Name),
		Id:          dbPipeline.BaseDynamic.ID.String(),
		DisplayName: dbPipeline.Name,
		Mode:        pipelinePB.Pipeline_Mode(dbPipeline.Mode),
		Status:      pipelinePB.Pipeline_Status(dbPipeline.Status),
		OwnerId:     dbPipeline.OwnerID.String(),
		FullName:    dbPipeline.FullName,
		CreateTime:  timestamppb.New(dbPipeline.CreateTime),
		UpdateTime:  timestamppb.New(dbPipeline.UpdateTime),
		Description: &dbPipeline.Description.String,

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

}
