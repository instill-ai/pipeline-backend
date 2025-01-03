package base

import (
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// convertFormatFields converts component definition fields from the new format to the legacy format.
// This maintains backwards compatibility while we transition the API to use the new field names.
// The main changes are:
// - uiOrder -> instillUiOrder
// - shortDescription -> instillShortDescription
// This function will be removed once the API is updated to use the new field names directly.
func convertFormatFields(input *structpb.Struct, isCompSpec bool) (*structpb.Struct, error) {
	if input == nil {
		return nil, nil
	}

	output := proto.Clone(input).(*structpb.Struct)
	convertFormatFieldsRecursive(output, isCompSpec)

	return output, nil
}

func convertFormatFieldsRecursive(s *structpb.Struct, isCompSpec bool) {

	if s == nil {
		return
	}

	fields := s.Fields
	for key, value := range fields {

		switch {
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

		// Recursively process nested structures
		switch {
		case value.GetStructValue() != nil:
			convertFormatFieldsRecursive(value.GetStructValue(), isCompSpec)
		case value.GetListValue() != nil:
			for idx := range value.GetListValue().Values {
				if value.GetListValue().Values[idx].GetStructValue() != nil {
					convertFormatFieldsRecursive(value.GetListValue().Values[idx].GetStructValue(), isCompSpec)
				}
			}
		}
	}
}
