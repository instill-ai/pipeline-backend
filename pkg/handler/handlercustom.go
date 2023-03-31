package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-redis/redis/v9"
	"github.com/gofrs/uuid"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/db"
	"github.com/instill-ai/pipeline-backend/pkg/external"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	"github.com/instill-ai/x/sterr"

	mgmtPB "github.com/instill-ai/protogen-go/vdp/mgmt/v1alpha"
	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// HandleTriggerPipelineBinaryFileUpload is for POST multipart form data
func HandleTriggerPipelineBinaryFileUpload(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
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

	mgmtPrivateServiceClient, mgmtPrivateServiceClientConn := external.InitMgmtPrivateServiceClient()
	if mgmtPrivateServiceClientConn != nil {
		defer mgmtPrivateServiceClientConn.Close()
	}

	connectorPublicServiceClient, connectorPublicServiceClientConn := external.InitConnectorPublicServiceClient()
	if connectorPublicServiceClientConn != nil {
		defer connectorPublicServiceClientConn.Close()
	}

	modelPublicServiceClient, modelPublicServiceClientConn := external.InitModelPublicServiceClient()
	if modelPublicServiceClientConn != nil {
		defer modelPublicServiceClientConn.Close()
	}

	redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
	defer redisClient.Close()

	service := service.NewService(
		repository.NewRepository(db.GetConnection()),
		mgmtPrivateServiceClient,
		connectorPublicServiceClient,
		modelPublicServiceClient,
		redisClient,
	)

	var owner *mgmtPB.User

	// Verify if "jwt-sub" is in the header
	headerOwnerUId := req.Header.Get(constant.HeaderOwnerUIDKey)
	if headerOwnerUId != "" {
		_, err := uuid.FromString(headerOwnerUId)
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Unauthenticated,
				"[handler] unauthenticated request",
				"pipelines",
				fmt.Sprintf("id %s", id),
				headerOwnerUId,
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			errorResponse(w, st)
			logger.Error(st.String())
			return
		}

		ownerPermalink := "users/" + headerOwnerUId
		resp, err := service.GetMgmtPrivateServiceClient().LookUpUserAdmin(req.Context(), &mgmtPB.LookUpUserAdminRequest{Permalink: ownerPermalink})
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Unauthenticated,
				"[handler] unauthenticated request",
				"pipelines",
				fmt.Sprintf("id %s", id),
				ownerPermalink,
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
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
			st, err := sterr.CreateErrorBadRequest("[handler] invalid owner field",
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "owner",
						Description: "required parameter Jwt-Sub not found in the header",
					},
				})
			if err != nil {
				logger.Error(err.Error())
			}
			errorResponse(w, st)
			logger.Error(st.String())
			return
		}

		ownerName := "users/" + headerOwnerId
		resp, err := service.GetMgmtPrivateServiceClient().GetUserAdmin(req.Context(), &mgmtPB.GetUserAdminRequest{Name: ownerName})
		if err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Unauthenticated,
				"[handler] unauthenticated request",
				"pipelines",
				fmt.Sprintf("id %s", id),
				ownerName,
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			errorResponse(w, st)
			logger.Error(st.String())
			return
		}
		owner = resp.GetUser()
	}

	dbPipeline, err := service.GetPipelineByID(id, owner, false)
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

	modelInstance, err := service.GetModelInstanceByName(dbPipeline.Recipe.ModelInstances[0])
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
			"[handler] error while get model instance information",
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
	switch modelInstance.Task {
	case modelPB.ModelInstance_TASK_CLASSIFICATION,
		modelPB.ModelInstance_TASK_DETECTION,
		modelPB.ModelInstance_TASK_KEYPOINT,
		modelPB.ModelInstance_TASK_OCR,
		modelPB.ModelInstance_TASK_INSTANCE_SEGMENTATION,
		modelPB.ModelInstance_TASK_SEMANTIC_SEGMENTATION:
		inp, err = parseImageFormDataInputsToBytes(req)
	case modelPB.ModelInstance_TASK_TEXT_TO_IMAGE:
		inp, err = parseImageFormDataTextToImageInputs(req)
	case modelPB.ModelInstance_TASK_TEXT_GENERATION:
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

	obj, err := service.TriggerPipelineBinaryFileUpload(owner, dbPipeline, modelInstance.Task, inp)
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
