package rpc

import (
	"github.com/instill-ai/pipeline-backend/pkg/model"
	pb "github.com/instill-ai/protogen-go/pipeline"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func marshalDataSource(d *model.DataSource) *pb.DataSource {
	return &pb.DataSource{
		Type: d.Type,
	}
}

func marshalDataDestination(d *model.DataDestination) *pb.DataDestination {
	return &pb.DataDestination{
		Type: d.Type,
	}
}

func marshalVisualDataOperator(v []*model.VisualDataOperator) []*pb.VisualDataOperator {
	var ret []*pb.VisualDataOperator
	for _, vv := range v {
		ret = append(ret, &pb.VisualDataOperator{
			ModelId: vv.ModelId,
			Version: vv.Version,
		})
	}
	return ret
}

func marshalLogicOperator(l []*model.LogicOperator) []*pb.LogicOperator {
	var ret []*pb.LogicOperator
	return ret
}

func marshalRecipe(recipe *model.Recipe) *pb.Recipe {
	return &pb.Recipe{
		DataSource:         marshalDataSource(recipe.DataSource),
		DataDestination:    marshalDataDestination(recipe.DataDestination),
		VisualDataOperator: marshalVisualDataOperator(recipe.VisualDataOperator),
		LogicOperator:      marshalLogicOperator(recipe.LogicOperator),
	}
}

func marshalPipeline(pipeline *model.Pipeline) *pb.PipelineInfo {
	return &pb.PipelineInfo{
		Id:          pipeline.Id,
		Name:        pipeline.Name,
		Description: pipeline.Description,
		Active:      pipeline.Active,
		CreatedAt:   timestamppb.New(pipeline.CreatedAt),
		UpdatedAt:   timestamppb.New(pipeline.UpdatedAt),
		Recipe:      marshalRecipe(pipeline.Recipe),
		FullName:    pipeline.FullName,
	}
}
