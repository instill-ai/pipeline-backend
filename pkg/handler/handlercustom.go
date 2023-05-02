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
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/sterr"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// HandleTriggerPipelineBinaryFileUpload is for POST multipart form data
func HandleTriggerPipelineBinaryFileUpload(s service.Service, w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
	logger, _ := logger.GetZapLogger()

	contentType := req.Header.Get("Content-Type")
	id := pathParams["id"]

	if !strings.Contains(contentType, "multipart/form-data") {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[handler] content-type not supported",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "TriggerPipelineBinaryFileUpload",
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
		return
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
		return
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
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		resp, err := s.GetMgmtPrivateServiceClient().LookUpUserAdmin(ctx, &mgmtPB.LookUpUserAdminRequest{Permalink: ownerPermalink})
		if err != nil {
			logger.Error(err.Error())
			return
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
			return
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
			return
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
			return
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
			return
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
		return
	}

	model, err := s.GetModelByName(owner, service.GetModelsFromRecipe(dbPipeline.Recipe)[0])
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
		return
	}

	if err := req.ParseMultipartForm(4 << 20); err != nil {
		st, err := sterr.CreateErrorPreconditionFailure(
			"[handler] error while get model information",
			[]*errdetails.PreconditionFailure_Violation{
				{
					Type:        "TriggerPipelineBinaryFileUpload",
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
		return
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
					Type:        "TriggerPipelineBinaryFileUpload",
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
		return
	}

	obj, err := s.TriggerPipelineBinaryFileUpload(owner, dbPipeline, model.Task, inp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		logger.Error(err.Error())
		return
	}

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
				case "TriggerPipelineBinaryFileUpload":
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
