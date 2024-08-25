package data

import (
	"encoding/base64"

	"google.golang.org/protobuf/types/known/structpb"
)

type ByteArray struct {
	Raw []byte
}

func NewByteArray(b []byte) *ByteArray {
	return &ByteArray{Raw: b}
}

func (ByteArray) isValue() {}

func (b *ByteArray) GetByteArray() []byte {
	return b.Raw
}

func (b ByteArray) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewStringValue(base64.StdEncoding.EncodeToString(b.Raw))
	return
}
