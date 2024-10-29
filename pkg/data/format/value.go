package format

import (
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type Value interface {
	IsValue()
	ToStructValue() (v *structpb.Value, err error)
	Get(p *path.Path) (v Value, err error)
	Equal(other Value) bool
}
