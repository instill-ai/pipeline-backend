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
	ListPipeline(owner string, view pipelinePB.View, pageSize int, pageToken string) ([]datamodel.Pipeline, string, error)
	GetPipeline(uid uuid.UUID, owner string) (*datamodel.Pipeline, error)
	GetPipelineByID(id string, owner string) (*datamodel.Pipeline, error)
	UpdatePipeline(uid uuid.UUID, owner string, pipeline *datamodel.Pipeline) error
	DeletePipeline(uid uuid.UUID, owner string) error
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
		return status.Errorf(codes.InvalidArgument, "%v", result.Error)
	}
	return nil
}

func (r *repository) ListPipeline(owner string, view pipelinePB.View, pageSize int, pageToken string) ([]datamodel.Pipeline, string, error) {
	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Order("create_time DESC, id DESC")

	if pageSize > 0 {
		queryBuilder = queryBuilder.Limit(pageSize)
	}

	if pageToken != "" {
		createTime, uuid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, "", status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createTime, uuid)
	}

	if view != pipelinePB.View_VIEW_FULL {
		queryBuilder.Omit("pipeline.recipe")
	}

	var pipelines []datamodel.Pipeline
	var createTime time.Time // only using one for all loops, we only need the latest one in the end
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
		createTime = item.CreateTime
		pipelines = append(pipelines, item)
	}

	if len(pipelines) > 0 {
		nextPageToken := paginate.EncodeToken(createTime, (pipelines)[len(pipelines)-1].UID.String())
		return pipelines, nextPageToken, nil
	}

	return nil, "", nil
}

func (r *repository) GetPipeline(uid uuid.UUID, owner string) (*datamodel.Pipeline, error) {
	var pipeline datamodel.Pipeline
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("uid = ? AND owner = ?", uid, owner).
		First(&pipeline); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "The pipeline uid \"%s\" you specified is not found", uid.String())
	}
	return &pipeline, nil
}

func (r *repository) GetPipelineByID(id string, owner string) (*datamodel.Pipeline, error) {
	var pipeline datamodel.Pipeline
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("id = ? AND owner = ?", id, owner).
		First(&pipeline); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "The pipeline id \"%s\" you specified is not found", id)
	}
	return &pipeline, nil
}

func (r *repository) UpdatePipeline(uid uuid.UUID, owner string, pipeline *datamodel.Pipeline) error {
	if result := r.db.Model(&datamodel.Pipeline{}).Select("*").Omit("Uid").
		Where("uid = ? AND owner = ?", uid, owner).
		Updates(pipeline); result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}
	return nil
}

func (r *repository) DeletePipeline(uid uuid.UUID, owner string) error {
	result := r.db.Model(&datamodel.Pipeline{}).
		Where("uid = ? AND owner = ?", uid, owner).
		Delete(&datamodel.Pipeline{})

	if result.Error != nil {
		return status.Errorf(codes.Internal, "Error %v", result.Error)
	}

	if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "The pipeline uid \"%s\" you specified is not found", uid.String())
	}

	return nil
}
