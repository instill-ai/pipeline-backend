package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type Null struct {
}

func NewNull() *Null {
	return &Null{}
}

func (Null) IsValue() {}

func (n *Null) Get(path string) (v value.Value, err error) {
	if path == "" {
		return n, nil
	}
	return nil, fmt.Errorf("wrong path %s for Null", path)
}

func (n Null) ToStructValue() (v *structpb.Value, err error) {
	return structpb.NewNullValue(), nil
}
