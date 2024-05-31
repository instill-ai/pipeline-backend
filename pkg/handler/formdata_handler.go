//nolint:stylecheck
package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

var forward_PipelinePublicService_TriggerUserPipeline_0 = runtime.ForwardResponseMessage
var forward_PipelinePublicService_TriggerUserPipelineRelease_0 = runtime.ForwardResponseMessage

func convertFormData(ctx context.Context, mux *runtime.ServeMux, req *http.Request) ([]*pb.TriggerData, error) {

	err := req.ParseMultipartForm(4 << 20)
	if err != nil {
		return nil, err
	}

	varMap := map[int]map[string]interface{}{}

	maxVarIdx := 0

	for k, v := range req.MultipartForm.Value {
		if strings.HasPrefix(k, "variables[") || strings.HasPrefix(k, "inputs[") {
			k = k[7:]

			varIdx, err := strconv.Atoi(k[:strings.Index(k, "]")])
			if err != nil {
				return nil, err
			}

			if varIdx > maxVarIdx {
				maxVarIdx = varIdx
			}

			k = k[strings.Index(k, "]")+2:]

			var key string
			isArray := false
			keyIdx := 0
			if strings.Contains(k, "[") {
				key = k[:strings.Index(k, "[")]
				keyIdx, err = strconv.Atoi(k[len(key)+1 : strings.Index(k, "]")])
				if err != nil {
					return nil, err
				}
				isArray = true
			} else {
				key = k
			}

			if _, ok := varMap[varIdx]; !ok {
				varMap[varIdx] = map[string]interface{}{}
			}

			if isArray {
				if _, ok := varMap[varIdx][key]; !ok {
					varMap[varIdx][key] = map[int]interface{}{}
				}
				var b interface{}
				unmarshalErr := json.Unmarshal([]byte(v[0]), &b)
				if unmarshalErr != nil {
					return nil, unmarshalErr
				}
				varMap[varIdx][key].(map[int]interface{})[keyIdx] = b
			} else {
				var b interface{}
				unmarshalErr := json.Unmarshal([]byte(v[0]), &b)
				if unmarshalErr != nil {
					return nil, unmarshalErr
				}
				varMap[varIdx][key] = b
			}

		}
	}

	for k, v := range req.MultipartForm.File {
		if strings.HasPrefix(k, "variables[") || strings.HasPrefix(k, "inputs[") {
			k = k[7:]

			varIdx, err := strconv.Atoi(k[:strings.Index(k, "]")])
			if err != nil {
				return nil, err
			}

			if varIdx > maxVarIdx {
				maxVarIdx = varIdx
			}

			k = k[strings.Index(k, "]")+2:]

			var key string
			isArray := false
			keyIdx := 0
			if strings.Contains(k, "[") {
				key = k[:strings.Index(k, "[")]
				keyIdx, err = strconv.Atoi(k[len(key)+1 : strings.Index(k, "]")])
				if err != nil {
					return nil, err
				}
				isArray = true
			} else {
				key = k
			}

			if _, ok := varMap[varIdx]; !ok {
				varMap[varIdx] = map[string]interface{}{}
			}

			file, err := v[0].Open()
			if err != nil {
				return nil, err
			}

			byteContainer, err := io.ReadAll(file)
			if err != nil {
				return nil, err
			}
			v := fmt.Sprintf("data:%s;base64,%s", v[0].Header.Get("Content-Type"), base64.StdEncoding.EncodeToString(byteContainer))
			if isArray {
				if _, ok := varMap[varIdx][key]; !ok {
					varMap[varIdx][key] = map[int]interface{}{}
				}

				varMap[varIdx][key].(map[int]interface{})[keyIdx] = v
			} else {
				varMap[varIdx][key] = v
			}

		}
	}

	data := make([]*pb.TriggerData, maxVarIdx+1)
	for varIdx, inputValue := range varMap {
		data[varIdx].Variable = &structpb.Struct{
			Fields: map[string]*structpb.Value{},
		}
		for key, value := range inputValue {

			switch value := value.(type) {
			case map[int]interface{}:
				maxItemIdx := 0
				for itemIdx := range value {
					if itemIdx > maxItemIdx {
						maxItemIdx = itemIdx
					}
				}
				vals := make([]interface{}, maxItemIdx+1)
				for itemIdx, itemValue := range value {
					vals[itemIdx] = itemValue
				}

				structVal, err := structpb.NewList(vals)
				if err != nil {
					return nil, err
				}

				data[varIdx].Variable.GetFields()[key] = structpb.NewListValue(structVal)

			default:
				structVal, err := structpb.NewValue(value)
				if err != nil {
					return nil, err
				}
				data[varIdx].Variable.GetFields()[key] = structVal
			}

		}
	}
	return data, nil
}

// HandleTrigger
func HandleTrigger(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/vdp.pipeline.v1beta.PipelinePublicService/TriggerUserPipeline", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*}/trigger"))
	if err != nil {
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		data, err := convertFormData(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err = request_PipelinePublicService_TriggerUserPipeline_0_form(annotatedContext, inboundMarshaler, client, &pb.TriggerUserPipelineRequest{
			Data: data,
		}, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = request_PipelinePublicService_TriggerUserPipeline_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forward_PipelinePublicService_TriggerUserPipeline_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

}

// HandleTriggerAsync
func HandleTriggerAsync(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/vdp.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipeline", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*}/triggerAsync"))
	if err != nil {
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		data, err := convertFormData(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err = request_PipelinePublicService_TriggerAsyncUserPipeline_0_form(annotatedContext, inboundMarshaler, client, &pb.TriggerAsyncUserPipelineRequest{
			Data: data,
		}, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = request_PipelinePublicService_TriggerAsyncUserPipeline_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forward_PipelinePublicService_TriggerUserPipeline_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerUserPipeline_0(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pb.TriggerUserPipelineRequest
	var metadata runtime.ServerMetadata

	newReader, berr := utilities.IOReaderFactory(req.Body)
	if berr != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", berr)
	}
	if err := marshaler.NewDecoder(newReader()).Decode(&protoReq); err != nil && err != io.EOF {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}

	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}

	msg, err := client.TriggerUserPipeline(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerUserPipeline_0_form(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, protoReq *pb.TriggerUserPipelineRequest, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}

	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}

	msg, err := client.TriggerUserPipeline(ctx, protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

func request_PipelinePublicService_TriggerAsyncUserPipeline_0(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pb.TriggerAsyncUserPipelineRequest
	var metadata runtime.ServerMetadata

	newReader, berr := utilities.IOReaderFactory(req.Body)
	if berr != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", berr)
	}
	if err := marshaler.NewDecoder(newReader()).Decode(&protoReq); err != nil && err != io.EOF {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}

	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}

	msg, err := client.TriggerAsyncUserPipeline(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerAsyncUserPipeline_0_form(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, protoReq *pb.TriggerAsyncUserPipelineRequest, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}

	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}

	msg, err := client.TriggerAsyncUserPipeline(ctx, protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// HandleTrigger
func HandleTriggerRelease(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/vdp.pipeline.v1beta.PipelinePublicService/TriggerUserPipelineRelease", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*/releases/*}/trigger"))
	if err != nil {
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		data, err := convertFormData(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err = request_PipelinePublicService_TriggerUserPipelineRelease_0_form(annotatedContext, inboundMarshaler, client, &pb.TriggerUserPipelineReleaseRequest{
			Data: data,
		}, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = request_PipelinePublicService_TriggerUserPipelineRelease_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forward_PipelinePublicService_TriggerUserPipelineRelease_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

}

// HandleTriggerAsync
func HandleTriggerAsyncRelease(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/vdp.pipeline.v1beta.PipelinePublicService/TriggerAsyncUserPipelineRelease", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*/releases/*}/triggerAsync"))
	if err != nil {
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		data, err := convertFormData(ctx, mux, req)
		if err != nil {
			runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
			return
		}
		resp, md, err = request_PipelinePublicService_TriggerAsyncUserPipelineRelease_0_form(annotatedContext, inboundMarshaler, client, &pb.TriggerAsyncUserPipelineReleaseRequest{
			Data: data,
		}, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = request_PipelinePublicService_TriggerAsyncUserPipelineRelease_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forward_PipelinePublicService_TriggerUserPipelineRelease_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerUserPipelineRelease_0(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pb.TriggerUserPipelineReleaseRequest
	var metadata runtime.ServerMetadata

	newReader, berr := utilities.IOReaderFactory(req.Body)
	if berr != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", berr)
	}
	if err := marshaler.NewDecoder(newReader()).Decode(&protoReq); err != nil && err != io.EOF {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}

	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}

	msg, err := client.TriggerUserPipelineRelease(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerUserPipelineRelease_0_form(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, protoReq *pb.TriggerUserPipelineReleaseRequest, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}

	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}

	msg, err := client.TriggerUserPipelineRelease(ctx, protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

func request_PipelinePublicService_TriggerAsyncUserPipelineRelease_0(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pb.TriggerAsyncUserPipelineReleaseRequest
	var metadata runtime.ServerMetadata

	newReader, berr := utilities.IOReaderFactory(req.Body)
	if berr != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", berr)
	}
	if err := marshaler.NewDecoder(newReader()).Decode(&protoReq); err != nil && err != io.EOF {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}

	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}

	msg, err := client.TriggerAsyncUserPipelineRelease(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerAsyncUserPipelineRelease_0_form(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, protoReq *pb.TriggerAsyncUserPipelineReleaseRequest, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	val, ok = pathParams["name"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "name")
	}

	protoReq.Name, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "name", err)
	}

	msg, err := client.TriggerAsyncUserPipelineRelease(ctx, protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}
