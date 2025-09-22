package data

import (
	"context"
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

func TestUnmarshal(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	binaryFetcher := external.NewBinaryFetcher()
	unmarshaler := NewUnmarshaler(binaryFetcher)

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
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

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
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

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
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

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
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

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

		imageBytes, err := os.ReadFile("testdata/small_sample.jpeg")
		c.Assert(err, qt.IsNil)

		// Create a new Image from bytes and verify format tag handling
		// The input is JPEG but NewImageFromBytes will auto convert to PNG.
		img, err := NewImageFromBytes(imageBytes, "image/jpeg", "small_sample.jpeg", true)
		c.Assert(err, qt.IsNil)

		input := Map{
			"image": img,
		}

		// Unmarshal the input into the TestStruct, since we set the format to be image/jpeg, the result will be a JPEG image.
		var result TestStruct
		err = unmarshaler.Unmarshal(context.Background(), input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.Image.ContentType().String(), qt.Equals, "image/bmp")
		c.Assert(result.Image.Width().Integer(), qt.Equals, 320)
		c.Assert(result.Image.Height().Integer(), qt.Equals, 240)
	})

	c.Run("Null", func(c *qt.C) {
		type TestStruct struct {
			NullField *format.String `instill:"null-field"`
		}

		input := Map{
			"null-field": NewNull(),
		}

		var result TestStruct
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.NullField, qt.IsNil)
	})

	c.Run("Default value", func(c *qt.C) {
		type TestStruct struct {
			StringField   format.String  `instill:"string-field,default=hello"`
			NumberField   format.Number  `instill:"number-field,default=42"`
			BooleanField  format.Boolean `instill:"boolean-field,default=true"`
			IntField      int            `instill:"int-field,default=123"`
			UintField     uint           `instill:"uint-field,default=456"`
			FloatField    float64        `instill:"float-field,default=3.14"`
			BoolField     bool           `instill:"bool-field,default=true"`
			StrField      string         `instill:"string-field,default=world"`
			IntPtrField   *int           `instill:"int-ptr-field,default=123"`
			UintPtrField  *uint          `instill:"uint-ptr-field,default=456"`
			FloatPtrField *float64       `instill:"float-ptr-field,default=3.14"`
			BoolPtrField  *bool          `instill:"bool-ptr-field,default=true"`
			StrPtrField   *string        `instill:"string-ptr-field,default=world"`
		}

		input := Map{}
		var result TestStruct
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

		c.Assert(err, qt.IsNil)

		// Test format.Value types
		c.Assert(result.StringField.String(), qt.Equals, "hello")
		c.Assert(result.NumberField.Float64(), qt.Equals, 42.0)
		c.Assert(result.BooleanField.Boolean(), qt.Equals, true)

		// Test primitive types
		c.Assert(result.IntField, qt.Equals, 123)
		c.Assert(result.UintField, qt.Equals, uint(456))
		c.Assert(result.FloatField, qt.Equals, 3.14)
		c.Assert(result.BoolField, qt.Equals, true)
		c.Assert(result.StrField, qt.Equals, "world")

		// Test pointer primitive types
		c.Assert(*result.IntPtrField, qt.Equals, 123)
		c.Assert(*result.UintPtrField, qt.Equals, uint(456))
		c.Assert(*result.FloatPtrField, qt.Equals, 3.14)
		c.Assert(*result.BoolPtrField, qt.Equals, true)
		c.Assert(*result.StrPtrField, qt.Equals, "world")

		// Test invalid default values
		type InvalidStruct struct {
			BadInt *int `instill:"bad-int,default=not-a-number"`
		}
		var invalid InvalidStruct
		err = unmarshaler.Unmarshal(context.Background(), Map{}, &invalid)
		c.Assert(err, qt.ErrorMatches, "error setting default value for field bad-int:.*")
	})

	c.Run("Without default value", func(c *qt.C) {
		type TestStruct struct {
			StringField   format.String  `instill:"string-field"`
			NumberField   format.Number  `instill:"number-field"`
			BooleanField  format.Boolean `instill:"boolean-field"`
			IntField      int            `instill:"int-field"`
			UintField     uint           `instill:"uint-field"`
			FloatField    float64        `instill:"float-field"`
			BoolField     bool           `instill:"bool-field"`
			StrField      string         `instill:"str-field"`
			IntPtrField   *int           `instill:"int-ptr-field"`
			UintPtrField  *uint          `instill:"uint-ptr-field"`
			FloatPtrField *float64       `instill:"float-ptr-field"`
			BoolPtrField  *bool          `instill:"bool-ptr-field"`
			StrPtrField   *string        `instill:"str-ptr-field"`
		}

		input := Map{}
		var result TestStruct
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

		c.Assert(err, qt.IsNil)

		// Test format.Value types have zero values
		c.Assert(result.StringField, qt.IsNil)
		c.Assert(result.NumberField, qt.IsNil)
		c.Assert(result.BooleanField, qt.IsNil)

		// Test primitive types have zero values
		c.Assert(result.IntField, qt.Equals, 0)
		c.Assert(result.UintField, qt.Equals, uint(0))
		c.Assert(result.FloatField, qt.Equals, 0.0)
		c.Assert(result.BoolField, qt.Equals, false)
		c.Assert(result.StrField, qt.Equals, "")

		// Test pointer primitive types are nil
		c.Assert(result.IntPtrField, qt.IsNil)
		c.Assert(result.UintPtrField, qt.IsNil)
		c.Assert(result.FloatPtrField, qt.IsNil)
		c.Assert(result.BoolPtrField, qt.IsNil)
		c.Assert(result.StrPtrField, qt.IsNil)
	})

	c.Run("Error cases", func(c *qt.C) {
		c.Run("Non-pointer input", func(c *qt.C) {
			var s struct{}
			err := unmarshaler.Unmarshal(context.Background(), Map{}, s)
			c.Assert(err, qt.ErrorMatches, "input must be a pointer")
		})

		c.Run("Non-struct input", func(c *qt.C) {
			var i int
			err := unmarshaler.Unmarshal(context.Background(), Map{}, &i)
			c.Assert(err, qt.ErrorMatches, "input must be a pointer to a struct, got pointer to int")
		})

		c.Run("Non-Map input", func(c *qt.C) {
			var s struct{}
			err := unmarshaler.Unmarshal(context.Background(), NewString("not a map"), &s)
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
			err := unmarshaler.Unmarshal(context.Background(), input, &result)
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
			err := unmarshaler.Unmarshal(context.Background(), input, &result)
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
			err := unmarshaler.Unmarshal(context.Background(), input, &result)
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
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

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
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

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
		err := unmarshaler.Unmarshal(context.Background(), stringInput, &stringResult)
		c.Assert(err, qt.IsNil)
		c.Assert(stringResult.Value.(format.String).String(), qt.Equals, "test")

		// Test number value
		numberInput := Map{
			"value": NewNumberFromFloat(42.5),
		}
		var numberResult TestStruct
		err = unmarshaler.Unmarshal(context.Background(), numberInput, &numberResult)
		c.Assert(err, qt.IsNil)
		c.Assert(numberResult.Value.(format.Number).Float64(), qt.Equals, 42.5)

		// Test boolean value
		boolInput := Map{
			"value": NewBoolean(true),
		}
		var boolResult TestStruct
		err = unmarshaler.Unmarshal(context.Background(), boolInput, &boolResult)
		c.Assert(err, qt.IsNil)
		c.Assert(boolResult.Value.(format.Boolean).Boolean(), qt.Equals, true)
	})

	c.Run("Compositional struct", func(c *qt.C) {
		type BaseStruct struct {
			BaseField format.String `instill:"base-field"`
		}

		type ComposedStruct struct {
			BaseStruct
			ExtraField format.Number `instill:"extra-field"`
		}

		input := Map{
			"base-field":  NewString("base value"),
			"extra-field": NewNumberFromFloat(123),
		}

		var result ComposedStruct
		err := unmarshaler.Unmarshal(context.Background(), input, &result)

		c.Assert(err, qt.IsNil)
		c.Assert(result.BaseField.String(), qt.Equals, "base value")
		c.Assert(result.ExtraField.Float64(), qt.Equals, 123.0)
	})

	// Test camelCase field name extraction
	c.Run("CamelCase field name extraction", func(c *qt.C) {
		// Test struct that mimics external package types with json tags
		type ExternalType struct {
			MIMEType    string `json:"mimeType"`
			FileURI     string `json:"fileUri"`
			DisplayName string `json:"displayName"`
			WithInstill string `instill:"custom-field-name" json:"jsonFieldName"`
			PlainField  string
		}

		inputData := Map{
			"mime-type":         NewString("application/pdf"),
			"file-uri":          NewString("gs://bucket/file.pdf"),
			"display-name":      NewString("document.pdf"),
			"custom-field-name": NewString("instill-value"), // Should match instill tag
			"PlainField":        NewString("plain-value"),   // No tag, uses field name
		}

		// Test with automatic detection (now default behavior)
		unmarshalerWithCamelCase := NewUnmarshaler(binaryFetcher)
		var resultWithCamelCase ExternalType
		err := unmarshalerWithCamelCase.Unmarshal(context.Background(), inputData, &resultWithCamelCase)
		c.Assert(err, qt.IsNil)

		// Verify kebab-case → camelCase mapping worked
		c.Check(resultWithCamelCase.MIMEType, qt.Equals, "application/pdf")
		c.Check(resultWithCamelCase.FileURI, qt.Equals, "gs://bucket/file.pdf")
		c.Check(resultWithCamelCase.DisplayName, qt.Equals, "document.pdf")
		// Verify instill tag takes precedence over json tag
		c.Check(resultWithCamelCase.WithInstill, qt.Equals, "instill-value")
		c.Check(resultWithCamelCase.PlainField, qt.Equals, "plain-value")

		// Test with automatic detection (now default behavior)
		unmarshalerDefault := NewUnmarshaler(binaryFetcher)
		var resultDefault ExternalType
		err = unmarshalerDefault.Unmarshal(context.Background(), inputData, &resultDefault)
		c.Assert(err, qt.IsNil)

		// With automatic detection, camelCase json tags are now automatically converted
		c.Check(resultDefault.MIMEType, qt.Equals, "application/pdf")     // Automatic kebab-case → camelCase mapping
		c.Check(resultDefault.FileURI, qt.Equals, "gs://bucket/file.pdf") // Automatic kebab-case → camelCase mapping
		c.Check(resultDefault.DisplayName, qt.Equals, "document.pdf")     // Automatic kebab-case → camelCase mapping
		c.Check(resultDefault.WithInstill, qt.Equals, "instill-value")    // instill tag still takes precedence
		c.Check(resultDefault.PlainField, qt.Equals, "plain-value")       // Field name match
	})

	c.Run("CamelCase with complex nested structures", func(c *qt.C) {
		type NestedType struct {
			InnerField string `json:"innerField"`
			DataCount  int    `json:"dataCount"`
		}

		type ComplexType struct {
			TopLevel    string                `json:"topLevel"`
			NestedArray []NestedType          `json:"nestedArray"`
			ConfigMap   map[string]NestedType `json:"configMap"`
		}

		inputData := Map{
			"top-level": NewString("top-value"),
			"nested-array": Array{
				Map{
					"inner-field": NewString("nested1"),
					"data-count":  NewNumberFromInteger(10),
				},
				Map{
					"inner-field": NewString("nested2"),
					"data-count":  NewNumberFromInteger(20),
				},
			},
			"config-map": Map{
				"key1": Map{
					"inner-field": NewString("config1"),
					"data-count":  NewNumberFromInteger(100),
				},
			},
		}

		unmarshaler := NewUnmarshaler(binaryFetcher)
		var result ComplexType
		err := unmarshaler.Unmarshal(context.Background(), inputData, &result)
		c.Assert(err, qt.IsNil)

		// Verify top-level camelCase conversion
		c.Check(result.TopLevel, qt.Equals, "top-value")

		// Verify nested array camelCase conversion
		c.Assert(result.NestedArray, qt.HasLen, 2)
		c.Check(result.NestedArray[0].InnerField, qt.Equals, "nested1")
		c.Check(result.NestedArray[0].DataCount, qt.Equals, 10)
		c.Check(result.NestedArray[1].InnerField, qt.Equals, "nested2")
		c.Check(result.NestedArray[1].DataCount, qt.Equals, 20)

		// Verify nested map camelCase conversion
		c.Assert(result.ConfigMap, qt.HasLen, 1)
		c.Check(result.ConfigMap["key1"].InnerField, qt.Equals, "config1")
		c.Check(result.ConfigMap["key1"].DataCount, qt.Equals, 100)
	})

	c.Run("SnakeCase field name extraction", func(c *qt.C) {
		// Test struct that mimics external package types with snake_case json tags
		type SnakeType struct {
			MIMEType    string `json:"mime_type"`
			FileURI     string `json:"file_uri"`
			DisplayName string `json:"display_name"`
			WithInstill string `instill:"custom-field-name" json:"json_field_name"`
			PlainField  string
		}

		inputData := Map{
			"mime-type":         NewString("application/json"),
			"file-uri":          NewString("gs://bucket/data.json"),
			"display-name":      NewString("data.json"),
			"custom-field-name": NewString("instill-value"), // Should match instill tag
			"PlainField":        NewString("plain-value"),   // No tag, uses field name
		}

		// Test with automatic detection (now default behavior)
		unmarshalerWithSnakeCase := NewUnmarshaler(binaryFetcher)
		var resultWithSnakeCase SnakeType
		err := unmarshalerWithSnakeCase.Unmarshal(context.Background(), inputData, &resultWithSnakeCase)
		c.Assert(err, qt.IsNil)

		// Verify kebab-case → snake_case mapping worked
		c.Check(resultWithSnakeCase.MIMEType, qt.Equals, "application/json")
		c.Check(resultWithSnakeCase.FileURI, qt.Equals, "gs://bucket/data.json")
		c.Check(resultWithSnakeCase.DisplayName, qt.Equals, "data.json")
		// Verify instill tag takes precedence over json tag
		c.Check(resultWithSnakeCase.WithInstill, qt.Equals, "instill-value")
		c.Check(resultWithSnakeCase.PlainField, qt.Equals, "plain-value")
	})

	c.Run("PascalCase field name extraction", func(c *qt.C) {
		// Test struct that mimics external package types with PascalCase json tags
		type PascalType struct {
			MIMEType    string `json:"MimeType"`
			FileURI     string `json:"FileUri"`
			DisplayName string `json:"DisplayName"`
			WithInstill string `instill:"custom-field-name" json:"JsonFieldName"`
			PlainField  string
		}

		inputData := Map{
			"mime-type":         NewString("text/plain"),
			"file-uri":          NewString("gs://bucket/file.txt"),
			"display-name":      NewString("file.txt"),
			"custom-field-name": NewString("instill-value"), // Should match instill tag
			"PlainField":        NewString("plain-value"),   // No tag, uses field name
		}

		// Test with automatic detection (now default behavior)
		unmarshalerWithPascalCase := NewUnmarshaler(binaryFetcher)
		var resultWithPascalCase PascalType
		err := unmarshalerWithPascalCase.Unmarshal(context.Background(), inputData, &resultWithPascalCase)
		c.Assert(err, qt.IsNil)

		// Verify kebab-case → PascalCase mapping worked
		c.Check(resultWithPascalCase.MIMEType, qt.Equals, "text/plain")
		c.Check(resultWithPascalCase.FileURI, qt.Equals, "gs://bucket/file.txt")
		c.Check(resultWithPascalCase.DisplayName, qt.Equals, "file.txt")
		// Verify instill tag takes precedence over json tag
		c.Check(resultWithPascalCase.WithInstill, qt.Equals, "instill-value")
		c.Check(resultWithPascalCase.PlainField, qt.Equals, "plain-value")
	})

	c.Run("Pattern validation", func(c *qt.C) {
		c.Run("String field pattern validation", func(c *qt.C) {
			type PatternStruct struct {
				Username string `instill:"username,pattern=^[a-zA-Z0-9]+$"`
				Code     string `instill:"code,pattern=^[0-9]{4}$"`
				Email    string `instill:"email,pattern=^[^@]+@[^@]+\\.[^@]+$"`
				Name     string `instill:"name"` // No pattern
			}

			// Test valid patterns
			validInput := Map{
				"username": NewString("user123"),
				"code":     NewString("1234"),
				"email":    NewString("test@example.com"),
				"name":     NewString("Any string @#$%^&*() should work!"),
			}

			var validResult PatternStruct
			err := unmarshaler.Unmarshal(context.Background(), validInput, &validResult)
			c.Assert(err, qt.IsNil)
			c.Check(validResult.Username, qt.Equals, "user123")
			c.Check(validResult.Code, qt.Equals, "1234")
			c.Check(validResult.Email, qt.Equals, "test@example.com")
			c.Check(validResult.Name, qt.Equals, "Any string @#$%^&*() should work!")

			// Test invalid username pattern
			invalidUsernameInput := Map{
				"username": NewString("user-123"), // Dash not allowed
				"name":     NewString("test"),
			}
			var invalidUsernameResult PatternStruct
			err = unmarshaler.Unmarshal(context.Background(), invalidUsernameInput, &invalidUsernameResult)
			c.Assert(err, qt.ErrorMatches, `error unmarshaling field username: pattern validation failed:.*`)

			// Test invalid code pattern
			invalidCodeInput := Map{
				"code": NewString("12345"), // 5 digits instead of 4
				"name": NewString("test"),
			}
			var invalidCodeResult PatternStruct
			err = unmarshaler.Unmarshal(context.Background(), invalidCodeInput, &invalidCodeResult)
			c.Assert(err, qt.ErrorMatches, `error unmarshaling field code: pattern validation failed:.*`)

			// Test invalid email pattern
			invalidEmailInput := Map{
				"email": NewString("invalid-email"), // Missing @ and domain
				"name":  NewString("test"),
			}
			var invalidEmailResult PatternStruct
			err = unmarshaler.Unmarshal(context.Background(), invalidEmailInput, &invalidEmailResult)
			c.Assert(err, qt.ErrorMatches, `error unmarshaling field email: pattern validation failed:.*`)
		})

		c.Run("Time.Duration field pattern validation", func(c *qt.C) {
			type DurationStruct struct {
				TTL            *time.Duration `instill:"ttl,pattern=^[0-9]+(\\.([0-9]{1,9}))?s$"`
				FlexibleTTL    *time.Duration `instill:"flexible-ttl"` // No pattern
				RegularTimeout string         `instill:"timeout,pattern=^[0-9]+ms$"`
			}

			validInput := Map{
				"ttl":          NewString("3600s"),
				"flexible-ttl": NewString("1h"),
				"timeout":      NewString("5000ms"),
			}

			var validResult DurationStruct
			err := unmarshaler.Unmarshal(context.Background(), validInput, &validResult)
			c.Assert(err, qt.IsNil)
			// TTL should be parsed as seconds format (3600s = 1 hour)
			c.Assert(validResult.TTL, qt.Not(qt.IsNil))
			c.Check(*validResult.TTL, qt.Equals, 1*time.Hour)
			// flexible-ttl should use standard Go duration parsing
			c.Assert(validResult.FlexibleTTL, qt.Not(qt.IsNil))
			c.Check(*validResult.FlexibleTTL, qt.Equals, 1*time.Hour)
			c.Check(validResult.RegularTimeout, qt.Equals, "5000ms")

			fractionalInput := Map{
				"ttl": NewString("0.123456789s"), // 9 fractional digits
			}
			var fractionalResult DurationStruct
			err = unmarshaler.Unmarshal(context.Background(), fractionalInput, &fractionalResult)
			c.Assert(err, qt.IsNil)
			c.Assert(fractionalResult.TTL, qt.Not(qt.IsNil))
			c.Check(*fractionalResult.TTL, qt.Equals, 123456789*time.Nanosecond)

			// Test other valid Google Duration formats
			validGoogleFormats := []struct {
				input    string
				expected time.Duration
			}{
				{"1s", 1 * time.Second},
				{"60s", 60 * time.Second},
				{"3.5s", 3500 * time.Millisecond},
				{"0.001s", 1 * time.Millisecond},
			}
			for _, test := range validGoogleFormats {
				input := Map{"ttl": NewString(test.input)}
				var result DurationStruct
				err := unmarshaler.Unmarshal(context.Background(), input, &result)
				c.Assert(err, qt.IsNil, qt.Commentf("Failed for input: %s", test.input))
				c.Assert(result.TTL, qt.Not(qt.IsNil))
				c.Check(*result.TTL, qt.Equals, test.expected, qt.Commentf("Failed for input: %s", test.input))
			}

			// Test invalid TTL patterns
			invalidTTLInputs := []string{
				"1h",            // Hours not allowed by pattern
				"30m",           // Minutes not allowed by pattern
				"1.1234567890s", // Too many fractional digits (10 digits)
				"3600",          // Missing 's' suffix
				"s",             // Missing number
				"3.5",           // Missing 's' suffix
				"abc",           // Not a number
				"",              // Empty string
			}
			for _, invalidTTL := range invalidTTLInputs {
				invalidInput := Map{"ttl": NewString(invalidTTL)}
				var invalidResult DurationStruct
				err = unmarshaler.Unmarshal(context.Background(), invalidInput, &invalidResult)
				c.Assert(err, qt.ErrorMatches, `error unmarshaling field ttl: pattern validation failed:.*`,
					qt.Commentf("Should have failed for input: %s", invalidTTL))
			}

			// Test invalid timeout pattern
			invalidTimeoutInput := Map{
				"timeout": NewString("5s"), // Should be milliseconds
			}
			var invalidTimeoutResult DurationStruct
			err = unmarshaler.Unmarshal(context.Background(), invalidTimeoutInput, &invalidTimeoutResult)
			c.Assert(err, qt.ErrorMatches, `error unmarshaling field timeout: pattern validation failed:.*`)

			// Test that JSON numbers are rejected for time.Duration fields
			jsonNumberInput := Map{
				"ttl": NewNumberFromInteger(60), // Should be rejected - must use string format
			}
			var jsonNumberResult DurationStruct
			err = unmarshaler.Unmarshal(context.Background(), jsonNumberInput, &jsonNumberResult)
			c.Assert(err, qt.ErrorMatches, `error unmarshaling field ttl: cannot unmarshal Number into \*time\.Duration: use string format like "60s"`)
		})

		c.Run("Time.Time field pattern validation (RFC 3339)", func(c *qt.C) {
			type TimeStruct struct {
				ExpireTime   *time.Time `instill:"expire-time,pattern=^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2})$"`
				FlexibleTime *time.Time `instill:"flexible-time"` // No pattern
				CreateTime   time.Time  `instill:"create-time,pattern=^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\\.[0-9]{1,9})?(Z|[+-][0-9]{2}:[0-9]{2})$"`
			}

			// Test valid RFC 3339 timestamps
			validRFC3339Inputs := []struct {
				input string
				desc  string
			}{
				{"2014-10-02T15:01:23Z", "Basic Z-normalized format"},
				{"2014-10-02T15:01:23.045123456Z", "With 9 fractional digits"},
				{"2014-10-02T15:01:23.123Z", "With 3 fractional digits"},
				{"2014-10-02T15:01:23.000000001Z", "With nanosecond precision"},
				{"2014-10-02T15:01:23+05:30", "With positive timezone offset"},
				{"2014-10-02T15:01:23-08:00", "With negative timezone offset"},
				{"2024-12-31T23:59:59Z", "End of year"},
				{"2024-01-01T00:00:00Z", "Start of year"},
			}

			for _, test := range validRFC3339Inputs {
				validInput := Map{
					"expire-time": NewString(test.input),
					"create-time": NewString(test.input),
				}
				var validResult TimeStruct
				err := unmarshaler.Unmarshal(context.Background(), validInput, &validResult)
				c.Assert(err, qt.IsNil, qt.Commentf("Failed for %s: %s", test.desc, test.input))
				c.Assert(validResult.ExpireTime, qt.Not(qt.IsNil))
				// Verify the time was parsed correctly
				expectedTime, parseErr := time.Parse(time.RFC3339Nano, test.input)
				c.Assert(parseErr, qt.IsNil)
				c.Check(validResult.ExpireTime.Equal(expectedTime), qt.IsTrue,
					qt.Commentf("Time mismatch for %s", test.input))
				c.Check(validResult.CreateTime.Equal(expectedTime), qt.IsTrue,
					qt.Commentf("Time mismatch for %s", test.input))
			}

			// Test invalid RFC 3339 timestamps that should fail pattern validation
			invalidPatternInputs := []struct {
				input string
				desc  string
			}{
				{"2024-1-1T00:00:00Z", "Single digit month/day"},
				{"2024-01-01 00:00:00", "Missing T separator"},
				{"2024-01-01T00:00:00", "Missing timezone"},
				{"2024-01-01T00:00:00.1234567890Z", "Too many fractional digits (10)"},
				{"not-a-date", "Invalid format"},
				{"", "Empty string"},
			}

			for _, test := range invalidPatternInputs {
				invalidInput := Map{"expire-time": NewString(test.input)}
				var invalidResult TimeStruct
				err := unmarshaler.Unmarshal(context.Background(), invalidInput, &invalidResult)
				c.Assert(err, qt.ErrorMatches, `error unmarshaling field expire-time: pattern validation failed:.*`,
					qt.Commentf("Should have failed for %s: %s", test.desc, test.input))
			}

			// Test invalid RFC 3339 timestamps that pass pattern but fail time parsing
			// These demonstrate that pattern validation + time parsing work together
			invalidTimeInputs := []struct {
				input string
				desc  string
			}{
				{"2024-01-01T25:00:00Z", "Invalid hour (25)"},
				{"2024-01-01T00:60:00Z", "Invalid minute (60)"},
				{"2024-01-01T00:00:60Z", "Invalid second (60)"},
				{"2024-13-01T00:00:00Z", "Invalid month (13)"},
				{"2024-01-32T00:00:00Z", "Invalid day (32)"},
			}

			for _, test := range invalidTimeInputs {
				invalidInput := Map{"expire-time": NewString(test.input)}
				var invalidResult TimeStruct
				err := unmarshaler.Unmarshal(context.Background(), invalidInput, &invalidResult)
				// These should fail during time parsing, not pattern validation
				c.Assert(err, qt.ErrorMatches, `error unmarshaling field expire-time: cannot unmarshal string .* into time\.Time:.*`,
					qt.Commentf("Should have failed during time parsing for %s: %s", test.desc, test.input))
			}

			// Test flexible time field (no pattern) should accept various formats
			flexibleInput := Map{
				"flexible-time": NewString("2024-01-01 00:00:00"), // Non-RFC3339 format
			}
			var flexibleResult TimeStruct
			err := unmarshaler.Unmarshal(context.Background(), flexibleInput, &flexibleResult)
			c.Assert(err, qt.IsNil) // Should succeed without pattern validation
		})

		c.Run("Complex pattern validation", func(c *qt.C) {
			type ComplexPatternStruct struct {
				Version   string `instill:"version,pattern=^\\d+\\.\\d+\\.\\d+$"`                                        // Semantic version
				UUID      string `instill:"uuid,pattern=^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$"` // UUID
				HexColor  string `instill:"color,pattern=^#[0-9A-Fa-f]{6}$"`                                             // Hex color
				NoPattern string `instill:"no-pattern"`                                                                  // No pattern constraint
			}

			// Test valid complex patterns
			validInput := Map{
				"version":    NewString("1.2.3"),
				"uuid":       NewString("123e4567-e89b-12d3-a456-426614174000"),
				"color":      NewString("#FF5733"),
				"no-pattern": NewString("anything goes here! 123 @#$"),
			}

			var validResult ComplexPatternStruct
			err := unmarshaler.Unmarshal(context.Background(), validInput, &validResult)
			c.Assert(err, qt.IsNil)
			c.Check(validResult.Version, qt.Equals, "1.2.3")
			c.Check(validResult.UUID, qt.Equals, "123e4567-e89b-12d3-a456-426614174000")
			c.Check(validResult.HexColor, qt.Equals, "#FF5733")
			c.Check(validResult.NoPattern, qt.Equals, "anything goes here! 123 @#$")

			// Test invalid version pattern
			invalidVersionInput := Map{
				"version": NewString("v1.2.3"), // 'v' prefix not allowed
			}
			var invalidVersionResult ComplexPatternStruct
			err = unmarshaler.Unmarshal(context.Background(), invalidVersionInput, &invalidVersionResult)
			c.Assert(err, qt.ErrorMatches, `error unmarshaling field version: pattern validation failed:.*`)

			// Test invalid UUID pattern
			invalidUUIDInput := Map{
				"uuid": NewString("not-a-uuid"),
			}
			var invalidUUIDResult ComplexPatternStruct
			err = unmarshaler.Unmarshal(context.Background(), invalidUUIDInput, &invalidUUIDResult)
			c.Assert(err, qt.ErrorMatches, `error unmarshaling field uuid: pattern validation failed:.*`)

			// Test invalid hex color pattern
			invalidColorInput := Map{
				"color": NewString("#GG5733"), // Invalid hex characters
			}
			var invalidColorResult ComplexPatternStruct
			err = unmarshaler.Unmarshal(context.Background(), invalidColorInput, &invalidColorResult)
			c.Assert(err, qt.ErrorMatches, `error unmarshaling field color: pattern validation failed:.*`)
		})

		c.Run("Pattern validation with other attributes", func(c *qt.C) {
			type MixedAttributesStruct struct {
				RequiredCode   string  `instill:"code,pattern=^[A-Z]{3}[0-9]{3}$"`
				OptionalField  *string `instill:"optional,pattern=^[a-z]+$,default=hello"`
				FormattedField string  `instill:"formatted,pattern=^[0-9]+$,format=number"`
			}

			// Test valid input with mixed attributes
			validInput := Map{
				"code":      NewString("ABC123"),
				"formatted": NewString("12345"),
			}

			var validResult MixedAttributesStruct
			err := unmarshaler.Unmarshal(context.Background(), validInput, &validResult)
			c.Assert(err, qt.IsNil)
			c.Check(validResult.RequiredCode, qt.Equals, "ABC123")
			c.Check(*validResult.OptionalField, qt.Equals, "hello") // Default value applied
			c.Check(validResult.FormattedField, qt.Equals, "12345")

			// Test invalid code pattern
			invalidInput := Map{
				"code": NewString("abc123"), // Lowercase not allowed
			}
			var invalidResult MixedAttributesStruct
			err = unmarshaler.Unmarshal(context.Background(), invalidInput, &invalidResult)
			c.Assert(err, qt.ErrorMatches, `error unmarshaling field code: pattern validation failed:.*`)
		})
	})

	c.Run("JSON string to struct conversion", func(c *qt.C) {
		c.Run("Basic struct conversion", func(c *qt.C) {
			type TargetStruct struct {
				Name  string `json:"name"`
				Age   int    `json:"age"`
				Email string `json:"email"`
			}

			type TestStruct struct {
				JSONData TargetStruct `instill:"json-data"`
			}

			// JSON string input
			jsonString := `{
				"name": "John Doe",
				"age": 30,
				"email": "john@example.com"
			}`

			input := Map{
				"json-data": NewString(jsonString),
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			c.Assert(err, qt.IsNil)
			c.Check(result.JSONData.Name, qt.Equals, "John Doe")
			c.Check(result.JSONData.Age, qt.Equals, 30)
			c.Check(result.JSONData.Email, qt.Equals, "john@example.com")
		})

		c.Run("Pointer to struct conversion", func(c *qt.C) {
			type TargetStruct struct {
				ID     int    `json:"id"`
				Status string `json:"status"`
			}

			type TestStruct struct {
				JSONData *TargetStruct `instill:"json-data"`
			}

			jsonString := `{
				"id": 123,
				"status": "active"
			}`

			input := Map{
				"json-data": NewString(jsonString),
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			c.Assert(err, qt.IsNil)
			c.Assert(result.JSONData, qt.Not(qt.IsNil))
			c.Check(result.JSONData.ID, qt.Equals, 123)
			c.Check(result.JSONData.Status, qt.Equals, "active")
		})

		c.Run("Nested struct conversion", func(c *qt.C) {
			type Address struct {
				Street string `json:"street"`
				City   string `json:"city"`
			}

			type Person struct {
				Name    string  `json:"name"`
				Address Address `json:"address"`
			}

			type TestStruct struct {
				PersonData Person `instill:"person-data"`
			}

			jsonString := `{
				"name": "Jane Smith",
				"address": {
					"street": "123 Main St",
					"city": "New York"
				}
			}`

			input := Map{
				"person-data": NewString(jsonString),
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			c.Assert(err, qt.IsNil)
			c.Check(result.PersonData.Name, qt.Equals, "Jane Smith")
			c.Check(result.PersonData.Address.Street, qt.Equals, "123 Main St")
			c.Check(result.PersonData.Address.City, qt.Equals, "New York")
		})

		c.Run("Array/slice conversion", func(c *qt.C) {
			type Item struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			}

			type ItemList struct {
				Items []Item `json:"items"`
			}

			type TestStruct struct {
				ItemData ItemList `instill:"item-data"`
			}

			jsonString := `{
				"items": [
					{"id": 1, "name": "Item 1"},
					{"id": 2, "name": "Item 2"}
				]
			}`

			input := Map{
				"item-data": NewString(jsonString),
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			c.Assert(err, qt.IsNil)
			c.Assert(result.ItemData.Items, qt.HasLen, 2)
			c.Check(result.ItemData.Items[0].ID, qt.Equals, 1)
			c.Check(result.ItemData.Items[0].Name, qt.Equals, "Item 1")
			c.Check(result.ItemData.Items[1].ID, qt.Equals, 2)
			c.Check(result.ItemData.Items[1].Name, qt.Equals, "Item 2")
		})

		c.Run("Complex nested structure with arrays", func(c *qt.C) {
			type Property struct {
				Type        string `json:"type"`
				Description string `json:"description"`
			}

			type Schema struct {
				Type       string              `json:"type"`
				Properties map[string]Property `json:"properties"`
				Required   []string            `json:"required"`
			}

			type TestStruct struct {
				ResponseSchema Schema `instill:"response-schema"`
			}

			jsonString := `{
				"type": "object",
				"properties": {
					"name": {
						"type": "string",
						"description": "The name field"
					},
					"age": {
						"type": "number",
						"description": "The age field"
					}
				},
				"required": ["name", "age"]
			}`

			input := Map{
				"response-schema": NewString(jsonString),
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			c.Assert(err, qt.IsNil)
			c.Check(result.ResponseSchema.Type, qt.Equals, "object")
			c.Assert(result.ResponseSchema.Properties, qt.HasLen, 2)
			c.Check(result.ResponseSchema.Properties["name"].Type, qt.Equals, "string")
			c.Check(result.ResponseSchema.Properties["name"].Description, qt.Equals, "The name field")
			c.Check(result.ResponseSchema.Properties["age"].Type, qt.Equals, "number")
			c.Check(result.ResponseSchema.Properties["age"].Description, qt.Equals, "The age field")
			c.Assert(result.ResponseSchema.Required, qt.HasLen, 2)
			c.Check(result.ResponseSchema.Required[0], qt.Equals, "name")
			c.Check(result.ResponseSchema.Required[1], qt.Equals, "age")
		})

		c.Run("Performance optimization - non-JSON strings", func(c *qt.C) {
			type TestStruct struct {
				RegularString string `instill:"regular-string"`
				JSONData      struct {
					Name string `json:"name"`
				} `instill:"json-data"`
			}

			// Test that regular strings don't trigger JSON parsing
			input := Map{
				"regular-string": NewString("This is just a regular string, not JSON"),
				"json-data":      NewString(`{"name": "test"}`),
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			c.Assert(err, qt.IsNil)
			c.Check(result.RegularString, qt.Equals, "This is just a regular string, not JSON")
			c.Check(result.JSONData.Name, qt.Equals, "test")
		})

		c.Run("Invalid JSON handling", func(c *qt.C) {
			type TargetStruct struct {
				Name string `json:"name"`
			}

			type TestStruct struct {
				JSONData TargetStruct `instill:"json-data"`
			}

			// Invalid JSON string - should fall back to regular string handling
			invalidJSONString := `{"name": "John", "invalid": }`

			input := Map{
				"json-data": NewString(invalidJSONString),
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			// Should fail because string can't be unmarshaled into struct normally
			c.Assert(err, qt.ErrorMatches, ".*cannot unmarshal String into.*")
		})

		c.Run("JSON object vs JSON string", func(c *qt.C) {
			type TargetStruct struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}

			type TestStruct struct {
				FromString TargetStruct `instill:"from-string"`
				FromObject TargetStruct `instill:"from-object"`
			}

			// Test both JSON string and direct object
			input := Map{
				"from-string": NewString(`{"name": "John", "age": 30}`),
				"from-object": Map{
					"name": NewString("Jane"),
					"age":  NewNumberFromInteger(25),
				},
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			c.Assert(err, qt.IsNil)
			c.Check(result.FromString.Name, qt.Equals, "John")
			c.Check(result.FromString.Age, qt.Equals, 30)
			c.Check(result.FromObject.Name, qt.Equals, "Jane")
			c.Check(result.FromObject.Age, qt.Equals, 25)
		})

		c.Run("Edge cases", func(c *qt.C) {
			type ArrayWrapper struct {
				Items []string `json:"items"`
			}

			type PrimitiveWrapper struct {
				BoolValue   bool    `json:"boolValue"`
				NumberValue float64 `json:"numberValue"`
				StringValue string  `json:"stringValue"`
			}

			type NullableWrapper struct {
				NullableString *string `json:"nullableString"`
			}

			type TestStruct struct {
				EmptyObject   struct{}         `instill:"empty-object"`
				EmptyArray    ArrayWrapper     `instill:"empty-array"`
				PrimitiveData PrimitiveWrapper `instill:"primitive-data"`
				NullableData  NullableWrapper  `instill:"nullable-data"`
			}

			input := Map{
				"empty-object": NewString(`{}`),
				"empty-array":  NewString(`{"items": []}`),
				"primitive-data": NewString(`{
					"boolValue": true,
					"numberValue": 42.5,
					"stringValue": "hello world"
				}`),
				"nullable-data": NewString(`{"nullableString": null}`),
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			c.Assert(err, qt.IsNil)
			c.Assert(result.EmptyArray.Items, qt.HasLen, 0)
			c.Check(result.PrimitiveData.BoolValue, qt.Equals, true)
			c.Check(result.PrimitiveData.NumberValue, qt.Equals, 42.5)
			c.Check(result.PrimitiveData.StringValue, qt.Equals, "hello world")
			c.Assert(result.NullableData.NullableString, qt.IsNil)
		})

		c.Run("Real-world Gemini API schema example", func(c *qt.C) {
			// Simulate the genai.Schema structure
			type Schema struct {
				Type        string             `json:"type"`
				Description string             `json:"description,omitempty"`
				Properties  map[string]*Schema `json:"properties,omitempty"`
				Items       *Schema            `json:"items,omitempty"`
				Required    []string           `json:"required,omitempty"`
				Enum        []string           `json:"enum,omitempty"`
				Pattern     string             `json:"pattern,omitempty"`
				Minimum     *float64           `json:"minimum,omitempty"`
				Maximum     *float64           `json:"maximum,omitempty"`
			}

			type GenerationConfig struct {
				ResponseSchema *Schema `json:"responseSchema,omitempty"`
			}

			type TestStruct struct {
				GenConfig GenerationConfig `instill:"generation-config"`
			}

			// Complex nested schema JSON string (similar to Gemini API usage)
			schemaString := `{
				"responseSchema": {
					"type": "array",
					"items": {
						"type": "object",
						"properties": {
							"time": {
								"type": "string",
								"pattern": "^[0-9]{2}:[0-9]{2}$",
								"description": "Timestamp in MM:SS format"
							},
							"transcript": {
								"type": "string",
								"description": "Transcript text for this segment"
							}
						},
						"required": ["time", "transcript"]
					}
				}
			}`

			input := Map{
				"generation-config": NewString(schemaString),
			}

			var result TestStruct
			err := unmarshaler.Unmarshal(context.Background(), input, &result)

			c.Assert(err, qt.IsNil)
			c.Assert(result.GenConfig.ResponseSchema, qt.Not(qt.IsNil))
			c.Check(result.GenConfig.ResponseSchema.Type, qt.Equals, "array")
			c.Assert(result.GenConfig.ResponseSchema.Items, qt.Not(qt.IsNil))
			c.Check(result.GenConfig.ResponseSchema.Items.Type, qt.Equals, "object")
			c.Assert(result.GenConfig.ResponseSchema.Items.Properties, qt.HasLen, 2)

			timeProperty := result.GenConfig.ResponseSchema.Items.Properties["time"]
			c.Assert(timeProperty, qt.Not(qt.IsNil))
			c.Check(timeProperty.Type, qt.Equals, "string")
			c.Check(timeProperty.Pattern, qt.Equals, "^[0-9]{2}:[0-9]{2}$")
			c.Check(timeProperty.Description, qt.Equals, "Timestamp in MM:SS format")

			transcriptProperty := result.GenConfig.ResponseSchema.Items.Properties["transcript"]
			c.Assert(transcriptProperty, qt.Not(qt.IsNil))
			c.Check(transcriptProperty.Type, qt.Equals, "string")
			c.Check(transcriptProperty.Description, qt.Equals, "Transcript text for this segment")

			c.Assert(result.GenConfig.ResponseSchema.Items.Required, qt.HasLen, 2)
			c.Check(result.GenConfig.ResponseSchema.Items.Required[0], qt.Equals, "time")
			c.Check(result.GenConfig.ResponseSchema.Items.Required[1], qt.Equals, "transcript")
		})
	})

}

func TestMarshal(t *testing.T) {
	t.Parallel()
	c := qt.New(t)

	marshaler := NewMarshaler()

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

		result, err := marshaler.Marshal(input)

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

		result, err := marshaler.Marshal(input)

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

		result, err := marshaler.Marshal(input)

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

		result, err := marshaler.Marshal(input)

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
		imageBytes, err := os.ReadFile("testdata/small_sample.jpeg")
		c.Assert(err, qt.IsNil)

		img, err := NewImageFromBytes(imageBytes, "image/jpeg", "small_sample.jpeg", true)
		c.Assert(err, qt.IsNil)

		input := struct {
			Image format.Image `instill:"image,image/jpeg"`
		}{
			Image: img,
		}

		result, err := marshaler.Marshal(input)
		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		image, ok := m["image"].(format.Image)
		c.Assert(ok, qt.IsTrue)
		c.Assert(image.ContentType().String(), qt.Equals, "image/jpeg")
		c.Assert(image.Width().Integer(), qt.Equals, 320)
		c.Assert(image.Height().Integer(), qt.Equals, 240)
	})

	c.Run("Null", func(c *qt.C) {
		input := struct {
			NullField *format.String `instill:"null-field"`
		}{
			NullField: nil,
		}

		result, err := marshaler.Marshal(input)

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

		result, err := marshaler.Marshal(input)

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

		result, err := marshaler.Marshal(input)

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
		stringResult, err := marshaler.Marshal(stringInput)
		c.Assert(err, qt.IsNil)
		stringMap, ok := stringResult.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(stringMap["value"].(format.String).String(), qt.Equals, "test")

		// Test number value
		numberInput := TestStruct{
			Value: NewNumberFromFloat(42.5),
		}
		numberResult, err := marshaler.Marshal(numberInput)
		c.Assert(err, qt.IsNil)
		numberMap, ok := numberResult.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(numberMap["value"].(format.Number).Float64(), qt.Equals, 42.5)

		// Test boolean value
		boolInput := TestStruct{
			Value: NewBoolean(true),
		}
		boolResult, err := marshaler.Marshal(boolInput)
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
			_, err := marshaler.Marshal(input)
			c.Assert(err, qt.ErrorMatches, "error marshaling field invalid: unsupported type: chan")
		})

		c.Run("Nil interface", func(c *qt.C) {
			var input interface{}
			_, err := marshaler.Marshal(input)
			c.Assert(err, qt.ErrorMatches, "input must not be nil")
		})

		c.Run("Invalid map key type", func(c *qt.C) {
			input := struct {
				InvalidMap map[int]string `instill:"invalid-map"`
			}{
				InvalidMap: map[int]string{1: "value"},
			}
			_, err := marshaler.Marshal(input)
			c.Assert(err, qt.ErrorMatches, "error marshaling field invalid-map: map key must be string type")
		})
	})

	c.Run("Empty struct", func(c *qt.C) {
		input := struct{}{}
		result, err := marshaler.Marshal(input)
		c.Assert(err, qt.IsNil)
		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)
		c.Assert(len(m), qt.Equals, 0)
	})

	c.Run("Compositional struct", func(c *qt.C) {
		type BaseStruct struct {
			BaseField format.String `instill:"base-field"`
		}

		type ComposedStruct struct {
			BaseStruct
			ExtraField format.Number `instill:"extra-field"`
		}

		input := ComposedStruct{
			BaseStruct: BaseStruct{
				BaseField: NewString("base value"),
			},
			ExtraField: NewNumberFromFloat(123),
		}

		result, err := marshaler.Marshal(input)
		c.Assert(err, qt.IsNil)

		m, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)

		// Test embedded struct field
		embeddedMap, exists := m["BaseStruct"].(Map)
		c.Assert(exists, qt.IsTrue)
		baseField, ok := embeddedMap["base-field"].(format.String)
		c.Assert(ok, qt.IsTrue)
		c.Assert(baseField.String(), qt.Equals, "base value")

		// Test regular field
		extraField, ok := m["extra-field"].(format.Number)
		c.Assert(ok, qt.IsTrue)
		c.Assert(extraField.Float64(), qt.Equals, 123.0)
	})

	c.Run("JSON tag automatic naming convention", func(c *qt.C) {
		type ExternalType struct {
			MIMEType    string `json:"mimeType"`
			FileURI     string `json:"fileUri"`
			DisplayName string `json:"displayName"`
			WithInstill string `instill:"custom-field-name" json:"jsonFieldName"`
			PlainField  string
		}

		input := ExternalType{
			MIMEType:    "application/pdf",
			FileURI:     "gs://bucket/file.pdf",
			DisplayName: "document.pdf",
			WithInstill: "instill-value",
			PlainField:  "plain-value",
		}

		marshaler := NewMarshaler()
		result, err := marshaler.Marshal(input)
		c.Assert(err, qt.IsNil)

		resultMap, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)

		// Verify json tags are converted to kebab-case
		c.Check(resultMap["mime-type"].(format.String).String(), qt.Equals, "application/pdf")
		c.Check(resultMap["file-uri"].(format.String).String(), qt.Equals, "gs://bucket/file.pdf")
		c.Check(resultMap["display-name"].(format.String).String(), qt.Equals, "document.pdf")

		// Verify instill tag takes precedence over json tag
		c.Check(resultMap["custom-field-name"].(format.String).String(), qt.Equals, "instill-value")

		// Verify field name is used when no tag is present
		c.Check(resultMap["PlainField"].(format.String).String(), qt.Equals, "plain-value")
	})

	c.Run("JSON tag naming conventions - snake_case and PascalCase", func(c *qt.C) {
		type MixedNamingType struct {
			CamelCaseField  string `json:"camelCaseField"`
			SnakeCaseField  string `json:"snake_case_field"`
			PascalCaseField string `json:"PascalCaseField"`
			WithInstill     string `instill:"custom-field-name" json:"anyJsonName"`
			NoTag           string
		}

		input := MixedNamingType{
			CamelCaseField:  "camel-value",
			SnakeCaseField:  "snake-value",
			PascalCaseField: "pascal-value",
			WithInstill:     "instill-value",
			NoTag:           "no-tag-value",
		}

		marshaler := NewMarshaler()
		result, err := marshaler.Marshal(input)
		c.Assert(err, qt.IsNil)

		resultMap, ok := result.(Map)
		c.Assert(ok, qt.IsTrue)

		// Verify all json tags are converted to kebab-case
		c.Check(resultMap["camel-case-field"].(format.String).String(), qt.Equals, "camel-value")
		c.Check(resultMap["snake-case-field"].(format.String).String(), qt.Equals, "snake-value")
		c.Check(resultMap["pascal-case-field"].(format.String).String(), qt.Equals, "pascal-value")

		// Verify instill tag takes precedence
		c.Check(resultMap["custom-field-name"].(format.String).String(), qt.Equals, "instill-value")

		// Verify field name is used when no tag is present
		c.Check(resultMap["NoTag"].(format.String).String(), qt.Equals, "no-tag-value")
	})

	c.Run("Round-trip marshaling and unmarshaling with JSON tags", func(c *qt.C) {
		type ExternalType struct {
			MIMEType    string `json:"mimeType"`
			FileURI     string `json:"fileUri"`
			DisplayName string `json:"displayName"`
			WithInstill string `instill:"custom-field-name" json:"jsonFieldName"`
			PlainField  string
		}

		original := ExternalType{
			MIMEType:    "application/pdf",
			FileURI:     "gs://bucket/file.pdf",
			DisplayName: "document.pdf",
			WithInstill: "instill-value",
			PlainField:  "plain-value",
		}

		// Marshal to Map
		marshaler := NewMarshaler()
		marshaled, err := marshaler.Marshal(original)
		c.Assert(err, qt.IsNil)

		// Unmarshal back to struct
		binaryFetcher := external.NewBinaryFetcher()
		unmarshaler := NewUnmarshaler(binaryFetcher)
		var unmarshaled ExternalType
		err = unmarshaler.Unmarshal(context.Background(), marshaled, &unmarshaled)
		c.Assert(err, qt.IsNil)

		// Verify round-trip preserves all values
		c.Check(unmarshaled.MIMEType, qt.Equals, original.MIMEType)
		c.Check(unmarshaled.FileURI, qt.Equals, original.FileURI)
		c.Check(unmarshaled.DisplayName, qt.Equals, original.DisplayName)
		c.Check(unmarshaled.WithInstill, qt.Equals, original.WithInstill)
		c.Check(unmarshaled.PlainField, qt.Equals, original.PlainField)
	})

	c.Run("Time types marshaling and unmarshaling", func(c *qt.C) {
		type TimeStruct struct {
			CreateTime *time.Time     `instill:"create-time"`
			UpdateTime time.Time      `instill:"update-time"`
			TTL        *time.Duration `instill:"ttl"`
			Timeout    time.Duration  `instill:"timeout"`
		}

		createTime := time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC)
		updateTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		ttl := 1 * time.Hour
		timeout := 30 * time.Second

		original := TimeStruct{
			CreateTime: &createTime,
			UpdateTime: updateTime,
			TTL:        &ttl,
			Timeout:    timeout,
		}

		// Marshal to Map
		marshaler := NewMarshaler()
		marshaled, err := marshaler.Marshal(original)
		c.Assert(err, qt.IsNil)

		// Verify marshaled values are strings
		marshaledMap, ok := marshaled.(Map)
		c.Assert(ok, qt.IsTrue)

		createTimeStr, ok := marshaledMap["create-time"].(format.String)
		c.Assert(ok, qt.IsTrue)
		c.Check(createTimeStr.String(), qt.Equals, "2023-12-25T10:30:00Z")

		updateTimeStr, ok := marshaledMap["update-time"].(format.String)
		c.Assert(ok, qt.IsTrue)
		c.Check(updateTimeStr.String(), qt.Equals, "2024-01-01T00:00:00Z")

		ttlStr, ok := marshaledMap["ttl"].(format.String)
		c.Assert(ok, qt.IsTrue)
		c.Check(ttlStr.String(), qt.Equals, "1h0m0s")

		timeoutStr, ok := marshaledMap["timeout"].(format.String)
		c.Assert(ok, qt.IsTrue)
		c.Check(timeoutStr.String(), qt.Equals, "30s")

		// Unmarshal back to struct
		binaryFetcher := external.NewBinaryFetcher()
		unmarshaler := NewUnmarshaler(binaryFetcher)
		var unmarshaled TimeStruct
		err = unmarshaler.Unmarshal(context.Background(), marshaled, &unmarshaled)
		c.Assert(err, qt.IsNil)

		// Verify round-trip preserves all values
		c.Assert(unmarshaled.CreateTime, qt.Not(qt.IsNil))
		c.Check(*unmarshaled.CreateTime, qt.Equals, createTime)
		c.Check(unmarshaled.UpdateTime, qt.Equals, updateTime)
		c.Assert(unmarshaled.TTL, qt.Not(qt.IsNil))
		c.Check(*unmarshaled.TTL, qt.Equals, ttl)
		c.Check(unmarshaled.Timeout, qt.Equals, timeout)
	})
}

// Performance Benchmarks

// BenchmarkReflectionTypeComparison benchmarks the difference between
// repeated reflect.TypeOf calls vs pre-computed types
func BenchmarkReflectionTypeComparison(b *testing.B) {
	b.Run("Old_RepeatedReflectTypeOf", func(b *testing.B) {
		for b.Loop() {
			// Simulate old behavior - repeated reflect.TypeOf calls
			_ = reflect.TypeOf(time.Time{})
			_ = reflect.TypeOf(time.Duration(0))
			_ = reflect.TypeOf((*format.Value)(nil)).Elem()
			_ = reflect.TypeOf((*format.String)(nil)).Elem()
			_ = reflect.TypeOf((*format.Number)(nil)).Elem()
		}
	})

	b.Run("New_PreComputedTypes", func(b *testing.B) {
		for b.Loop() {
			// Simulate new behavior - using pre-computed types
			_ = timeTimeType
			_ = timeDurationType
			_ = formatValueType
			_ = formatStringType
			_ = formatNumberType
		}
	})
}

// BenchmarkRegexPatternValidation benchmarks regex compilation caching
func BenchmarkRegexPatternValidation(b *testing.B) {
	testValue := "test@example.com"
	emailPattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`

	b.Run("Old_RepeatedRegexCompile", func(b *testing.B) {
		for b.Loop() {
			// Simulate old behavior - compile regex every time
			regex, err := regexp.Compile(emailPattern)
			if err != nil {
				b.Fatal(err)
			}
			_ = regex.MatchString(testValue)
		}
	})

	b.Run("New_CachedRegex", func(b *testing.B) {
		for b.Loop() {
			// Use the new cached regex compilation
			err := validatePattern(testValue, emailPattern)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkTagParsing benchmarks instill tag parsing with and without caching
func BenchmarkTagParsing(b *testing.B) {
	complexTag := "field-name,format=image/jpeg,pattern=^[a-zA-Z0-9._-]+$,default=test"

	b.Run("Old_RepeatedTagParsing", func(b *testing.B) {
		for b.Loop() {
			// Simulate old behavior - parse tag every time
			_, _, _, _ = parseInstillTagUncached(complexTag)
		}
	})

	b.Run("New_CachedTagParsing", func(b *testing.B) {
		for b.Loop() {
			// Use the new cached tag parsing
			_, _, _, _ = parseInstillTag(complexTag)
		}
	})
}

// BenchmarkTimeFormatParsing benchmarks time parsing with pre-compiled formats
func BenchmarkTimeFormatParsing(b *testing.B) {
	timeString := "2023-12-25T15:30:45Z"

	b.Run("Old_RepeatedFormatSliceCreation", func(b *testing.B) {
		for b.Loop() {
			// Simulate old behavior - create format slice every time
			formats := []string{
				time.RFC3339,
				time.RFC3339Nano,
				"2006-01-02T15:04:05Z07:00",
				"2006-01-02 15:04:05",
				"2006-01-02",
			}
			for _, format := range formats {
				if _, err := time.Parse(format, timeString); err == nil {
					break
				}
			}
		}
	})

	b.Run("New_PreCompiledFormats", func(b *testing.B) {
		for b.Loop() {
			// Use the new pre-compiled formats
			_, _ = parseTimeValue(timeString, "")
		}
	})
}

// BenchmarkFileTypeChecking benchmarks file type checking optimization
func BenchmarkFileTypeChecking(b *testing.B) {
	imageType := formatImageType

	b.Run("Old_RepeatedFileTypeSliceCreation", func(b *testing.B) {
		for b.Loop() {
			// Simulate old behavior - create file types slice every time
			fileTypes := []reflect.Type{
				reflect.TypeOf((*format.Image)(nil)).Elem(),
				reflect.TypeOf((*format.Audio)(nil)).Elem(),
				reflect.TypeOf((*format.Video)(nil)).Elem(),
				reflect.TypeOf((*format.Document)(nil)).Elem(),
				reflect.TypeOf((*format.File)(nil)).Elem(),
			}
			for _, fileType := range fileTypes {
				if imageType == fileType {
					break
				}
			}
		}
	})

	b.Run("New_PreComputedFileTypes", func(b *testing.B) {
		for b.Loop() {
			// Use the new pre-computed file types
			_ = isFileType(imageType)
		}
	})
}

// BenchmarkCompleteStructUnmarshaling benchmarks the overall unmarshaling performance
func BenchmarkCompleteStructUnmarshaling(b *testing.B) {
	type TestStruct struct {
		Name        string        `instill:"name,pattern=^[a-zA-Z ]+$"`
		Email       string        `instill:"email,pattern=^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$"`
		Age         int           `instill:"age"`
		Score       float64       `instill:"score"`
		IsActive    bool          `instill:"is-active"`
		CreatedAt   time.Time     `instill:"created-at"`
		Duration    time.Duration `instill:"duration"`
		Description *string       `instill:"description,default=No description"`
	}

	input := Map{
		"name":        NewString("John Doe"),
		"email":       NewString("john@example.com"),
		"age":         NewNumberFromInteger(30),
		"score":       NewNumberFromFloat(95.5),
		"is-active":   NewBoolean(true),
		"created-at":  NewString("2023-12-25T15:30:45Z"),
		"duration":    NewString("1h30m"),
		"description": NewString("Test user"),
	}

	ctx := context.Background()
	unmarshaler := NewUnmarshaler(nil)

	b.ResetTimer()
	for b.Loop() {
		var result TestStruct
		err := unmarshaler.Unmarshal(ctx, input, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCompleteStructMarshaling benchmarks the overall marshaling performance
func BenchmarkCompleteStructMarshaling(b *testing.B) {
	type TestStruct struct {
		Name        string        `instill:"name"`
		Email       string        `instill:"email"`
		Age         int           `instill:"age"`
		Score       float64       `instill:"score"`
		IsActive    bool          `instill:"is-active"`
		CreatedAt   time.Time     `instill:"created-at"`
		Duration    time.Duration `instill:"duration"`
		Description *string       `instill:"description"`
	}

	desc := "Test user"
	input := TestStruct{
		Name:        "John Doe",
		Email:       "john@example.com",
		Age:         30,
		Score:       95.5,
		IsActive:    true,
		CreatedAt:   time.Date(2023, 12, 25, 15, 30, 45, 0, time.UTC),
		Duration:    90 * time.Minute,
		Description: &desc,
	}

	marshaler := NewMarshaler()

	b.ResetTimer()
	for b.Loop() {
		_, err := marshaler.Marshal(input)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCacheContention benchmarks cache performance under concurrent access
func BenchmarkCacheContention(b *testing.B) {
	patterns := []string{
		`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
		`^[0-9]{3}-[0-9]{2}-[0-9]{4}$`,
		`^[a-zA-Z ]+$`,
		`^[0-9]+$`,
		`^https?://[^\s]+$`,
	}

	tags := []string{
		"field1,format=image/jpeg,pattern=^[a-zA-Z]+$",
		"field2,format=video/mp4,default=test",
		"field3,pattern=^[0-9]+$,format=text/plain",
		"field4,default=value,pattern=^[a-zA-Z0-9]+$",
		"field5,format=audio/mpeg",
	}

	b.Run("RegexCacheContention", func(b *testing.B) {
		testValues := []string{
			"test@example.com",
			"123-45-6789",
			"John Doe",
			"12345",
			"https://example.com",
		}

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				pattern := patterns[i%len(patterns)]
				testValue := testValues[i%len(testValues)]
				_ = validatePattern(testValue, pattern) // Ignore validation errors for benchmark
				i++
			}
		})
	})

	b.Run("TagCacheContention", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				tag := tags[i%len(tags)]
				_, _, _, _ = parseInstillTag(tag)
				i++
			}
		})
	})
}

// BenchmarkMemoryAllocation benchmarks memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.Run("StringOperations", func(b *testing.B) {
		tag := "field-name,format=image/jpeg,pattern=^[a-zA-Z0-9._-]+$,default=test"

		b.ResetTimer()
		for b.Loop() {
			// Test string operations that were optimized
			parts := strings.Split(tag, ",")
			for _, part := range parts {
				if strings.HasPrefix(part, "pattern=") {
					_ = strings.TrimPrefix(part, "pattern=")
				}
			}
		}
	})

	b.Run("MapCreation", func(b *testing.B) {
		b.ResetTimer()
		for b.Loop() {
			// Test map creation patterns
			attributes := make(map[string]string)
			attributes["default"] = "test"
			attributes["format"] = "image/jpeg"
			attributes["pattern"] = "^[a-zA-Z0-9._-]+$"
		}
	})
}
