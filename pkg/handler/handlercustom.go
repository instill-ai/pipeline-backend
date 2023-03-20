package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-redis/redis/v9"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/db"
	"github.com/instill-ai/pipeline-backend/internal/external"
	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/internal/sterr"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	modelPB "github.com/instill-ai/protogen-go/vdp/model/v1alpha"
)

// HandleTriggerPipelineBinaryFileUpload is for POST multipart form data
func HandleTriggerPipelineBinaryFileUpload(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	logger, _ := logger.GetZapLogger()

	contentType := req.Header.Get("Content-Type")

	owner := req.Header.Get("owner")
	id := pathParams["id"]

	if owner == "" {
		st := sterr.CreateErrorBadRequest("[handler] invalid owner field", "owner", "required parameter Jwt-Sub not found in the header")
		errorResponse(w, st)
		logger.Error(st.String())
		return
	}

	if id == "" {
		st := sterr.CreateErrorBadRequest("[handler] invalid id field", "id", "required parameter pipeline id not found in the path")
		errorResponse(w, st)
		logger.Error(st.String())
		return
	}

	if strings.Contains(contentType, "multipart/form-data") {

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

		dbPipeline, err := service.GetPipelineByID(id, owner, false)
		if err != nil {
			st := sterr.CreateErrorResourceInfo(
				codes.NotFound,
				"[handler] cannot get pipeline by id",
				"pipelines",
				fmt.Sprintf("id %s", id),
				owner,
				err.Error(),
			)
			errorResponse(w, st)
			logger.Error(st.String())
			return
		}

		modelInstance, err := service.GetModelInstanceByName(dbPipeline.Recipe.ModelInstances[0])
		if err != nil {
			st := sterr.CreateErrorResourceInfo(
				codes.NotFound,
				"[handler] cannot get pipeline by id",
				"pipelines",
				fmt.Sprintf("id %s", id),
				owner,
				err.Error(),
			)
			errorResponse(w, st)
			logger.Error(st.String())
			return
		}

		if err := req.ParseMultipartForm(4 << 20); err != nil {
			st := sterr.CreateErrorPreconditionFailure(
				"[handler] error while get model instance information",
				"TriggerPipelineBinaryFileUpload",
				fmt.Sprintf("id %s", id),
				err.Error(),
			)
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
			st := sterr.CreateErrorPreconditionFailure(
				"[handler] error while reading file from request",
				"TriggerPipelineBinaryFileUpload",
				fmt.Sprintf("id %s", id),
				err.Error(),
			)
			errorResponse(w, st)
			logger.Error(st.String())
			return
		}

		obj, err := service.TriggerPipelineBinaryFileUpload(dbPipeline, modelInstance.Task, inp)
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

	} else {
		st := sterr.CreateErrorPreconditionFailure(
			"[handler] content-type not supported",
			"TriggerPipelineBinaryFileUpload",
			fmt.Sprintf("id %s", id),
			fmt.Sprintf("content-type %s not supported", contentType),
		)
		errorResponse(w, st)
		logger.Error(st.String())
		return
	}
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
