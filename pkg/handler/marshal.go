package handler

import (
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

func marshalRecipeSource(d *datamodel.Source) *pipelinePB.Source {
	return &pipelinePB.Source{
		Type: d.Type,
	}
}

func marshalRecipeDestination(d *datamodel.Destination) *pipelinePB.Destination {
	return &pipelinePB.Destination{
		Type: d.Type,
	}
}

func marshalRecipeModel(v []*datamodel.Model) []*pipelinePB.Model {
	var ret []*pipelinePB.Model
	for _, vv := range v {
		ret = append(ret, &pipelinePB.Model{
			Name:    vv.Name,
			Version: vv.Version,
		})
	}
	return ret
}

func marshalRecipe(recipe *datamodel.Recipe) *pipelinePB.Recipe {
	return &pipelinePB.Recipe{
		Source:      marshalRecipeSource(recipe.Source),
		Destination: marshalRecipeDestination(recipe.Destination),
		Models:      marshalRecipeModel(recipe.Model),
	}
}

func marshalPipeline(pipeline *datamodel.Pipeline) *pipelinePB.Pipeline {
	ret := &pipelinePB.Pipeline{
		Id:          pipeline.Id,
		Name:        pipeline.Name,
		Description: pipeline.Description,
		Active:      pipeline.Active,
		CreatedAt:   timestamppb.New(pipeline.CreatedAt),
		UpdatedAt:   timestamppb.New(pipeline.UpdatedAt),
		FullName:    pipeline.FullName,
	}

	if pipeline.Recipe != nil {
		ret.Recipe = marshalRecipe(pipeline.Recipe)
	}

	return ret
}
