package data

import (
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestUnmarshal(t *testing.T) {
	c := qt.New(t)

	c.Run("Basic types", func(c *qt.C) {
		type TestStruct struct {
			StringField   format.String  `key:"string-field"`
			NumberField   format.Number  `key:"number-field"`
			BooleanField  format.Boolean `key:"boolean-field"`
			FloatField    float64        `key:"float-field"`
			FloatPtrField *float64       `key:"float-ptr-field"`
		}

		floatVal := 42.5
		input := Map{
			"string-field":    NewString("test"),
			"number-field":    NewNumberFromFloat(42.5),
			"boolean-field":   NewBoolean(true),
			"float-field":     NewNumberFromFloat(123.456),
			"float-ptr-field": NewNumberFromFloat(42.5),
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.StringField.String(), qt.Equals, "test")
		c.Assert(result.NumberField.Float64(), qt.Equals, 42.5)
		c.Assert(result.BooleanField.Boolean(), qt.Equals, true)
		c.Assert(result.FloatField, qt.Equals, 123.456)
		c.Assert(result.FloatPtrField, qt.DeepEquals, &floatVal)
	})

	c.Run("Nested struct", func(c *qt.C) {
		type NestedStruct struct {
			NestedField format.String `key:"nested-field"`
		}

		type TestStruct struct {
			TopField     format.String `key:"top-field"`
			NestedStruct NestedStruct  `key:"nested-struct"`
			NestedPtr    *NestedStruct `key:"nested-ptr"`
		}

		input := Map{
			"top-field": NewString("top"),
			"nested-struct": Map{
				"nested-field": NewString("nested"),
			},
			"nested-ptr": Map{
				"nested-field": NewString("nested-ptr"),
			},
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.TopField.String(), qt.Equals, "top")
		c.Assert(result.NestedStruct.NestedField.String(), qt.Equals, "nested")
		c.Assert(result.NestedPtr.NestedField.String(), qt.Equals, "nested-ptr")
	})

	c.Run("Array", func(c *qt.C) {
		type TestStruct struct {
			ArrayField  Array           `key:"array-field"`
			StringArray []format.String `key:"string-array"`
			NumberArray []format.Number `key:"number-array"`
		}

		input := Map{
			"array-field":  Array{NewString("one"), NewString("two"), NewString("three")},
			"string-array": Array{NewString("a"), NewString("b"), NewString("c")},
			"number-array": Array{NewNumberFromFloat(1), NewNumberFromFloat(2), NewNumberFromFloat(3)},
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(len(result.ArrayField), qt.Equals, 3)
		c.Assert(result.ArrayField[0].(format.String).String(), qt.Equals, "one")
		c.Assert(result.ArrayField[1].(format.String).String(), qt.Equals, "two")
		c.Assert(result.ArrayField[2].(format.String).String(), qt.Equals, "three")

		c.Assert(len(result.StringArray), qt.Equals, 3)
		c.Assert(result.StringArray[0].String(), qt.Equals, "a")
		c.Assert(result.StringArray[1].String(), qt.Equals, "b")
		c.Assert(result.StringArray[2].String(), qt.Equals, "c")

		c.Assert(len(result.NumberArray), qt.Equals, 3)
		c.Assert(result.NumberArray[0].Float64(), qt.Equals, 1.0)
		c.Assert(result.NumberArray[1].Float64(), qt.Equals, 2.0)
		c.Assert(result.NumberArray[2].Float64(), qt.Equals, 3.0)
	})

	c.Run("Map", func(c *qt.C) {
		type TestStruct struct {
			MapField  Map                      `key:"map-field"`
			StringMap map[string]format.String `key:"string-map"`
			ValueMap  map[string]format.Value  `key:"value-map"`
		}

		input := Map{
			"map-field": Map{
				"key1": NewString("value1"),
				"key2": NewString("value2"),
			},
			"string-map": Map{
				"a": NewString("A"),
				"b": NewString("B"),
			},
			"value-map": Map{
				"str":  NewString("string"),
				"num":  NewNumberFromFloat(42),
				"bool": NewBoolean(true),
			},
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(len(result.MapField), qt.Equals, 2)
		c.Assert(result.MapField["key1"].(format.String).String(), qt.Equals, "value1")
		c.Assert(result.MapField["key2"].(format.String).String(), qt.Equals, "value2")

		c.Assert(len(result.StringMap), qt.Equals, 2)
		c.Assert(result.StringMap["a"].String(), qt.Equals, "A")
		c.Assert(result.StringMap["b"].String(), qt.Equals, "B")

		c.Assert(len(result.ValueMap), qt.Equals, 3)
		c.Assert(result.ValueMap["str"].(format.String).String(), qt.Equals, "string")
		c.Assert(result.ValueMap["num"].(format.Number).Float64(), qt.Equals, 42.0)
		c.Assert(result.ValueMap["bool"].(format.Boolean).Boolean(), qt.Equals, true)
	})

	c.Run("Null", func(c *qt.C) {
		type TestStruct struct {
			NullField *format.String `key:"null-field"`
		}

		input := Map{
			"null-field": NewNull(),
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.NullField, qt.IsNil)
	})

	c.Run("Error cases", func(c *qt.C) {
		c.Run("Non-pointer input", func(c *qt.C) {
			var s struct{}
			err := Unmarshal(Map{}, s)
			c.Assert(err, qt.ErrorMatches, "input must be a pointer to a struct")
		})

		c.Run("Non-struct input", func(c *qt.C) {
			var i int
			err := Unmarshal(Map{}, &i)
			c.Assert(err, qt.ErrorMatches, "input must be a pointer to a struct")
		})

		c.Run("Non-Map input", func(c *qt.C) {
			var s struct{}
			err := Unmarshal(NewString("not a map"), &s)
			c.Assert(err, qt.ErrorMatches, "input value must be a Map")
		})
	})
}

func TestMarshal(t *testing.T) {
	c := qt.New(t)

	c.Run("Basic types", func(c *qt.C) {
		input := struct {
			StringField  format.String  `key:"string-field"`
			NumberField  format.Number  `key:"number-field"`
			BooleanField format.Boolean `key:"boolean-field"`
		}{
			StringField:  NewString("test"),
			NumberField:  NewNumberFromFloat(42.5),
			BooleanField: NewBoolean(true),
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(m["string-field"].(format.String).String(), qt.Equals, "test")
		c.Assert(m["number-field"].(format.Number).Float64(), qt.Equals, 42.5)
		c.Assert(m["boolean-field"].(format.Boolean).Boolean(), qt.Equals, true)
	})

	c.Run("Nested struct", func(c *qt.C) {
		input := struct {
			TopField     format.String `key:"top-field"`
			NestedStruct struct {
				NestedField format.String `key:"nested-field"`
			} `key:"nested-struct"`
		}{
			TopField: NewString("top"),
			NestedStruct: struct {
				NestedField format.String `key:"nested-field"`
			}{
				NestedField: NewString("nested"),
			},
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(m["top-field"].(format.String).String(), qt.Equals, "top")
		nestedMap, ok := m["nested-struct"].(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(nestedMap["nested-field"].(format.String).String(), qt.Equals, "nested")
	})

	c.Run("Array", func(c *qt.C) {
		input := struct {
			ArrayField Array `key:"array-field"`
		}{
			ArrayField: Array{NewString("one"), NewString("two"), NewString("three")},
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		arr, ok := m["array-field"].(Array)
		c.Assert(ok, qt.IsTrue)
		c.Assert(len(arr), qt.Equals, 3)
		c.Assert(arr[0].(format.String).String(), qt.Equals, "one")
		c.Assert(arr[1].(format.String).String(), qt.Equals, "two")
		c.Assert(arr[2].(format.String).String(), qt.Equals, "three")
	})

	c.Run("Map", func(c *qt.C) {
		input := struct {
			MapField Map `key:"map-field"`
		}{
			MapField: Map{
				"key1": NewString("value1"),
				"key2": NewString("value2"),
			},
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		mapField, ok := m["map-field"].(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(len(mapField), qt.Equals, 2)
		c.Assert(mapField["key1"].(format.String).String(), qt.Equals, "value1")
		c.Assert(mapField["key2"].(format.String).String(), qt.Equals, "value2")
	})

	c.Run("Null", func(c *qt.C) {
		input := struct {
			NullField *format.String `key:"null-field"`
		}{
			NullField: nil,
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(m["null-field"], qt.Equals, NewNull())
	})

	c.Run("Pointer fields", func(c *qt.C) {
		floatVal := 42.5
		input := struct {
			FloatPtr  *float64      `key:"float-ptr"`
			StringPtr format.String `key:"string-ptr"`
		}{
			FloatPtr:  &floatVal,
			StringPtr: NewString("pointer string"),
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(m["float-ptr"].(format.Number).Float64(), qt.Equals, 42.5)
		c.Assert(m["string-ptr"].(format.String).String(), qt.Equals, "pointer string")
	})

	c.Run("Complex nested structure", func(c *qt.C) {
		type NestedStruct struct {
			NestedField format.String `key:"nested-field"`
		}

		input := struct {
			Text    format.String           `key:"text"`
			Numbers []format.Number         `key:"numbers"`
			Object  NestedStruct            `key:"object"`
			TextMap map[string]format.Value `key:"text-map"`
		}{
			Text:    NewString("example text"),
			Numbers: []format.Number{NewNumberFromFloat(1), NewNumberFromFloat(2), NewNumberFromFloat(3)},
			Object: NestedStruct{
				NestedField: NewString("nested text"),
			},
			TextMap: map[string]format.Value{
				"key1": NewString("value1"),
				"key2": NewNumberFromFloat(42),
			},
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(m["text"].(format.String).String(), qt.Equals, "example text")

		numbers, ok := m["numbers"].(Array)
		c.Assert(ok, qt.IsTrue)
		c.Assert(len(numbers), qt.Equals, 3)
		c.Assert(numbers[0].(format.Number).Float64(), qt.Equals, 1.0)
		c.Assert(numbers[1].(format.Number).Float64(), qt.Equals, 2.0)
		c.Assert(numbers[2].(format.Number).Float64(), qt.Equals, 3.0)

		object, ok := m["object"].(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(object["nested-field"].(format.String).String(), qt.Equals, "nested text")

		textMap, ok := m["text-map"].(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(textMap["key1"].(format.String).String(), qt.Equals, "value1")
		c.Assert(textMap["key2"].(format.Number).Float64(), qt.Equals, 42.0)
	})
}
