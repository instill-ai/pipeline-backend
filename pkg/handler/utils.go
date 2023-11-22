package handler

import (
	"github.com/instill-ai/pipeline-backend/pkg/service"
	// pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

func parseView(view int32) service.View {
	// switch view.(type) {
	// case pipelinePB.ListPipelinesRequest_View:
	// 	return service.View(view.(pipelinePB.ListPipelinesRequest_View))
	// default:
	if view == 0 {
		return service.VIEW_BASIC
	}
	return service.View(view)
	// }
	// return service.View(0)
}
