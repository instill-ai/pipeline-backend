package data

import "google.golang.org/protobuf/types/known/structpb"

type String struct {
	Raw string
}

func (String) isValue() {}

func NewString(t string) *String {
	return &String{Raw: t}
}

func (s *String) GetString() string {
	return s.Raw
}

func (s String) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewStringValue(s.Raw)
	return
}
