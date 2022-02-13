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

	configs "github.com/instill-ai/pipeline-backend/configs"
	database "github.com/instill-ai/pipeline-backend/internal/db"
	metadataUtil "github.com/instill-ai/pipeline-backend/internal/grpc/metadata"
	"github.com/instill-ai/pipeline-backend/internal/logger"
	paginate "github.com/instill-ai/pipeline-backend/internal/paginate"
	"github.com/instill-ai/pipeline-backend/pkg/model"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	"github.com/instill-ai/pipeline-backend/pkg/service"
	modelPB "github.com/instill-ai/protogen-go/model"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	structpb "google.golang.org/protobuf/types/known/structpb"
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
	pipelineService service.PipelineService
	paginateTocken  paginate.TokenGenerator
}

func NewPipelineServiceHandlers(pipelineService service.PipelineService) pipelinePB.PipelineServer {
	return &pipelineServiceHandlers{
		pipelineService: pipelineService,
		paginateTocken:  paginate.TokenGeneratorWithSalt(configs.Config.Server.Paginate.Salt),
	}
}

func (s *pipelineServiceHandlers) Liveness(ctx context.Context, in *emptypb.Empty) (*pipelinePB.HealthCheckResponse, error) {
	return &pipelinePB.HealthCheckResponse{Status: "ok", Code: pipelinePB.HealthCheckResponse_SERVING}, nil
}

func (s *pipelineServiceHandlers) Readiness(ctx context.Context, in *emptypb.Empty) (*pipelinePB.HealthCheckResponse, error) {
	return &pipelinePB.HealthCheckResponse{Status: "ok", Code: pipelinePB.HealthCheckResponse_SERVING}, nil
}

func (s *pipelineServiceHandlers) CreatePipeline(ctx context.Context, in *pipelinePB.CreatePipelineRequest) (*pipelinePB.PipelineInfo, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.PipelineInfo{}, err
	}

	// Covert to model
	entity := model.Pipeline{
		Name:        in.Name,
		Description: in.Description,
		Recipe:      unmarshalRecipe(in.Recipe),
		Active:      in.Active,
		Namespace:   username,
	}

	pipeline, err := s.pipelineService.CreatePipeline(entity)
	if err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusCreated)))

	return marshalPipeline(&pipeline), nil
}

func (s *pipelineServiceHandlers) ListPipelines(ctx context.Context, in *pipelinePB.ListPipelinesRequest) (*pipelinePB.ListPipelinesResponse, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.ListPipelinesResponse{}, err
	}

	cursor, err := s.paginateTocken.Decode(in.PageToken)
	if err != nil {
		return nil, err
	}

	query := model.ListPipelineQuery{
		Namespace:  username,
		WithRecipe: in.View == pipelinePB.ListPipelinesRequest_WITH_RECIPE,
		PageSize:   in.PageSize,
		Cursor:     cursor,
	}

	pipelines, _, min, err := s.pipelineService.ListPipelines(query)
	if err != nil {
		return nil, err
	}

	var resp pipelinePB.ListPipelinesResponse

	var nextCorsor uint64
	for _, pipeline := range pipelines {
		resp.Contents = append(resp.Contents, marshalPipeline(&pipeline))
		nextCorsor = pipeline.Id
	}

	if min != nextCorsor {
		resp.NextPageToken = s.paginateTocken.Encode(nextCorsor)
	}

	return &resp, nil
}

func (s *pipelineServiceHandlers) GetPipeline(ctx context.Context, in *pipelinePB.GetPipelineRequest) (*pipelinePB.PipelineInfo, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.PipelineInfo{}, err
	}

	pipeline, err := s.pipelineService.GetPipelineByName(username, in.Name)
	if err != nil {
		return nil, err
	}

	return marshalPipeline(&pipeline), nil
}

func (s *pipelineServiceHandlers) UpdatePipeline(ctx context.Context, in *pipelinePB.UpdatePipelineRequest) (*pipelinePB.PipelineInfo, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &pipelinePB.PipelineInfo{}, err
	}

	// Covert to model
	entity := model.Pipeline{
		Name:      in.Pipeline.Name,
		Namespace: username,
	}
	if in.UpdateMask != nil && len(in.UpdateMask.Paths) > 0 {
		entity.UpdatedAt = time.Now()

		for _, field := range in.UpdateMask.Paths {
			switch field {
			case "description":
				entity.Description = in.Pipeline.Description
			case "active":
				entity.Active = in.Pipeline.Active
			}
			if strings.Contains(field, "recipe") {
				entity.Recipe = unmarshalRecipe(in.Pipeline.Recipe)
			}
		}
	}

	pipeline, err := s.pipelineService.UpdatePipeline(entity)
	if err != nil {
		return nil, err
	}

	return marshalPipeline(&pipeline), nil
}

func (s *pipelineServiceHandlers) DeletePipeline(ctx context.Context, in *pipelinePB.DeletePipelineRequest) (*emptypb.Empty, error) {

	username, err := getUsername(ctx)
	if err != nil {
		return &emptypb.Empty{}, err
	}

	if err := s.pipelineService.DeletePipeline(username, in.Name); err != nil {
		return nil, err
	}

	// We need to manually set the custom header to have a StatusCreated http response for REST endpoint
	grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusNoContent)))

	return &emptypb.Empty{}, nil
}

func (s *pipelineServiceHandlers) TriggerPipelineByUpload(stream pipelinePB.Pipeline_TriggerPipelineByUploadServer) error {

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

			return status.Errorf(codes.Internal, "failed unexpectadely while reading chunks from stream: %s", err.Error())
		}

		if data.Contents == nil {
			continue
		}

		if len(data.Contents) > 1 {
			return status.Error(codes.InvalidArgument, "only accept upload single file")
		}

		if _, err := buf.Write(data.Contents[0].Chunk); err != nil {
			return status.Errorf(codes.Internal, "failed unexpectadely while reading chunks from stream: %s", err.Error())
		}
	}

	var obj interface{}
	if obj, err = s.pipelineService.TriggerPipelineByUpload(username, buf, pipeline); err != nil {
		return err
	}

	stream.SendAndClose(obj.(*structpb.Struct))

	return nil
}

func HandleUploadOutput(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {

	logger, _ := logger.GetZapLogger()

	contentType := r.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {
		username := r.Header.Get("Username")
		pipelineName := pathParams["name"]

		if username == "" {
			w.Header().Add("Content-Type", "application/json+problem")
			w.WriteHeader(422)
			obj, _ := json.Marshal(model.Error{
				Status: 422,
				Title:  "Required parameter missing",
				Detail: "Required parameter Jwt-Sub not found in your header",
			})
			w.Write(obj)
		}
		if pipelineName == "" {
			w.Header().Add("Content-Type", "application/json+problem")
			w.WriteHeader(422)
			obj, _ := json.Marshal(model.Error{
				Status: 422,
				Title:  "Required parameter missing",
				Detail: "Required parameter pipeline id not found in your path",
			})
			w.Write(obj)
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

		modelServiceClient := modelPB.NewModelClient(clientConn)

		pipelineService := service.NewPipelineService(pipelineRepository, modelServiceClient)

		pipeline, err := pipelineService.GetPipelineByName(username, pipelineName)
		if err != nil {
			w.Header().Add("Content-Type", "application/json+problem")
			w.WriteHeader(400)
			obj, _ := json.Marshal(model.Error{
				Status: 400,
				Title:  "Required parameter missing",
				Detail: "Required parameter pipeline id not found in your path",
			})
			w.Write(obj)
		}

		r.ParseMultipartForm(4 << 20)
		file, _, err := r.FormFile("contents")
		if err != nil {
			w.Header().Add("Content-Type", "application/json+problem")
			w.WriteHeader(400)
			obj, _ := json.Marshal(model.Error{
				Status: 500,
				Title:  "Internal Error",
				Detail: "Error while reading file from request",
			})
			w.Write(obj)
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
			w.Header().Add("Content-Type", "application/json+problem")
			w.WriteHeader(400)
			obj, _ := json.Marshal(model.Error{
				Status: 400,
				Title:  "Error Reading",
			})
			w.Write(obj)
		}

		var obj interface{}
		if obj, err = pipelineService.TriggerPipelineByUpload(username, *buf, pipeline); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(200)
		ret, _ := json.Marshal(obj)
		w.Write(ret)
	} else {
		w.Header().Add("Content-Type", "application/json+problem")
		w.WriteHeader(405)
	}
}
