package handler

import (
	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func parseView(view pipelinePB.View) pipelinePB.View {
	parsedView := pipelinePB.View_VIEW_BASIC
	if view != pipelinePB.View_VIEW_UNSPECIFIED {
		parsedView = view
	}
	return parsedView
}
