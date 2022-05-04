package handler

import (
	"reflect"
	"regexp"

	"github.com/gogo/status"
	"google.golang.org/grpc/codes"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

// requiredFields are Protobuf message fields with REQUIRED field_behavior annotation
var requiredFields = []string{"Recipe"}

// immutableFields are Protobuf message fields with IMMUTABLE field_behavior annotation
var immutableFields = []string{"Id"}

// outputOnlyFields are Protobuf message fields with OUTPUT_ONLY field_behavior annotation
var outputOnlyFields = []string{"Name", "Uid", "Mode", "State", "Owner", "CreateTime", "UpdateTime"}

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
func checkOutputOnlyFields(pbPipeline *pipelinePB.Pipeline) error {
	for _, field := range outputOnlyFields {
		f := reflect.Indirect(reflect.ValueOf(pbPipeline)).FieldByName(field)
		switch f.Kind() {
		case reflect.Int32:
			reflect.ValueOf(pbPipeline).Elem().FieldByName(field).SetInt(0)
		case reflect.String:
			reflect.ValueOf(pbPipeline).Elem().FieldByName(field).SetString("")
		case reflect.Ptr:
			reflect.ValueOf(pbPipeline).Elem().FieldByName(field).Set(reflect.Zero(f.Type()))
		}
	}
	return nil
}

// Implementation follows https://google.aip.dev/203#immutable
func checkImmutableFields(pbPipelineReq *pipelinePB.Pipeline, pbPipelineToUpdate *pipelinePB.Pipeline) error {
	for _, field := range immutableFields {
		f := reflect.Indirect(reflect.ValueOf(pbPipelineReq)).FieldByName(field)
		switch f.Kind() {
		case reflect.String:
			if f.String() != "" {
				if f.String() != reflect.Indirect(reflect.ValueOf(pbPipelineToUpdate)).FieldByName(field).String() {
					return status.Errorf(codes.InvalidArgument, "Field %s is immutable", field)
				}
			}
		}
	}
	return nil
}

// Implementation follows https://google.aip.dev/122#resource-id-segments
func checkResourceID(id string) error {
	if match, _ := regexp.MatchString("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$", id); !match {
		return status.Error(codes.FailedPrecondition, "The id of pipeline needs to be within ASCII-only 63 characters following RFC-1034 with a regexp (^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$)")
	}
	return nil
}
