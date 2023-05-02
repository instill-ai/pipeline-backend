package service

import (
	"context"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	"google.golang.org/grpc/metadata"
)

type DefinitionType int64

const (
	DefinitionTypeUnspecified DefinitionType = 0
	SourceConnector           DefinitionType = 1
	DestinationConnector      DefinitionType = 2
	Model                     DefinitionType = 3
)

func GenOwnerPermalink(owner *mgmtPB.User) string {
	return "users/" + owner.GetUid()
}

func InjectOwnerToContext(ctx context.Context, owner *mgmtPB.User) context.Context {
	ctx = metadata.AppendToOutgoingContext(ctx, "Jwt-Sub", owner.GetUid())
	return ctx
}

func GetDefinitionType(component *datamodel.Component) DefinitionType {
	if i := strings.Index(component.ResourceName, "/"); i >= 0 {
		switch component.ResourceName[:i] {
		case "source-connectors":
			return SourceConnector
		case "destination-connectors":
			return DestinationConnector
		case "models":
			return Model
		}
	}
	return DefinitionTypeUnspecified
}

func GetResourceFromRecipe(recipe *datamodel.Recipe, t DefinitionType) []string {
	resources := []string{}
	for _, component := range recipe.Components {
		switch GetDefinitionType(component) {
		case t:
			resources = append(resources, component.ResourceName)
		}
	}
	return resources
}

func GetModelsFromRecipe(recipe *datamodel.Recipe) []string {
	return GetResourceFromRecipe(recipe, Model)
}

func GetDestinationsFromRecipe(recipe *datamodel.Recipe) []string {
	return GetResourceFromRecipe(recipe, DestinationConnector)
}

func GetSourcesFromRecipe(recipe *datamodel.Recipe) []string {
	return GetResourceFromRecipe(recipe, SourceConnector)
}
