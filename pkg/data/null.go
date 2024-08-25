package data

import "google.golang.org/protobuf/types/known/structpb"

type Null struct {
}

func NewNull() *Null {
	return &Null{}
}

func (Null) isValue() {}

func (n Null) ToStructValue() (v *structpb.Value, err error) {
	return structpb.NewNullValue(), nil
}
