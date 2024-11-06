package data

import (
	"bytes"
	"encoding/base64"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type byteArrayData struct {
	Raw []byte
}

func NewByteArray(b []byte) *byteArrayData {
	return &byteArrayData{Raw: b}
}

func (byteArrayData) IsValue() {}

func (b *byteArrayData) ByteArray() []byte {
	return b.Raw
}

func (b *byteArrayData) String() (val string) {
	return base64.StdEncoding.EncodeToString(b.Raw)
}

func (b *byteArrayData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return b, nil
	}
	return nil, fmt.Errorf("path not found: %s", p)
}

// Deprecated: ToStructValue() is deprecated and will be removed in a future
// version. structpb is not suitable for handling binary data and will be phased
// out gradually.
func (b byteArrayData) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewStringValue(base64.StdEncoding.EncodeToString(b.Raw))
	return
}

func (b *byteArrayData) Equal(other format.Value) bool {
	if other, ok := other.(format.ByteArray); ok {
		return bytes.Equal(b.Raw, other.ByteArray())
	}
	return false
}
