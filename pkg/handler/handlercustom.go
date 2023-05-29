package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	custom_otel "github.com/instill-ai/pipeline-backend/pkg/logger/otel"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/pipeline-backend/pkg/utils"
	"github.com/instill-ai/x/sterr"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

func injectSpanInRequest(r *http.Request) *http.Request {
	if len(r.Header["X-B3-Traceid"]) > 0 {
		traceID, _ := trace.TraceIDFromHex(r.Header["X-B3-Traceid"][0])
		spanID, _ := trace.SpanIDFromHex(r.Header["X-B3-Spanid"][0])
		var traceFlags trace.TraceFlags
		if r.Header["X-B3-Sampled"][0] == "1" {
			traceFlags = trace.FlagsSampled
		}

		spanContext := trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: traceFlags,
		})

		ctx := trace.ContextWithSpanContext(r.Context(), spanContext)
		r = r.WithContext(ctx)
	}
	return r
}

// HandleTriggerPipelineBinaryFileUpload is for POST multipart form data
func HandleTriggerPipelineBinaryFileUpload(ctx context.Context, s service.Service, w http.ResponseWriter, req *http.Request, pathParams map[string]string) (*mgmtPB.User, *datamodel.Pipeline, *modelPB.Model, interface{}, bool) {

	logger, _ := logger.GetZapLogger(ctx)

	contentType := req.Header.Get("Content-Type")
	id := pathParams["id"]

	if !strings.Contains(contentType, "multipart/form-data") {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[handler] content-type not supported",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "TriggerSyncPipelineBinaryFileUpload",
					Subject:     fmt.Sprintf("id %s", id),
					Description: fmt.Sprintf("content-type %s not supported", contentType),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		errorResponse(w, st)
		logger.Error(st.String())
		return nil, nil, nil, nil, false
	}

	if id == "" {
		st, err := sterr.CreateErrorBadRequest("[handler] invalid id field",
			[]*errdetails.BadRequest_FieldViolation{
				{
					Field:       "id",
					Description: "required parameter pipeline id not found in the path",
				},
			})
		if err != nil {
			logger.Error(err.Error())
		}
		errorResponse(w, st)
		logger.Error(st.String())
		return nil, nil, nil, nil, false
	}

	var owner *mgmtPB.User

	// Verify if "Authorization" is in the header
	authorization := req.Header.Get(strings.ToLower(constant.HeaderAuthorization))

	// Verify if "jwt-sub" is in the header
	headerOwnerUId := req.Header.Get(constant.HeaderOwnerUIDKey)

	apiToken := strings.Replace(authorization, "Bearer ", "", 1)
	if apiToken != "" {
		ownerPermalink, err := s.GetRedisClient().Get(context.Background(), fmt.Sprintf(constant.AccessTokenKeyFormat, apiToken)).Result()
		if err != nil {
			logger.Error(err.Error())
			return nil, nil, nil, nil, false
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := s.GetMgmtPrivateServiceClient().LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: ownerPermalink})
		if err != nil {
			logger.Error(err.Error())
			return nil, nil, nil, nil, false
		}
		owner = resp.GetUser()
	} else if headerOwnerUId != "" {
		_, err := uuid.FromString(headerOwnerUId)
		if err != nil {
			logger.Error(err.Error())
			st, e := sterr.CreateErrorResourceInfo(
				codes.NotFound,
				"Not found",
				"user",
				fmt.Sprintf("uid %s", headerOwnerUId),
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			errorResponse(w, st)
			logger.Error(st.String())
			return nil, nil, nil, nil, false
		}

		ownerPermalink := "users/" + headerOwnerUId
		resp, err := s.GetMgmtPrivateServiceClient().LookUpUserAdmin(req.Context(), &mgmtPB.LookUpUserAdminRequest{Permalink: ownerPermalink})
		if err != nil {
			logger.Error(err.Error())
			st, e := sterr.CreateErrorResourceInfo(
				codes.NotFound,
				"Not found",
				"user",
				fmt.Sprintf("uid %s", headerOwnerUId),
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			errorResponse(w, st)
			logger.Error(st.String())
			return nil, nil, nil, nil, false
		}
		owner = resp.GetUser()

	} else {
		// Verify "owner-id" in the header if there is no "jwt-sub"
		headerOwnerId := req.Header.Get(constant.HeaderOwnerIDKey)
		if headerOwnerId == "" {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Unauthenticated,
				"Unauthorized",
				"pipeline",
				fmt.Sprintf("id %s", id),
				"",
				"",
			)
			if err != nil {
				logger.Error(err.Error())
			}
			errorResponse(w, st)
			logger.Error(st.String())
			return nil, nil, nil, nil, false
		}

		ownerName := "users/" + headerOwnerId
		resp, err := s.GetMgmtPrivateServiceClient().GetUserAdmin(req.Context(), &mgmtPB.GetUserAdminRequest{Name: ownerName})
		if err != nil {
			logger.Error(err.Error())
			st, e := sterr.CreateErrorResourceInfo(
				codes.NotFound,
				"Not found",
				"user",
				fmt.Sprintf("id %s", headerOwnerId),
				"",
				err.Error(),
			)
			if e != nil {
				logger.Error(e.Error())
			}
			errorResponse(w, st)
			logger.Error(st.String())
			return nil, nil, nil, nil, false
		}
		owner = resp.GetUser()
	}

	dbPipeline, err := s.GetPipelineByID(id, owner, false)
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			"[handler] cannot get pipeline by id",
			"pipelines",
			fmt.Sprintf("id %s", id),
			owner.GetName(),
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		errorResponse(w, st)
		logger.Error(st.String())
		return nil, nil, nil, nil, false
	}

	model, err := s.GetModelByName(owner, utils.GetModelsFromRecipe(dbPipeline.Recipe)[0])
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			"[handler] cannot get pipeline by id",
			"pipelines",
			fmt.Sprintf("id %s", id),
			owner.GetName(),
			err.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		errorResponse(w, st)
		logger.Error(st.String())
		return nil, nil, nil, nil, false
	}

	if err := req.ParseMultipartForm(4 << 20); err != nil {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[handler] error while get model information",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "TriggerSyncPipelineBinaryFileUpload",
					Subject:     fmt.Sprintf("id %s", id),
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		errorResponse(w, st)
		logger.Error(st.String())
		return nil, nil, nil, nil, false
	}

	var inp interface{}
	switch model.Task {
	case modelPB.Model_TASK_CLASSIFICATION,
		modelPB.Model_TASK_DETECTION,
		modelPB.Model_TASK_KEYPOINT,
		modelPB.Model_TASK_OCR,
		modelPB.Model_TASK_INSTANCE_SEGMENTATION,
		modelPB.Model_TASK_SEMANTIC_SEGMENTATION:
		inp, err = parseImageFormDataInputsToBytes(req)
	case modelPB.Model_TASK_TEXT_TO_IMAGE:
		inp, err = parseImageFormDataTextToImageInputs(req)
	case modelPB.Model_TASK_TEXT_GENERATION:
		inp, err = parseTextFormDataTextGenerationInputs(req)
	}
	if err != nil {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[handler] error while reading file from request",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "TriggerSyncPipelineBinaryFileUpload",
					Subject:     fmt.Sprintf("id %s", id),
					Description: err.Error(),
				},
			},
		)
		if err != nil {
			logger.Error(err.Error())
		}
		errorResponse(w, st)
		logger.Error(st.String())
		return nil, nil, nil, nil, false
	}

	return owner, dbPipeline, model, inp, true

}

func getHttpStatus(err error) int {
	s := status.Convert(err)
	var httpStatus int
	switch {
	case s.Code() == codes.FailedPrecondition:
		if len(s.Details()) > 0 {
			switch v := s.Details()[0].(type) {
			case *errdetails.PreconditionFailure:
				switch v.Violations[0].Type {
				case "UPDATE", "DELETE", "STATE", "RENAME", "TRIGGER":
					httpStatus = http.StatusUnprocessableEntity
				}
			}
		} else {
			httpStatus = http.StatusBadRequest
		}
	default:
		httpStatus = runtime.HTTPStatusFromCode(s.Code())
	}
	return httpStatus
}

// HandleTriggerSyncPipelineBinaryFileUpload is for POST multipart form data
func HandleTriggerSyncPipelineBinaryFileUpload(s service.Service, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	req = injectSpanInRequest(req)
	ctx, span := tracer.Start(req.Context(), "HandleTriggerSyncPipelineBinaryFileUpload",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	owner, dbPipeline, model, inp, success := HandleTriggerPipelineBinaryFileUpload(ctx, s, w, req, pathParams)
	if !success {
		return
	}

	obj, err := s.TriggerSyncPipelineBinaryFileUpload(owner, dbPipeline, model.Task, inp)
	if err != nil {
		http.Error(w, err.Error(), getHttpStatus(err))
		logger.Error(err.Error())
		return
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		*owner,
		*dbPipeline,
		"HandleTriggerSyncPipelineBinaryFileUpload",
		true,
		"",
	)))
	custom_otel.SetupSyncTriggerCounter().Add(
		ctx,
		1,
		metric.WithAttributeSet(
			attribute.NewSet(
				attribute.KeyValue{
					Key:   "ownerId",
					Value: attribute.StringValue(owner.Id),
				},
				attribute.KeyValue{
					Key:   "ownerUid",
					Value: attribute.StringValue(*owner.Uid),
				},
				attribute.KeyValue{
					Key:   "pipelineId",
					Value: attribute.StringValue(dbPipeline.ID),
				},
				attribute.KeyValue{
					Key:   "pipelineUid",
					Value: attribute.StringValue(dbPipeline.UID.String()),
				},
				attribute.KeyValue{
					Key:   "response",
					Value: attribute.StringValue(obj.String()),
				},
			),
		),
	)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	ret, _ := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}.Marshal(obj)
	_, _ = w.Write(ret)

}

// HandleTriggerAsyncPipelineBinaryFileUpload is for POST multipart form data
func HandleTriggerAsyncPipelineBinaryFileUpload(s service.Service, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	req = injectSpanInRequest(req)
	ctx, span := tracer.Start(req.Context(), "HandleTriggerAsyncPipelineBinaryFileUpload",
		trace.WithSpanKind(trace.SpanKindServer))
	defer span.End()

	logger, _ := logger.GetZapLogger(ctx)

	owner, dbPipeline, model, inp, success := HandleTriggerPipelineBinaryFileUpload(ctx, s, w, req, pathParams)

	if !success {
		return
	}

	obj, err := s.TriggerAsyncPipelineBinaryFileUpload(ctx, owner, dbPipeline, model.Task, inp)
	if err != nil {
		http.Error(w, err.Error(), getHttpStatus(err))
		logger.Error(err.Error())
		return
	}

	logger.Info(string(utils.ConstructAuditLog(
		span,
		*owner,
		*dbPipeline,
		"HandleTriggerAsyncPipelineBinaryFileUpload",
		true,
		"",
	)))
	custom_otel.SetupAsyncTriggerCounter().Add(
		ctx,
		1,
		metric.WithAttributeSet(
			attribute.NewSet(
				attribute.KeyValue{
					Key:   "ownerId",
					Value: attribute.StringValue(owner.Id),
				},
				attribute.KeyValue{
					Key:   "ownerUid",
					Value: attribute.StringValue(*owner.Uid),
				},
				attribute.KeyValue{
					Key:   "pipelineId",
					Value: attribute.StringValue(dbPipeline.ID),
				},
				attribute.KeyValue{
					Key:   "pipelineUid",
					Value: attribute.StringValue(dbPipeline.UID.String()),
				},
				attribute.KeyValue{
					Key:   "response",
					Value: attribute.StringValue(obj.String()),
				},
			),
		),
	)

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(200)
	ret, _ := protojson.MarshalOptions{
		EmitUnpopulated: true,
		UseProtoNames:   true,
	}.Marshal(obj)
	_, _ = w.Write(ret)

}

func errorResponse(w http.ResponseWriter, s *status.Status) {
	w.Header().Add("Content-Type", "application/problem+json")
	switch {
	case s.Code() == codes.FailedPrecondition:
		if len(s.Details()) > 0 {
			switch v := s.Details()[0].(type) {
			case *errdetails.PreconditionFailure:
				switch v.Violations[0].Type {
				case "TriggerSyncPipelineBinaryFileUpload":
					if strings.Contains(v.Violations[0].Description, "content-type") {
						w.WriteHeader(http.StatusUnsupportedMediaType)
					} else {
						w.WriteHeader(http.StatusUnprocessableEntity)
					}
				}
			}
		}
	default:
		w.WriteHeader(runtime.HTTPStatusFromCode(s.Code()))
	}
	obj, _ := json.Marshal(s.Proto())
	_, _ = w.Write(obj)
}
