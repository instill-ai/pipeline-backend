package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type stringData struct {
	Raw string
}

func (stringData) IsValue()    {}
func (stringData) Resolvable() {}

func NewString(t string) *stringData {
	return &stringData{Raw: t}
}

func (s *stringData) String() string {
	return s.Raw
}

func (s *stringData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return s, nil
	}
	return nil, fmt.Errorf("path not found: %s", p)
}

// Deprecated: ToStructValue() is deprecated and will be removed in a future
// version. structpb is not suitable for handling binary data and will be phased
// out gradually.
func (s stringData) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewStringValue(s.Raw)
	return
}

func (s *stringData) Equal(other format.Value) bool {
	if other, ok := other.(format.String); ok {
		return s.Raw == other.String()
	}
	return false
}

func (s *stringData) ToJSONValue() (v any, err error) {
	return s.Raw, nil
}
