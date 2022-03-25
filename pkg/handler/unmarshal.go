package handler

import (
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

func unmarshalRecipeSource(d *pipelinePB.Source) *datamodel.Source {
	return &datamodel.Source{
		Type: d.Type,
	}
}

func unmarshalDataDestination(d *pipelinePB.Destination) *datamodel.Destination {
	return &datamodel.Destination{
		Type: d.Type,
	}
}

func unmarshalRecipeModel(v []*pipelinePB.Model) []*datamodel.Model {
	var ret []*datamodel.Model
	for _, vv := range v {
		ret = append(ret, &datamodel.Model{
			Name:    vv.Name,
			Version: vv.Version,
		})
	}
	return ret
}

func unmarshalRecipe(recipe *pipelinePB.Recipe) *datamodel.Recipe {
	return &datamodel.Recipe{
		Source:      unmarshalRecipeSource(recipe.Source),
		Destination: unmarshalDataDestination(recipe.Destination),
		Model:       unmarshalRecipeModel(recipe.Models),
	}
}
