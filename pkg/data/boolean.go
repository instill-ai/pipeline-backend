package data

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
	"google.golang.org/protobuf/types/known/structpb"
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

func (b *booleanData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return b, nil
	}
	return nil, fmt.Errorf("path not found: %s", p)
}

// Deprecated: ToStructValue() is deprecated and will be removed in a future
// version. structpb is not suitable for handling binary data and will be phased
// out gradually.
func (b booleanData) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewBoolValue(b.Raw)
	return
}

func (b *booleanData) Equal(other format.Value) bool {
	if other, ok := other.(format.Boolean); ok {
		return b.Raw == other.Boolean()
	}
	return false
}

func (b *booleanData) Hash() string {
	return fmt.Sprintf("%t", b.Raw)
}

func (b *booleanData) ToJSONValue() (v any, err error) {
	return b.Raw, nil
}
