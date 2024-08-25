package data

import "google.golang.org/protobuf/types/known/structpb"

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

func (b Boolean) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewBoolValue(b.Raw)
	return
}
