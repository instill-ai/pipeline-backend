package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"
)

type Number struct {
	Raw float64
}

func NewNumberFromFloat(f float64) *Number {
	return &Number{Raw: f}
}

func NewNumberFromInteger(i int) *Number {
	return &Number{Raw: float64(i)}
}

func (Number) isValue() {}

func (n *Number) GetInteger() int {
	return int(n.Raw)
}

func (n *Number) GetFloat() float64 {
	return n.Raw
}

func (n *Number) Get(path string) (v Value, err error) {
	if path == "" {
		return n, nil
	}
	return nil, fmt.Errorf("wrong path %s for Number", path)
}

func (n Number) ToStructValue() (v *structpb.Value, err error) {
	v = structpb.NewNumberValue(n.Raw)
	return
}
