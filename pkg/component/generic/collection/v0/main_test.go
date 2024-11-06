package collection

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/base"
	"github.com/instill-ai/pipeline-backend/pkg/component/internal/mock"
	"github.com/instill-ai/pipeline-backend/pkg/data"
	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestAppend(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    appendInput
		expected []format.Value
	}{
		{
			name: "append string to array",
			input: appendInput{
				Array:   []format.Value{data.NewString("a"), data.NewString("b")},
				Element: data.NewString("c"),
			},
			expected: []format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c")},
		},
		{
			name: "append to empty array",
			input: appendInput{
				Array:   []format.Value{},
				Element: data.NewNumberFromInteger(1),
			},
			expected: []format.Value{data.NewNumberFromInteger(1)},
		},
		{
			name: "append number to string array",
			input: appendInput{
				Array:   []format.Value{data.NewString("a"), data.NewString("b")},
				Element: data.NewNumberFromInteger(1),
			},
			expected: []format.Value{data.NewString("a"), data.NewString("b"), data.NewNumberFromInteger(1)},
		},
		{
			name: "append boolean to array",
			input: appendInput{
				Array:   []format.Value{data.NewString("a"), data.NewNumberFromInteger(1)},
				Element: data.NewBoolean(true),
			},
			expected: []format.Value{data.NewString("a"), data.NewNumberFromInteger(1), data.NewBoolean(true)},
		},
		{
			name: "append array to array",
			input: appendInput{
				Array:   []format.Value{data.NewString("a")},
				Element: data.Array{data.NewString("b"), data.NewString("c")},
			},
			expected: []format.Value{data.NewString("a"), data.Array{data.NewString("b"), data.NewString("c")}},
		},
		{
			name: "append map to array",
			input: appendInput{
				Array: []format.Value{data.NewString("a")},
				Element: data.Map{
					"key": data.NewString("value"),
				},
			},
			expected: []format.Value{data.NewString("a"), data.Map{
				"key": data.NewString("value"),
			}},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_APPEND",
			})
			c.Assert(err, qt.IsNil)

			ir, ow, _, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *appendInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput *appendOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(*appendOutput)
				return nil
			})

			err = execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
			c.Assert(capturedOutput.Array, qt.DeepEquals, tc.expected)
		})
	}
}

func TestAssign(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    assignInput
		expected format.Value
	}{
		{
			name: "assign string value",
			input: assignInput{
				Data: data.NewString("test"),
			},
			expected: data.NewString("test"),
		},
		{
			name: "assign number value",
			input: assignInput{
				Data: data.NewNumberFromInteger(42),
			},
			expected: data.NewNumberFromInteger(42),
		},
		{
			name: "assign array value",
			input: assignInput{
				Data: data.Array{
					data.NewString("a"),
					data.NewString("b"),
					data.NewNumberFromInteger(1),
				},
			},
			expected: data.Array{
				data.NewString("a"),
				data.NewString("b"),
				data.NewNumberFromInteger(1),
			},
		},
		{
			name: "assign empty array",
			input: assignInput{
				Data: data.Array{},
			},
			expected: data.Array{},
		},
		{
			name: "assign map value",
			input: assignInput{
				Data: data.Map{
					"name":  data.NewString("test"),
					"age":   data.NewNumberFromInteger(25),
					"items": data.Array{data.NewString("a"), data.NewString("b")},
				},
			},
			expected: data.Map{
				"name":  data.NewString("test"),
				"age":   data.NewNumberFromInteger(25),
				"items": data.Array{data.NewString("a"), data.NewString("b")},
			},
		},
		{
			name: "assign empty map",
			input: assignInput{
				Data: data.Map{},
			},
			expected: data.Map{},
		},
		{
			name: "assign nested map",
			input: assignInput{
				Data: data.Map{
					"user": data.Map{
						"details": data.Map{
							"name":   data.NewString("test"),
							"active": data.NewBoolean(true),
							"scores": data.Array{data.NewNumberFromInteger(85), data.NewNumberFromInteger(90)},
						},
					},
				},
			},
			expected: data.Map{
				"user": data.Map{
					"details": data.Map{
						"name":   data.NewString("test"),
						"active": data.NewBoolean(true),
						"scores": data.Array{data.NewNumberFromInteger(85), data.NewNumberFromInteger(90)},
					},
				},
			},
		},
		{
			name: "assign boolean value",
			input: assignInput{
				Data: data.NewBoolean(true),
			},
			expected: data.NewBoolean(true),
		},
		{
			name: "assign null value",
			input: assignInput{
				Data: data.NewNull(),
			},
			expected: data.NewNull(),
		},
		{
			name: "assign complex nested structure",
			input: assignInput{
				Data: data.Map{
					"metadata": data.Map{
						"created": data.NewString("2023-01-01"),
						"tags": data.Array{
							data.NewString("tag1"),
							data.NewString("tag2"),
						},
						"settings": data.Map{
							"enabled": data.NewBoolean(true),
							"config": data.Map{
								"timeout": data.NewNumberFromInteger(30),
								"retries": data.NewNumberFromInteger(3),
							},
						},
					},
				},
			},
			expected: data.Map{
				"metadata": data.Map{
					"created": data.NewString("2023-01-01"),
					"tags": data.Array{
						data.NewString("tag1"),
						data.NewString("tag2"),
					},
					"settings": data.Map{
						"enabled": data.NewBoolean(true),
						"config": data.Map{
							"timeout": data.NewNumberFromInteger(30),
							"retries": data.NewNumberFromInteger(3),
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_ASSIGN",
			})
			c.Assert(err, qt.IsNil)

			ir, ow, _, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *assignInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput *assignOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(*assignOutput)
				return nil
			})

			err = execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
			c.Assert(capturedOutput.Data, qt.DeepEquals, tc.expected)
		})
	}
}

func TestConcat(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    concatInput
		expected []format.Value
	}{
		{
			name: "concat two arrays",
			input: concatInput{
				Arrays: [][]format.Value{
					{data.NewString("a"), data.NewString("b")},
					{data.NewString("c"), data.NewString("d")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c"), data.NewString("d")},
		},
		{
			name: "concat empty arrays",
			input: concatInput{
				Arrays: [][]format.Value{{}, {}},
			},
			expected: []format.Value{},
		},
		{
			name: "concat multiple arrays",
			input: concatInput{
				Arrays: [][]format.Value{
					{data.NewNumberFromInteger(1)},
					{data.NewNumberFromInteger(2)},
					{data.NewNumberFromInteger(3)},
				},
			},
			expected: []format.Value{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
		},
		{
			name: "concat mixed type arrays",
			input: concatInput{
				Arrays: [][]format.Value{
					{data.NewString("a"), data.NewNumberFromInteger(1)},
					{data.NewNumberFromInteger(2), data.NewString("b")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewString("b")},
		},
		{
			name: "concat arrays with boolean values",
			input: concatInput{
				Arrays: [][]format.Value{
					{data.NewBoolean(true), data.NewBoolean(false)},
					{data.NewBoolean(true)},
				},
			},
			expected: []format.Value{data.NewBoolean(true), data.NewBoolean(false), data.NewBoolean(true)},
		},
		{
			name: "concat arrays with null values",
			input: concatInput{
				Arrays: [][]format.Value{
					{data.NewString("a"), data.NewNull()},
					{data.NewNull(), data.NewString("b")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewNull(), data.NewNull(), data.NewString("b")},
		},
		{
			name: "concat arrays with nested structures",
			input: concatInput{
				Arrays: [][]format.Value{
					{data.Array{data.NewString("a"), data.NewString("b")}},
					{data.Map{"key": data.NewString("value")}},
				},
			},
			expected: []format.Value{
				data.Array{data.NewString("a"), data.NewString("b")},
				data.Map{"key": data.NewString("value")},
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_CONCAT",
			})
			c.Assert(err, qt.IsNil)

			ir, ow, _, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *concatInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput *concatOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(*concatOutput)
				return nil
			})

			err = execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
			c.Assert(capturedOutput.Array, qt.DeepEquals, tc.expected)
		})
	}
}

func TestDifference(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    differenceInput
		expected []format.Value
	}{
		{
			name: "difference of two sets",
			input: differenceInput{
				SetA: []format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c")},
				SetB: []format.Value{data.NewString("b"), data.NewString("c")},
			},
			expected: []format.Value{data.NewString("a")},
		},
		{
			name: "empty difference",
			input: differenceInput{
				SetA: []format.Value{data.NewString("a"), data.NewString("b")},
				SetB: []format.Value{data.NewString("a"), data.NewString("b")},
			},
			expected: []format.Value{},
		},
		{
			name: "difference with numbers",
			input: differenceInput{
				SetA: []format.Value{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
				SetB: []format.Value{data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
			},
			expected: []format.Value{data.NewNumberFromInteger(1)},
		},
		{
			name: "difference with mixed types",
			input: differenceInput{
				SetA: []format.Value{data.NewString("a"), data.NewNumberFromInteger(1), data.NewString("b")},
				SetB: []format.Value{data.NewString("b"), data.NewNumberFromInteger(1)},
			},
			expected: []format.Value{data.NewString("a")},
		},
		{
			name: "difference with boolean values",
			input: differenceInput{
				SetA: []format.Value{data.NewBoolean(true), data.NewBoolean(false), data.NewString("a")},
				SetB: []format.Value{data.NewBoolean(true), data.NewString("b")},
			},
			expected: []format.Value{data.NewBoolean(false), data.NewString("a")},
		},
		{
			name: "difference with null values",
			input: differenceInput{
				SetA: []format.Value{data.NewString("a"), data.NewNull(), data.NewString("b")},
				SetB: []format.Value{data.NewNull(), data.NewString("b")},
			},
			expected: []format.Value{data.NewString("a")},
		},
		{
			name: "difference with nested structures",
			input: differenceInput{
				SetA: []format.Value{
					data.Array{data.NewString("a")},
					data.Map{"key": data.NewString("value")},
					data.NewString("c"),
				},
				SetB: []format.Value{
					data.Map{"key": data.NewString("value")},
					data.NewString("d"),
				},
			},
			expected: []format.Value{
				data.Array{data.NewString("a")},
				data.NewString("c"),
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_DIFFERENCE",
			})
			c.Assert(err, qt.IsNil)

			ir, ow, _, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *differenceInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput *differenceOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(*differenceOutput)
				return nil
			})

			err = execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
			c.Assert(capturedOutput.Set, qt.DeepEquals, tc.expected)
		})
	}
}

func TestSplit(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    splitInput
		expected [][]format.Value
	}{
		{
			name: "split array into groups",
			input: splitInput{
				Array:     []format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c"), data.NewString("d")},
				GroupSize: 2,
			},
			expected: [][]format.Value{
				{data.NewString("a"), data.NewString("b")},
				{data.NewString("c"), data.NewString("d")},
			},
		},
		{
			name: "split with uneven groups",
			input: splitInput{
				Array:     []format.Value{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
				GroupSize: 2,
			},
			expected: [][]format.Value{
				{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)},
				{data.NewNumberFromInteger(3)},
			},
		},
		{
			name: "split empty array",
			input: splitInput{
				Array:     []format.Value{},
				GroupSize: 2,
			},
			expected: [][]format.Value{},
		},
		{
			name: "split mixed type array",
			input: splitInput{
				Array:     []format.Value{data.NewString("a"), data.NewNumberFromInteger(1), data.NewString("b"), data.NewNumberFromInteger(2)},
				GroupSize: 2,
			},
			expected: [][]format.Value{
				{data.NewString("a"), data.NewNumberFromInteger(1)},
				{data.NewString("b"), data.NewNumberFromInteger(2)},
			},
		},
		{
			name: "split with group size 1",
			input: splitInput{
				Array:     []format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c")},
				GroupSize: 1,
			},
			expected: [][]format.Value{
				{data.NewString("a")},
				{data.NewString("b")},
				{data.NewString("c")},
			},
		},
		{
			name: "split with boolean values",
			input: splitInput{
				Array:     []format.Value{data.NewBoolean(true), data.NewBoolean(false), data.NewBoolean(true)},
				GroupSize: 2,
			},
			expected: [][]format.Value{
				{data.NewBoolean(true), data.NewBoolean(false)},
				{data.NewBoolean(true)},
			},
		},
		{
			name: "split with null values",
			input: splitInput{
				Array:     []format.Value{data.NewNull(), data.NewString("a"), data.NewNull(), data.NewString("b")},
				GroupSize: 2,
			},
			expected: [][]format.Value{
				{data.NewNull(), data.NewString("a")},
				{data.NewNull(), data.NewString("b")},
			},
		},
		{
			name: "split with nested structures",
			input: splitInput{
				Array: []format.Value{
					data.Array{data.NewString("a")},
					data.Map{"key": data.NewString("value")},
					data.NewString("c"),
				},
				GroupSize: 2,
			},
			expected: [][]format.Value{
				{
					data.Array{data.NewString("a")},
					data.Map{"key": data.NewString("value")},
				},
				{data.NewString("c")},
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_SPLIT",
			})
			c.Assert(err, qt.IsNil)

			ir, ow, _, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *splitInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput *splitOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(*splitOutput)
				return nil
			})

			err = execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
			c.Assert(capturedOutput.Arrays, qt.DeepEquals, tc.expected)
		})
	}
}

func TestUnion(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    unionInput
		expected []format.Value
	}{
		{
			name: "union of two sets",
			input: unionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewString("b")},
					{data.NewString("b"), data.NewString("c")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c")},
		},
		{
			name: "union of empty sets",
			input: unionInput{
				Sets: [][]format.Value{},
			},
			expected: []format.Value{},
		},
		{
			name: "union with duplicate values",
			input: unionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewString("a")},
					{data.NewString("a"), data.NewString("b")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewString("b")},
		},
		{
			name: "union of multiple sets",
			input: unionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewString("b")},
					{data.NewString("c"), data.NewString("d")},
					{data.NewString("e"), data.NewString("f")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c"),
				data.NewString("d"), data.NewString("e"), data.NewString("f")},
		},
		{
			name: "union with numbers",
			input: unionInput{
				Sets: [][]format.Value{
					{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2)},
					{data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
				},
			},
			expected: []format.Value{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
		},
		{
			name: "union with single set",
			input: unionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewString("b"), data.NewString("c")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c")},
		},
		{
			name: "union with mixed types",
			input: unionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewNumberFromInteger(1)},
					{data.NewNumberFromInteger(2), data.NewString("b")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewString("b")},
		},
		{
			name: "union with boolean values",
			input: unionInput{
				Sets: [][]format.Value{
					{data.NewBoolean(true), data.NewBoolean(false)},
					{data.NewBoolean(true), data.NewString("a")},
				},
			},
			expected: []format.Value{data.NewBoolean(true), data.NewBoolean(false), data.NewString("a")},
		},
		{
			name: "union with null values",
			input: unionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewNull()},
					{data.NewNull(), data.NewString("b")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewNull(), data.NewString("b")},
		},
		{
			name: "union with nested structures",
			input: unionInput{
				Sets: [][]format.Value{
					{
						data.Array{data.NewString("a")},
						data.Map{"key1": data.NewString("value1")},
					},
					{
						data.Map{"key2": data.NewString("value2")},
						data.NewString("b"),
					},
				},
			},
			expected: []format.Value{
				data.Array{data.NewString("a")},
				data.Map{"key1": data.NewString("value1")},
				data.Map{"key2": data.NewString("value2")},
				data.NewString("b"),
			},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_UNION",
			})
			c.Assert(err, qt.IsNil)

			ir, ow, _, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *unionInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput *unionOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(*unionOutput)
				return nil
			})

			err = execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
			c.Assert(capturedOutput.Set, qt.DeepEquals, tc.expected)
		})
	}
}

func TestIntersection(t *testing.T) {
	c := qt.New(t)

	testCases := []struct {
		name     string
		input    intersectionInput
		expected []format.Value
	}{
		{
			name: "intersection of two sets",
			input: intersectionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewString("b")},
					{data.NewString("b"), data.NewString("c")},
				},
			},
			expected: []format.Value{data.NewString("b")},
		},
		{
			name: "empty intersection",
			input: intersectionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewString("b")},
					{data.NewString("c"), data.NewString("d")},
				},
			},
			expected: []format.Value{},
		},
		{
			name: "intersection with empty sets",
			input: intersectionInput{
				Sets: [][]format.Value{},
			},
			expected: []format.Value{},
		},
		{
			name: "intersection of multiple sets",
			input: intersectionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewString("b"), data.NewString("c")},
					{data.NewString("b"), data.NewString("c"), data.NewString("d")},
					{data.NewString("c"), data.NewString("d"), data.NewString("e")},
				},
			},
			expected: []format.Value{data.NewString("c")},
		},
		{
			name: "intersection with numbers",
			input: intersectionInput{
				Sets: [][]format.Value{
					{data.NewNumberFromInteger(1), data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
					{data.NewNumberFromInteger(2), data.NewNumberFromInteger(3), data.NewNumberFromInteger(4)},
				},
			},
			expected: []format.Value{data.NewNumberFromInteger(2), data.NewNumberFromInteger(3)},
		},
		{
			name: "intersection with single set",
			input: intersectionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewString("b"), data.NewString("c")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewString("b"), data.NewString("c")},
		},
		{
			name: "intersection with duplicate values",
			input: intersectionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewString("a"), data.NewString("b")},
					{data.NewString("a"), data.NewString("b"), data.NewString("b")},
				},
			},
			expected: []format.Value{data.NewString("a"), data.NewString("b")},
		},
		{
			name: "intersection with mixed types",
			input: intersectionInput{
				Sets: [][]format.Value{
					{data.NewString("a"), data.NewNumberFromInteger(1), data.NewString("b")},
					{data.NewNumberFromInteger(1), data.NewString("b"), data.NewString("c")},
				},
			},
			expected: []format.Value{data.NewNumberFromInteger(1), data.NewString("b")},
		},
	}

	for _, tc := range testCases {
		c.Run(tc.name, func(c *qt.C) {
			component := Init(base.Component{})
			c.Assert(component, qt.IsNotNil)

			execution, err := component.CreateExecution(base.ComponentExecution{
				Component: component,
				Task:      "TASK_INTERSECTION",
			})
			c.Assert(err, qt.IsNil)

			ir, ow, _, job := mock.GenerateMockJob(c)
			ir.ReadDataMock.Set(func(ctx context.Context, input any) error {
				switch input := input.(type) {
				case *intersectionInput:
					*input = tc.input
				}
				return nil
			})

			var capturedOutput *intersectionOutput
			ow.WriteDataMock.Set(func(ctx context.Context, output any) error {
				capturedOutput = output.(*intersectionOutput)
				return nil
			})

			err = execution.Execute(context.Background(), []*base.Job{job})
			c.Assert(err, qt.IsNil)
			c.Assert(capturedOutput.Set, qt.DeepEquals, tc.expected)
		})
	}
}
