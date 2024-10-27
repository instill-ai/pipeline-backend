package data

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
)

// Package data provides functionality for marshaling and unmarshaling between
// Go structs and a custom Map type that represents structured data.
//
// The main functions in this file are:
//
// - Unmarshal: Converts a Map value into a provided struct using `key` tags.
// - Marshal: Converts a struct into a Map that represents the struct fields as
// values.
//
// These functions use reflection to handle various types, including nested
// structs, slices, maps, and custom types that implement the format.Value
// interface.
//
// The following struct tags are supported:
//
// - `key`: Specifies the key name to use when marshaling/unmarshaling the field.
//   If not provided, the field name will be used. For example:
//   type Person struct {
//     FirstName string `key:"first_name"`  // Will use "first_name" as the key
//     LastName  string                     // Will use "LastName" as the key
//   }
//
// - `format`: Specifies MIME type conversions for File types:
//   - For Image: "image/png", "image/jpeg", etc
//   - For Video: "video/mp4", "video/webm", etc
//   - For Audio: "audio/mp3", "audio/wav", etc
//   - For Document: "application/pdf", "text/plain", etc

// Unmarshal converts a Map value into the provided struct s using `key` tags.
func Unmarshal(d format.Value, s any) error {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("input must be a pointer to a struct")
	}
	m, ok := d.(Map)
	if !ok {
		return errors.New("input value must be a Map")
	}
	return unmarshalStruct(m, v.Elem())
}

// unmarshalStruct iterates through struct fields and unmarshals corresponding values.
func unmarshalStruct(m Map, v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		if !fieldValue.CanSet() {
			continue
		}
		fieldName := getFieldName(field)
		val, ok := m[fieldName]
		if !ok {
			continue
		}
		if err := unmarshalValue(val, fieldValue, field); err != nil {
			return fmt.Errorf("error unmarshaling field %s: %w", fieldName, err)
		}
	}
	return nil
}

// unmarshalValue dispatches to type-specific unmarshal functions based on the value type.
func unmarshalValue(val format.Value, field reflect.Value, structField reflect.StructField) error {
	switch v := val.(type) {
	case format.File, format.Document, format.Image, format.Video, format.Audio:
		return unmarshalInterface(v, field, structField)
	case format.Boolean:
		return unmarshalBoolean(v, field)
	case format.Number:
		return unmarshalNumber(v, field)
	case format.String:
		return unmarshalString(v, field)
	case Array:
		return unmarshalArray(v, field)
	case Map:
		return unmarshalMap(v, field)
	case format.Null:
		if field.Type().Implements(reflect.TypeOf((*format.Value)(nil)).Elem()) {
			field.Set(reflect.ValueOf(v))
			return nil
		}
		return unmarshalNull(v, field)
	default:
		return fmt.Errorf("unsupported type: %T", val)
	}
}

// unmarshalString handles unmarshaling of String values.
func unmarshalString(v format.String, field reflect.Value) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(v.String())
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return unmarshalString(v, field.Elem())
	default:
		switch field.Type() {

		// If the string is a URL, create a file from the URL
		case reflect.TypeOf((*format.Image)(nil)).Elem(),
			reflect.TypeOf((*format.Audio)(nil)).Elem(),
			reflect.TypeOf((*format.Video)(nil)).Elem(),
			reflect.TypeOf((*format.Document)(nil)).Elem(),
			reflect.TypeOf((*format.File)(nil)).Elem():
			f, err := createFileFromURL(field.Type(), v.String())
			if err == nil {
				field.Set(reflect.ValueOf(f))
				return nil
			}
		case reflect.TypeOf(v), reflect.TypeOf((*format.String)(nil)).Elem():
			field.Set(reflect.ValueOf(v))
		case reflect.TypeOf((*format.Value)(nil)).Elem():
			field.Set(reflect.ValueOf(v))
		default:
			return fmt.Errorf("cannot unmarshal String into %v", field.Type())
		}
	}
	return nil
}

func createFileFromURL(t reflect.Type, url string) (format.Value, error) {
	switch t {
	case reflect.TypeOf((*format.Image)(nil)).Elem():
		return NewImageFromURL(url)
	case reflect.TypeOf((*format.Audio)(nil)).Elem():
		return NewAudioFromURL(url)
	case reflect.TypeOf((*format.Video)(nil)).Elem():
		return NewVideoFromURL(url)
	case reflect.TypeOf((*format.Document)(nil)).Elem():
		return NewDocumentFromURL(url)
	case reflect.TypeOf((*format.File)(nil)).Elem():
		return NewBinaryFromURL(url)
	}
	return nil, fmt.Errorf("unsupported type: %v", t)
}

// unmarshalBoolean handles unmarshaling of Boolean values.
func unmarshalBoolean(v format.Boolean, field reflect.Value) error {
	switch field.Kind() {
	case reflect.Bool:
		field.SetBool(v.Boolean())
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return unmarshalBoolean(v, field.Elem())
	default:
		switch field.Type() {
		case reflect.TypeOf(v), reflect.TypeOf((*format.Boolean)(nil)).Elem():
			field.Set(reflect.ValueOf(v))
		case reflect.TypeOf((*format.Value)(nil)).Elem():
			field.Set(reflect.ValueOf(v))
		default:
			return fmt.Errorf("cannot unmarshal Boolean into %v", field.Type())
		}
	}
	return nil
}

// unmarshalNumber handles unmarshaling of Number values.
func unmarshalNumber(v format.Number, field reflect.Value) error {
	switch field.Kind() {
	case reflect.Float32, reflect.Float64:
		field.SetFloat(v.Float64())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(int64(v.Integer()))
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return unmarshalNumber(v, field.Elem())
	default:
		switch field.Type() {
		case reflect.TypeOf(v), reflect.TypeOf((*format.Number)(nil)).Elem():
			field.Set(reflect.ValueOf(v))
		case reflect.TypeOf((*format.Value)(nil)).Elem():
			field.Set(reflect.ValueOf(v))
		default:
			return fmt.Errorf("cannot unmarshal Number into %v", field.Type())
		}
	}
	return nil
}

// unmarshalArray handles unmarshaling of Array values.
func unmarshalArray(v Array, field reflect.Value) error {
	if field.Kind() != reflect.Slice {
		return fmt.Errorf("cannot unmarshal Array into %v", field.Type())
	}
	slice := reflect.MakeSlice(field.Type(), len(v), len(v))
	for i, elem := range v {
		elemValue := slice.Index(i)
		if err := unmarshalValue(elem, elemValue, reflect.StructField{}); err != nil {
			return fmt.Errorf("error unmarshaling array element %d: %w", i, err)
		}
	}
	field.Set(slice)
	return nil
}

// unmarshalMap handles unmarshaling of Map values.
func unmarshalMap(v Map, field reflect.Value) error {
	switch field.Kind() {
	case reflect.Map:
		return unmarshalToReflectMap(v, field)
	case reflect.Struct:
		return unmarshalToStruct(v, field)
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return unmarshalMap(v, field.Elem())
	default:
		return fmt.Errorf("cannot unmarshal Map into %v", field.Type())
	}
}

// unmarshalToReflectMap handles unmarshaling of Map values into reflect.Map.
func unmarshalToReflectMap(v Map, field reflect.Value) error {
	mapValue := reflect.MakeMap(field.Type())
	for k, val := range v {
		keyValue := reflect.ValueOf(k)
		elemType := field.Type().Elem()
		elemValue := reflect.New(elemType).Elem()

		if err := unmarshalValue(val, elemValue, reflect.StructField{}); err != nil {
			return fmt.Errorf("error unmarshaling map value for key %s: %w", k, err)
		}

		mapValue.SetMapIndex(keyValue, elemValue)
	}
	field.Set(mapValue)
	return nil
}

// unmarshalToStruct handles unmarshaling of Map values into struct.
func unmarshalToStruct(v Map, field reflect.Value) error {
	for i := 0; i < field.NumField(); i++ {
		structField := field.Type().Field(i)
		fieldValue := field.Field(i)
		if !fieldValue.CanSet() {
			continue
		}
		fieldName := getFieldName(structField)
		val, ok := v[fieldName]
		if !ok {
			continue
		}
		if err := unmarshalValue(val, fieldValue, structField); err != nil {
			return fmt.Errorf("error unmarshaling field %s: %w", fieldName, err)
		}
	}
	return nil
}

// unmarshalNull handles unmarshaling of Null values.
func unmarshalNull(v format.Null, field reflect.Value) error {
	if field.Kind() == reflect.Ptr {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}
	return fmt.Errorf("cannot unmarshal Null into non-pointer %v", field.Type())
}

// unmarshalInterface handles unmarshaling of interface values.
func unmarshalInterface(v format.Value, field reflect.Value, structField reflect.StructField) error {
	if field.Kind() == reflect.String {
		field.SetString(v.(format.String).String())
		return nil
	}
	if field.Type() == reflect.TypeOf((*format.String)(nil)).Elem() {
		field.SetString(v.(format.String).String())
		return nil
	}
	if field.Type().Implements(reflect.TypeOf((*format.Value)(nil)).Elem()) {
		// Check for format tag and convert if needed
		if formatTag := structField.Tag.Get("format"); formatTag != "" {
			switch val := v.(type) {
			case format.Image:
				converted, err := val.Convert(formatTag)
				if err != nil {
					return err
				}
				field.Set(reflect.ValueOf(converted))
				return nil
			case format.Video:
				converted, err := val.Convert(formatTag)
				if err != nil {
					return err
				}
				field.Set(reflect.ValueOf(converted))
				return nil
			case format.Audio:
				converted, err := val.Convert(formatTag)
				if err != nil {
					return err
				}
				field.Set(reflect.ValueOf(converted))
				return nil
			case format.Document:
				if formatTag == "application/pdf" {
					converted, err := val.PDF()
					if err != nil {
						return err
					}
					field.Set(reflect.ValueOf(converted))
					return nil
				} else if formatTag == "text/plain" {
					converted, err := val.Text()
					if err != nil {
						return err
					}
					field.Set(reflect.ValueOf(converted))
					return nil
				}
			}
		}
		field.Set(reflect.ValueOf(v))
		return nil
	}
	return fmt.Errorf("cannot unmarshal %T into %v", v, field.Type())
}

// getFieldName returns the field name from the struct tag or the field name itself.
func getFieldName(field reflect.StructField) string {
	fieldName := field.Tag.Get("key")
	if fieldName == "" {
		fieldName = field.Name
	}
	return fieldName
}

// Marshal converts a struct into a Map that represents the struct fields as values.
func Marshal(val any) (format.Value, error) {
	if val == nil {
		return nil, fmt.Errorf("input must not be nil")
	}
	v := reflect.ValueOf(val)
	return marshalValue(v)
}

// marshalValue handles marshaling of different value types.
func marshalValue(v reflect.Value) (format.Value, error) {
	if !v.IsValid() {
		return NewNull(), nil
	}

	if v.CanInterface() {
		if val, ok := v.Interface().(format.Value); ok {
			return val, nil
		}
	}

	// Dereference pointer if necessary
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return NewNull(), nil
		}
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		return marshalStruct(v)
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map key must be string type")
		}
		return marshalMap(v)
	case reflect.Slice, reflect.Array:
		return marshalSlice(v)
	case reflect.Float32, reflect.Float64:
		return NewNumberFromFloat(v.Float()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return NewNumberFromInteger(int(v.Int())), nil
	case reflect.Bool:
		return NewBoolean(v.Bool()), nil
	case reflect.String:
		return NewString(v.String()), nil
	default:
		return nil, fmt.Errorf("unsupported type: %v", v.Kind())
	}
}

// marshalStruct handles marshaling of struct values.
func marshalStruct(v reflect.Value) (Map, error) {
	t := v.Type()
	m := Map{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		fieldName := field.Tag.Get("key")
		if fieldName == "" {
			fieldName = field.Name
		}

		// Handle format tag conversion before marshaling
		if formatTag := field.Tag.Get("format"); formatTag != "" && fieldValue.CanInterface() {
			if val, ok := fieldValue.Interface().(format.Value); ok {
				switch v := val.(type) {
				case format.Image:
					converted, err := v.Convert(formatTag)
					if err != nil {
						return nil, err
					}
					fieldValue = reflect.ValueOf(converted)
				case format.Video:
					converted, err := v.Convert(formatTag)
					if err != nil {
						return nil, err
					}
					fieldValue = reflect.ValueOf(converted)
				case format.Audio:
					converted, err := v.Convert(formatTag)
					if err != nil {
						return nil, err
					}
					fieldValue = reflect.ValueOf(converted)
				case format.Document:
					if formatTag == "application/pdf" {
						converted, err := v.PDF()
						if err != nil {
							return nil, err
						}
						fieldValue = reflect.ValueOf(converted)
					} else if formatTag == "text/plain" {
						converted, err := v.Text()
						if err != nil {
							return nil, err
						}
						fieldValue = reflect.ValueOf(converted)
					}
				}
			}
		}

		marshaledValue, err := marshalValue(fieldValue)
		if err != nil {
			return nil, fmt.Errorf("error marshaling field %s: %w", fieldName, err)
		}

		m[fieldName] = marshaledValue
	}

	return m, nil
}

// marshalMap handles marshaling of map values.
func marshalMap(v reflect.Value) (Map, error) {
	m := Map{}
	for _, key := range v.MapKeys() {
		keyStr := key.String()

		marshaledValue, err := marshalValue(v.MapIndex(key))
		if err != nil {
			return nil, fmt.Errorf("error marshaling map value: %w", err)
		}

		m[keyStr] = marshaledValue
	}
	return m, nil
}

// marshalSlice handles marshaling of slice values.
func marshalSlice(v reflect.Value) (Array, error) {
	arr := make(Array, v.Len())
	for i := 0; i < v.Len(); i++ {
		marshaledValue, err := marshalValue(v.Index(i))
		if err != nil {
			return nil, fmt.Errorf("error marshaling slice element %d: %w", i, err)
		}
		arr[i] = marshaledValue
	}
	return arr, nil
}
