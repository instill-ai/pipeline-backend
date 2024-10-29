package data

import (
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/data/path"
)

type numberData struct {
	Raw float64
}

func NewNumberFromFloat(f float64) *numberData {
	return &numberData{Raw: f}
}

func NewNumberFromInteger(i int) *numberData {
	return &numberData{Raw: float64(i)}
}

func (numberData) IsValue() {}

func (n *numberData) Integer() int {
	return int(n.Raw)
}

func (n *numberData) Float64() float64 {
	return n.Raw
}

func (n *numberData) String() string {
	return fmt.Sprintf("%f", n.Raw)
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
	v = structpb.NewNumberValue(n.Raw)
	return
}

func (n *numberData) Equal(other format.Value) bool {
	if other, ok := other.(format.Number); ok {
		return n.Raw == other.Float64()
	}
	return false
}
