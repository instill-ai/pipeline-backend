package handler

import (
	"bytes"
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
	"github.com/instill-ai/pipeline-backend/internal/constant"
	"github.com/instill-ai/pipeline-backend/internal/db"
	"github.com/instill-ai/pipeline-backend/internal/external"
	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/internal/sterr"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
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

		userServiceClient, userServiceClientConn := external.InitMgmtAdminServiceClient()
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
			logger.Error(st.String())
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
			logger.Error(st.String())
			return
		}

		content, fileNames, fileLengths, err := parseImageFormDataInputsToBytes(req)
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

		obj, err := service.TriggerPipelineBinaryFileUpload(*bytes.NewBuffer(content), fileNames, fileLengths, dbPipeline)
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

func parseImageFormDataInputsToBytes(req *http.Request) (content []byte, fileNames []string, fileLengths []uint64, err error) {

	inputs := req.MultipartForm.File["file"]

	for _, input := range inputs {
		file, err := input.Open()
		defer func() {
			err = file.Close()
		}()

		if err != nil {
			return nil, nil, nil, fmt.Errorf("Unable to open file for image")
		}

		buff := new(bytes.Buffer)
		numBytes, err := buff.ReadFrom(file)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("Unable to read content body from image")
		}
		if numBytes > int64(config.Config.Server.MaxDataSize*constant.MB) {
			return nil, nil, nil, fmt.Errorf(
				"Image size must be smaller than %vMB. Got %vMB",
				config.Config.Server.MaxDataSize,
				float32(numBytes)/float32(constant.MB),
			)
		}

		content = append(content, buff.Bytes()...)
		fileNames = append(fileNames, input.Filename)
		fileLengths = append(fileLengths, uint64(buff.Len()))
	}

	return content, fileNames, fileLengths, nil
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
