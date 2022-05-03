package handler

import (
	"reflect"

	"github.com/gogo/status"
	"github.com/iancoleman/strcase"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
	fieldmask_utils "github.com/mennanov/fieldmask-utils"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// requiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var requiredFields = []string{"DisplayName", "Recipe"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation for UPDATE rpc
var outputOnlyFields = []string{"Name", "Id", "Mode", "OwnerId", "FullName", "CreateTime", "UpdateTime"}

// Implementation follows https://google.aip.dev/203#required
func checkRequiredFields(pbPipeline *pipelinePB.Pipeline) error {
	for _, field := range requiredFields {
		f := reflect.Indirect(reflect.ValueOf(pbPipeline)).FieldByName(field)
		switch f.Kind() {
		case reflect.String:
			if f.String() == "" {
				return status.Errorf(codes.FailedPrecondition, "Required field %s is not provided", field)
			}
		case reflect.Ptr:
			if f.IsNil() {
				return status.Errorf(codes.FailedPrecondition, "Required field %s is not provided", field)
			}
		}
	}

	return nil
}

// Implementation follows https://google.aip.dev/203#output-only
func checkUpdateMaskForOutputOnlyFields(updateMask *fieldmaskpb.FieldMask) (*fieldmask_utils.Mask, error) {

	mask, err := fieldmask_utils.MaskFromProtoFieldMask(updateMask, strcase.ToCamel)
	if err != nil {
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	for _, field := range outputOnlyFields {
		_, ok := mask.Filter(field)
		if ok {
			delete(mask, field)
		}
	}

	return &mask, nil
}
