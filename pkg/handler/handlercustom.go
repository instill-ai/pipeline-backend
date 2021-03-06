package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/go-redis/redis/v9"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/constant"
	"github.com/instill-ai/pipeline-backend/internal/db"
	"github.com/instill-ai/pipeline-backend/internal/external"
	"github.com/instill-ai/pipeline-backend/internal/sterr"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
)

// HandleTriggerPipelineBinaryFileUpload is for POST multipart form data
func HandleTriggerPipelineBinaryFileUpload(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	contentType := req.Header.Get("Content-Type")

	owner := req.Header.Get("owner")
	id := pathParams["id"]

	if owner == "" {
		st := sterr.CreateErrorBadRequest("[handler] invalid owner field", "owner", "required parameter Jwt-Sub not found in the header")
		errorResponse(w, st)
		return
	}

	if id == "" {
		st := sterr.CreateErrorBadRequest("[handler] invalid id field", "id", "required parameter pipeline id not found in the path")
		errorResponse(w, st)
		return
	}

	if strings.Contains(contentType, "multipart/form-data") {

		userServiceClient, userServiceClientConn := external.InitUserServiceClient()
		defer userServiceClientConn.Close()

		connectorServiceClient, connectorServiceClientConn := external.InitConnectorServiceClient()
		defer connectorServiceClientConn.Close()

		modelServiceClient, modelServiceClientConn := external.InitModelServiceClient()
		defer modelServiceClientConn.Close()

		redisClient := redis.NewClient(&config.Config.Cache.Redis.RedisOptions)
		defer redisClient.Close()

		service := service.NewService(
			repository.NewRepository(db.GetConnection()),
			userServiceClient,
			connectorServiceClient,
			modelServiceClient,
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
			return
		}

		if err := req.ParseMultipartForm(4 << 20); err != nil {
			st := sterr.CreateErrorPreconditionFailure(
				"[handler] error while reading file from request",
				"TriggerPipelineBinaryFileUpload",
				fmt.Sprintf("id %s", id),
				err.Error(),
			)
			errorResponse(w, st)
			return
		}

		fileBytes, fileLengths, err := parseImageFormDataInputsToBytes(req)
		if err != nil {
			st := sterr.CreateErrorPreconditionFailure(
				"[handler] error while reading file from request",
				"TriggerPipelineBinaryFileUpload",
				fmt.Sprintf("id %s", id),
				err.Error(),
			)
			errorResponse(w, st)
			return
		}

		var obj interface{}
		if obj, err = service.TriggerPipelineBinaryFileUpload(*bytes.NewBuffer(fileBytes), fileLengths, dbPipeline); err != nil {
			// TODO: return ResourceInfo error
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		ret, _ := json.Marshal(obj)
		_, _ = w.Write(ret)
	} else {
		st := sterr.CreateErrorPreconditionFailure(
			"[handler] content-type not supported",
			"TriggerPipelineBinaryFileUpload",
			fmt.Sprintf("id %s", id),
			fmt.Sprintf("content-type %s not supported", contentType),
		)
		errorResponse(w, st)
		return
	}
}

func parseImageFormDataInputsToBytes(req *http.Request) (fileBytes []byte, fileLengths []uint64, err error) {
	inputs := req.MultipartForm.File["file"]
	var file multipart.File
	for _, content := range inputs {
		file, err = content.Open()
		defer func() {
			err = file.Close()
		}()

		if err != nil {
			return nil, nil, fmt.Errorf("Unable to open file for image")
		}

		buff := new(bytes.Buffer)
		numBytes, err := buff.ReadFrom(file)
		if err != nil {
			return nil, nil, fmt.Errorf("Unable to read content body from image")
		}

		if numBytes > int64(constant.MaxImageSizeBytes) {
			return nil, nil, fmt.Errorf(
				"Image size must be smaller than %vMB. Got %vMB",
				float32(constant.MaxImageSizeBytes)/float32(constant.MB),
				float32(numBytes)/float32(constant.MB),
			)
		}

		fileBytes = append(fileBytes, buff.Bytes()...)
		fileLengths = append(fileLengths, uint64(buff.Len()))
	}

	return fileBytes, fileLengths, nil
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
