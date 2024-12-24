package base

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// convertFormatFields converts component definition fields from the new format to the legacy format.
// This maintains backwards compatibility while we transition the API to use the new field names.
// The main changes are:
// - format -> type + instillFormat
// - acceptFormats -> instillAcceptFormats
// - upstreamTypes -> instillUpstreamTypes
// - uiOrder -> instillUiOrder
// - shortDescription -> instillShortDescription
// This function will be removed once the API is updated to use the new field names directly.
func convertFormatFields(input *structpb.Struct) (*structpb.Struct, error) {
	if input == nil {
		return nil, nil
	}

	output := proto.Clone(input).(*structpb.Struct)
	convertFormatFieldsRecursive(output)

	return output, nil
}

func convertFormatFieldsRecursive(s *structpb.Struct) {
	// return

	if s == nil {
		return
	}

	fields := s.Fields
	for key, value := range fields {

		switch {
		case key == "format":
			fields["type"] = value
			fields["instillFormat"] = value
			delete(fields, "format")
		case key == "acceptFormats":
			fields["instillAcceptFormats"] = value
			delete(fields, "acceptFormats")
		case key == "upstreamTypes":
			fields["instillUpstreamTypes"] = value
			delete(fields, "upstreamTypes")
		case key == "upstreamType":
			fields["instillUpstreamType"] = value
			delete(fields, "upstreamType")
		case key == "uiOrder":
			fields["instillUIOrder"] = value
			delete(fields, "uiOrder")
		case key == "shortDescription":
			fields["instillShortDescription"] = value
			delete(fields, "shortDescription")
		}
		if _, ok := fields["description"]; ok {
			if _, ok := fields["instillShortDescription"]; !ok {
				fields["instillShortDescription"] = fields["description"]
			}
		}
		if _, ok := fields["type"]; !ok {
			if _, ok := fields["instillFormat"]; ok {
				f := fields["instillFormat"].GetStringValue()
				switch f {
				case "file", "document", "image", "audio", "video":
					f = "string"
				}
				fields["type"] = structpb.NewStringValue(f)
			}

			if _, ok := fields["instillAcceptFormats"]; ok {
				f := fields["instillAcceptFormats"].GetListValue().Values[0].GetStringValue()
				switch f {
				case "file", "document", "image", "audio", "video":
					f = "string"
				}
				fields["type"] = structpb.NewStringValue(f)
			}
		}

		// Recursively process nested structures
		switch {
		case value.GetStructValue() != nil:
			convertFormatFieldsRecursive(value.GetStructValue())
		case value.GetListValue() != nil:
			for idx := range value.GetListValue().Values {
				if value.GetListValue().Values[idx].GetStructValue() != nil {
					convertFormatFieldsRecursive(value.GetListValue().Values[idx].GetStructValue())
				}
			}
		}
	}
}
