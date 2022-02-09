package service

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/instill-ai/pipeline-backend/configs"
	"github.com/instill-ai/pipeline-backend/internal/cache"
	httpUtils "github.com/instill-ai/pipeline-backend/internal/http"
	structUtil "github.com/instill-ai/pipeline-backend/internal/struct/util"
	"github.com/instill-ai/pipeline-backend/internal/temporal"
	model "github.com/instill-ai/pipeline-backend/pkg/model"
	"github.com/instill-ai/pipeline-backend/pkg/repository"
	pipelinePB "github.com/instill-ai/protogen-go/pipeline"
	workerModel "github.com/instill-ai/visual-data-pipeline/pkg/models"
	"github.com/rs/xid"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/structpb"
)

type PipelineService interface {
	CreatePipeline(pipeline model.Pipeline) (model.Pipeline, error)
	ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, error)
	GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error)
	UpdatePipeline(pipeline model.Pipeline) (model.Pipeline, error)
	DeletePipeline(namespace string, pipelineName string) error
	TriggerPipeline(namespace string, trigger pipelinePB.TriggerPipelineRequest, pipeline model.Pipeline) (interface{}, error)
	ValidateTriggerPipeline(namespace string, pipelineName string, pipeline model.Pipeline) error
	TriggerPipelineByUpload(namespace string, buf bytes.Buffer, pipeline model.Pipeline) (interface{}, error)
}

type pipelineService struct {
	pipelineRepository repository.PipelineRepository
}

func NewPipelineService(r repository.PipelineRepository) PipelineService {
	return &pipelineService{
		pipelineRepository: r,
	}
}

func (p *pipelineService) CreatePipeline(pipeline model.Pipeline) (model.Pipeline, error) {

	// Validate the naming rule of pipeline
	if match, _ := regexp.MatchString("^[A-Za-z0-9][a-zA-Z0-9_.-]*$", pipeline.Name); !match {
		return model.Pipeline{}, status.Error(codes.FailedPrecondition, "The name of pipeline is invalid")
	}

	// TODO: validation

	if existingPipeline, _ := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); existingPipeline.Name != "" {
		return model.Pipeline{}, status.Errorf(codes.FailedPrecondition, "The name %s is existing in your namespace", pipeline.Name)
	}

	if err := p.pipelineRepository.CreatePipeline(pipeline); err != nil {
		return model.Pipeline{}, err
	}

	if createdPipeline, err := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); err != nil {
		return model.Pipeline{}, err
	} else {
		return createdPipeline, nil
	}
}

func (p *pipelineService) ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, error) {
	return p.pipelineRepository.ListPipelines(query)
}

func (p *pipelineService) GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error) {
	return p.pipelineRepository.GetPipelineByName(namespace, pipelineName)
}

func (p *pipelineService) UpdatePipeline(pipeline model.Pipeline) (model.Pipeline, error) {

	// TODO: validation

	if existingPipeline, _ := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); existingPipeline.Name == "" {
		return model.Pipeline{}, status.Errorf(codes.NotFound, "The pipeline name %s you specified is not found", pipeline.Name)
	}

	if err := p.pipelineRepository.UpdatePipeline(pipeline); err != nil {
		return model.Pipeline{}, err
	}

	if updatedPipeline, err := p.GetPipelineByName(pipeline.Namespace, pipeline.Name); err != nil {
		return model.Pipeline{}, err
	} else {
		return updatedPipeline, nil
	}
}

func (p *pipelineService) DeletePipeline(namespace string, pipelineName string) error {
	return p.pipelineRepository.DeletePipeline(namespace, pipelineName)
}

func (p *pipelineService) ValidateTriggerPipeline(namespace string, pipelineName string, pipeline model.Pipeline) error {

	// Specified pipeline not exists
	if pipeline.Name == "" {
		return status.Errorf(codes.NotFound, "The pipeline name %s you specified is not found", pipelineName)
	}

	// Pipeline is inactive
	if !pipeline.Active {
		return status.Error(codes.FailedPrecondition, "This pipeline has been deactivated")
	}

	// Pipeline not belong to this requester
	if !strings.Contains(pipeline.FullName, namespace) {
		return status.Error(codes.PermissionDenied, "You are not allowed to trigger this pipeline")
	}

	// TODO: The model that pipeline used is offline

	return nil
}

func (p *pipelineService) TriggerPipeline(namespace string, trigger pipelinePB.TriggerPipelineRequest, pipeline model.Pipeline) (interface{}, error) {

	// TODO: The model that pipeline used is offline

	if temporal.IsDirect(pipeline.Recipe) {

		body, err := json.Marshal(trigger)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "Error while decode request:", err.Error())
		}

		vdo := pipeline.Recipe.VisualDataOperator[0]
		vdoEndpoint := fmt.Sprintf("%s://%s:%d/%s",
			configs.Config.VDO.Scheme,
			configs.Config.VDO.Host,
			configs.Config.VDO.Port,
			fmt.Sprintf(configs.Config.VDO.Path, vdo.ModelId, strconv.FormatInt(int64(vdo.Version), 10)))

		client := &http.Client{}

		proxyReq, err := http.NewRequest(http.MethodPost, vdoEndpoint, bytes.NewReader(body))
		if err != nil {
			return &structpb.Struct{}, status.Error(codes.PermissionDenied, "You are not allowed to trigger this pipeline")
		}

		proxyReq.Header = make(http.Header)
		proxyReq.Header["Content-Type"] = []string{"application/json"}

		var resp *http.Response
		resp, err = client.Do(proxyReq)
		if err != nil {
			return &structpb.Struct{}, status.Error(codes.Internal, err.Error())
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			return &structpb.Struct{}, status.Error(codes.Internal, "Inference error")
		}

		var respBody []byte
		if respBody, err = ioutil.ReadAll(resp.Body); err != nil {
			return &structpb.Struct{}, status.Error(codes.Internal, err.Error())
		}

		if obj, err := structUtil.BytesToInterface(respBody); err != nil {
			return &structpb.Struct{}, status.Error(codes.Internal, "Can not process response")
		} else {
			return obj, nil
		}
	} else {
		buf := new(bytes.Buffer)
		enc := gob.NewEncoder(buf)
		httpBodyCache := workerModel.VDOHttpBodyCache{
			ContentType: "application/json",
			Body:        trigger,
		}

		// Serialized struct
		gob.Register(pipelinePB.TriggerPipelineRequest{})
		if err := enc.Encode(httpBodyCache); err != nil {
			return &structpb.Struct{}, status.Errorf(codes.Internal, "Error when deserialize trigger content: %+v", err)
		}

		uid := xid.New().String()

		if err := cache.Redis.Set(context.Background(), uid, buf.Bytes(), 10*time.Minute).Err(); err != nil {
			return &structpb.Struct{}, status.Error(codes.Internal, err.Error())
		}

		result, _ := temporal.TriggerTemporalWorkflow(pipeline.Name, pipeline.Recipe, uid, namespace)

		if obj, err := structUtil.BytesToInterface([]byte(result["visualDataOperatorResult"])); err != nil {
			return &structpb.Struct{}, status.Error(codes.Internal, "Can not process response")
		} else {
			return obj, nil
		}
	}
}

func (p *pipelineService) TriggerPipelineByUpload(namespace string, buf bytes.Buffer, pipeline model.Pipeline) (interface{}, error) {

	vdo := pipeline.Recipe.VisualDataOperator[0]
	vdoEndpoint := fmt.Sprintf("%s://%s:%d/%s",
		configs.Config.VDO.Scheme,
		configs.Config.VDO.Host,
		configs.Config.VDO.Port,
		fmt.Sprintf(configs.Config.VDO.Path, vdo.ModelId, strconv.FormatInt(int64(vdo.Version), 10)))

	httpCode, respBody, err := httpUtils.MultiPart(vdoEndpoint, nil, nil, "contents", "file", buf.Bytes())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed unexpectadely while writing to byte buffer: %s", err.Error())
	}

	if httpCode != 200 {
		return nil, status.Error(codes.Internal, "failed to perform inference")
	}

	var obj interface{}
	if obj, err = structUtil.BytesToInterface(respBody); err != nil {
		return nil, status.Error(codes.Internal, "Can not process response")
	}

	return obj, nil
}
