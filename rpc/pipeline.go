package rpc

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
	"google.golang.org/grpc/status"

	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/model"
	"github.com/instill-ai/pipeline-backend/pkg/repository"

	configs "github.com/instill-ai/pipeline-backend/configs"
	database "github.com/instill-ai/pipeline-backend/internal/db"
	metadataUtil "github.com/instill-ai/pipeline-backend/internal/grpc/metadata"
	paginate "github.com/instill-ai/pipeline-backend/internal/paginate"
	pipelineService "github.com/instill-ai/pipeline-backend/pkg/service"
	modelPB "github.com/instill-ai/protogen-go/model/v1alpha"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

func getUsername(ctx context.Context) (string, error) {
	if metadatas, ok := metadataUtil.ExtractFromMetadata(ctx, "Username"); ok {
		if len(metadatas) == 0 {
			return "", status.Error(codes.FailedPrecondition, "Username not found in your request")
		}
		return metadatas[0], nil
	} else {
		return "", status.Error(codes.FailedPrecondition, "Error when extract metadata")
	}
}

type pipelineServiceHandlers struct {
	pipelineService pipelineService.Services
	paginateTocken  paginate.TokenGenerator
}

func NewPipelineServiceHandlers(pipelineService pipelineService.Services) pipelinePB.PipelineServiceServer {
	return &pipelineServiceHandlers{
		pipelineService: pipelineService,
		paginateTocken:  paginate.TokenGeneratorWithSalt(configs.Config.Server.Paginate.Salt),
	}
}

func (s *pipelineServiceHandlers) Liveness(ctx context.Context, in *pipelinePB.LivenessRequest) (*pipelinePB.LivenessResponse, error) {
	return &pipelinePB.LivenessResponse{Status: pipelinePB.LivenessResponse_SERVING_STATUS_SERVING}, nil
}

func (s *pipelineServiceHandlers) Readiness(ctx context.Context, in *pipelinePB.ReadinessRequest) (*pipelinePB.ReadinessResponse, error) {
	return &pipelinePB.ReadinessResponse{Status: pipelinePB.ReadinessResponse_SERVING_STATUS_SERVING}, nil
}

func (s *pipelineServiceHandlers) CreatePipeline(ctx context.Context, in *pipelinePB.CreatePipelineRequest) (*pipelinePB.CreatePipelineResponse, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.CreatePipelineResponse{}, err
	}

	// Covert to model
	entity := model.Pipeline{
		Name:        in.Name,
		Description: in.Description,
		Active:      in.Active,
		Namespace:   username,
	}
	if in.Recipe != nil {
		entity.Recipe = unmarshalRecipe(in.Recipe)
	}

	pipeline, err := s.pipelineService.CreatePipeline(entity)
	if err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated))); err != nil {
		return nil, err
	}

	return &pipelinePB.CreatePipelineResponse{Pipeline: marshalPipeline(&pipeline)}, nil
}

func (s *pipelineServiceHandlers) ListPipeline(ctx context.Context, in *pipelinePB.ListPipelineRequest) (*pipelinePB.ListPipelineResponse, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.ListPipelineResponse{}, err
	}

	cursor, err := s.paginateTocken.Decode(in.PageToken)
	if err != nil {
		return nil, err
	}

	query := model.ListPipelineQuery{
		Namespace:  username,
		WithRecipe: in.View == pipelinePB.ListPipelineRequest_VIEW_RECIPE,
		PageSize:   in.PageSize,
		Cursor:     cursor,
	}

	pipelines, _, min, err := s.pipelineService.ListPipelines(query)
	if err != nil {
		return nil, err
	}

	var resp pipelinePB.ListPipelineResponse

	var nextCursor uint64
	for _, pipeline := range pipelines {
		resp.Pipelines = append(resp.Pipelines, marshalPipeline(&pipeline))
		nextCursor = pipeline.Id
	}

	if min != nextCursor {
		resp.NextPageToken = s.paginateTocken.Encode(nextCursor)
	}

	return &resp, nil
}

func (s *pipelineServiceHandlers) GetPipeline(ctx context.Context, in *pipelinePB.GetPipelineRequest) (*pipelinePB.GetPipelineResponse, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.GetPipelineResponse{}, err
	}

	pipeline, err := s.pipelineService.GetPipelineByName(username, in.Name)
	if err != nil {
		return nil, err
	}

	return &pipelinePB.GetPipelineResponse{Pipeline: marshalPipeline(&pipeline)}, nil
}

func (s *pipelineServiceHandlers) UpdatePipeline(ctx context.Context, in *pipelinePB.UpdatePipelineRequest) (*pipelinePB.UpdatePipelineResponse, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	// Covert to model
	entity := model.Pipeline{
		Name:      in.Name,
		Namespace: username,
	}
	if in.FieldMask != nil && len(in.FieldMask.Paths) > 0 {
		entity.UpdatedAt = time.Now()

		for _, field := range in.FieldMask.Paths {
			switch field {
			case "description":
				entity.Description = in.PipelinePatch.Description
			case "active":
				entity.Active = in.PipelinePatch.Active
			}
			if strings.Contains(field, "recipe") {
				entity.Recipe = unmarshalRecipe(in.PipelinePatch.Recipe)
			}
		}
	}

	pipeline, err := s.pipelineService.UpdatePipeline(entity)
	if err != nil {
		return nil, err
	}

	return &pipelinePB.UpdatePipelineResponse{Pipeline: marshalPipeline(&pipeline)}, nil
}

func (s *pipelineServiceHandlers) DeletePipeline(ctx context.Context, in *pipelinePB.DeletePipelineRequest) (*pipelinePB.DeletePipelineResponse, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	if err := s.pipelineService.DeletePipeline(username, in.Name); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent))); err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	return &pipelinePB.DeletePipelineResponse{}, nil
}

func (s *pipelineServiceHandlers) TriggerPipeline(ctx context.Context, in *pipelinePB.TriggerPipelineRequest) (*pipelinePB.TriggerPipelineResponse, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	pipeline, err := s.pipelineService.GetPipelineByName(username, in.Name)
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	if err := s.pipelineService.ValidateTriggerPipeline(username, in.Name, pipeline); err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	if obj, err := s.pipelineService.TriggerPipeline(username, in, pipeline); err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	} else {
		return &pipelinePB.TriggerPipelineResponse{Output: obj.Output}, nil
	}
}

func (s *pipelineServiceHandlers) TriggerPipelineBinaryFileUpload(stream pipelinePB.PipelineService_TriggerPipelineBinaryFileUploadServer) error {

	username, err := getUsername(stream.Context())
	if err != nil {
		return err
	}

	data, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "cannot receive trigger info")
	}

	pipeline, err := s.pipelineService.GetPipelineByName(username, data.Name)
	if err != nil {
		return err
	}

	if err := s.pipelineService.ValidateTriggerPipeline(username, data.Name, pipeline); err != nil {
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

		if data.Chunk == nil {
			continue
		}

		if _, err := buf.Write(data.Chunk); err != nil {
			return status.Errorf(codes.Internal, "failed unexpectedly while reading chunks from stream: %s", err.Error())
		}
	}

	var obj *modelPB.TriggerModelBinaryFileUploadResponse
	if obj, err = s.pipelineService.TriggerPipelineByUpload(username, buf, pipeline); err != nil {
		return err
	}

	stream.SendAndClose(&pipelinePB.TriggerPipelineBinaryFileUploadResponse{Output: obj.Output})

	return nil
}

func errorResponse(w http.ResponseWriter, status int, title string, detail string) {
	w.Header().Add("Content-Type", "application/json+problem")
	w.WriteHeader(status)
	obj, _ := json.Marshal(model.Error{
		Status: int32(status),
		Title:  title,
		Detail: detail,
	})
	_, _ = w.Write(obj)
}

func HandleUploadOutput(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {

	logger, _ := logger.GetZapLogger()

	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		username := r.Header.Get("Username")
		pipelineName := pathParams["name"]

		if username == "" {
			errorResponse(w, 422, "Required parameter missing", "Required parameter Jwt-Sub not found in your header")
		}
		if pipelineName == "" {
			errorResponse(w, 422, "Required parameter missing", "Required parameter pipeline name not found in your path")
		}

		db := database.GetConnection()
		pipelineRepository := repository.NewPipelineRepository(db)

		// Create tls based credential.
		var creds credentials.TransportCredentials
		var err error
		if configs.Config.Server.HTTPS.Enabled {
			creds, err = credentials.NewServerTLSFromFile(configs.Config.Server.HTTPS.Cert, configs.Config.Server.HTTPS.Key)
			if err != nil {
				logger.Fatal(fmt.Sprintf("failed to create credentials: %v", err))
			}
		}

		var modelClientDialOpts grpc.DialOption
		if configs.Config.ModelService.TLS {
			modelClientDialOpts = grpc.WithTransportCredentials(creds)
		} else {
			modelClientDialOpts = grpc.WithTransportCredentials(insecure.NewCredentials())
		}

		clientConn, err := grpc.Dial(fmt.Sprintf("%v:%v", configs.Config.ModelService.Host, configs.Config.ModelService.Port), modelClientDialOpts)
		if err != nil {
			logger.Fatal(err.Error())
		}

		modelServiceClient := modelPB.NewModelServiceClient(clientConn)

		pipelineService := pipelineService.NewPipelineService(pipelineRepository, modelServiceClient)

		pipeline, err := pipelineService.GetPipelineByName(username, pipelineName)
		if err != nil {
			errorResponse(w, 400, "Pipeline not found", "Pipeline not found")
		}

		if err := r.ParseMultipartForm(4 << 20); err != nil {
			errorResponse(w, 500, "Internal Error", "Error while reading file from request")
		}

		file, _, err := r.FormFile("contents")
		if err != nil {
			errorResponse(w, 500, "Internal Error", "Error while reading file from request")
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
		}

		var obj interface{}
		if obj, err = pipelineService.TriggerPipelineByUpload(username, *buf, pipeline); err != nil {
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
