package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type String struct {
	Raw string
}

func (String) IsValue() {}

func NewString(t string) *String {
	return &String{Raw: t}
}

func (s *String) GetString() string {
	return s.Raw
}

func (s *String) Get(path string) (v value.Value, err error) {
	if path == "" {
		return s, nil
	}
	return nil, fmt.Errorf("wrong path %s for String", path)
}

func (s String) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewStringValue(s.Raw)
	return
}
