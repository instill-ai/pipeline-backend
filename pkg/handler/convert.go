package handler

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

func convertDBPipelineToPBPipeline(dbPipeline *datamodel.Pipeline) *pipelinePB.Pipeline {
	logger, _ := logger.GetZapLogger()

	return &pipelinePB.Pipeline{
		Id:          dbPipeline.BaseDynamic.ID.String(),
		Name:        dbPipeline.Name,
		Description: dbPipeline.Description,
		Status:      pipelinePB.Pipeline_Status(dbPipeline.Status),
		OwnerId:     dbPipeline.OwnerID.String(),
		FullName:    dbPipeline.FullName,
		CreatedAt:   timestamppb.New(dbPipeline.CreatedAt),
		UpdatedAt:   timestamppb.New(dbPipeline.UpdatedAt),

		Recipe: func() *pipelinePB.Recipe {

			if dbPipeline.Recipe != nil {

				pbRecipeByte, err := json.Marshal(dbPipeline.Recipe)
				if err != nil {
					logger.Fatal(err.Error())
				}

				pbRecipe := pipelinePB.Recipe{}
				err = protojson.Unmarshal(pbRecipeByte, &pbRecipe)
				if err != nil {
					logger.Fatal(err.Error())
				}

				return &pbRecipe
			}
			return nil
		}(),
	}

}
