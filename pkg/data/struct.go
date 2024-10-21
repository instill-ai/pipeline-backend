package data

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
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
// structs, slices, maps, and custom types that implement the value.Value
// interface.

// Unmarshal converts a Map value into the provided struct s using `key` tags.
func Unmarshal(d value.Value, s any) error {
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
		if err := unmarshalValue(val, fieldValue); err != nil {
			return fmt.Errorf("error unmarshaling field %s: %w", fieldName, err)
		}
	}
	return nil
}

// unmarshalValue dispatches to type-specific unmarshal functions based on the value type.
func unmarshalValue(val value.Value, field reflect.Value) error {
	switch v := val.(type) {
	case File, Document, Image, Video, Audio:
		return unmarshalInterface(v, field)
	case Boolean:
		return unmarshalBoolean(v, field)
	case Number:
		return unmarshalNumber(v, field)
	case String:
		return unmarshalString(v, field)
	case Array:
		return unmarshalArray(v, field)
	case Map:
		return unmarshalMap(v, field)
	case Null:
		return unmarshalNull(v, field)
	default:
		return fmt.Errorf("unsupported type: %T", val)
	}
}

// unmarshalString handles unmarshaling of String values.
func unmarshalString(v String, field reflect.Value) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(v.String())
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return unmarshalString(v, field.Elem())
	default:
		if field.Type() == reflect.TypeOf(v) || field.Type() == reflect.TypeOf((*String)(nil)).Elem() {
			field.Set(reflect.ValueOf(v))
		} else {
			return fmt.Errorf("cannot unmarshal String into %v", field.Type())
		}
	}
	return nil
}

// unmarshalBoolean handles unmarshaling of Boolean values.
func unmarshalBoolean(v Boolean, field reflect.Value) error {
	switch field.Kind() {
	case reflect.Bool:
		field.SetBool(v.Boolean())
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return unmarshalBoolean(v, field.Elem())
	default:
		if field.Type() == reflect.TypeOf(v) || field.Type() == reflect.TypeOf((*Boolean)(nil)).Elem() {
			field.Set(reflect.ValueOf(v))
		} else {
			return fmt.Errorf("cannot unmarshal Boolean into %v", field.Type())
		}
	}
	return nil
}

// unmarshalNumber handles unmarshaling of Number values.
func unmarshalNumber(v Number, field reflect.Value) error {
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
		if field.Type() == reflect.TypeOf(v) || field.Type() == reflect.TypeOf((*Number)(nil)).Elem() {
			field.Set(reflect.ValueOf(v))
		} else {
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
		if elemValue.Type().Implements(reflect.TypeOf((*value.Value)(nil)).Elem()) {
			elemValue.Set(reflect.ValueOf(elem))
		} else {
			if err := unmarshalValue(elem, elemValue); err != nil {
				return fmt.Errorf("error unmarshaling array element %d: %w", i, err)
			}
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

		if elemType.Implements(reflect.TypeOf((*value.Value)(nil)).Elem()) {
			elemValue.Set(reflect.ValueOf(val))
		} else {
			if err := unmarshalValue(val, elemValue); err != nil {
				return fmt.Errorf("error unmarshaling map value for key %s: %w", k, err)
			}
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
		if err := unmarshalValue(val, fieldValue); err != nil {
			return fmt.Errorf("error unmarshaling field %s: %w", fieldName, err)
		}
	}
	return nil
}

// unmarshalNull handles unmarshaling of Null values.
func unmarshalNull(v Null, field reflect.Value) error {
	if field.Kind() == reflect.Ptr {
		field.Set(reflect.Zero(field.Type()))
		return nil
	}
	return fmt.Errorf("cannot unmarshal Null into non-pointer %v", field.Type())
}

// unmarshalInterface handles unmarshaling of interface values.
func unmarshalInterface(v value.Value, field reflect.Value) error {
	if field.Type().Implements(reflect.TypeOf((*value.Value)(nil)).Elem()) {
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
func Marshal(val any) (value.Value, error) {
	v := reflect.ValueOf(val)
	return marshalValue(v)
}

// marshalValue handles marshaling of different value types.
func marshalValue(v reflect.Value) (value.Value, error) {
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
		if v.CanInterface() {
			if val, ok := v.Interface().(value.Value); ok {
				return val, nil
			}
		}
		return nil, fmt.Errorf("unsupported type: %v", v.Type())
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
		marshaledKey, err := marshalValue(key)
		if err != nil {
			return nil, fmt.Errorf("error marshaling map key: %w", err)
		}

		stringKey, ok := marshaledKey.(String)
		if !ok {
			return nil, fmt.Errorf("map key must be a string, got %T", marshaledKey)
		}

		marshaledValue, err := marshalValue(v.MapIndex(key))
		if err != nil {
			return nil, fmt.Errorf("error marshaling map value: %w", err)
		}

		m[stringKey.String()] = marshaledValue
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
