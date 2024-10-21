package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
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

func (s *stringData) Get(path string) (v value.Value, err error) {
	if path == "" {
		return s, nil
	}
	return nil, fmt.Errorf("wrong path %s for String", path)
}

func (s stringData) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewStringValue(s.Raw)
	return
}
