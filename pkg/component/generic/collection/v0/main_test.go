package collection

import (
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"
)

func Test_Concat(t *testing.T) {
	c := qt.New(t)

	c.Run("Concat", func(c *qt.C) {
		e := &execution{}

		inputArrays := []*structpb.Value{
			structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStringValue("a"),
					structpb.NewStringValue("b"),
				},
			}),
			structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStringValue("c"),
					structpb.NewStringValue("d"),
				},
			}),
		}

		inputStruct := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"arrays": structpb.NewListValue(&structpb.ListValue{
					Values: inputArrays,
				}),
			},
		}

		expectedOutput := &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"array": structpb.NewListValue(&structpb.ListValue{
					Values: []*structpb.Value{
						structpb.NewStringValue("a"),
						structpb.NewStringValue("b"),
						structpb.NewStringValue("c"),
						structpb.NewStringValue("d"),
					},
				}),
			},
		}

		out, err := e.concat(inputStruct)

		c.Assert(err, qt.IsNil)

		c.Assert(proto.Equal(out, expectedOutput), qt.IsTrue)

	})

}
