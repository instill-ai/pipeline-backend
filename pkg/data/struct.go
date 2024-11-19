package data

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/instill-ai/pipeline-backend/pkg/data/format"
	"github.com/instill-ai/pipeline-backend/pkg/external"
)

// Package data provides functionality for marshaling and unmarshaling between
// Go structs and a custom Map type that represents structured data.
//
// The main functions in this file are:
//
// - Unmarshal: Converts a Map value into a provided struct using `instill` tags.
// - Marshal: Converts a struct into a Map that represents the struct fields as
// values.
//
// These functions use reflection to handle various types, including nested
// structs, slices, maps, and custom types that implement the format.Value
// interface.
//
// The following struct tags are supported:
//
// - `instill`: Specifies the key name and optional format when marshaling/unmarshaling the field.
//   If not provided, the field name will be used. For example:
//   type Person struct {
//     FirstName string `instill:"first_name"`           // Will use "first_name" as the key
//     LastName  string                                  // Will use "LastName" as the key
//     Avatar   format.Image `instill:"photo,image/png"` // Will use "photo" as key and convert to PNG
//   }
//
// The format portion of the tag supports:
//   - For Image: "image/png", "image/jpeg", etc
//   - For Video: "video/mp4", "video/webm", etc
//   - For Audio: "audio/mpeg", "audio/wav", etc
//   - For Document: "application/pdf", "text/plain", etc

type Marshaler struct {
}

type Unmarshaler struct {
	binaryFetcher external.BinaryFetcher
}

func NewMarshaler() *Marshaler {
	return &Marshaler{}
}

func NewUnmarshaler(binaryFetcher external.BinaryFetcher) *Unmarshaler {
	return &Unmarshaler{binaryFetcher}
}

// Unmarshal converts a Map value into the provided struct s using `instill` tags.
func (u *Unmarshaler) Unmarshal(ctx context.Context, d format.Value, s any) error {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("input must be a pointer to a struct")
	}
	m, ok := d.(Map)
	if !ok {
		return errors.New("input value must be a Map")
	}
	return u.unmarshalStruct(ctx, m, v.Elem())
}

// unmarshalStruct iterates through struct fields and unmarshals corresponding values.
func (u *Unmarshaler) unmarshalStruct(ctx context.Context, m Map, v reflect.Value) error {
	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)
		if !fieldValue.CanSet() {
			continue
		}
		fieldName := u.getFieldName(field)
		val, ok := m[fieldName]
		if !ok {
			continue
		}
		if err := u.unmarshalValue(ctx, val, fieldValue, field); err != nil {
			return fmt.Errorf("error unmarshaling field %s: %w", fieldName, err)
		}
	}
	return nil
}

// unmarshalValue dispatches to type-specific unmarshal functions based on the value type.
func (u *Unmarshaler) unmarshalValue(ctx context.Context, val format.Value, field reflect.Value, structField reflect.StructField) error {
	switch v := val.(type) {
	case *fileData, *documentData, *imageData, *videoData, *audioData:
		return u.unmarshalInterface(v, field, structField)
	case *booleanData:
		return u.unmarshalBoolean(v, field)
	case *numberData:
		return u.unmarshalNumber(v, field)
	case *stringData:
		return u.unmarshalString(ctx, v, field)
	case Array:
		if field.Type().Implements(reflect.TypeOf((*format.Value)(nil)).Elem()) {
			field.Set(reflect.ValueOf(v))
			return nil
		}
		return u.unmarshalArray(ctx, v, field)
	case Map:
		if field.Type().Implements(reflect.TypeOf((*format.Value)(nil)).Elem()) {
			field.Set(reflect.ValueOf(v))
			return nil
		}
		return u.unmarshalMap(ctx, v, field)
	case *nullData:
		if field.Type().Implements(reflect.TypeOf((*format.Value)(nil)).Elem()) {
			field.Set(reflect.ValueOf(v))
			return nil
		}
		return u.unmarshalNull(v, field)
	default:
		return fmt.Errorf("unsupported type: %T", val)
	}
}

// unmarshalString handles unmarshaling of String values.
func (u *Unmarshaler) unmarshalString(ctx context.Context, v format.String, field reflect.Value) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(v.String())
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return u.unmarshalString(ctx, v, field.Elem())
	default:
		switch field.Type() {

		// If the string is a URL, create a file from the URL
		case reflect.TypeOf((*format.Image)(nil)).Elem(),
			reflect.TypeOf((*format.Audio)(nil)).Elem(),
			reflect.TypeOf((*format.Video)(nil)).Elem(),
			reflect.TypeOf((*format.Document)(nil)).Elem(),
			reflect.TypeOf((*format.File)(nil)).Elem():
			f, err := u.createFileFromURL(ctx, field.Type(), v.String())
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

func (u *Unmarshaler) createFileFromURL(ctx context.Context, t reflect.Type, url string) (format.Value, error) {
	switch t {
	case reflect.TypeOf((*format.Image)(nil)).Elem():
		return NewImageFromURL(ctx, u.binaryFetcher, url)
	case reflect.TypeOf((*format.Audio)(nil)).Elem():
		return NewAudioFromURL(ctx, u.binaryFetcher, url)
	case reflect.TypeOf((*format.Video)(nil)).Elem():
		return NewVideoFromURL(ctx, u.binaryFetcher, url)
	case reflect.TypeOf((*format.Document)(nil)).Elem():
		return NewDocumentFromURL(ctx, u.binaryFetcher, url)
	case reflect.TypeOf((*format.File)(nil)).Elem():
		return NewBinaryFromURL(ctx, u.binaryFetcher, url)
	}
	return nil, fmt.Errorf("unsupported type: %v", t)
}

// unmarshalBoolean handles unmarshaling of Boolean values.
func (u *Unmarshaler) unmarshalBoolean(v format.Boolean, field reflect.Value) error {
	switch field.Kind() {
	case reflect.Bool:
		field.SetBool(v.Boolean())
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return u.unmarshalBoolean(v, field.Elem())
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
func (u *Unmarshaler) unmarshalNumber(v format.Number, field reflect.Value) error {
	switch field.Kind() {
	case reflect.Float32, reflect.Float64:
		field.SetFloat(v.Float64())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		field.SetInt(int64(v.Integer()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.SetUint(uint64(v.Integer()))
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return u.unmarshalNumber(v, field.Elem())
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
func (u *Unmarshaler) unmarshalArray(ctx context.Context, v Array, field reflect.Value) error {
	if field.Kind() != reflect.Slice {
		return fmt.Errorf("cannot unmarshal Array into %v", field.Type())
	}
	slice := reflect.MakeSlice(field.Type(), len(v), len(v))
	for i, elem := range v {
		elemValue := slice.Index(i)
		if err := u.unmarshalValue(ctx, elem, elemValue, reflect.StructField{}); err != nil {
			return fmt.Errorf("error unmarshaling array element %d: %w", i, err)
		}
	}
	field.Set(slice)
	return nil
}

// unmarshalMap handles unmarshaling of Map values.
func (u *Unmarshaler) unmarshalMap(ctx context.Context, v Map, field reflect.Value) error {
	switch field.Kind() {
	case reflect.Map:
		return u.unmarshalToReflectMap(ctx, v, field)
	case reflect.Struct:
		return u.unmarshalToStruct(ctx, v, field)
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return u.unmarshalMap(ctx, v, field.Elem())
	default:
		return fmt.Errorf("cannot unmarshal Map into %v", field.Type())
	}
}

// unmarshalToReflectMap handles unmarshaling of Map values into reflect.Map.
func (u *Unmarshaler) unmarshalToReflectMap(ctx context.Context, v Map, field reflect.Value) error {
	mapValue := reflect.MakeMap(field.Type())
	for k, val := range v {
		keyValue := reflect.ValueOf(k)
		elemType := field.Type().Elem()
		elemValue := reflect.New(elemType).Elem()

		if err := u.unmarshalValue(ctx, val, elemValue, reflect.StructField{}); err != nil {
			return fmt.Errorf("error unmarshaling map value for key %s: %w", k, err)
		}

		mapValue.SetMapIndex(keyValue, elemValue)
	}
	field.Set(mapValue)
	return nil
}

// unmarshalToStruct handles unmarshaling of Map values into struct.
func (u *Unmarshaler) unmarshalToStruct(ctx context.Context, v Map, field reflect.Value) error {
	for i := 0; i < field.NumField(); i++ {
		structField := field.Type().Field(i)
		fieldValue := field.Field(i)
		if !fieldValue.CanSet() {
			continue
		}
		fieldName := u.getFieldName(structField)
		val, ok := v[fieldName]
		if !ok {
			continue
		}
		if err := u.unmarshalValue(ctx, val, fieldValue, structField); err != nil {
			return fmt.Errorf("error unmarshaling field %s: %w", fieldName, err)
		}
	}
	return nil
}

// unmarshalNull handles unmarshaling of Null values.
func (u *Unmarshaler) unmarshalNull(_ format.Null, field reflect.Value) error {
	if field.Kind() == reflect.Ptr {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}
	return fmt.Errorf("cannot unmarshal Null into non-pointer %v", field.Type())
}

// unmarshalInterface handles unmarshaling of interface values.
func (u *Unmarshaler) unmarshalInterface(v format.Value, field reflect.Value, structField reflect.StructField) error {
	if field.Kind() == reflect.String {
		field.SetString(v.(format.String).String())
		return nil
	}
	if field.Type() == reflect.TypeOf((*format.String)(nil)).Elem() {
		field.SetString(v.(format.String).String())
		return nil
	}
	if field.Type().Implements(reflect.TypeOf((*format.Value)(nil)).Elem()) {
		// Check for format in instill tag and convert if needed
		if tag := structField.Tag.Get("instill"); tag != "" {
			parts := strings.Split(tag, ",")
			if len(parts) > 1 {
				formatTag := parts[1]
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
		}
		if f, ok := v.(*fileData); ok {
			file, err := NewBinaryFromBytes(f.raw, f.contentType, f.filename)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(file))
			return nil
		} else {
			field.Set(reflect.ValueOf(v))
		}

		return nil
	}
	return fmt.Errorf("cannot unmarshal %T into %v", v, field.Type())
}

// getFieldName returns the field name from the struct tag or the field name itself.
func (u *Unmarshaler) getFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("instill")
	if tag == "" {
		return field.Name
	}
	parts := strings.Split(tag, ",")
	return parts[0]
}

// Marshal converts a struct into a Map that represents the struct fields as values.
func (m *Marshaler) Marshal(val any) (format.Value, error) {
	if val == nil {
		return nil, fmt.Errorf("input must not be nil")
	}
	v := reflect.ValueOf(val)
	return m.marshalValue(v)
}

// marshalValue handles marshaling of different value types.
func (m *Marshaler) marshalValue(v reflect.Value) (format.Value, error) {
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
		return m.marshalStruct(v)
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return nil, fmt.Errorf("map key must be string type")
		}
		return m.marshalMap(v)
	case reflect.Slice, reflect.Array:
		return m.marshalSlice(v)
	case reflect.Float32, reflect.Float64:
		return NewNumberFromFloat(v.Float()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return NewNumberFromInteger(int(v.Int())), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return NewNumberFromInteger(int(v.Uint())), nil
	case reflect.Bool:
		return NewBoolean(v.Bool()), nil
	case reflect.String:
		return NewString(v.String()), nil
	case reflect.Interface:
		if v.IsNil() {
			return NewNull(), nil
		}
		return m.marshalValue(v.Elem())
	default:
		return nil, fmt.Errorf("unsupported type: %v", v.Kind())
	}
}

// marshalStruct handles marshaling of struct values.
func (m *Marshaler) marshalStruct(v reflect.Value) (Map, error) {
	t := v.Type()
	mp := Map{}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		tag := field.Tag.Get("instill")
		var fieldName string
		var formatTag string

		if tag != "" {
			parts := strings.Split(tag, ",")
			fieldName = parts[0]
			if len(parts) > 1 {
				formatTag = parts[1]
			}
		} else {
			fieldName = field.Name
		}

		// Handle format conversion before marshaling
		if formatTag != "" && fieldValue.CanInterface() {
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

		marshaledValue, err := m.marshalValue(fieldValue)
		if err != nil {
			return nil, fmt.Errorf("error marshaling field %s: %w", fieldName, err)
		}

		mp[fieldName] = marshaledValue
	}

	return mp, nil
}

// marshalMap handles marshaling of map values.
func (m *Marshaler) marshalMap(v reflect.Value) (Map, error) {
	mp := Map{}
	for _, key := range v.MapKeys() {
		keyStr := key.String()

		marshaledValue, err := m.marshalValue(v.MapIndex(key))
		if err != nil {
			return nil, fmt.Errorf("error marshaling map value: %w", err)
		}

		mp[keyStr] = marshaledValue
	}
	return mp, nil
}

// marshalSlice handles marshaling of slice values.
func (m *Marshaler) marshalSlice(v reflect.Value) (Array, error) {
	arr := make(Array, v.Len())
	for i := 0; i < v.Len(); i++ {
		marshaledValue, err := m.marshalValue(v.Index(i))
		if err != nil {
			return nil, fmt.Errorf("error marshaling slice element %d: %w", i, err)
		}
		arr[i] = marshaledValue
	}
	return arr, nil
}
