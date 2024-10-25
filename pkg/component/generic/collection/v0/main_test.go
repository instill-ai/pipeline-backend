package collection

import (
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	qt "github.com/frankban/quicktest"
)

const (
	arrays1 = `
{
	"arrays": [
		["a", "b"],
		["c", "d"]
	]
}`

	arrays2 = `
{
	"arrays": [
		["a", "b"],
		["c", "d"],
		["e"]
	]
}`

	array1 = `
{
	"array": ["a", "b", "c", "d"]
}`

	array2 = `
{
	"array": ["a", "b", "c", "d", "e"]
}`
)

func Test_Concat(t *testing.T) {
	c := qt.New(t)

	c.Run("Concat", func(c *qt.C) {
		e := &execution{}

		inputStruct := &structpb.Struct{}
		err := protojson.Unmarshal([]byte(arrays1), inputStruct)
		c.Assert(err, qt.IsNil)

		expectedOutput := &structpb.Struct{}
		err = protojson.Unmarshal([]byte(array1), expectedOutput)
		c.Assert(err, qt.IsNil)

		out, err := e.concat(inputStruct)

		c.Assert(err, qt.IsNil)

		c.Assert(proto.Equal(out, expectedOutput), qt.IsTrue)

	})
}

func Test_Split(t *testing.T) {
	c := qt.New(t)

	testcases := []struct {
		name       string
		groupSize  int
		arrayInput string
		output     string
	}{
		{
			name:       "Split without remaining",
			groupSize:  2,
			arrayInput: array1,
			output:     arrays1,
		},
		{
			name:       "Split with remaining",
			groupSize:  2,
			arrayInput: array2,
			output:     arrays2,
		},
	}

	for _, tc := range testcases {

		c.Run(tc.name, func(c *qt.C) {
			e := &execution{}

			inputStruct := &structpb.Struct{}
			err := protojson.Unmarshal([]byte(tc.arrayInput), inputStruct)

			c.Assert(err, qt.IsNil)

			// set group-size to 2
			inputStruct.Fields["group-size"] = &structpb.Value{
				Kind: &structpb.Value_NumberValue{NumberValue: float64(tc.groupSize)},
			}

			expectedOutput := &structpb.Struct{}
			err = protojson.Unmarshal([]byte(tc.output), expectedOutput)
			c.Assert(err, qt.IsNil)

			out, err := e.split(inputStruct)

			c.Assert(err, qt.IsNil)

			c.Assert(proto.Equal(out, expectedOutput), qt.IsTrue)
		})

	}

}
