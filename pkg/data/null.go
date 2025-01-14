package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type nullData struct {
}

func NewNull() *nullData {
	return &nullData{}
}

func (nullData) IsValue()   {}
func (nullData) Omittable() {}

func (n *nullData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return n, nil
	}
	return nil, fmt.Errorf("path not found: %s", p)
}

// Deprecated: ToStructValue() is deprecated and will be removed in a future
// version. structpb is not suitable for handling binary data and will be phased
// out gradually.
func (n nullData) ToStructValue() (v *structpb.Value, err error) {
	return structpb.NewNullValue(), nil
}

func (n *nullData) Equal(other format.Value) bool {
	if _, ok := other.(*nullData); ok {
		return true
	}
	return false
}

func (n *nullData) String() string {
	return "null"
}

func (n *nullData) ToJSONValue() (v any, err error) {
	return nil, nil
}
