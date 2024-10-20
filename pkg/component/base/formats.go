package base

import (
	"encoding/base64"
	"mime"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/santhosh-tekuri/jsonschema/v5"
	"google.golang.org/protobuf/types/known/structpb"
)

type InstillAcceptFormatsCompiler struct{}

func (InstillAcceptFormatsCompiler) Compile(ctx jsonschema.CompilerContext, m map[string]interface{}) (jsonschema.ExtSchema, error) {
	if instillAcceptFormats, ok := m["instillAcceptFormats"]; ok {

		formats := []string{}
		for _, instillAcceptFormat := range instillAcceptFormats.([]interface{}) {
			formats = append(formats, instillAcceptFormat.(string))
		}
		return InstillAcceptFormatsSchema(formats), nil
	}

	return nil, nil
}

type InstillAcceptFormatsSchema []string

func (s InstillAcceptFormatsSchema) Validate(ctx jsonschema.ValidationContext, v interface{}) error {

	// TODO: We should design a better approach to validate the Base64 data.
	switch v := v.(type) {

	case []any:
		// TODO: We should validate the data type as well, not just check if
		// it's an array.
		ok := false
		for _, instillAcceptFormat := range s {
			if strings.HasPrefix(instillAcceptFormat, "array:") {
				ok = true
			}
			if instillAcceptFormat == "semi-structured/*" || instillAcceptFormat == "semi-structured/json" || instillAcceptFormat == "json" ||
				instillAcceptFormat == "*" || instillAcceptFormat == "*/*" {
				ok = true
			}
		}
		if !ok {
			return ctx.Error("instillAcceptFormats", "expected one of %v", s)
		}

		return nil
	case string:
		mimeType := ""
		for _, instillAcceptFormat := range s {

			switch {
			case instillAcceptFormat == "string",
				instillAcceptFormat == "*",
				instillAcceptFormat == "*/*",
				strings.HasPrefix(instillAcceptFormat, "semi-structured"),
				strings.HasPrefix(instillAcceptFormat, "structured"):
				return nil

			// For other types, we assume they are Base64 strings and need to validate the Base64 encoding.
			default:

				b, err := base64.StdEncoding.DecodeString(TrimBase64Mime(v))
				if err != nil {
					return ctx.Error("instillAcceptFormats", "can not decode file")
				}

				mimeType = strings.Split(mimetype.Detect(b).String(), ";")[0]
				if strings.Split(mimeType, "/")[0] == strings.Split(instillAcceptFormat, "/")[0] && strings.Split(instillAcceptFormat, "/")[1] == "*" {
					return nil
				} else if mimeType == instillAcceptFormat {
					return nil
				}
			}
		}
		return ctx.Error("instillAcceptFormats", "expected one of %v, but got %s", s, mimeType)

	default:
		return nil
	}
}

var InstillAcceptFormatsMeta = jsonschema.MustCompileString("instillAcceptFormats.json", `{
	"properties" : {
		"instillAcceptFormats": {
			"type": "array",
			"items": {
				"type": "string"
			}
		}
	}
}`)

type InstillFormatCompiler struct{}

func (InstillFormatCompiler) Compile(ctx jsonschema.CompilerContext, m map[string]interface{}) (jsonschema.ExtSchema, error) {
	if _, ok := m["instillFormat"]; ok {

		return InstillFormatSchema(m["instillFormat"].(string)), nil
	}

	return nil, nil
}

type InstillFormatSchema string

func (s InstillFormatSchema) Validate(ctx jsonschema.ValidationContext, v interface{}) error {

	// TODO: We should design a better approach to validate the Base64 data.
	switch v := v.(type) {

	case string:
		switch {
		case s == "string",
			s == "*",
			s == "*/*",
			strings.HasPrefix(string(s), "semi-structured"),
			strings.HasPrefix(string(s), "structured"):
			return nil

		// For other types, we assume they are Base64 strings and need to validate the Base64 encoding.
		default:
			mimeType := ""
			if !strings.HasPrefix(v, "data:") {
				b, err := base64.StdEncoding.DecodeString(TrimBase64Mime(v))
				if err != nil {
					return ctx.Error("instillFormat", "can not decode file")
				}
				mimeType = strings.Split(mimetype.Detect(b).String(), ";")[0]
			} else {
				mimeType = strings.Split(strings.Split(v, ";")[0], ":")[1]
			}

			if strings.Split(mimeType, "/")[0] == strings.Split(string(s), "/")[0] && strings.Split(string(s), "/")[1] == "*" {
				return nil
			} else if mimeType == string(s) {
				return nil
			} else {
				return ctx.Error("instillFormat", "expected %v, but got %s", s, mimeType)
			}

		}

	default:
		return nil
	}
}

var InstillFormatMeta = jsonschema.MustCompileString("instillFormat.json", `{
	"properties" : {
		"instillFormat": {
			"type": "string"
		}
	}
}`)

func CompileInstillAcceptFormats(sch *structpb.Struct) error {
	var err error
	for k, v := range sch.Fields {
		if v.GetStructValue() != nil {
			err = CompileInstillAcceptFormats(v.GetStructValue())
			if err != nil {
				return err
			}
		}
		if k == "instillAcceptFormats" {
			itemInstillAcceptFormats := []interface{}{}
			for _, item := range v.GetListValue().AsSlice() {
				if strings.HasPrefix(item.(string), "array:") {
					_, itemInstillAcceptFormat, _ := strings.Cut(item.(string), ":")
					itemInstillAcceptFormats = append(itemInstillAcceptFormats, itemInstillAcceptFormat)
				}
			}
			if len(itemInstillAcceptFormats) > 0 {
				if _, ok := sch.Fields["items"]; !ok {
					sch.Fields["items"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})
				}
				sch.Fields["items"].GetStructValue().Fields["instillAcceptFormats"], err = structpb.NewValue(itemInstillAcceptFormats)
				if err != nil {
					return err
				}
			}
		}

	}
	return nil
}

func CompileInstillFormat(sch *structpb.Struct) error {
	var err error
	for k, v := range sch.Fields {
		if v.GetStructValue() != nil {
			err = CompileInstillFormat(v.GetStructValue())
			if err != nil {
				return err
			}
		}
		if k == "instillFormat" {
			if strings.HasPrefix(v.GetStringValue(), "array:") {
				_, itemInstillFormat, _ := strings.Cut(v.GetStringValue(), ":")
				sch.Fields["items"] = structpb.NewStructValue(&structpb.Struct{Fields: make(map[string]*structpb.Value)})
				sch.Fields["items"].GetStructValue().Fields["instillFormat"], err = structpb.NewValue(itemInstillFormat)
				if err != nil {
					return err
				}
			}
		}

	}
	return nil
}

func TrimBase64Mime(b64 string) string {
	splitB64 := strings.Split(b64, ",")
	return splitB64[len(splitB64)-1]
}

// return the extension of the file from the base64 string, in the "jpeg" , "png" format, check with provided header
func GetBase64FileExtension(b64 string) string {
	splitB64 := strings.Split(b64, ",")
	header := splitB64[0]
	header = strings.TrimPrefix(header, "data:")
	header = strings.TrimSuffix(header, ";base64")
	mtype, _, err := mime.ParseMediaType(header)
	if err != nil {
		return err.Error()
	}
	return strings.Split(mtype, "/")[1]
}
