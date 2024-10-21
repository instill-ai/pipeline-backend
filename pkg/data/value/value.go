package value

import "google.golang.org/protobuf/types/known/structpb"

type Value interface {
	IsValue()
	ToStructValue() (v *structpb.Value, err error)
	Get(path string) (v Value, err error)
}
