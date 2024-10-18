package data

import (
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestUnmarshal(t *testing.T) {
	c := qt.New(t)

	c.Run("Basic types", func(c *qt.C) {
		type TestStruct struct {
			StringField  *String  `key:"string-field"`
			NumberField  *Number  `key:"number-field"`
			BooleanField *Boolean `key:"boolean-field"`
		}

		input := Map{
			"string-field":  NewString("test"),
			"number-field":  NewNumberFromFloat(42.5),
			"boolean-field": NewBoolean(true),
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.StringField.GetString(), qt.Equals, "test")
		c.Assert(result.NumberField.GetFloat(), qt.Equals, 42.5)
		c.Assert(result.BooleanField.GetBoolean(), qt.Equals, true)
	})

	c.Run("Nested struct", func(c *qt.C) {
		type NestedStruct struct {
			NestedField *String `key:"nested-field"`
		}

		type TestStruct struct {
			TopField     *String      `key:"top-field"`
			NestedStruct NestedStruct `key:"nested-struct"`
		}

		input := Map{
			"top-field": NewString("top"),
			"nested-struct": Map{
				"nested-field": NewString("nested"),
			},
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.TopField.GetString(), qt.Equals, "top")
		c.Assert(result.NestedStruct.NestedField.GetString(), qt.Equals, "nested")
	})

	c.Run("Slice", func(c *qt.C) {
		type TestStruct struct {
			SliceField []*String `key:"slice-field"`
		}

		input := Map{
			"slice-field": Array{NewString("one"), NewString("two"), NewString("three")},
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(len(result.SliceField), qt.Equals, 3)
		c.Assert(result.SliceField[0].GetString(), qt.Equals, "one")
		c.Assert(result.SliceField[1].GetString(), qt.Equals, "two")
		c.Assert(result.SliceField[2].GetString(), qt.Equals, "three")
	})

	c.Run("Map", func(c *qt.C) {
		type TestStruct struct {
			MapField map[string]*String `key:"map-field"`
		}

		input := Map{
			"map-field": Map{
				"key1": NewString("value1"),
				"key2": NewString("value2"),
			},
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(len(result.MapField), qt.Equals, 2)
		c.Assert(result.MapField["key1"].GetString(), qt.Equals, "value1")
		c.Assert(result.MapField["key2"].GetString(), qt.Equals, "value2")
	})
}

func TestMarshal(t *testing.T) {
	c := qt.New(t)

	c.Run("Basic types", func(c *qt.C) {
		type TestStruct struct {
			StringField  *String  `key:"string-field"`
			NumberField  *Number  `key:"number-field"`
			BooleanField *Boolean `key:"boolean-field"`
		}

		input := TestStruct{
			StringField:  NewString("test"),
			NumberField:  NewNumberFromFloat(42.5),
			BooleanField: NewBoolean(true),
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(m["string-field"].(*String).GetString(), qt.Equals, "test")
		c.Assert(m["number-field"].(*Number).GetFloat(), qt.Equals, 42.5)
		c.Assert(m["boolean-field"].(*Boolean).GetBoolean(), qt.Equals, true)
	})

	c.Run("Nested struct", func(c *qt.C) {
		type NestedStruct struct {
			NestedField *String `key:"nested-field"`
		}

		type TestStruct struct {
			TopField     *String      `key:"top-field"`
			NestedStruct NestedStruct `key:"nested-struct"`
		}

		input := TestStruct{
			TopField: NewString("top"),
			NestedStruct: NestedStruct{
				NestedField: NewString("nested"),
			},
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(m["top-field"].(*String).GetString(), qt.Equals, "top")
		nestedMap, ok := m["nested-struct"].(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(nestedMap["nested-field"].(*String).GetString(), qt.Equals, "nested")
	})

	c.Run("Slice", func(c *qt.C) {
		type TestStruct struct {
			SliceField []*String `key:"slice-field"`
		}

		input := TestStruct{
			SliceField: []*String{NewString("one"), NewString("two"), NewString("three")},
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		sliceValue, ok := m["slice-field"].(Array)
		c.Assert(ok, qt.IsTrue)
		c.Assert(len(sliceValue), qt.Equals, 3)
		c.Assert(sliceValue[0].(*String).GetString(), qt.Equals, "one")
		c.Assert(sliceValue[1].(*String).GetString(), qt.Equals, "two")
		c.Assert(sliceValue[2].(*String).GetString(), qt.Equals, "three")
	})

	c.Run("Map", func(c *qt.C) {
		type TestStruct struct {
			MapField map[string]*String `key:"map-field"`
		}

		input := TestStruct{
			MapField: map[string]*String{
				"key1": NewString("value1"),
				"key2": NewString("value2"),
			},
		}

		result, err := Marshal(input)

		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		mapValue, ok := m["map-field"].(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(len(mapValue), qt.Equals, 2)
		c.Assert(mapValue["key1"].(*String).GetString(), qt.Equals, "value1")
		c.Assert(mapValue["key2"].(*String).GetString(), qt.Equals, "value2")
	})
}
