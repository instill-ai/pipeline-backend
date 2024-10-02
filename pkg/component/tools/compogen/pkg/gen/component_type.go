package gen

import (
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// ComponentType holds the possible subtypes of a component (e.g.
// "operator", "AI", "data") and implements several helper methods.
type ComponentType string

const (
	cstOperator    ComponentType = "operator"
	cstAI          ComponentType = "AI"
	cstApplication ComponentType = "application"
	cstData        ComponentType = "data"
	cstGeneric     ComponentType = "generic"
)

var toComponentType = map[string]ComponentType{
	pb.ComponentType_COMPONENT_TYPE_AI.String():          cstAI,
	pb.ComponentType_COMPONENT_TYPE_APPLICATION.String(): cstApplication,
	pb.ComponentType_COMPONENT_TYPE_DATA.String():        cstData,
	pb.ComponentType_COMPONENT_TYPE_OPERATOR.String():    cstOperator,
	pb.ComponentType_COMPONENT_TYPE_GENERIC.String():     cstGeneric,
}

var modifiesArticle = map[ComponentType]bool{
	cstOperator:    true,
	cstAI:          true,
	cstApplication: true,
}

// IndefiniteArticle returns the correct indefinite article (in English) for a
// component subtype, e.g., "an" (operator), "an" (AI), "a" (data
// connector).
func (ct ComponentType) IndefiniteArticle() string {
	if modifiesArticle[ct] {
		return "an"
	}

	return "a"
}
