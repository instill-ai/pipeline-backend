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
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	"github.com/instill-ai/pipeline-backend/pkg/pubsub"

	pipelinepb "github.com/instill-ai/protogen-go/pipeline/pipeline/v1beta"
	logx "github.com/instill-ai/x/log"
)

// StreamingHandler intercepts pipeline trigger requests to stream the
// response.
type StreamingHandler struct {
	mux    *runtime.ServeMux
	client pipelinepb.PipelinePublicServiceClient
	sub    pubsub.EventSubscriber
}

// NewStreamingHandler returns an initialized StreamingHandler.
func NewStreamingHandler(mux *runtime.ServeMux, cli pipelinepb.PipelinePublicServiceClient, sub pubsub.EventSubscriber) *StreamingHandler {
	return &StreamingHandler{
		mux:    mux,
		client: cli,
		sub:    sub,
	}
}

// HandleTrigger intercepts TriggerNamespacePipeline endpoints.
func (h *StreamingHandler) HandleTrigger(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	ctx := req.Context()

	var sh *streamingHandler
	if req.Header.Get(constant.HeaderAccept) == "text/event-stream" {
		sh = newStreamingHandler(w, h.sub)
	}

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(h.mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, h.mux, req, "/pipeline.pipeline.v1beta.PipelinePublicService/TriggerNamespacePipeline", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*}/trigger"))
	if err != nil {
		runtime.HTTPError(ctx, h.mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		resp, md, err = requestPipelinePublicServiceTriggerNamespacePipeline0form(annotatedContext, inboundMarshaler, h.client, req, pathParams, sh)
		if err != nil {
			runtime.HTTPError(annotatedContext, h.mux, outboundMarshaler, w, req, err)
			return
		}
	} else {
		resp, md, err = requestPipelinePublicServiceTriggerNamespacePipeline0(annotatedContext, inboundMarshaler, h.client, req, pathParams, sh)
		if err != nil {
			runtime.HTTPError(annotatedContext, h.mux, outboundMarshaler, w, req, err)
			return
		}
	}
	// When using `streamHandler`, we should directly close the response once
	// the event stream is completed to prevent redundant events.
	if sh != nil {
		return
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forwardPipelinePublicServiceTriggerNamespacePipeline0(annotatedContext, h.mux, outboundMarshaler, w, req, resp, h.mux.GetForwardResponseOptions()...)
}

// HandleTriggerAsync intercepts TriggerAsyncNamespacePipeline endpoints.
func (h *StreamingHandler) HandleTriggerAsync(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	ctx := req.Context()

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(h.mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, h.mux, req, "/pipeline.pipeline.v1beta.PipelinePublicService/TriggerAsyncNamespacePipeline", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*}/triggerAsync"))
	if err != nil {
		runtime.HTTPError(ctx, h.mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {

		resp, md, err = requestPipelinePublicServiceTriggerAsyncNamespacePipeline0form(annotatedContext, inboundMarshaler, h.client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, h.mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = requestPipelinePublicServiceTriggerAsyncNamespacePipeline0(annotatedContext, inboundMarshaler, h.client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, h.mux, outboundMarshaler, w, req, err)
			return
		}
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forwardPipelinePublicServiceTriggerNamespacePipeline0(annotatedContext, h.mux, outboundMarshaler, w, req, resp, h.mux.GetForwardResponseOptions()...)
}

// HandleTriggerRelease intercepts TriggerNamespacePipelineRelease endpoints.
func (h *StreamingHandler) HandleTriggerRelease(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	ctx := req.Context()
	var sh *streamingHandler
	if req.Header.Get(constant.HeaderAccept) == "text/event-stream" {
		sh = newStreamingHandler(w, h.sub)
	}

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(h.mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, h.mux, req, "/pipeline.pipeline.v1beta.PipelinePublicService/TriggerNamespacePipelineRelease", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*/releases/*}/trigger"))
	if err != nil {
		runtime.HTTPError(ctx, h.mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		resp, md, err = requestPipelinePublicServiceTriggerNamespacePipelineRelease0form(annotatedContext, inboundMarshaler, h.client, req, pathParams, sh)
		if err != nil {
			runtime.HTTPError(annotatedContext, h.mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = requestPipelinePublicServiceTriggerNamespacePipelineRelease0(annotatedContext, inboundMarshaler, h.client, req, pathParams, sh)
		if err != nil {
			runtime.HTTPError(annotatedContext, h.mux, outboundMarshaler, w, req, err)
			return
		}
	}
	// When using `streamHandler`, we should directly close the response once
	// the event stream is completed to prevent redundant events.
	if sh != nil {
		return
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forwardPipelinePublicServiceTriggerNamespacePipelineRelease0(annotatedContext, h.mux, outboundMarshaler, w, req, resp, h.mux.GetForwardResponseOptions()...)
}

// HandleTriggerAsyncRelease intercepts TriggerAsyncNamespacePipelineRelease
// endpoints.
func (h *StreamingHandler) HandleTriggerAsyncRelease(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	ctx := req.Context()

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(h.mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, h.mux, req, "/pipeline.pipeline.v1beta.PipelinePublicService/TriggerAsyncNamespacePipelineRelease", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*/releases/*}/triggerAsync"))
	if err != nil {
		runtime.HTTPError(ctx, h.mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		resp, md, err = requestPipelinePublicServiceTriggerAsyncNamespacePipelineRelease0form(annotatedContext, inboundMarshaler, h.client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, h.mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = requestPipelinePublicServiceTriggerAsyncNamespacePipelineRelease0(annotatedContext, inboundMarshaler, h.client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, h.mux, outboundMarshaler, w, req, err)
			return
		}
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forwardPipelinePublicServiceTriggerNamespacePipelineRelease0(annotatedContext, h.mux, outboundMarshaler, w, req, resp, h.mux.GetForwardResponseOptions()...)
}

var forwardPipelinePublicServiceTriggerNamespacePipeline0 = runtime.ForwardResponseMessage
var forwardPipelinePublicServiceTriggerNamespacePipelineRelease0 = runtime.ForwardResponseMessage

func convertFormData(ctx context.Context, req *http.Request) ([]*pipelinepb.TriggerData, error) {
	err := req.ParseMultipartForm(4 << 20)
	if err != nil {
		return nil, err
	}

	varMap := map[int]map[string]interface{}{}

	maxVarIdx := 0

	for k, v := range req.MultipartForm.Value {
		if strings.HasPrefix(k, "variables[") || strings.HasPrefix(k, "inputs[") {
			if strings.HasPrefix(k, "variables[") {
				k = k[10:]
			} else {
				k = k[7:]
			}

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
			if strings.HasPrefix(k, "variables[") {
				k = k[10:]
			} else {
				k = k[7:]
			}

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

	data := make([]*pipelinepb.TriggerData, maxVarIdx+1)
	for varIdx, inputValue := range varMap {
		data[varIdx] = &pipelinepb.TriggerData{}
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

// ref: the generated protogen-go files
func requestPipelinePublicServiceTriggerNamespacePipeline0(ctx context.Context, marshaler runtime.Marshaler, client pipelinepb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string, sh *streamingHandler) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pipelinepb.TriggerNamespacePipelineRequest
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

	val, ok = pathParams["namespaceID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "namespaceID")
	}
	protoReq.NamespaceId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "namespace_id", err)
	}

	val, ok = pathParams["pipelineID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "pipelineID")
	}
	protoReq.PipelineId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "pipeline_id", err)
	}

	if sh != nil {
		asyncReq := pipelinepb.TriggerAsyncNamespacePipelineRequest{
			NamespaceId: protoReq.NamespaceId,
			PipelineId:  protoReq.PipelineId,
			Inputs:      protoReq.Inputs,
			Data:        protoReq.Data,
		}
		resp, err := client.TriggerAsyncNamespacePipeline(ctx, &asyncReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
		if err != nil {
			sendPipelineError(ctx, sh, err)
			return nil, metadata, err
		}
		triggerID := strings.Split(resp.Operation.Name, "/")[1]
		sh.handle(ctx, triggerID)
		return nil, metadata, nil
	}
	msg, err := client.TriggerNamespacePipeline(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func requestPipelinePublicServiceTriggerNamespacePipeline0form(ctx context.Context, marshaler runtime.Marshaler, client pipelinepb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string, sh *streamingHandler) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	data, err := convertFormData(ctx, req)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "form-data error")
	}
	protoReq := &pipelinepb.TriggerNamespacePipelineRequest{
		Data: data,
	}

	val, ok = pathParams["namespaceID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "namespaceID")
	}
	protoReq.NamespaceId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "namespace_id", err)
	}

	val, ok = pathParams["pipelineID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "pipelineID")
	}
	protoReq.PipelineId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "pipeline_id", err)
	}

	msg, err := client.TriggerNamespacePipeline(ctx, protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

func requestPipelinePublicServiceTriggerAsyncNamespacePipeline0(ctx context.Context, marshaler runtime.Marshaler, client pipelinepb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pipelinepb.TriggerAsyncNamespacePipelineRequest
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

	val, ok = pathParams["namespaceID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "namespaceID")
	}
	protoReq.NamespaceId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "namespace_id", err)
	}

	val, ok = pathParams["pipelineID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "pipelineID")
	}
	protoReq.PipelineId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "pipeline_id", err)
	}

	msg, err := client.TriggerAsyncNamespacePipeline(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func requestPipelinePublicServiceTriggerAsyncNamespacePipeline0form(ctx context.Context, marshaler runtime.Marshaler, client pipelinepb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	data, err := convertFormData(ctx, req)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "form-data error")
	}

	protoReq := &pipelinepb.TriggerAsyncNamespacePipelineRequest{
		Data: data,
	}

	val, ok = pathParams["namespaceID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "namespaceID")
	}
	protoReq.NamespaceId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "namespace_id", err)
	}

	val, ok = pathParams["pipelineID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "pipelineID")
	}
	protoReq.PipelineId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "pipeline_id", err)
	}

	msg, err := client.TriggerAsyncNamespacePipeline(ctx, protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func requestPipelinePublicServiceTriggerNamespacePipelineRelease0(ctx context.Context, marshaler runtime.Marshaler, client pipelinepb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string, sh *streamingHandler) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pipelinepb.TriggerNamespacePipelineReleaseRequest
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

	val, ok = pathParams["namespaceID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "namespaceID")
	}
	protoReq.NamespaceId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "namespace_id", err)
	}

	val, ok = pathParams["pipelineID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "pipelineID")
	}
	protoReq.PipelineId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "pipeline_id", err)
	}

	val, ok = pathParams["releaseID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "releaseID")
	}
	protoReq.ReleaseId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "release_id", err)
	}

	if sh != nil {
		asyncReq := pipelinepb.TriggerAsyncNamespacePipelineReleaseRequest{
			NamespaceId: protoReq.NamespaceId,
			PipelineId:  protoReq.PipelineId,
			ReleaseId:   protoReq.ReleaseId,
			Inputs:      protoReq.Inputs,
			Data:        protoReq.Data,
		}
		resp, err := client.TriggerAsyncNamespacePipelineRelease(ctx, &asyncReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
		if err != nil {
			sendPipelineError(ctx, sh, err)
			return nil, metadata, err
		}
		triggerID := strings.Split(resp.Operation.Name, "/")[1]
		sh.handle(ctx, triggerID)
		return nil, metadata, nil
	}

	msg, err := client.TriggerNamespacePipelineRelease(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func requestPipelinePublicServiceTriggerNamespacePipelineRelease0form(ctx context.Context, marshaler runtime.Marshaler, client pipelinepb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string, sh *streamingHandler) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	data, err := convertFormData(ctx, req)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "form-data error")
	}
	protoReq := &pipelinepb.TriggerNamespacePipelineReleaseRequest{
		Data: data,
	}

	val, ok = pathParams["namespaceID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "namespaceID")
	}
	protoReq.NamespaceId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "namespace_id", err)
	}

	val, ok = pathParams["pipelineID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "pipelineID")
	}
	protoReq.PipelineId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "pipeline_id", err)
	}

	val, ok = pathParams["releaseID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "releaseID")
	}
	protoReq.ReleaseId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "release_id", err)
	}

	msg, err := client.TriggerNamespacePipelineRelease(ctx, protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

func requestPipelinePublicServiceTriggerAsyncNamespacePipelineRelease0(ctx context.Context, marshaler runtime.Marshaler, client pipelinepb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pipelinepb.TriggerAsyncNamespacePipelineReleaseRequest
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

	val, ok = pathParams["namespaceID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "namespaceID")
	}
	protoReq.NamespaceId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "namespace_id", err)
	}

	val, ok = pathParams["pipelineID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "pipelineID")
	}
	protoReq.PipelineId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "pipeline_id", err)
	}

	val, ok = pathParams["releaseID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "releaseID")
	}
	protoReq.ReleaseId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "release_id", err)
	}

	msg, err := client.TriggerAsyncNamespacePipelineRelease(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func requestPipelinePublicServiceTriggerAsyncNamespacePipelineRelease0form(ctx context.Context, marshaler runtime.Marshaler, client pipelinepb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	var (
		val string
		ok  bool
		err error
		_   = err
	)

	data, err := convertFormData(ctx, req)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "form-data error")
	}
	protoReq := &pipelinepb.TriggerAsyncNamespacePipelineReleaseRequest{
		Data: data,
	}

	val, ok = pathParams["namespaceID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "namespaceID")
	}
	protoReq.NamespaceId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "namespace_id", err)
	}

	val, ok = pathParams["pipelineID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "pipelineID")
	}
	protoReq.PipelineId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "pipeline_id", err)
	}

	val, ok = pathParams["releaseID"]
	if !ok {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "missing parameter %s", "releaseID")
	}
	protoReq.ReleaseId, err = runtime.String(val)
	if err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "type mismatch, parameter: %s, error: %v", "release_id", err)
	}

	msg, err := client.TriggerAsyncNamespacePipelineRelease(ctx, protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

type streamingHandler struct {
	writer     http.ResponseWriter
	subscriber pubsub.EventSubscriber
}

func newStreamingHandler(writer http.ResponseWriter, sub pubsub.EventSubscriber) *streamingHandler {
	return &streamingHandler{
		writer:     writer,
		subscriber: sub,
	}
}

// TODO streamingHandler's methods should be merged into StreamingHandler as
// unexported methods.
func (sh *streamingHandler) handle(ctx context.Context, triggerID string) {
	logger, _ := logx.GetZapLogger(ctx)
	logger.Info("StreamingHandler", zap.String("triggerID", triggerID))

	sh.writer.Header().Set("Content-Type", "text/event-stream")
	sh.writer.Header().Set("Cache-Control", "no-cache")
	sh.writer.Header().Set("Connection", "keep-alive")

	topic := pubsub.WorkflowStatusTopic(triggerID)
	sub := sh.subscriber.Subscribe(ctx, topic)
	defer func() {
		if ctx.Err() != nil {
			ctx = context.Background()
		}

		if err := sub.Cleanup(ctx); err != nil {
			logger.Error("Couldn't unsubscribe from topic", zap.Error(err))
		}
	}()

	ch := sub.Channel()
	for {
		var event pubsub.Event
		select {
		case <-ctx.Done():
			logger.Error("Context cancelled while waiting for event", zap.Error(ctx.Err()))
			return
		case event = <-ch:
		}

		if event.Name == string(memory.PipelineClosed) {
			break
		}

		b, err := json.Marshal(event.Data)
		if err != nil {
			logger.Error("Couldn't marshal data", zap.Error(err))
			return
		}

		fmt.Fprintf(sh.writer, "event: %s\n", event.Name)
		fmt.Fprintf(sh.writer, "data: %s\n", string(b))
		fmt.Fprintf(sh.writer, "\n")
		if flusher, ok := sh.writer.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

// sendPipelineError is a helper function to send a pipeline error to the client
func sendPipelineError(_ context.Context, sh *streamingHandler, err error) {

	sh.writer.Header().Set("Content-Type", "text/event-stream")
	sh.writer.Header().Set("Cache-Control", "no-cache")
	sh.writer.Header().Set("Connection", "keep-alive")

	startEvent := pubsub.Event{
		Name: string(memory.PipelineStatusUpdated),
		Data: memory.PipelineStatusUpdatedEventData{
			PipelineEventData: memory.PipelineEventData{
				UpdateTime: time.Now(),
				BatchIndex: 0,
				Status: map[memory.PipelineStatusType]bool{
					memory.PipelineStatusStarted:   true,
					memory.PipelineStatusErrored:   false,
					memory.PipelineStatusCompleted: false,
				},
			},
		},
	}
	errEvent := pubsub.Event{
		Name: string(memory.PipelineErrorUpdated),
		Data: memory.PipelineErrorUpdatedEventData{
			PipelineEventData: memory.PipelineEventData{
				UpdateTime: time.Now(),
				BatchIndex: 0,
				Status: map[memory.PipelineStatusType]bool{
					memory.PipelineStatusStarted:   true,
					memory.PipelineStatusErrored:   true,
					memory.PipelineStatusCompleted: false,
				},
			},
			Error: memory.MessageError{
				Message: err.Error(),
			},
		},
	}
	startData, err := json.Marshal(startEvent.Data)
	if err != nil {
		return
	}
	errData, err := json.Marshal(errEvent.Data)
	if err != nil {
		return
	}
	fmt.Fprintf(sh.writer, "event: %s\n", startEvent.Name)
	fmt.Fprintf(sh.writer, "data: %s\n", startData)
	fmt.Fprintf(sh.writer, "\n")
	fmt.Fprintf(sh.writer, "event: %s\n", errEvent.Name)
	fmt.Fprintf(sh.writer, "data: %s\n", errData)
	fmt.Fprintf(sh.writer, "\n")
	if flusher, ok := sh.writer.(http.Flusher); ok {
		flusher.Flush()
	}

}
