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

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/memory"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

var forward_PipelinePublicService_TriggerNamespacePipeline_0 = runtime.ForwardResponseMessage
var forward_PipelinePublicService_TriggerNamespacePipelineRelease_0 = runtime.ForwardResponseMessage

type streamingHandlerFunc func(triggerID string) error

func convertFormData(ctx context.Context, req *http.Request) ([]*pb.TriggerData, error) {

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

	data := make([]*pb.TriggerData, maxVarIdx+1)
	for varIdx, inputValue := range varMap {
		data[varIdx] = &pb.TriggerData{}
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
func HandleTrigger(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, w http.ResponseWriter, req *http.Request, pathParams map[string]string, ms memory.MemoryStore) {

	ctx := req.Context()

	var sh streamingHandlerFunc
	if req.Header.Get(constant.HeaderAccept) == "text/event-stream" {
		sh = func(triggerID string) (err error) {

			wfm, err := ms.GetWorkflowMemory(ctx, triggerID)
			if err != nil {
				return err
			}
			defer func() {
				_ = ms.PurgeWorkflowMemory(ctx, triggerID)
			}()
			ch := wfm.ListenEvent(ctx)

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			// defer cancel()
			closed := false
			for !closed {
				select {
				// Check if the main context is canceled to stop the goroutine
				case <-ctx.Done():
					return nil
				case event := <-ch:
					if event.Event == string(memory.PipelineClosed) {
						closed = true
						break
					}

					b, err := json.Marshal(event.Data)
					if err != nil {
						return err
					}
					fmt.Fprintf(w, "event: %s\n", event.Event)
					fmt.Fprintf(w, "data: %s\n", string(b))
					fmt.Fprintf(w, "\n")
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}

				}
			}
			return nil

		}
	}

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/vdp.pipeline.v1beta.PipelinePublicService/TriggerNamespacePipeline", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*}/trigger"))
	if err != nil {
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		resp, md, err = request_PipelinePublicService_TriggerNamespacePipeline_0_form(annotatedContext, inboundMarshaler, client, req, pathParams, sh)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = request_PipelinePublicService_TriggerNamespacePipeline_0(annotatedContext, inboundMarshaler, client, req, pathParams, sh)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
	}
	// When using `streamHandler`, we should directly close the response once
	// the event stream is completed to prevent redundant events.
	if sh != nil {
		return
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forward_PipelinePublicService_TriggerNamespacePipeline_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

}

// HandleTriggerAsync
func HandleTriggerAsync(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, w http.ResponseWriter, req *http.Request, pathParams map[string]string, _ memory.MemoryStore) {

	ctx := req.Context()

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/vdp.pipeline.v1beta.PipelinePublicService/TriggerAsyncNamespacePipeline", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*}/triggerAsync"))
	if err != nil {
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {

		resp, md, err = request_PipelinePublicService_TriggerAsyncNamespacePipeline_0_form(annotatedContext, inboundMarshaler, client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = request_PipelinePublicService_TriggerAsyncNamespacePipeline_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forward_PipelinePublicService_TriggerNamespacePipeline_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerNamespacePipeline_0(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string, sh streamingHandlerFunc) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pb.TriggerNamespacePipelineRequest
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
		asyncReq := pb.TriggerAsyncNamespacePipelineRequest{
			NamespaceId: protoReq.NamespaceId,
			PipelineId:  protoReq.PipelineId,
			Inputs:      protoReq.Inputs,
			Data:        protoReq.Data,
		}
		resp, err := client.TriggerAsyncNamespacePipeline(ctx, &asyncReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
		if err != nil {
			return nil, metadata, err
		}
		triggerID := strings.Split(resp.Operation.Name, "/")[1]
		err = sh(triggerID)
		if err != nil {
			return nil, metadata, err
		}
		return nil, metadata, nil
	}
	msg, err := client.TriggerNamespacePipeline(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerNamespacePipeline_0_form(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string, sh streamingHandlerFunc) (proto.Message, runtime.ServerMetadata, error) {
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
	protoReq := &pb.TriggerNamespacePipelineRequest{
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

func request_PipelinePublicService_TriggerAsyncNamespacePipeline_0(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pb.TriggerAsyncNamespacePipelineRequest
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
func request_PipelinePublicService_TriggerAsyncNamespacePipeline_0_form(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
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

	protoReq := &pb.TriggerAsyncNamespacePipelineRequest{
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

// HandleTrigger
func HandleTriggerRelease(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, w http.ResponseWriter, req *http.Request, pathParams map[string]string, ms memory.MemoryStore) {

	ctx := req.Context()
	var sh streamingHandlerFunc
	if req.Header.Get(constant.HeaderAccept) == "text/event-stream" {
		sh = func(triggerID string) (err error) {

			wfm, err := ms.GetWorkflowMemory(ctx, triggerID)
			if err != nil {
				return err
			}
			defer func() {
				_ = ms.PurgeWorkflowMemory(ctx, triggerID)
			}()
			ch := wfm.ListenEvent(ctx)

			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			closed := false
			for !closed {
				select {
				// Check if the main context is canceled to stop the goroutine
				case <-ctx.Done():
					return nil
				case event := <-ch:
					if event.Event == string(memory.PipelineClosed) {
						closed = true
						break
					}

					b, err := json.Marshal(event.Data)
					if err != nil {
						return err
					}
					fmt.Fprintf(w, "event: %s\n", event.Event)
					fmt.Fprintf(w, "data: %s\n", string(b))
					fmt.Fprintf(w, "\n")
					if flusher, ok := w.(http.Flusher); ok {
						flusher.Flush()
					}

				}
			}
			return nil

		}
	}

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/vdp.pipeline.v1beta.PipelinePublicService/TriggerNamespacePipelineRelease", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*/releases/*}/trigger"))
	if err != nil {
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		resp, md, err = request_PipelinePublicService_TriggerNamespacePipelineRelease_0_form(annotatedContext, inboundMarshaler, client, req, pathParams, sh)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = request_PipelinePublicService_TriggerNamespacePipelineRelease_0(annotatedContext, inboundMarshaler, client, req, pathParams, sh)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
	}
	// When using `streamHandler`, we should directly close the response once
	// the event stream is completed to prevent redundant events.
	if sh != nil {
		return
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forward_PipelinePublicService_TriggerNamespacePipelineRelease_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

}

// HandleTriggerAsync
func HandleTriggerAsyncRelease(mux *runtime.ServeMux, client pb.PipelinePublicServiceClient, w http.ResponseWriter, req *http.Request, pathParams map[string]string, _ memory.MemoryStore) {

	ctx := req.Context()

	inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
	var err error
	var annotatedContext context.Context
	var resp protoreflect.ProtoMessage
	var md runtime.ServerMetadata

	annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, "/vdp.pipeline.v1beta.PipelinePublicService/TriggerAsyncNamespacePipelineRelease", runtime.WithHTTPPathPattern("/v1beta/{name=users/*/pipelines/*/releases/*}/triggerAsync"))
	if err != nil {
		runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		resp, md, err = request_PipelinePublicService_TriggerAsyncNamespacePipelineRelease_0_form(annotatedContext, inboundMarshaler, client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}

	} else {
		resp, md, err = request_PipelinePublicService_TriggerAsyncNamespacePipelineRelease_0(annotatedContext, inboundMarshaler, client, req, pathParams)
		if err != nil {
			runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
			return
		}
	}

	annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)

	forward_PipelinePublicService_TriggerNamespacePipelineRelease_0(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerNamespacePipelineRelease_0(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string, sh streamingHandlerFunc) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pb.TriggerNamespacePipelineReleaseRequest
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
		asyncReq := pb.TriggerAsyncNamespacePipelineReleaseRequest{
			NamespaceId: protoReq.NamespaceId,
			PipelineId:  protoReq.PipelineId,
			ReleaseId:   protoReq.ReleaseId,
			Inputs:      protoReq.Inputs,
			Data:        protoReq.Data,
		}
		resp, err := client.TriggerAsyncNamespacePipelineRelease(ctx, &asyncReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
		if err != nil {
			return nil, metadata, err
		}
		triggerID := strings.Split(resp.Operation.Name, "/")[1]
		err = sh(triggerID)
		if err != nil {
			return nil, metadata, err
		}
		return nil, metadata, nil
	}

	msg, err := client.TriggerNamespacePipelineRelease(ctx, &protoReq, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return msg, metadata, err

}

// ref: the generated protogen-go files
func request_PipelinePublicService_TriggerNamespacePipelineRelease_0_form(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string, sh streamingHandlerFunc) (proto.Message, runtime.ServerMetadata, error) {
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
	protoReq := &pb.TriggerNamespacePipelineReleaseRequest{
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

func request_PipelinePublicService_TriggerAsyncNamespacePipelineRelease_0(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var protoReq pb.TriggerAsyncNamespacePipelineReleaseRequest
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
func request_PipelinePublicService_TriggerAsyncNamespacePipelineRelease_0_form(ctx context.Context, marshaler runtime.Marshaler, client pb.PipelinePublicServiceClient, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
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
	protoReq := &pb.TriggerAsyncNamespacePipelineReleaseRequest{
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
