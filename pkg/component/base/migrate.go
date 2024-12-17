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
	convertFormatFieldsRecursive(output.Fields)

	return output, nil
}

func convertFormatFieldsRecursive(fields map[string]*structpb.Value) {
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
		case key == "uiOrder":
			fields["instillUiOrder"] = value
			delete(fields, "uiOrder")
		case key == "shortDescription":
			fields["instillShortDescription"] = value
			delete(fields, "shortDescription")
		}

		// Recursively process nested structures
		switch {
		case value.GetStructValue() != nil:
			convertFormatFieldsRecursive(value.GetStructValue().Fields)
		case value.GetListValue() != nil:
			for _, elem := range value.GetListValue().Values {
				if elem.GetStructValue() != nil {
					convertFormatFieldsRecursive(elem.GetStructValue().Fields)
				}
			}
		}
	}
}
