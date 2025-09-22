package data

import (
	"container/list"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/iancoleman/strcase"

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
//     Age      *int `instill:"age,default=18"`          // Will use 18 as default if nil
//   }
//
// The format portion of the tag supports:
//   - For Image: "image/png", "image/jpeg", etc
//   - For Video: "video/mp4", "video/webm", etc
//   - For Audio: "audio/mpeg", "audio/wav", etc
//   - For Document: "application/pdf", "text/plain", etc
//   - For pointers: "default=value" to specify default value when nil

// fieldMappingCache implements an LRU cache for struct field name mappings
type fieldMappingCache struct {
	cache   map[reflect.Type]map[string]string // Type -> FieldName -> ResolvedName
	lru     *list.List                         // LRU list for eviction
	items   map[reflect.Type]*list.Element     // Type -> List element mapping
	maxSize int                                // Maximum number of cached types
	mu      sync.RWMutex                       // Read-write mutex for thread safety
}

// newFieldMappingCache creates a new LRU cache with the specified maximum size
func newFieldMappingCache(maxSize int) *fieldMappingCache {
	return &fieldMappingCache{
		cache:   make(map[reflect.Type]map[string]string),
		lru:     list.New(),
		items:   make(map[reflect.Type]*list.Element),
		maxSize: maxSize,
	}
}

// get retrieves field mappings for a struct type from cache
func (c *fieldMappingCache) get(structType reflect.Type) (map[string]string, bool) {
	if c == nil {
		return nil, false
	}
	c.mu.RLock()
	mappings, exists := c.cache[structType]
	if exists {
		// Move to front (most recently used)
		c.lru.MoveToFront(c.items[structType])
	}
	c.mu.RUnlock()
	return mappings, exists
}

// set stores field mappings for a struct type in cache
func (c *fieldMappingCache) set(structType reflect.Type, mappings map[string]string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	// If already exists, update and move to front
	if elem, exists := c.items[structType]; exists {
		c.cache[structType] = mappings
		c.lru.MoveToFront(elem)
		return
	}

	// Check if we need to evict
	if c.lru.Len() >= c.maxSize {
		// Remove least recently used
		oldest := c.lru.Back()
		if oldest != nil {
			oldestType := oldest.Value.(reflect.Type)
			delete(c.cache, oldestType)
			delete(c.items, oldestType)
			c.lru.Remove(oldest)
		}
	}

	// Add new entry
	c.cache[structType] = mappings
	elem := c.lru.PushFront(structType)
	c.items[structType] = elem
}

// Marshaler is used to marshal a struct into a Map.
type Marshaler struct {
	fieldCache *fieldMappingCache
}

// Unmarshaler is used to unmarshal data into a struct.
type Unmarshaler struct {
	binaryFetcher external.BinaryFetcher
	fieldCache    *fieldMappingCache
}

// NewMarshaler creates a new Marshaler with field name caching enabled.
func NewMarshaler() *Marshaler {
	return &Marshaler{
		fieldCache: newFieldMappingCache(200), // Cache up to 200 struct types
	}
}

// NewUnmarshaler creates a new Unmarshaler with automatic naming convention detection.
// The unmarshaler automatically detects the correct naming convention for each field
// based on available input data, providing seamless integration with any external package.
// Field name mappings are cached for improved performance on repeated operations.
func NewUnmarshaler(binaryFetcher external.BinaryFetcher) *Unmarshaler {
	return &Unmarshaler{
		binaryFetcher: binaryFetcher,
		fieldCache:    newFieldMappingCache(200), // Cache up to 200 struct types
	}
}

// Unmarshal converts a Map value into the provided struct s using `instill` tags.
func (u *Unmarshaler) Unmarshal(ctx context.Context, d format.Value, s any) error {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Ptr {
		return errors.New("input must be a pointer")
	}

	elem := v.Elem()

	// Handle both direct structs and embedded structs
	switch elem.Kind() {
	case reflect.Struct:
		// Direct struct case
		m, ok := d.(Map)
		if !ok {
			return errors.New("input value must be a Map")
		}
		return u.unmarshalStruct(ctx, m, elem)
	case reflect.Interface:
		// Handle interface case
		if elem.IsNil() {
			return errors.New("input interface is nil")
		}
		return u.Unmarshal(ctx, d, elem.Interface())
	default:
		return fmt.Errorf("input must be a pointer to a struct, got pointer to %v", elem.Kind())
	}
}

// unmarshalStruct iterates through struct fields and unmarshals corresponding values.
func (u *Unmarshaler) unmarshalStruct(ctx context.Context, m Map, v reflect.Value) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Handle embedded structs by flattening their fields
		if field.Anonymous && fieldValue.Kind() == reflect.Struct {
			// Iterate through the embedded struct's fields
			for j := 0; j < fieldValue.NumField(); j++ {
				embField := fieldValue.Type().Field(j)

				embValue := fieldValue.Field(j)

				if !embValue.CanSet() {
					continue
				}

				// Get the field name from the embedded struct's field
				embFieldName := u.getFieldNameFromMap(embField, m)
				if val, ok := m[embFieldName]; ok {
					if err := u.unmarshalValue(ctx, val, embValue, embField); err != nil {
						return fmt.Errorf("error unmarshaling embedded field %s: %w", embFieldName, err)
					}
				}
			}
			continue
		}

		if !fieldValue.CanSet() {
			continue
		}

		fieldName := u.getFieldNameFromMap(field, m)
		val, ok := m[fieldName]
		if !ok {
			// Check for default value if field is nil pointer or zero value
			if (fieldValue.Kind() == reflect.Ptr && fieldValue.IsNil()) ||
				fieldValue.IsZero() {
				tag := field.Tag.Get("instill")
				parts := strings.Split(tag, ",")
				for _, part := range parts {
					if strings.HasPrefix(part, "default=") {
						defaultVal := strings.TrimPrefix(part, "default=")
						if err := u.setDefaultValue(fieldValue, defaultVal); err != nil {
							return fmt.Errorf("error setting default value for field %s: %w", fieldName, err)
						}
					}
				}
			}
			continue
		}

		if err := u.unmarshalValue(ctx, val, fieldValue, field); err != nil {
			return fmt.Errorf("error unmarshaling field %s: %w", fieldName, err)
		}
	}
	return nil
}

// setDefaultValue sets the default value for a nil pointer field
func (u *Unmarshaler) setDefaultValue(field reflect.Value, defaultVal string) error {
	// Handle format.Value types first
	if field.Type().Implements(reflect.TypeOf((*format.Value)(nil)).Elem()) {
		elemType := field.Type()
		if elemType == reflect.TypeOf((*format.String)(nil)).Elem() {
			field.Set(reflect.ValueOf(NewString(defaultVal)))
			return nil
		} else if elemType == reflect.TypeOf((*format.Number)(nil)).Elem() {
			f, err := strconv.ParseFloat(defaultVal, 64)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(NewNumberFromFloat(f)))
			return nil
		} else if elemType == reflect.TypeOf((*format.Boolean)(nil)).Elem() {
			b, err := strconv.ParseBool(defaultVal)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(NewBoolean(b)))
			return nil
		}
		return fmt.Errorf("unsupported format.Value type: %v", elemType)
	}

	// Handle pointer types
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		field = field.Elem()
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultVal)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(defaultVal, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		i, err := strconv.ParseUint(defaultVal, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(i)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(defaultVal, 64)
		if err != nil {
			return err
		}
		field.SetFloat(f)
	case reflect.Bool:
		b, err := strconv.ParseBool(defaultVal)
		if err != nil {
			return err
		}
		field.SetBool(b)
	default:
		// Handle special types that don't match basic kinds
		switch field.Type() {
		case reflect.TypeOf(time.Duration(0)):
			duration, err := time.ParseDuration(defaultVal)
			if err != nil {
				return fmt.Errorf("cannot parse default duration %q: %w", defaultVal, err)
			}
			field.Set(reflect.ValueOf(duration))
		case reflect.TypeOf(time.Time{}):
			// Try multiple time formats for default values
			parsedTime, err := parseTimeValue(defaultVal, "")
			if err != nil {
				return fmt.Errorf("cannot parse default time %q: %w", defaultVal, err)
			}
			field.Set(reflect.ValueOf(parsedTime))
		default:
			return fmt.Errorf("unsupported default value type: %v", field.Type())
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
		return u.unmarshalString(ctx, v, field, structField)
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

// parseInstillTag parses the instill tag and returns field name, format, pattern, and other attributes
func parseInstillTag(tag string) (fieldName, format, pattern string, attributes map[string]string) {
	attributes = make(map[string]string)
	if tag == "" {
		return
	}

	// First, extract the field name (everything before the first comma)
	firstCommaIdx := strings.Index(tag, ",")
	if firstCommaIdx == -1 {
		fieldName = tag
		return
	}

	fieldName = tag[:firstCommaIdx]
	remaining := tag[firstCommaIdx+1:]

	// Parse the remaining attributes using a simple approach that handles patterns better
	parts := strings.Split(remaining, ",")

	for i := 0; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if part == "" {
			continue
		}

		switch {
		case strings.HasPrefix(part, "default="):
			attributes["default"] = strings.TrimPrefix(part, "default=")
		case strings.HasPrefix(part, "pattern="):
			// For patterns, we may need to rejoin parts if the pattern contains commas
			patternValue := strings.TrimPrefix(part, "pattern=")
			// Check if this looks like an incomplete regex (missing closing bracket/paren)
			if strings.Contains(patternValue, "(") && !strings.Contains(patternValue, ")") && i+1 < len(parts) {
				// Likely a pattern split by comma, rejoin with next parts until we find a closing paren or end
				for j := i + 1; j < len(parts); j++ {
					patternValue += "," + parts[j]
					if strings.Contains(parts[j], ")") {
						i = j // Skip the parts we've consumed
						break
					}
				}
			}
			pattern = patternValue
		case strings.HasPrefix(part, "format="):
			format = strings.TrimPrefix(part, "format=")
		case strings.Contains(part, "/") && !strings.Contains(part, "="):
			// Legacy format specification without "format=" prefix
			format = part
		}
	}

	return
}

// validatePattern validates a string against a regex pattern
func validatePattern(value, pattern string) error {
	if pattern == "" {
		return nil
	}

	// Unescape the pattern (convert \\. to \.)
	unescapedPattern := strings.ReplaceAll(pattern, "\\\\", "\\")

	regex, err := regexp.Compile(unescapedPattern)
	if err != nil {
		return fmt.Errorf("invalid pattern %q: %w", pattern, err)
	}

	if !regex.MatchString(value) {
		return fmt.Errorf("value %q does not match pattern %q", value, pattern)
	}

	return nil
}

// parseTimeValue parses a time string using appropriate formats based on the format hint
func parseTimeValue(timeStr, format string) (time.Time, error) {
	var timeFormats []string

	// If format is "date-time" or similar, prioritize RFC3339 formats
	if format == "date-time" || format == "datetime" {
		timeFormats = []string{
			time.RFC3339,
			time.RFC3339Nano,
		}
	} else {
		// Try multiple time formats
		timeFormats = []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z07:00",
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
	}

	for _, timeFormat := range timeFormats {
		if parsedTime, err := time.Parse(timeFormat, timeStr); err == nil {
			return parsedTime, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time string with any supported format")
}

// isFileType checks if a type is a file-related format type
func isFileType(t reflect.Type) bool {
	fileTypes := []reflect.Type{
		reflect.TypeOf((*format.Image)(nil)).Elem(),
		reflect.TypeOf((*format.Audio)(nil)).Elem(),
		reflect.TypeOf((*format.Video)(nil)).Elem(),
		reflect.TypeOf((*format.Document)(nil)).Elem(),
		reflect.TypeOf((*format.File)(nil)).Elem(),
	}

	for _, fileType := range fileTypes {
		if t == fileType {
			return true
		}
	}
	return false
}

// handleTimePointer handles marshaling of time pointer types
func handleTimePointer(v reflect.Value) (format.Value, bool) {
	elemType := v.Type().Elem()
	switch elemType {
	case reflect.TypeOf(time.Time{}):
		timeVal := v.Interface().(*time.Time)
		return NewString(timeVal.Format(time.RFC3339)), true
	case reflect.TypeOf(time.Duration(0)):
		durationVal := v.Interface().(*time.Duration)
		return NewString(durationVal.String()), true
	}
	return nil, false
}

// unmarshalString handles unmarshaling of String values.
func (u *Unmarshaler) unmarshalString(ctx context.Context, v format.String, field reflect.Value, structField reflect.StructField) error {
	stringValue := v.String()

	// Parse instill tag for validation rules
	_, _, pattern, _ := parseInstillTag(structField.Tag.Get("instill"))

	// Validate against pattern if specified (applies to all string fields)
	if err := validatePattern(stringValue, pattern); err != nil {
		return fmt.Errorf("pattern validation failed: %w", err)
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(stringValue)
	case reflect.Ptr:
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return u.unmarshalString(ctx, v, field.Elem(), structField)
	default:
		// Check if we can unmarshal JSON string into struct
		if field.Kind() == reflect.Struct || (field.Kind() == reflect.Ptr && field.Type().Elem().Kind() == reflect.Struct) {
			// Fast pre-check: only attempt JSON parsing if string looks like JSON
			if len(stringValue) > 1 && (stringValue[0] == '{' || stringValue[0] == '[') {
				// Try to parse the string as JSON and unmarshal into the struct
				if u.tryUnmarshalJSONString(stringValue, field) == nil {
					return nil
				}
			}
			// If not JSON-like or parsing fails, continue with other type handling
		}

		switch field.Type() {
		// Handle time.Duration
		case reflect.TypeOf(time.Duration(0)):
			// Parse instill tag for parsing hints
			_, _, pattern, _ := parseInstillTag(structField.Tag.Get("instill"))

			// If pattern suggests seconds format, parse as seconds
			if pattern != "" && strings.Contains(pattern, "s$") {
				// Pattern suggests seconds format like "3600s" or "3600.5s"
				// Remove the 's' suffix and parse as float, then convert to duration
				if strings.HasSuffix(stringValue, "s") {
					secondsStr := strings.TrimSuffix(stringValue, "s")
					seconds, err := strconv.ParseFloat(secondsStr, 64)
					if err != nil {
						return fmt.Errorf("cannot parse seconds value %q: %w", secondsStr, err)
					}
					duration := time.Duration(seconds * float64(time.Second))
					field.Set(reflect.ValueOf(duration))
				} else {
					return fmt.Errorf("duration string %q does not end with 's' as required by pattern", stringValue)
				}
			} else {
				// No pattern or different pattern, use standard Go duration parsing
				duration, err := time.ParseDuration(stringValue)
				if err != nil {
					return fmt.Errorf("cannot unmarshal string %q into time.Duration: %w", stringValue, err)
				}
				field.Set(reflect.ValueOf(duration))
			}
		// Handle time.Time
		case reflect.TypeOf(time.Time{}):
			// Parse instill tag for format specification
			_, format, _, _ := parseInstillTag(structField.Tag.Get("instill"))

			parsedTime, err := parseTimeValue(stringValue, format)
			if err != nil {
				return fmt.Errorf("cannot unmarshal string %q into time.Time: %w", stringValue, err)
			}
			field.Set(reflect.ValueOf(parsedTime))

		default:
			// Try to create file from URL for media/document types
			if isFileType(field.Type()) {
				f, err := u.createFileFromURL(ctx, field.Type(), v.String())
				if err == nil {
					field.Set(reflect.ValueOf(f))
					return nil
				}
				// If URL creation fails, return a helpful error message
				return fmt.Errorf("cannot unmarshal string into %v: expected valid URL, not base64 string: %w", field.Type(), err)
			}

			// Handle format.Value types
			if field.Type() == reflect.TypeOf(v) ||
				field.Type() == reflect.TypeOf((*format.String)(nil)).Elem() ||
				field.Type() == reflect.TypeOf((*format.Value)(nil)).Elem() {
				field.Set(reflect.ValueOf(v))
				return nil
			}

			return fmt.Errorf("cannot unmarshal String into %v", field.Type())
		}
	}
	return nil
}

func (u *Unmarshaler) createFileFromURL(ctx context.Context, t reflect.Type, url string) (format.Value, error) {
	switch t {
	case reflect.TypeOf((*format.Image)(nil)).Elem():
		return NewImageFromURL(ctx, u.binaryFetcher, url, true)
	case reflect.TypeOf((*format.Audio)(nil)).Elem():
		return NewAudioFromURL(ctx, u.binaryFetcher, url, true)
	case reflect.TypeOf((*format.Video)(nil)).Elem():
		return NewVideoFromURL(ctx, u.binaryFetcher, url, true)
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
		// Special handling for time.Duration - should only accept string format
		if field.Type() == reflect.TypeOf(time.Duration(0)) {
			return fmt.Errorf("cannot unmarshal Number into time.Duration: use string format like \"60s\"")
		}
		field.SetInt(int64(v.Integer()))
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		field.SetUint(uint64(v.Integer()))
	case reflect.Ptr:
		// Special handling for *time.Duration - should only accept string format
		if field.Type().Elem() == reflect.TypeOf(time.Duration(0)) {
			return fmt.Errorf("cannot unmarshal Number into *time.Duration: use string format like \"60s\"")
		}
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
	structType := field.Type()

	// Get cached field mappings for this struct type
	fieldMappings := u.getFieldMappingsForType(structType, v)

	for i := 0; i < field.NumField(); i++ {
		structField := structType.Field(i)
		fieldValue := field.Field(i)
		if !fieldValue.CanSet() {
			continue
		}

		// Use cached mapping instead of computing each time
		fieldName := fieldMappings[structField.Name]
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
					switch formatTag {
					case "application/pdf":
						converted, err := val.PDF()
						if err != nil {
							return err
						}
						field.Set(reflect.ValueOf(converted))
					case "text/plain":
						converted, err := val.Text()
						if err != nil {
							return err
						}
						field.Set(reflect.ValueOf(converted))
					}
				}
			}
		}
		// Ensure assignable types are properly reconstructed to the target interface
		// to avoid panics like: reflect.Set: *data.fileData is not assignable to type format.Document
		// 1) If target field expects a Document, make sure we set a Document
		if field.Type() == reflect.TypeOf((*format.Document)(nil)).Elem() {
			// Already a Document → set directly
			if doc, ok := v.(format.Document); ok {
				field.Set(reflect.ValueOf(doc))
				return nil
			}
			// If it's a raw internal fileData, rebuild explicitly as Document
			if f, ok := v.(*fileData); ok {
				// Prefer constructing a Document directly to avoid misclassification (e.g., octet-stream)
				rebuilt, err := NewDocumentFromBytes(f.raw, f.contentType, f.filename)
				if err != nil {
					return err
				}
				// rebuilt is *documentData which implements format.Document
				field.Set(reflect.ValueOf(rebuilt))
				return nil
			}
			// If it's any File implementation, rebuild from its bytes and content type
			if f, ok := v.(format.File); ok {
				ba, err := f.Binary()
				if err != nil {
					return err
				}
				rebuilt, err := NewDocumentFromBytes(ba.ByteArray(), f.ContentType().String(), f.Filename().String())
				if err != nil {
					return err
				}
				// rebuilt is *documentData which implements format.Document
				field.Set(reflect.ValueOf(rebuilt))
				return nil
			}
			return fmt.Errorf("cannot unmarshal %T into %v (expected Document)", v, field.Type())
		}

		// Fallback: if v is our internal fileData, rebuild to the proper specific type
		if f, ok := v.(*fileData); ok {
			file, err := NewBinaryFromBytes(f.raw, f.contentType, f.filename)
			if err != nil {
				return err
			}
			field.Set(reflect.ValueOf(file))
			return nil
		}

		// If value is assignable to field type, set directly; otherwise error to avoid panic
		val := reflect.ValueOf(v)
		if val.Type().AssignableTo(field.Type()) {
			field.Set(val)
			return nil
		}
		return fmt.Errorf("cannot unmarshal %T into %v", v, field.Type())
	}
	return fmt.Errorf("cannot unmarshal %T into %v", v, field.Type())
}

// getFieldMappingsForType returns cached field name mappings for a struct type.
// If not cached, it computes the mappings using automatic naming convention detection.
func (u *Unmarshaler) getFieldMappingsForType(structType reflect.Type, inputMap Map) map[string]string {
	// Initialize cache if nil (for backward compatibility with tests)
	if u.fieldCache == nil {
		u.fieldCache = newFieldMappingCache(200)
	}

	// Try to get from cache first
	if mappings, exists := u.fieldCache.get(structType); exists {
		return mappings
	}

	// Cache miss - compute mappings for all fields in this struct type
	mappings := make(map[string]string)
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		mappings[field.Name] = u.computeFieldName(field, inputMap)
	}

	// Cache the computed mappings
	u.fieldCache.set(structType, mappings)
	return mappings
}

// computeFieldName computes the field name using automatic naming convention detection.
// This is the core logic that tries different naming conventions and returns the one that exists in the input map.
func (u *Unmarshaler) computeFieldName(field reflect.StructField, inputMap Map) string {
	// First priority: instill tag (always takes precedence)
	if tag := field.Tag.Get("instill"); tag != "" {
		parts := strings.Split(tag, ",")
		return parts[0]
	}

	// Second priority: try json tag with automatic convention detection
	jsonTag := field.Tag.Get("json")
	if jsonTag != "" && jsonTag != "-" {
		parts := strings.Split(jsonTag, ",")
		jsonFieldName := parts[0]
		if jsonFieldName != "" {
			// Try different naming conventions and return the one that exists in input
			conversions := []struct {
				name      string
				converted string
			}{
				{"kebab-case", jsonFieldName},                  // No conversion
				{"camelCase", strcase.ToKebab(jsonFieldName)},  // camelCase -> kebab-case
				{"snake_case", strcase.ToKebab(jsonFieldName)}, // snake_case -> kebab-case
				{"PascalCase", strcase.ToKebab(jsonFieldName)}, // PascalCase -> kebab-case
			}

			for _, conv := range conversions {
				if _, exists := inputMap[conv.converted]; exists {
					return conv.converted
				}
			}
		}
	}

	// Fallback: use field name as-is
	return field.Name
}

// getFieldNameFromMap returns the field name using cached mappings when possible.
// This method maintains backward compatibility while leveraging caching for performance.
func (u *Unmarshaler) getFieldNameFromMap(field reflect.StructField, inputMap Map) string {
	// For single field lookups, we still use the direct computation to avoid
	// computing mappings for the entire struct when only one field is needed.
	// The cache is most beneficial for full struct unmarshaling.
	return u.computeFieldName(field, inputMap)
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

	// Handle special pointer types before dereferencing
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		if timeVal, ok := handleTimePointer(v); ok {
			return timeVal, nil
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
		// Handle special struct types before generic struct marshaling
		switch v.Type() {
		case reflect.TypeOf(time.Time{}):
			// Marshal time.Time as RFC3339 string
			timeVal := v.Interface().(time.Time)
			return NewString(timeVal.Format(time.RFC3339)), nil
		default:
			return m.marshalStruct(v)
		}
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
		// Handle time.Duration before generic int64 handling
		if v.Type() == reflect.TypeOf(time.Duration(0)) {
			durationVal := v.Interface().(time.Duration)
			return NewString(durationVal.String()), nil
		}
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

// getMarshalFieldMappings returns cached field name mappings for marshaling a struct type.
// If not cached, it computes the mappings for consistent kebab-case output.
func (m *Marshaler) getMarshalFieldMappings(structType reflect.Type) map[string]marshalFieldInfo {
	// Initialize cache if nil (for backward compatibility with tests)
	if m.fieldCache == nil {
		m.fieldCache = newFieldMappingCache(200)
	}

	// Try to get from cache first
	if mappings, exists := m.fieldCache.get(structType); exists {
		// Convert cached string mappings to marshalFieldInfo
		result := make(map[string]marshalFieldInfo)
		for fieldName, resolvedName := range mappings {
			// We need to recompute format tags since they're not cached in string form
			field, _ := structType.FieldByName(fieldName)
			var formatTag string
			if instillTag := field.Tag.Get("instill"); instillTag != "" {
				parts := strings.Split(instillTag, ",")
				if len(parts) > 1 {
					formatTag = parts[1]
				}
			}
			result[fieldName] = marshalFieldInfo{
				resolvedName: resolvedName,
				formatTag:    formatTag,
			}
		}
		return result
	}

	// Cache miss - compute mappings for all fields in this struct type
	mappings := make(map[string]string)
	fieldInfos := make(map[string]marshalFieldInfo)

	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		var fieldName string
		var formatTag string

		// First priority: instill tag
		if instillTag := field.Tag.Get("instill"); instillTag != "" {
			parts := strings.Split(instillTag, ",")
			fieldName = parts[0]
			if len(parts) > 1 {
				formatTag = parts[1]
			}
		} else if jsonTag := field.Tag.Get("json"); jsonTag != "" && jsonTag != "-" {
			// Second priority: json tag, convert to kebab-case
			parts := strings.Split(jsonTag, ",")
			jsonFieldName := parts[0]
			if jsonFieldName != "" {
				fieldName = strcase.ToKebab(jsonFieldName)
			} else {
				fieldName = field.Name
			}
		} else {
			// Fallback: use field name as-is
			fieldName = field.Name
		}

		mappings[field.Name] = fieldName
		fieldInfos[field.Name] = marshalFieldInfo{
			resolvedName: fieldName,
			formatTag:    formatTag,
		}
	}

	// Cache the computed mappings (string form for compatibility with cache)
	m.fieldCache.set(structType, mappings)
	return fieldInfos
}

// marshalFieldInfo contains field mapping information for marshaling
type marshalFieldInfo struct {
	resolvedName string
	formatTag    string
}

// marshalStruct handles marshaling of struct values.
func (m *Marshaler) marshalStruct(v reflect.Value) (Map, error) {
	t := v.Type()
	mp := Map{}

	// Get cached field mappings for this struct type
	fieldMappings := m.getMarshalFieldMappings(t)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fieldValue := v.Field(i)

		// Skip unexported fields
		if field.PkgPath != "" {
			continue
		}

		// Use cached mapping
		fieldInfo := fieldMappings[field.Name]
		fieldName := fieldInfo.resolvedName
		formatTag := fieldInfo.formatTag

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
					switch formatTag {
					case "application/pdf":
						converted, err := v.PDF()
						if err != nil {
							return nil, err
						}
						fieldValue = reflect.ValueOf(converted)
					case "text/plain":
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

// tryUnmarshalJSONString attempts to unmarshal a JSON string directly into a struct field
func (u *Unmarshaler) tryUnmarshalJSONString(jsonStr string, field reflect.Value) error {
	// Create a new instance if the field is a nil pointer
	if field.Kind() == reflect.Ptr {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		// For pointer types, unmarshal directly into the pointed-to value
		return json.Unmarshal([]byte(jsonStr), field.Interface())
	}

	// For non-pointer struct types, we need to unmarshal into a temporary value
	// then set it, because we can't get the address of field directly
	tempValue := reflect.New(field.Type())
	if err := json.Unmarshal([]byte(jsonStr), tempValue.Interface()); err != nil {
		return err // Not valid JSON or incompatible struct
	}
	field.Set(tempValue.Elem())
	return nil
}
