package data

import (
	"encoding/base64"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
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

func (b *byteArrayData) Get(path string) (v value.Value, err error) {
	if path == "" {
		return b, nil
	}
	return nil, fmt.Errorf("wrong path %s for byteArray", path)
}

func (b byteArrayData) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewStringValue(base64.StdEncoding.EncodeToString(b.Raw))
	return
}
