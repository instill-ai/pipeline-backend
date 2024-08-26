package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

type Boolean struct {
	Raw bool
}

func NewBoolean(b bool) *Boolean {
	return &Boolean{Raw: b}
}

func (Boolean) isValue() {}

func (b *Boolean) GetBoolean() bool {
	return b.Raw
}

func (b *Boolean) Get(path string) (v Value, err error) {
	if path == "" {
		return b, nil
	}
	return nil, fmt.Errorf("wrong path %s for Boolean", path)
}

func (b Boolean) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewBoolValue(b.Raw)
	return
}
