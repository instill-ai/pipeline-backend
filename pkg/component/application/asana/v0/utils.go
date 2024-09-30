package asana

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-resty/resty/v2"
)

func turnToStringQueryParams(val any) string {
	var stringVal string
	switch val := val.(type) {
	case string:
		stringVal = val
	case int:
		stringVal = fmt.Sprintf("%d", val)
	case bool:
		stringVal = fmt.Sprintf("%t", val)
	case []string:
		stringVal = strings.Join(val, ",")
	case []int:
		var strVals []string
		for _, v := range val {
			strVals = append(strVals, fmt.Sprintf("%d", v))
		}
		stringVal = strings.Join(strVals, ",")
	default:
		return ""
	}
	return stringVal
}

func addQueryOptions(req *resty.Request, opt interface{}) error {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return nil
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			if !v.MapIndex(key).IsValid() || !v.MapIndex(key).CanInterface() {
				continue
			}
			val := v.MapIndex(key).Interface()
			stringVal := turnToStringQueryParams(val)
			if stringVal == fmt.Sprintf("%v", reflect.Zero(reflect.TypeOf(val))) {
				continue
			}
			paramName := key.String()
			req.SetQueryParam(paramName, stringVal)
		}
	} else if v.Kind() == reflect.Struct {
		typeOfS := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).IsValid() || !v.Field(i).CanInterface() {
				continue
			}
			val := v.Field(i).Interface()
			stringVal := turnToStringQueryParams(val)
			if stringVal == fmt.Sprintf("%v", reflect.Zero(reflect.TypeOf(val))) {
				continue
			}
			paramName := typeOfS.Field(i).Tag.Get("api")
			if paramName == "" {
				paramName = typeOfS.Field(i).Name
			}
			req.SetQueryParam(paramName, stringVal)
		}
	}
	return nil
}

func parseWantOptionFields(opt interface{}) string {
	v := reflect.ValueOf(opt)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return ""
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	var options string
	if v.Kind() == reflect.Map {
		for _, key := range v.MapKeys() {
			if !v.MapIndex(key).IsValid() || !v.MapIndex(key).CanInterface() {
				continue
			}
			options = fmt.Sprintf("%s,%s", options, key.String())
		}
	} else if v.Kind() == reflect.Struct {
		typeOfS := v.Type()
		for i := 0; i < v.NumField(); i++ {
			if !v.Field(i).IsValid() || !v.Field(i).CanInterface() {
				continue
			}
			paramName := typeOfS.Field(i).Tag.Get("api")
			if paramName == "" {
				paramName = typeOfS.Field(i).Name
			}
			options = fmt.Sprintf("%s,%s", options, paramName)
		}
	}
	options = strings.TrimLeft(options, ",")
	return options
}
