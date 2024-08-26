package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

type Null struct {
}

func NewNull() *Null {
	return &Null{}
}

func (Null) isValue() {}

func (n *Null) Get(path string) (v Value, err error) {
	if path == "" {
		return n, nil
	}
	return nil, fmt.Errorf("wrong path %s for Null", path)
}

func (n Null) ToStructValue() (v *structpb.Value, err error) {
	return structpb.NewNullValue(), nil
}
