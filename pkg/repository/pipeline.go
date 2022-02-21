package repository

import (
	"fmt"
	"math"

	"github.com/instill-ai/pipeline-backend/internal/logger"
	"github.com/instill-ai/pipeline-backend/pkg/model"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

type Operations interface {
	CreatePipeline(pipeline model.Pipeline) error
	ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, uint64, uint64, error)
	GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error)
	UpdatePipeline(pipeline model.Pipeline) error
	DeletePipeline(namespace string, pipelineName string) error
}

type PipelineRepository struct {
	DB *gorm.DB
}

func NewPipelineRepository(db *gorm.DB) Operations {
	return &PipelineRepository{
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

func (r *PipelineRepository) CreatePipeline(pipeline model.Pipeline) error {
	l, _ := logger.GetZapLogger()

	// We ignore the full_name column since it's a virtual column
	if result := r.DB.Model(&model.Pipeline{}).
		Omit(`"pipelines"."full_name"`).
		Create(&pipeline); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	return nil
}

func (r *PipelineRepository) ListPipelines(query model.ListPipelineQuery) ([]model.Pipeline, uint64, uint64, error) {
	var pipelines []model.Pipeline

	var count int64
	r.DB.Model(&model.Pipeline{}).Where("namespace = ?", query.Namespace).Count(&count)

	var min uint64
	var max uint64
	if count > 0 {
		rows, err := r.DB.Model(&model.Pipeline{}).
			Select("MIN(id) AS min, MAX(id) as max").
			Where("namespace = ?", query.Namespace).
			Rows()
		if err != nil {
			rows.Close()
			return nil, 0, 0, status.Errorf(codes.Internal, "Error when query min & max value", err.Error())
		}
		if rows.Next() {
			if err := rows.Scan(&min, &max); err != nil {
				rows.Close()
				return nil, 0, 0, status.Errorf(codes.Internal, "Can not fetch the min & max value: %s", err.Error())
			}
		}
		rows.Close()
	}

	cursor := query.Cursor
	if cursor <= 0 {
		cursor = math.MaxInt64
	}

	if query.WithRecipe {
		r.DB.Model(&model.Pipeline{}).
			Select(GetPipelineWithRecipeSelectField).
			Where("namespace = ? AND id < ?", query.Namespace, cursor).
			Limit(int(query.PageSize)).
			Order("id desc").
			Find(&pipelines)
	} else {
		r.DB.Model(&model.Pipeline{}).
			Select(GetPipelineSelectField).
			Where("namespace = ? AND id < ?", query.Namespace, cursor).
			Limit(int(query.PageSize)).
			Order("id desc").
			Find(&pipelines)
	}

	return pipelines, max, min, nil
}

func (r *PipelineRepository) GetPipelineByName(namespace string, pipelineName string) (model.Pipeline, error) {
	var pipeline model.Pipeline
	if result := r.DB.Model(&model.Pipeline{}).
		Select(GetPipelineWithRecipeSelectField).
		Where(map[string]interface{}{"name": pipelineName, "namespace": namespace}).
		First(&pipeline); result.Error != nil {
		return model.Pipeline{}, status.Errorf(codes.NotFound, "The pipeline name %s you specified is not found", pipelineName)
	}

	return pipeline, nil
}

func (r *PipelineRepository) UpdatePipeline(pipeline model.Pipeline) error {
	l, _ := logger.GetZapLogger()

	// We ignore the name column since it can not be updated
	if result := r.DB.Model(&model.Pipeline{}).
		Omit(`"pipelines"."name"`).
		Where(map[string]interface{}{"name": pipeline.Name, "namespace": pipeline.Namespace}).
		Updates(pipeline); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}
	return nil
}

func (r *PipelineRepository) DeletePipeline(namespace string, pipelineName string) error {
	l, _ := logger.GetZapLogger()

	if result := r.DB.Model(&model.Pipeline{}).
		Where(map[string]interface{}{"name": pipelineName, "namespace": namespace}).
		Delete(&model.Pipeline{}); result.Error != nil {
		l.Error(fmt.Sprintf("Error occur: %v", result.Error))
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	} else {
		if result.RowsAffected == 0 {
			return status.Errorf(codes.NotFound, "The pipeline name %s does not exist", pipelineName)
		}
	}
	return nil
}
