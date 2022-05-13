package repository

import (
	"time"

	"github.com/gofrs/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/internal/paginate"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
)

// Repository interface
type Repository interface {
	CreatePipeline(pipeline *datamodel.Pipeline) error
	ListPipeline(owner string, pageSize int, pageToken string, isBasicView bool) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByUID(uid uuid.UUID, owner string, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByID(id string, owner string, isBasicView bool) (*datamodel.Pipeline, error)
	UpdatePipeline(id string, owner string, pipeline *datamodel.Pipeline) error
	UpdatePipelineState(id string, owner string, state datamodel.PipelineState) error
	UpdatePipelineID(id string, owner string, newID string) error
	DeletePipeline(id string, owner string) error
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

func (r *repository) ListPipeline(owner string, pageSize int, pageToken string, isBasicView bool) (pipelines []datamodel.Pipeline, totalSize int64, nextPageToken string, err error) {

	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Where("owner = ?", owner).Order("create_time DESC, id DESC")

	if pageSize == 0 {
		queryBuilder = queryBuilder.Limit(10)
	} else if pageSize > 0 && pageSize <= 100 {
		queryBuilder = queryBuilder.Limit(pageSize)
	} else {
		queryBuilder = queryBuilder.Limit(100)
	}

	if pageToken != "" {
		createTime, uuid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, 0, "", status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,id) < (?::timestamp, ?)", createTime, uuid)
	}

	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Pipeline
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", status.Error(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		pipelines = append(pipelines, item)
	}

	if len(pipelines) > 0 {
		r.db.Model(&datamodel.Pipeline{}).Where("owner = ?", owner).Count(&totalSize)
		nextPageToken := paginate.EncodeToken(createTime, (pipelines)[len(pipelines)-1].UID.String())
		return pipelines, totalSize, nextPageToken, nil
	}

	return nil, 0, "", nil
}

func (r *repository) GetPipelineByUID(uid uuid.UUID, owner string, isBasicView bool) (*datamodel.Pipeline, error) {
	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Where("uid = ? AND owner = ?", uid, owner)
	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}
	var pipeline datamodel.Pipeline
	if result := queryBuilder.First(&pipeline); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "The pipeline uid \"%s\" you specified is not found", uid.String())
	}
	return &pipeline, nil
}

func (r *repository) GetPipelineByID(id string, owner string, isBasicView bool) (*datamodel.Pipeline, error) {
	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Where("id = ? AND owner = ?", id, owner)
	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}
	var pipeline datamodel.Pipeline
	if result := queryBuilder.First(&pipeline); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "The pipeline id \"%s\" you specified is not found", id)
	}
	return &pipeline, nil
}

func (r *repository) UpdatePipeline(id string, owner string, pipeline *datamodel.Pipeline) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("id = ? AND owner = ?", id, owner).
		Updates(pipeline); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	}
	return nil
}

func (r *repository) UpdatePipelineState(id string, owner string, state datamodel.PipelineState) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("id = ? AND owner = ?", id, owner).
		Update("state", state); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	}
	return nil
}

func (r *repository) UpdatePipelineID(id string, owner string, newID string) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("id = ? AND owner = ?", id, owner).
		Update("id", newID); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	}
	return nil
}

func (r *repository) DeletePipeline(id string, owner string) error {
	result := r.db.Model(&datamodel.Pipeline{}).
		Where("id = ? AND owner = ?", id, owner).
		Delete(&datamodel.Pipeline{})

	if result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "The pipeline id \"%s\" you specified is not found", id)
	}

	return nil
}
