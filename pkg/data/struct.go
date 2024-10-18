package data

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/instill-ai/pipeline-backend/pkg/data/value"
)

// Unmarshal converts a Map value into the provided struct s using `key` tags.
func Unmarshal(d value.Value, s any) (err error) {
	v := reflect.ValueOf(s)

	// Ensure the input is a pointer to a struct
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		return errors.New("input must be a pointer to a struct")
	}

	// Dereference the pointer to the struct
	v = v.Elem()
	m, ok := d.(Map)
	if !ok {
		return errors.New("input value must be a Map")
	}

	// Iterate through the struct fields
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		fieldName := field.Tag.Get("key") // Get the key from the `key` tag
		if fieldName == "" {
			fieldName = field.Name // Fallback to field name if no tag is present
		}

		// Check if the map contains the key
		val, ok := m[fieldName]
		if !ok {
			continue // Skip if the key doesn't exist in the map
		}

		structField := v.FieldByName(field.Name)
		if !structField.IsValid() || !structField.CanSet() {
			return fmt.Errorf("cannot set field %q", fieldName)
		}

		valReflect := reflect.ValueOf(val)

		switch structField.Kind() {
		case reflect.Slice, reflect.Array:
			// Ensure the value is a slice
			if valReflect.Kind() != reflect.Slice {
				return fmt.Errorf("field %q: expected slice, but got %s", fieldName, valReflect.Kind())
			}

			// Create and assign a new slice with the appropriate type
			sliceType := structField.Type().Elem()
			newSlice := reflect.MakeSlice(structField.Type(), valReflect.Len(), valReflect.Len())

			for i := 0; i < valReflect.Len(); i++ {
				elem := valReflect.Index(i)
				switch sliceType {
				case reflect.TypeOf(&Null{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*Null)))
				case reflect.TypeOf(&String{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*String)))
				case reflect.TypeOf(&Number{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*Number)))
				case reflect.TypeOf(&Boolean{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*Boolean)))
				case reflect.TypeOf(&ByteArray{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*ByteArray)))
				case reflect.TypeOf(&File{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*File)))
				case reflect.TypeOf(&Document{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*Document)))
				case reflect.TypeOf(&Image{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*Image)))
				case reflect.TypeOf(&Video{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*Video)))
				case reflect.TypeOf(&Audio{}):
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface().(*Audio)))
				case reflect.TypeOf((*value.Value)(nil)).Elem():
					newSlice.Index(i).Set(reflect.ValueOf(elem.Interface()))
				}
			}
			structField.Set(newSlice)

		case reflect.Struct:
			// Recursively convert nested structs
			err = Unmarshal(d.(Map)[fieldName], structField.Addr().Interface())
			if err != nil {
				return fmt.Errorf("field %q: error converting nested struct: %w", fieldName, err)
			}

		case reflect.Map:
			// Ensure the value is a map
			if valReflect.Kind() != reflect.Map {
				return fmt.Errorf("field %q: expected map, but got %s", fieldName, valReflect.Kind())
			}

			// Create and assign a new map
			mapValueType := structField.Type().Elem()
			newMap := reflect.MakeMap(structField.Type())

			for _, mapKey := range valReflect.MapKeys() {
				mapVal := valReflect.MapIndex(mapKey)

				switch mapValueType {
				case reflect.TypeOf(&Null{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*Null)))
				case reflect.TypeOf(&String{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*String)))
				case reflect.TypeOf(&Number{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*Number)))
				case reflect.TypeOf(&Boolean{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*Boolean)))
				case reflect.TypeOf(&ByteArray{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*ByteArray)))
				case reflect.TypeOf(&File{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*File)))
				case reflect.TypeOf(&Document{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*Document)))
				case reflect.TypeOf(&Image{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*Image)))
				case reflect.TypeOf(&Video{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*Video)))
				case reflect.TypeOf(&Audio{}):
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface().(*Audio)))
				case reflect.TypeOf((*value.Value)(nil)).Elem():
					newMap.SetMapIndex(reflect.ValueOf(mapKey.Interface()), reflect.ValueOf(mapVal.Interface()))
				}
			}
			structField.Set(newMap)

		default:
			// Ensure type compatibility between the field and the value
			if structField.Type() != valReflect.Type() {
				return fmt.Errorf("field %q: type mismatch (expected %s, got %s)", fieldName, structField.Type(), valReflect.Type())
			}
			structField.Set(valReflect)
		}
	}
	return nil
}

// Marshal converts a struct into a Map that represents the struct fields as values.
func Marshal(val any) (d value.Value, err error) {
	v := reflect.ValueOf(val)

	// Dereference pointer if necessary
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()
	m := Map{}

	// Iterate over struct fields
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		fieldName := field.Tag.Get("key") // Get the key from the `key` tag
		if fieldName == "" {
			fieldName = field.Name // Fallback to field name if no tag is present
		}

		switch field.Type.Kind() {
		case reflect.Map:
			// Convert map fields to Map
			mapKeys := fieldValue.MapKeys()
			mapResult := Map{}

			for _, key := range mapKeys {
				mapValue := fieldValue.MapIndex(key)

				if mapValue.Kind() == reflect.Struct {
					mapResult[key.Interface().(string)], err = Marshal(mapValue.Interface())
					if err != nil {
						return nil, err
					}
				} else {
					mapResult[key.Interface().(string)] = mapValue.Interface().(value.Value)
				}
			}
			m[fieldName] = mapResult

		case reflect.Struct:
			// Recursively convert struct fields
			m[fieldName], err = Marshal(fieldValue.Interface())
			if err != nil {
				return nil, err
			}

		case reflect.Slice, reflect.Array:
			// Convert slice/array fields to Array
			elemType := field.Type.Elem()
			arr := make(Array, fieldValue.Len())

			if elemType.Kind() == reflect.Struct {
				for j := 0; j < fieldValue.Len(); j++ {
					arr[j], err = Marshal(fieldValue.Index(j).Interface())
					if err != nil {
						return nil, err
					}
				}
			} else {
				for j := 0; j < fieldValue.Len(); j++ {
					arr[j] = fieldValue.Index(j).Interface().(value.Value)
				}
			}
			m[fieldName] = arr

		default:
			// Convert basic types to value.Value
			m[fieldName] = fieldValue.Interface().(value.Value)
		}
	}
	return m, nil
}
