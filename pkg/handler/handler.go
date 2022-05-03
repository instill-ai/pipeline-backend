package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/gofrs/uuid"
	"github.com/gogo/status"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	fieldmask_utils "github.com/mennanov/fieldmask-utils"

	"github.com/instill-ai/pipeline-backend/configs"
	"github.com/instill-ai/pipeline-backend/internal/constant"
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

	if err := checkRequiredFields(req.Pipeline); err != nil {
		return &pipelinePB.CreatePipelineResponse{}, err
	}

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &pipelinePB.CreatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.CreatePipeline(PBPipelineToDBPipeline(ownerID, req.GetPipeline()))
	if err != nil {
		// Manually set the custom header to have a StatusBadRequest http response for REST endpoint
		if err := grpc.SetHeader(ctx, metadata.Pairs("x-http-code", strconv.Itoa(http.StatusBadRequest))); err != nil {
			return &pipelinePB.CreatePipelineResponse{Pipeline: &pipelinePB.Pipeline{Recipe: &pipelinePB.Recipe{}}}, err
		}
		return &pipelinePB.CreatePipelineResponse{Pipeline: &pipelinePB.Pipeline{}}, err
	}

	pbPipeline := DBPipelineToPBPipeline(dbPipeline)
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

	dbPipelines, NextPageToken, err := h.service.ListPipeline(ownerID, req.GetView(), int(req.GetPageSize()), req.GetPageToken())
	if err != nil {
		return &pipelinePB.ListPipelineResponse{}, err
	}

	pbPipelines := []*pipelinePB.Pipeline{}
	for _, dbPipeline := range dbPipelines {
		pbPipelines = append(pbPipelines, DBPipelineToPBPipeline(&dbPipeline))
	}

	resp := pipelinePB.ListPipelineResponse{
		Pipelines:     pbPipelines,
		NextPageToken: NextPageToken,
	}

	return &resp, nil
}

func (h *handler) GetPipeline(ctx context.Context, req *pipelinePB.GetPipelineRequest) (*pipelinePB.GetPipelineResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &pipelinePB.GetPipelineResponse{}, err
	}

	name := strings.TrimPrefix(req.GetName(), "pipelines/")

	dbPipeline, err := h.service.GetPipelineByName(name, ownerID)
	if err != nil {
		return &pipelinePB.GetPipelineResponse{}, err
	}

	pbPipeline := DBPipelineToPBPipeline(dbPipeline)
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

	pbPipelineReq := req.GetPipeline()
	pbUpdateMask := req.GetUpdateMask()

	// Validate the field mask
	if !pbUpdateMask.IsValid(pbPipelineReq) {
		return &pipelinePB.UpdatePipelineResponse{}, status.Error(codes.FailedPrecondition, "The update_mask is invalid")
	}

	getResp, err := h.GetPipeline(ctx, &pipelinePB.GetPipelineRequest{Name: pbPipelineReq.GetName()})
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	mask, err := checkUpdateMaskForOutputOnlyFields(pbUpdateMask)
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	if mask.IsEmpty() {
		return &pipelinePB.UpdatePipelineResponse{
			Pipeline: getResp.GetPipeline(),
		}, nil
	}

	pbPipelineToUpdate := getResp.GetPipeline()
	id, err := uuid.FromString(pbPipelineToUpdate.GetId())
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	// Only the fields mentioned in the field mask will be copied to `pbPipelineToUpdate`, other fields are left intact
	err = fieldmask_utils.StructToStruct(mask, pbPipelineReq, pbPipelineToUpdate)
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	dbPipeline, err := h.service.UpdatePipeline(id, ownerID, PBPipelineToDBPipeline(ownerID, pbPipelineToUpdate))
	if err != nil {
		return &pipelinePB.UpdatePipelineResponse{}, err
	}

	resp := pipelinePB.UpdatePipelineResponse{
		Pipeline: DBPipelineToPBPipeline(dbPipeline),
	}

	return &resp, nil
}

func (h *handler) DeletePipeline(ctx context.Context, req *pipelinePB.DeletePipelineRequest) (*pipelinePB.DeletePipelineResponse, error) {

	ownerID, err := getOwnerID(ctx)
	if err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	existPipeline, err := h.GetPipeline(ctx, &pipelinePB.GetPipelineRequest{Name: req.GetName()})
	if err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	id, err := uuid.FromString(existPipeline.GetPipeline().Id)
	if err != nil {
		return &pipelinePB.DeletePipelineResponse{}, err
	}

	if err := h.service.DeletePipeline(id, ownerID); err != nil {
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

	name := strings.TrimPrefix(req.GetName(), "pipelines/")

	dbPipeline, err := h.service.GetPipelineByName(name, ownerID)
	if err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	if err := h.service.ValidatePipeline(dbPipeline); err != nil {
		return &pipelinePB.TriggerPipelineResponse{}, err
	}

	triggerModelResp, err := h.service.TriggerPipeline(req, dbPipeline)
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

	name := strings.TrimPrefix(data.GetName(), "pipelines/")

	dbPipeline, err := h.service.GetPipelineByName(name, ownerID)
	if err != nil {
		return err
	}

	if err := h.service.ValidatePipeline(dbPipeline); err != nil {
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
	if obj, err = h.service.TriggerPipelineBinaryFileUpload(buf, data.GetFileLengths(), dbPipeline); err != nil {
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
func HandleTriggerPipelineBinaryFileUpload(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {

	logger, _ := logger.GetZapLogger()

	contentType := req.Header.Get("Content-Type")

	if strings.Contains(contentType, "multipart/form-data") {

		ownerIDString := req.Header.Get("owner_id")
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

		dbPipeline, err := service.GetPipelineByName(pipelineName, ownerID)
		if err != nil {
			errorResponse(w, 400, "Bad Request", "Pipeline not found")
			return
		}

		if err := req.ParseMultipartForm(4 << 20); err != nil {
			errorResponse(w, 500, "Internal Error", "Error while reading file from request")
			return
		}

		fileBytes, fileLengths, err := parseImageFormDataInputsToBytes(req)
		if err != nil {
			errorResponse(w, 500, "Internal Error", "Error while reading files from request")
			return
		}

		var obj interface{}
		if obj, err = service.TriggerPipelineBinaryFileUpload(*bytes.NewBuffer(fileBytes), fileLengths, dbPipeline); err != nil {
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
