package rpc

import (
	"github.com/instill-ai/pipeline-backend/pkg/model"

	pb "github.com/instill-ai/protogen-go/pipeline/v1alpha"
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
		Model:       unmarshalRecipeModel(recipe.Models),
	}
}
