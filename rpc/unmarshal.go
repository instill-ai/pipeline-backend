package rpc

import (
	"github.com/instill-ai/pipeline-backend/pkg/model"
	pb "github.com/instill-ai/protogen-go/pipeline"
)

func unmarshalDataSource(d *pb.DataSource) *model.DataSource {
	return &model.DataSource{
		Type: d.Type,
	}
}

func unmarshalDataDestination(d *pb.DataDestination) *model.DataDestination {
	return &model.DataDestination{
		Type: d.Type,
	}
}

func unmarshalVisualDataOperator(v []*pb.VisualDataOperator) []*model.VisualDataOperator {
	var ret []*model.VisualDataOperator
	for _, vv := range v {
		ret = append(ret, &model.VisualDataOperator{
			ModelId: vv.ModelId,
			Version: vv.Version,
		})
	}
	return ret
}

func unmarshalLogicOperator(l []*pb.LogicOperator) []*model.LogicOperator {
	var ret []*model.LogicOperator
	return ret
}

func unmarshalRecipe(recipe *pb.Recipe) *model.Recipe {
	return &model.Recipe{
		DataSource:         unmarshalDataSource(recipe.DataSource),
		DataDestination:    unmarshalDataDestination(recipe.DataDestination),
		VisualDataOperator: unmarshalVisualDataOperator(recipe.VisualDataOperator),
		LogicOperator:      unmarshalLogicOperator(recipe.LogicOperator),
	}
}

func unmarshalPipeline(pipeline *pb.PipelineInfo) *model.Pipeline {
	return &model.Pipeline{
		Id:          pipeline.Id,
		Name:        pipeline.Name,
		Description: pipeline.Description,
		Active:      pipeline.Active,
		CreatedAt:   pipeline.CreatedAt.AsTime(),
		UpdatedAt:   pipeline.UpdatedAt.AsTime(),
		Recipe:      unmarshalRecipe(pipeline.Recipe),
		FullName:    pipeline.FullName,
	}
}

func unmarshalPipelineTriggerContent(content *pb.TriggerPipelineContent) *model.TriggerPipelineContent {
	return &model.TriggerPipelineContent{
		Url:    content.Url,
		Base64: content.Base64,
		Chunk:  content.Chunk,
	}
}
