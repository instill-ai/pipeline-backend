package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

// TODO(huitang): Currently, we put float and int in the same struct.
// We can consider separating them into two structs.

type numberData struct {
	RawFloat   float64
	RawInteger int
	IsInteger  bool
}

func NewNumberFromFloat(f float64) *numberData {
	return &numberData{RawFloat: f, IsInteger: false}
}

func NewNumberFromInteger(i int) *numberData {
	return &numberData{RawInteger: i, IsInteger: true}
}

func (numberData) IsValue() {}

func (n *numberData) Integer() int {
	if n.IsInteger {
		return n.RawInteger
	}
	return int(n.RawFloat)
}

func (n *numberData) Float64() float64 {
	if n.IsInteger {
		return float64(n.RawInteger)
	}
	return n.RawFloat
}

func (n *numberData) String() string {
	if n.IsInteger {
		return fmt.Sprintf("%d", n.RawInteger)
	}
	return fmt.Sprintf("%f", n.RawFloat)
}

func (n *numberData) Get(p *path.Path) (v format.Value, err error) {
	if p == nil || p.IsEmpty() {
		return n, nil
	}
	return nil, fmt.Errorf("path not found: %s", p)
}

// Deprecated: ToStructValue() is deprecated and will be removed in a future
// version. structpb is not suitable for handling binary data and will be phased
// out gradually.
func (n numberData) ToStructValue() (v *structpb.Value, err error) {
	if n.IsInteger {
		v = structpb.NewNumberValue(float64(n.RawInteger))
	} else {
		v = structpb.NewNumberValue(n.RawFloat)
	}
	return
}

func (n *numberData) Equal(other format.Value) bool {
	if other, ok := other.(format.Number); ok {
		return n.Float64() == other.Float64()
	}
	return false
}
