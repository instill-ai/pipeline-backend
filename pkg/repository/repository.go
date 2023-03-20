package repository

import (
	"errors"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgconn"
	"go.einride.tech/aip/filtering"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/x/paginate"
)

// DefaultPageSize is the default pagination page size when page size is not assigned
const DefaultPageSize = 10

// MaxPageSize is the maximum pagination page size if the assigned value is over this number
const MaxPageSize = 100

// Repository interface
type Repository interface {
	CreatePipeline(pipeline *datamodel.Pipeline) error
	ListPipeline(owner string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByID(id string, owner string, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUID(uid uuid.UUID, owner string, isBasicView bool) (*datamodel.Pipeline, error)
	UpdatePipeline(id string, owner string, pipeline *datamodel.Pipeline) error
	DeletePipeline(id string, owner string) error
	UpdatePipelineID(id string, owner string, newID string) error
	UpdatePipelineState(id string, owner string, state datamodel.PipelineState) error

	ListPipelineAdmin(pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]datamodel.Pipeline, int64, string, error)
	GetPipelineByIDAdmin(id string, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error)
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
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == "23505" {
				return status.Errorf(codes.AlreadyExists, pgErr.Message)
			}
		}
	}
	return nil
}

func (r *repository) ListPipeline(owner string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (pipelines []datamodel.Pipeline, totalSize int64, nextPageToken string, err error) {

	if result := r.db.Model(&datamodel.Pipeline{}).Where("owner = ?", owner).Count(&totalSize); result.Error != nil {
		return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Order("create_time DESC, uid DESC").Where("owner = ?", owner)

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createTime, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, 0, "", status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createTime, uid)
	}

	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}

	if expr, err := r.transpileFilter(filter); err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	} else if expr != nil {
		queryBuilder.Clauses(expr)
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
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
		lastUID := (pipelines)[len(pipelines)-1].UID
		lastItem := &datamodel.Pipeline{}
		if result := r.db.Model(&datamodel.Pipeline{}).
			Where("owner = ?", owner).
			Order("create_time ASC, uid ASC").
			Limit(1).Find(lastItem); result.Error != nil {
			return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
		}
		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return pipelines, totalSize, nextPageToken, nil
}

func (r *repository) ListPipelineAdmin(pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (pipelines []datamodel.Pipeline, totalSize int64, nextPageToken string, err error) {

	if result := r.db.Model(&datamodel.Pipeline{}).Count(&totalSize); result.Error != nil {
		return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Order("create_time DESC, uid DESC")

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createTime, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, 0, "", status.Errorf(codes.InvalidArgument, "Invalid page token: %s", err.Error())
		}
		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createTime, uid)
	}

	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}

	if expr, err := r.transpileFilter(filter); err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	} else if expr != nil {
		queryBuilder.Clauses(expr)
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
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
		lastUID := (pipelines)[len(pipelines)-1].UID
		lastItem := &datamodel.Pipeline{}
		if result := r.db.Model(&datamodel.Pipeline{}).
			Order("create_time ASC, uid ASC").
			Limit(1).Find(lastItem); result.Error != nil {
			return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
		}
		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return pipelines, totalSize, nextPageToken, nil
}

func (r *repository) GetPipelineByID(id string, owner string, isBasicView bool) (*datamodel.Pipeline, error) {
	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Where("id = ? AND owner = ?", id, owner)
	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}
	var pipeline datamodel.Pipeline
	if result := queryBuilder.First(&pipeline); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetPipelineByID] The pipeline id %s you specified is not found", id)
	}
	return &pipeline, nil
}

func (r *repository) GetPipelineByIDAdmin(id string, isBasicView bool) (*datamodel.Pipeline, error) {
	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Where("id = ?", id)
	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}
	var pipeline datamodel.Pipeline
	if result := queryBuilder.First(&pipeline); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetPipelineByID] The pipeline id %s you specified is not found", id)
	}
	return &pipeline, nil
}

func (r *repository) GetPipelineByUID(uid uuid.UUID, owner string, isBasicView bool) (*datamodel.Pipeline, error) {
	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Where("uid = ? AND owner = ?", uid, owner)
	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}
	var pipeline datamodel.Pipeline
	if result := queryBuilder.First(&pipeline); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetPipelineByUID] The pipeline uid %s you specified is not found", uid.String())
	}
	return &pipeline, nil
}

func (r *repository) GetPipelineByUIDAdmin(uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error) {
	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Where("uid = ?", uid)
	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}
	var pipeline datamodel.Pipeline
	if result := queryBuilder.First(&pipeline); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetPipelineByUID] The pipeline uid %s you specified is not found", uid.String())
	}
	return &pipeline, nil
}

func (r *repository) UpdatePipeline(id string, owner string, pipeline *datamodel.Pipeline) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("id = ? AND owner = ?", id, owner).
		Updates(pipeline); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipeline] The pipeline id %s you specified is not found", id)
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
		return status.Errorf(codes.NotFound, "[DeletePipeline] The pipeline id %s you specified is not found", id)
	}

	return nil
}

func (r *repository) UpdatePipelineID(id string, owner string, newID string) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("id = ? AND owner = ?", id, owner).
		Update("id", newID); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipelineID] The pipeline id %s you specified is not found", id)
	}
	return nil
}

func (r *repository) UpdatePipelineState(id string, owner string, state datamodel.PipelineState) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("id = ? AND owner = ?", id, owner).
		Update("state", state); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipelineState] The pipeline id %s you specified is not found", id)
	}
	return nil
}

// TranspileFilter transpiles a parsed AIP filter expression to GORM DB clauses
func (r *repository) transpileFilter(filter filtering.Filter) (*clause.Expr, error) {
	return (&Transpiler{
		filter: filter,
	}).Transpile()
}
