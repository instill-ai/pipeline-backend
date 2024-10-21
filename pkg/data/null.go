package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type nullData struct {
}

func NewNull() *nullData {
	return &nullData{}
}

func (nullData) IsValue()   {}
func (nullData) Omittable() {}

func (n *nullData) Get(path string) (v value.Value, err error) {
	if path == "" {
		return n, nil
	}
	return nil, fmt.Errorf("wrong path %s for Null", path)
}

func (n nullData) ToStructValue() (v *structpb.Value, err error) {
	return structpb.NewNullValue(), nil
}
