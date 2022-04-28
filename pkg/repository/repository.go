package repository

import (
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/internal/paginate"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	pipelinePB "github.com/instill-ai/protogen-go/pipeline/v1alpha"
)

// Repository interface
type Repository interface {
	CreatePipeline(pipeline *datamodel.Pipeline) error
	ListPipeline(ownerID uuid.UUID, view pipelinePB.PipelineView, pageSize int, pageCursor string) ([]datamodel.Pipeline, string, error)
	GetPipeline(ownerID uuid.UUID, name string) (*datamodel.Pipeline, error)
	UpdatePipeline(ownerID uuid.UUID, name string, pipeline *datamodel.Pipeline) error
	DeletePipeline(ownerID uuid.UUID, name string) error
}

type repository struct {
	db *gorm.DB
}

// NewRepository initiates a repository instance
func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) CreatePipeline(pipeline *datamodel.Pipeline) error {
	if result := r.db.Model(&datamodel.Pipeline{}).Create(pipeline); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}
	return nil
}

func (r *repository) ListPipeline(ownerID uuid.UUID, view pipelinePB.PipelineView, pageSize int, pageCursor string) ([]datamodel.Pipeline, string, error) {
	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Order("created_at DESC, id DESC")

	if pageSize > 0 {
		queryBuilder = queryBuilder.Limit(pageSize)
	}

	if pageCursor != "" {
		createdAt, uuid, err := paginate.DecodeCursor(pageCursor)
		if err != nil {
			return nil, "", status.Errorf(codes.InvalidArgument, "Invalid page cursor: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(created_at,id) < (?::timestamp, ?)", createdAt, uuid)
	}

	if view != pipelinePB.PipelineView_PIPELINE_VIEW_FULL {
		queryBuilder.Omit("pipeline.recipe")
	}

	var pipelines []datamodel.Pipeline
	var createdAt time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Pipeline
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, "", status.Errorf(codes.Internal, "Error %v", err.Error())
		}
		createdAt = item.CreatedAt
		pipelines = append(pipelines, item)
	}

	if len(pipelines) > 0 {
		nextPageCursor := paginate.EncodeCursor(createdAt, (pipelines)[len(pipelines)-1].ID.String())
		return pipelines, nextPageCursor, nil
	}

	return nil, "", nil
}

func (r *repository) GetPipeline(ownerID uuid.UUID, name string) (*datamodel.Pipeline, error) {
	var pipeline datamodel.Pipeline
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("owner_id = ? AND name = ?", ownerID, name).
		First(&pipeline); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "The pipeline name \"%s\" you specified is not found", name)
	}
	return &pipeline, nil
}

func (r *repository) UpdatePipeline(ownerID uuid.UUID, name string, pipeline *datamodel.Pipeline) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("owner_id = ? AND name = ?", ownerID, name).
		Updates(pipeline); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}
	return nil
}

func (r *repository) DeletePipeline(ownerID uuid.UUID, name string) error {
	result := r.db.Model(&datamodel.Pipeline{}).
		Where("owner_id = ? AND name = ?", ownerID, name).
		Delete(&datamodel.Pipeline{})

	if result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "The pipeline with name \"%s\" you specified is not found", name)
	}

	return nil
}
