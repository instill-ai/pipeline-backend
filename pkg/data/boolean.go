package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

type booleanData struct {
	Raw bool
}

func NewBoolean(b bool) *booleanData {
	return &booleanData{Raw: b}
}

func (booleanData) IsValue() {}

func (b *booleanData) Boolean() bool {
	return b.Raw
}
func (b *booleanData) String() (val string) {
	if b.Raw {
		return "true"
	} else {
		return "false"
	}
}

func (b *booleanData) Get(path string) (v value.Value, err error) {
	if path == "" {
		return b, nil
	}
	return nil, fmt.Errorf("wrong path %s for boolean", path)
}

func (b booleanData) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewBoolValue(b.Raw)
	return
}
