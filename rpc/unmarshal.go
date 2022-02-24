package rpc

import (
	"github.com/instill-ai/pipeline-backend/pkg/model"
	pb "github.com/instill-ai/protogen-go/pipeline"
)

func unmarshalRecipeSource(d *pb.Source) *model.Source {
	return &model.Source{
		Type: d.Type,
	}
}

func unmarshalDataDestination(d *pb.Destination) *model.Destination {
	return &model.Destination{
		Type: d.Type,
	}
}

func unmarshalRecipeModel(v []*pb.Model) []*model.Model {
	var ret []*model.Model
	for _, vv := range v {
		ret = append(ret, &model.Model{
			Name:    vv.Name,
			Version: vv.Version,
		})
	}
	return ret
}

func unmarshalRecipe(recipe *pb.Recipe) *model.Recipe {
	return &model.Recipe{
		Source:      unmarshalRecipeSource(recipe.Source),
		Destination: unmarshalDataDestination(recipe.Destination),
		Model:       unmarshalRecipeModel(recipe.Model),
	}
}

func unmarshalPipeline(pipeline *pb.PipelineInfo) *model.Pipeline {
	ret := &model.Pipeline{
		Id:          pipeline.Id,
		Name:        pipeline.Name,
		Description: pipeline.Description,
		Active:      pipeline.Active,
		CreatedAt:   pipeline.CreatedAt.AsTime(),
		UpdatedAt:   pipeline.UpdatedAt.AsTime(),
		FullName:    pipeline.FullName,
	}

	if pipeline.Recipe != nil {
		ret.Recipe = unmarshalRecipe(pipeline.Recipe)
	}

	return ret
}

func unmarshalPipelineTriggerContent(content *pb.TriggerPipelineContent) *model.TriggerPipelineContent {
	return &model.TriggerPipelineContent{
		Url:    content.Url,
		Base64: content.Base64,
	}
}
