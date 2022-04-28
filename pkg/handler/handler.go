package handler

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"github.com/instill-ai/pipeline-backend/configs"
	"github.com/instill-ai/pipeline-backend/internal/db"
	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

type handler struct {
	pipelinePB.UnimplementedPipelineServiceServer
	service service.Service
}

// NewHandler initiates a handler instance
func NewHandler(s service.Service) pipelinePB.PipelineServiceServer {
	return &handler{
		service: s,
	}
}

func (h *handler) Liveness(ctx context.Context, in *pipelinePB.LivenessRequest) (*pipelinePB.LivenessResponse, error) {
	return &pipelinePB.LivenessResponse{
		HealthCheckResponse: &pipelinePB.HealthCheckResponse{
			Status: pipelinePB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *handler) Readiness(ctx context.Context, in *pipelinePB.ReadinessRequest) (*pipelinePB.ReadinessResponse, error) {
	return &pipelinePB.ReadinessResponse{
		HealthCheckResponse: &pipelinePB.HealthCheckResponse{
			Status: pipelinePB.HealthCheckResponse_SERVING_STATUS_SERVING,
		},
	}, nil
}

func (h *handler) CreatePipeline(ctx context.Context, req *pipelinePB.CreatePipelineRequest) (*pipelinePB.CreatePipelineResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &pipelinePB.CreatePipelineResponse{}, err
	}

	dbRecipeByte, err := protojson.Marshal(req.Recipe)
	if err != nil {
		return &pipelinePB.CreatePipelineResponse{}, err
	}

	dbRecipe := datamodel.Recipe{}
	if err := json.Unmarshal(dbRecipeByte, &dbRecipe); err != nil {
		return &pipelinePB.CreatePipelineResponse{}, err
	}

	dbPipeline := &datamodel.Pipeline{
		OwnerID:     ownerID,
		Name:        req.Name,
		Description: req.Description,
		Recipe:      &dbRecipe,
	}

	dbPipeline, err = h.service.CreatePipeline(dbPipeline)
	if err != nil {
		return &pipelinePB.CreatePipelineResponse{}, err
	}

	pbPipeline := convertDBPipelineToPBPipeline(dbPipeline)
	resp := pipelinePB.CreatePipelineResponse{
		Pipeline: pbPipeline,
	}

	// Manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (h *handler) ListPipeline(ctx context.Context, req *pipelinePB.ListPipelineRequest) (*pipelinePB.ListPipelineResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &pipelinePB.ListPipelineResponse{}, err
	}

	dbPipelines, nextPageCursor, err := h.service.ListPipeline(ownerID, req.GetView(), int(req.PageSize), req.PageCursor)
	if err != nil {
		return &pipelinePB.ListPipelineResponse{}, err
	}

	pbPipelines := []*pipelinePB.Pipeline{}
	for _, dbPipeline := range dbPipelines {
		pbPipelines = append(pbPipelines, convertDBPipelineToPBPipeline(&dbPipeline))
	}

	resp := pipelinePB.ListPipelineResponse{
		Pipelines:      pbPipelines,
		NextPageCursor: nextPageCursor,
	}

	return &resp, nil
}

func (h *handler) GetPipeline(ctx context.Context, req *pipelinePB.GetPipelineRequest) (*pipelinePB.GetPipelineResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &pipelinePB.GetPipelineResponse{}, err
	}

	dbPipeline, err := h.service.GetPipeline(ownerID, req.GetName())
	if err != nil {
		return &pipelinePB.GetPipelineResponse{}, err
	}

	pbPipeline := convertDBPipelineToPBPipeline(dbPipeline)
	resp := pipelinePB.GetPipelineResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}

func (h *handler) UpdatePipeline(ctx context.Context, req *pipelinePB.UpdatePipelineRequest) (*pipelinePB.UpdatePipelineResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	dbPipeline := &datamodel.Pipeline{
		OwnerID: ownerID,
		Name:    req.Name,
	}

	if req.FieldMask != nil && len(req.FieldMask.Paths) > 0 {
		dbPipeline.UpdatedAt = time.Now()

		for _, field := range req.FieldMask.Paths {
			switch field {
			case "name":
				dbPipeline.Name = req.PipelinePatch.Name
			case "description":
				dbPipeline.Description = req.PipelinePatch.Description
			case "status":
				dbPipeline.Status = datamodel.PipelineStatus(req.PipelinePatch.Status)
			}
			if strings.Contains(field, "recipe") {

				dbRecipeByte, err := protojson.Marshal(req.PipelinePatch.Recipe)
				if err != nil {
					return &pipelinePB.UpdatePipelineResponse{}, err
				}

				dbRecipe := datamodel.Recipe{}
				if err := json.Unmarshal(dbRecipeByte, &dbRecipe); err != nil {
					return &pipelinePB.UpdatePipelineResponse{}, err
				}

				dbPipeline.Recipe = &dbRecipe
			}
		}
	}

	dbPipeline, err = h.service.UpdatePipeline(ownerID, req.GetName(), dbPipeline)
	if err != nil {
		return nil, err
	}

	pbPipeline := convertDBPipelineToPBPipeline(dbPipeline)
	resp := pipelinePB.UpdatePipelineResponse{
		Pipeline: pbPipeline,
	}

	return &resp, nil
}

func (h *handler) DeletePipeline(ctx context.Context, req *pipelinePB.DeletePipelineRequest) (*pipelinePB.DeletePipelineResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	if err := h.service.DeletePipeline(ownerID, req.GetName()); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	return &pipelinePB.DeletePipelineResponse{}, nil
}

func (h *handler) TriggerPipeline(ctx context.Context, req *pipelinePB.TriggerPipelineRequest) (*pipelinePB.TriggerPipelineResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	pipeline, err := h.service.GetPipeline(ownerID, req.Name)
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	if err := h.service.ValidateTriggerPipeline(ownerID, req.Name, pipeline); err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	triggerModelResp, err := h.service.TriggerPipeline(ownerID, req, pipeline)
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	if triggerModelResp == nil {
		return &pipelinePB.TriggerPipelineResponse{}, nil
	}

	resp := pipelinePB.TriggerPipelineResponse{
		Output: triggerModelResp.Output,
	}

	return &resp, nil

}

func (h *handler) TriggerPipelineBinaryFileUpload(stream pipelinePB.PipelineService_TriggerPipelineBinaryFileUploadServer) error {

	ownerID, err := getOwnerID(stream.Context())
	if err != nil {
		return err
	}

	data, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "Cannot receive trigger info")
	}

	pipeline, err := h.service.GetPipeline(ownerID, data.Name)
	if err != nil {
		return err
	}

	if err := h.service.ValidateTriggerPipeline(ownerID, data.Name, pipeline); err != nil {
		return err
	}

	// Read chuck
	buf := bytes.Buffer{}
	for {
		data, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}

			return status.Errorf(codes.Internal, "failed unexpectedly while reading chunks from stream: %s", err.Error())
		}

		if data.Bytes == nil {
			continue
		}

		if _, err := buf.Write(data.Bytes); err != nil {
			return status.Errorf(codes.Internal, "failed unexpectedly while reading chunks from stream: %s", err.Error())
		}
	}

	var obj *modelPB.TriggerModelBinaryFileUploadResponse
	if obj, err = h.service.TriggerPipelineByUpload(ownerID, buf, pipeline); err != nil {
		return err
	}

	stream.SendAndClose(&pipelinePB.TriggerPipelineBinaryFileUploadResponse{Output: obj.Output})

	return nil
}

func errorResponse(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(status)
	obj, _ := json.Marshal(datamodel.Error{
		Status: int32(status),
		Title:  title,
		Detail: detail,
	})
	_, _ = w.Write(obj)
}

// HandleTriggerPipelineBinaryFileUpload is for POST multipart form data
func HandleTriggerPipelineBinaryFileUpload(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {

	logger, _ := logger.GetZapLogger()

	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {

		ownerIDString := r.Header.Get("owner_id")
		pipelineName := pathParams["name"]

		if ownerIDString == "" {
			errorResponse(w, 400, "Bad Request", "Required parameter Jwt-Sub not found in the header")
			return
		}

		if pipelineName == "" {
			errorResponse(w, 400, "Bad Request", "Required parameter pipeline name not found in the path")
			return
		}

		ownerID, err := uuid.FromString(ownerIDString)
		if err != nil {
			errorResponse(w, 400, "Bad Request", "Required parameter Jwt-Sub is not UUID")
			return
		}

		pipelineRepository := repository.NewRepository(db.GetConnection())

		// Create tls based credential.
		var creds credentials.TransportCredentials
		if configs.Config.Server.HTTPS.Cert != "" && configs.Config.Server.HTTPS.Key != "" {
			creds, err = credentials.NewServerTLSFromFile(configs.Config.Server.HTTPS.Cert, configs.Config.Server.HTTPS.Key)
			if err != nil {
				logger.Fatal(fmt.Sprintf("failed to create credentials: %v", err))
				return
			}
		}

		var modelClientDialOpts grpc.DialOption
		if configs.Config.ModelBackend.TLS {
			modelClientDialOpts = grpc.WithTransportCredentials(creds)
		} else {
			modelClientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
		}

		clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", configs.Config.ModelBackend.Host, configs.Config.ModelBackend.Port), modelClientDialOpts)
		if err != nil {
			logger.Fatal(err.Error())
			return
		}

		modelServiceClient := modelPB.NewModelServiceClient(clientConn)

		service := service.NewService(pipelineRepository, modelServiceClient)

		pipeline, err := service.GetPipeline(ownerID, pipelineName)
		if err != nil {
			errorResponse(w, 400, "Bad Request", "Pipeline not found")
			return
		}

		if err := r.ParseMultipartForm(4 << 20); err != nil {
			errorResponse(w, 500, "Internal Error", "Error while reading file from request")
			return
		}

		file, _, err := r.FormFile("contents")
		if err != nil {
			errorResponse(w, 500, "Internal Error", "Error while reading file from request")
			return
		}
		defer file.Close()

		reader := bufio.NewReader(file)
		buf := bytes.NewBuffer(make([]byte, 0))
		part := make([]byte, 1024)

		count := 0
		for {
			if count, err = reader.Read(part); err != nil {
				break
			}
			buf.Write(part[:count])
		}
		if err != io.EOF {
			errorResponse(w, 500, "Internal Error", "Error while reading response from multipart")
			return
		}

		var obj interface{}
		if obj, err = service.TriggerPipelineByUpload(ownerID, *buf, pipeline); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(200)
		ret, _ := json.Marshal(obj)
		_, _ = w.Write(ret)
	} else {
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(405)
	}
}
