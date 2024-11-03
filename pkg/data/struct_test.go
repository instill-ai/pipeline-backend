package data

import (
	"os"
	"testing"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	c.Run("Basic types", func(c *qt.C) {
		type TestStruct struct {
			StringField   format.String  `instill:"string-field"`
			NumberField   format.Number  `instill:"number-field"`
			BooleanField  format.Boolean `instill:"boolean-field"`
			FloatField    float64        `instill:"float-field"`
			FloatPtrField *float64       `instill:"float-ptr-field"`
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
			NestedField format.String `instill:"nested-field"`
		}

		type TestStruct struct {
			TopField     format.String `instill:"top-field"`
			NestedStruct NestedStruct  `instill:"nested-struct"`
			NestedPtr    *NestedStruct `instill:"nested-ptr"`
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
			ArrayField  Array           `instill:"array-field"`
			StringArray []format.String `instill:"string-array"`
			NumberArray []format.Number `instill:"number-array"`
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
			MapField  Map                      `instill:"map-field"`
			StringMap map[string]format.String `instill:"string-map"`
			ValueMap  map[string]format.Value  `instill:"value-map"`
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

	c.Run("Format tag", func(c *qt.C) {
		type TestStruct struct {
			Image format.Image `instill:"image,image/bmp"`
		}

		imageBytes, err := os.ReadFile("testdata/sample_640_426.jpeg")
		c.Assert(err, qt.IsNil)

		// Create a new Image from bytes and verify format tag handling
		// The input is JPEG but NewImageFromBytes will auto convert to PNG.
		img, err := NewImageFromBytes(imageBytes, "image/jpeg", "sample_640_426.jpeg")
		c.Assert(err, qt.IsNil)

		input := Map{
			"image": img,
		}

		// Unmarshal the input into the TestStruct, since we set the format to be image/jpeg, the result will be a JPEG image.
		var result TestStruct
		err = Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.Image.ContentType().String(), qt.Equals, "image/bmp")
		c.Assert(result.Image.Width().Integer(), qt.Equals, 640)
		c.Assert(result.Image.Height().Integer(), qt.Equals, 426)
	})

	c.Run("Null", func(c *qt.C) {
		type TestStruct struct {
			NullField *format.String `instill:"null-field"`
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

		c.Run("Invalid field type", func(c *qt.C) {
			type InvalidStruct struct {
				Field int `instill:"field"`
			}
			input := Map{
				"field": NewString("not a number"),
			}
			var result InvalidStruct
			err := Unmarshal(input, &result)
			c.Assert(err, qt.ErrorMatches, "error unmarshaling field field:.*")
		})

		c.Run("Invalid array element type", func(c *qt.C) {
			type ArrayStruct struct {
				Numbers []format.Number `instill:"numbers"`
			}
			input := Map{
				"numbers": Array{NewString("not a number")},
			}
			var result ArrayStruct
			err := Unmarshal(input, &result)
			c.Assert(err, qt.ErrorMatches, "error unmarshaling field numbers:.*")
		})

		c.Run("Invalid map value type", func(c *qt.C) {
			type MapStruct struct {
				Values map[string]format.Number `instill:"values"`
			}
			input := Map{
				"values": Map{
					"key": NewString("not a number"),
				},
			}
			var result MapStruct
			err := Unmarshal(input, &result)
			c.Assert(err, qt.ErrorMatches, "error unmarshaling field values:.*")
		})
	})

	c.Run("Empty input", func(c *qt.C) {
		type TestStruct struct {
			OptionalField format.String  `instill:"optional"`
			RequiredPtr   *format.String `instill:"required"`
		}

		input := Map{}
		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.RequiredPtr, qt.IsNil)
	})

	c.Run("Mixed types array", func(c *qt.C) {
		type TestStruct struct {
			MixedArray Array `instill:"mixed"`
		}

		input := Map{
			"mixed": Array{
				NewString("text"),
				NewNumberFromFloat(42),
				NewBoolean(true),
				NewNull(),
			},
		}

		var result TestStruct
		err := Unmarshal(input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(len(result.MixedArray), qt.Equals, 4)
		c.Assert(result.MixedArray[0].(format.String).String(), qt.Equals, "text")
		c.Assert(result.MixedArray[1].(format.Number).Float64(), qt.Equals, 42.0)
		c.Assert(result.MixedArray[2].(format.Boolean).Boolean(), qt.Equals, true)
		c.Assert(result.MixedArray[3], qt.Equals, NewNull())
	})

	c.Run("Undetermined type", func(c *qt.C) {
		type TestStruct struct {
			Value format.Value `instill:"value"`
		}

		// Test string value
		stringInput := Map{
			"value": NewString("test"),
		}
		var stringResult TestStruct
		err := Unmarshal(stringInput, &stringResult)
		c.Assert(err, qt.IsNil)
		c.Assert(stringResult.Value.(format.String).String(), qt.Equals, "test")

		// Test number value
		numberInput := Map{
			"value": NewNumberFromFloat(42.5),
		}
		var numberResult TestStruct
		err = Unmarshal(numberInput, &numberResult)
		c.Assert(err, qt.IsNil)
		c.Assert(numberResult.Value.(format.Number).Float64(), qt.Equals, 42.5)

		// Test boolean value
		boolInput := Map{
			"value": NewBoolean(true),
		}
		var boolResult TestStruct
		err = Unmarshal(boolInput, &boolResult)
		c.Assert(err, qt.IsNil)
		c.Assert(boolResult.Value.(format.Boolean).Boolean(), qt.Equals, true)

	})

	c.Run("Undetermined type", func(c *qt.C) {

		type TestStruct struct {
			Value format.Value `instill:"value"`
		}

		// Test string value
		stringInput := Map{
			"value": NewString("test"),
		}
		var stringResult TestStruct
		err := Unmarshal(stringInput, &stringResult)
		c.Assert(err, qt.IsNil)
		c.Assert(stringResult.Value.(format.String).String(), qt.Equals, "test")

		// Test number value
		numberInput := Map{
			"value": NewNumberFromFloat(42.5),
		}
		var numberResult TestStruct
		err = Unmarshal(numberInput, &numberResult)
		c.Assert(err, qt.IsNil)
		c.Assert(numberResult.Value.(format.Number).Float64(), qt.Equals, 42.5)

		// Test boolean value
		boolInput := Map{
			"value": NewBoolean(true),
		}
		var boolResult TestStruct
		err = Unmarshal(boolInput, &boolResult)
		c.Assert(err, qt.IsNil)
		c.Assert(boolResult.Value.(format.Boolean).Boolean(), qt.Equals, true)

	})
}

func TestMarshal(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	c.Run("Basic types", func(c *qt.C) {
		input := struct {
			StringField  format.String  `instill:"string-field"`
			NumberField  format.Number  `instill:"number-field"`
			BooleanField format.Boolean `instill:"boolean-field"`
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
			TopField     format.String `instill:"top-field"`
			NestedStruct struct {
				NestedField format.String `instill:"nested-field"`
			} `instill:"nested-struct"`
		}{
			TopField: NewString("top"),
			NestedStruct: struct {
				NestedField format.String `instill:"nested-field"`
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
			ArrayField Array `instill:"array-field"`
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
			MapField Map `instill:"map-field"`
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

	c.Run("Format tag", func(c *qt.C) {
		imageBytes, err := os.ReadFile("testdata/sample_640_426.jpeg")
		c.Assert(err, qt.IsNil)

		img, err := NewImageFromBytes(imageBytes, "image/jpeg", "sample_640_426.jpeg")
		c.Assert(err, qt.IsNil)

		input := struct {
			Image format.Image `instill:"image,image/jpeg"`
		}{
			Image: img,
		}

		result, err := Marshal(input)
		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		image, ok := m["image"].(format.Image)
		c.Assert(ok, qt.IsTrue)
		c.Assert(image.ContentType().String(), qt.Equals, "image/jpeg")
		c.Assert(image.Width().Integer(), qt.Equals, 640)
		c.Assert(image.Height().Integer(), qt.Equals, 426)
	})

	c.Run("Null", func(c *qt.C) {
		input := struct {
			NullField *format.String `instill:"null-field"`
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
			FloatPtr  *float64      `instill:"float-ptr"`
			StringPtr format.String `instill:"string-ptr"`
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
			NestedField format.String `instill:"nested-field"`
		}

		input := struct {
			Text    format.String           `instill:"text"`
			Numbers []format.Number         `instill:"numbers"`
			Object  NestedStruct            `instill:"object"`
			TextMap map[string]format.Value `instill:"text-map"`
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

	c.Run("Undetermined type", func(c *qt.C) {
		type TestStruct struct {
			Value format.Value `instill:"value"`
		}

		// Test string value
		stringInput := TestStruct{
			Value: NewString("test"),
		}
		stringResult, err := Marshal(stringInput)
		c.Assert(err, qt.IsNil)
		stringMap, ok := stringResult.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(stringMap["value"].(format.String).String(), qt.Equals, "test")

		// Test number value
		numberInput := TestStruct{
			Value: NewNumberFromFloat(42.5),
		}
		numberResult, err := Marshal(numberInput)
		c.Assert(err, qt.IsNil)
		numberMap, ok := numberResult.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(numberMap["value"].(format.Number).Float64(), qt.Equals, 42.5)

		// Test boolean value
		boolInput := TestStruct{
			Value: NewBoolean(true),
		}
		boolResult, err := Marshal(boolInput)
		c.Assert(err, qt.IsNil)
		boolMap, ok := boolResult.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(boolMap["value"].(format.Boolean).Boolean(), qt.Equals, true)
	})

	c.Run("Error cases", func(c *qt.C) {
		c.Run("Invalid field type", func(c *qt.C) {
			input := struct {
				InvalidField chan int `instill:"invalid"`
			}{
				InvalidField: make(chan int),
			}
			_, err := Marshal(input)
			c.Assert(err, qt.ErrorMatches, "error marshaling field invalid: unsupported type: chan")
		})

		c.Run("Nil interface", func(c *qt.C) {
			var input interface{}
			_, err := Marshal(input)
			c.Assert(err, qt.ErrorMatches, "input must not be nil")
		})

		c.Run("Invalid map key type", func(c *qt.C) {
			input := struct {
				InvalidMap map[int]string `instill:"invalid-map"`
			}{
				InvalidMap: map[int]string{1: "value"},
			}
			_, err := Marshal(input)
			c.Assert(err, qt.ErrorMatches, "error marshaling field invalid-map: map key must be string type")
		})
	})

	c.Run("Empty struct", func(c *qt.C) {
		input := struct{}{}
		result, err := Marshal(input)
		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(len(m), qt.Equals, 0)
	})

}
