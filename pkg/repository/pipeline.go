package repository

import (
	"fmt"

	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type PipelineRepository interface {
	CreatePipeline(pipeline model.Pipeline) error
	ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, error)
	GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error)
	UpdatePipeline(pipeline model.Pipeline) error
	DeletePipeline(namespace string, pipelineName string) error
}

type pipelineRepository struct {
	DB *gorm.DB
}

func NewPipelineRepository(db *gorm.DB) PipelineRepository {
	return &pipelineRepository{
		DB: db,
	}
}

var GetPipelineSelectField = []string{
	`"pipelines"."id" as id`,
	`"pipelines"."name"`,
	`"pipelines"."description"`,
	`"pipelines"."active"`,
	`"pipelines"."created_at"`,
	`"pipelines"."updated_at"`,
	`'Pipeline' as kind`,
	`CONCAT(namespace, '/', name) as full_name`,
}

var GetPipelineWithRecipeSelectField = []string{
	`"pipelines"."id" as id`,
	`"pipelines"."name"`,
	`"pipelines"."description"`,
	`"pipelines"."active"`,
	`"pipelines"."created_at"`,
	`"pipelines"."updated_at"`,
	`"pipelines"."recipe"`,
	`'Pipeline' as kind`,
	`CONCAT(namespace, '/', name) as full_name`,
}

func (r *pipelineRepository) CreatePipeline(pipeline model.Pipeline) error {
	l, _ := logger.GetZapLogger()

	// We ignore the full_name column since it's a virtual column
	if result := r.DB.Model(&model.Pipeline{}).Omit(`"pipelines"."full_name"`).Create(&pipeline); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *pipelineRepository) ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, error) {
	var pipelines []model.Pipeline
	if query.WithRecipe {
		r.DB.Model(&model.Pipeline{}).Select(GetPipelineWithRecipeSelectField).Where("namespace", query.Namespace).Find(&pipelines)
	} else {
		r.DB.Model(&model.Pipeline{}).Select(GetPipelineSelectField).Where("namespace", query.Namespace).Find(&pipelines)
	}

	return pipelines, nil
}

func (r *pipelineRepository) GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error) {
	var pipeline model.Pipeline
	if result := r.DB.Model(&model.Pipeline{}).Select(GetPipelineWithRecipeSelectField).Where(map[string]interface{}{"name": pipelineName, "namespace": namespace}).First(&pipeline); result.Error != nil {
		return model.Pipeline{}, status.Errorf(codes.NotFound, "The pipeline name %s you specified is not found", pipelineName)
	}

	return pipeline, nil
}

func (r *pipelineRepository) UpdatePipeline(pipeline model.Pipeline) error {
	l, _ := logger.GetZapLogger()

	// We ignore the name column since it can not be updated
	if result := r.DB.Model(&model.Pipeline{}).Omit(`"pipelines"."name"`).Where(map[string]interface{}{"name": pipeline.Name, "namespace": pipeline.Namespace}).Updates(pipeline); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}
	return nil
}

func (r *pipelineRepository) DeletePipeline(namespace string, pipelineName string) error {
	l, _ := logger.GetZapLogger()

	if result := r.DB.Where(map[string]interface{}{"name": pipelineName, "namespace": namespace}).Delete(&model.Pipeline{}); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	} else {
		if result.RowsAffected == 0 {
			return status.Errorf(codes.NotFound, "The pipeline name %s does not exist", pipelineName)
		}
	}
	return nil
}
