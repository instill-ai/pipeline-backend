package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgconn"
	"go.einride.tech/aip/filtering"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/x/paginate"
	"github.com/instill-ai/x/sterr"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1alpha"
)

// TODO: in the repository, we'd better use uid as our function params

// DefaultPageSize is the default pagination page size when page size is not assigned
const DefaultPageSize = 10

// MaxPageSize is the maximum pagination page size if the assigned value is over this number
const MaxPageSize = 100

const VisibilityPublic = datamodel.PipelineVisibility(pipelinePB.Visibility_VISIBILITY_PUBLIC)

// Repository interface
type Repository interface {
	ListPipelines(ctx context.Context, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Pipeline, int64, string, error)

	CreateUserPipeline(ctx context.Context, ownerPermalink string, userPermalink string, pipeline *datamodel.Pipeline) error
	ListUserPipelines(ctx context.Context, ownerPermalink string, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Pipeline, int64, string, error)
	GetUserPipelineByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, isBasicView bool) (*datamodel.Pipeline, error)
	GetUserPipelineByUID(ctx context.Context, ownerPermalink string, userPermalink string, uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error)
	UpdateUserPipelineByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, pipeline *datamodel.Pipeline) error
	DeleteUserPipelineByID(ctx context.Context, ownerPermalink string, userPermalink string, id string) error
	UpdateUserPipelineIDByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, newID string) error

	CreateUserPipelineRelease(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, pipelineRelease *datamodel.PipelineRelease) error
	ListUserPipelineReleases(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.PipelineRelease, int64, string, error)
	GetUserPipelineReleaseByID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, id string, isBasicView bool) (*datamodel.PipelineRelease, error)
	GetUserPipelineReleaseByUID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error)
	UpdateUserPipelineReleaseByID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, id string, pipelineRelease *datamodel.PipelineRelease) error
	DeleteUserPipelineReleaseByID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, id string) error
	UpdateUserPipelineReleaseIDByID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, id string, newID string) error

	ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Pipeline, int64, string, error)
	GetPipelineByIDAdmin(ctx context.Context, id string, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error)
	ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.PipelineRelease, int64, string, error)
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

func (r *repository) CreateUserPipeline(ctx context.Context, ownerPermalink string, userPermalink string, pipeline *datamodel.Pipeline) error {
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

func (r *repository) listPipelines(ctx context.Context, where string, whereArgs []interface{}, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (pipelines []*datamodel.Pipeline, totalSize int64, nextPageToken string, err error) {
	var expr *clause.Expr
	if expr, err = r.transpileFilter(filter); err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	}
	if expr != nil {
		if len(whereArgs) == 0 {
			where = "(?)"
			whereArgs = append(whereArgs, expr)
		} else {
			where = fmt.Sprintf("((%s) AND ?)", where)
			whereArgs = append(whereArgs, expr)
		}
	}

	logger, _ := logger.GetZapLogger(ctx)

	r.db.Model(&datamodel.Pipeline{}).Where(where, whereArgs...).Count(&totalSize)

	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Order("create_time DESC, uid DESC").Where(where, whereArgs...)

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createdAt, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			st, err := sterr.CreateErrorBadRequest(
				fmt.Sprintf("[db] list Pipeline error: %s", err.Error()),
				[]*errdetails.BadRequest_FieldViolation{
					{
						Field:       "page_token",
						Description: fmt.Sprintf("Invalid page token: %s", err.Error()),
					},
				},
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
		}

		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createdAt, uid)
	}

	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] list Pipeline error: %s", err.Error()),
			"pipeline",
			"",
			"",
			err.Error(),
		)

		if err != nil {
			logger.Error(err.Error())
		}
		return nil, 0, "", st.Err()
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Pipeline
		if err = r.db.ScanRows(rows, &item); err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list Pipeline error: %s", err.Error()),
				"pipeline",
				"",
				"",
				err.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
		}
		createTime = item.CreateTime
		pipelines = append(pipelines, &item)
	}

	if len(pipelines) > 0 {
		lastUID := (pipelines)[len(pipelines)-1].UID
		lastItem := &datamodel.Pipeline{}

		if result := r.db.Model(&datamodel.Pipeline{}).
			Where(where, whereArgs...).
			Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list Pipeline error: %s", err.Error()),
				"pipeline",
				"",
				"",
				result.Error.Error(),
			)
			if err != nil {
				logger.Error(err.Error())
			}
			return nil, 0, "", st.Err()
		}

		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return pipelines, totalSize, nextPageToken, nil
}

func (r *repository) ListPipelines(ctx context.Context, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx,
		"(owner = ? OR visibility = ?)",
		[]interface{}{userPermalink, VisibilityPublic},
		pageSize, pageToken, isBasicView, filter)
}
func (r *repository) ListUserPipelines(ctx context.Context, ownerPermalink string, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx,
		"(owner = ? AND (visibility = ? OR ? = ?))",
		[]interface{}{ownerPermalink, VisibilityPublic, ownerPermalink, userPermalink},
		pageSize, pageToken, isBasicView, filter)
}

func (r *repository) ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx, "", []interface{}{}, pageSize, pageToken, isBasicView, filter)
}

func (r *repository) getUserPipeline(ctx context.Context, where string, whereArgs []interface{}, isBasicView bool) (*datamodel.Pipeline, error) {
	logger, _ := logger.GetZapLogger(ctx)

	var pipeline datamodel.Pipeline

	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}

	if result := queryBuilder.First(&pipeline); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] getUserPipeline error: %s", result.Error.Error()),
			"pipeline",
			"",
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}
	return &pipeline, nil
}

func (r *repository) GetUserPipelineByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, isBasicView bool) (*datamodel.Pipeline, error) {
	return r.getUserPipeline(ctx,
		"(id = ? AND (owner = ? AND (visibility = ? OR ? = ?)))",
		[]interface{}{id, ownerPermalink, VisibilityPublic, ownerPermalink, userPermalink},
		isBasicView)
}

func (r *repository) GetUserPipelineByUID(ctx context.Context, ownerPermalink string, userPermalink string, uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error) {
	return r.getUserPipeline(ctx,
		"(uid = ? AND (owner = ? AND (visibility = ? OR ? = ?)))",
		[]interface{}{uid, ownerPermalink, VisibilityPublic, ownerPermalink, userPermalink},
		isBasicView)
}

func (r *repository) GetPipelineByIDAdmin(ctx context.Context, id string, isBasicView bool) (*datamodel.Pipeline, error) {
	return r.getUserPipeline(ctx,
		"(id = ?)",
		[]interface{}{id},
		isBasicView)
}

func (r *repository) GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error) {
	return r.getUserPipeline(ctx,
		"(uid = ?)",
		[]interface{}{uid},
		isBasicView)
}

func (r *repository) UpdateUserPipelineByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, pipeline *datamodel.Pipeline) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("(id = ? AND owner = ? AND ? = ? )", id, ownerPermalink, ownerPermalink, userPermalink).
		Updates(pipeline); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipeline] The pipeline id %s you specified is not found", id)
	}
	return nil
}

func (r *repository) DeleteUserPipelineByID(ctx context.Context, ownerPermalink string, userPermalink string, id string) error {
	result := r.db.Model(&datamodel.Pipeline{}).
		Where("(id = ? AND owner = ? AND ? = ? )", id, ownerPermalink, ownerPermalink, userPermalink).
		Delete(&datamodel.Pipeline{})

	if result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[DeletePipeline] The pipeline id %s you specified is not found", id)
	}

	return nil
}

func (r *repository) UpdateUserPipelineIDByID(ctx context.Context, ownerPermalink string, userPermalink string, id string, newID string) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("(id = ? AND owner = ? AND ? = ? )", id, ownerPermalink, ownerPermalink, userPermalink).
		Update("id", newID); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipelineID] The pipeline id %s you specified is not found", id)
	}
	return nil
}

// TranspileFilter transpiles a parsed AIP filter expression to GORM DB clauses
func (r *repository) transpileFilter(filter filtering.Filter) (*clause.Expr, error) {
	return (&Transpiler{
		filter: filter,
	}).Transpile()
}

func (r *repository) CreateUserPipelineRelease(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, pipelineRelease *datamodel.PipelineRelease) error {
	if result := r.db.Model(&datamodel.PipelineRelease{}).Create(pipelineRelease); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == "23505" {
				return status.Errorf(codes.AlreadyExists, pgErr.Message)
			}
		}
	}
	return nil
}

func (r *repository) ListUserPipelineReleases(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (pipelineReleases []*datamodel.PipelineRelease, totalSize int64, nextPageToken string, err error) {

	if result := r.db.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid = ?", pipelineUid).Count(&totalSize); result.Error != nil {
		return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := r.db.Model(&datamodel.PipelineRelease{}).Order("create_time DESC, uid DESC").Where("pipeline_uid = ?", pipelineUid)

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
		queryBuilder.Omit("pipeline_release.recipe")
	}

	if expr, err := r.transpileFilter(filter); err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	} else if expr != nil {
		queryBuilder.Where("(?)", expr)
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", status.Errorf(codes.Internal, err.Error())
	}
	defer rows.Close()
	for rows.Next() {
		var item *datamodel.PipelineRelease
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", status.Error(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		pipelineReleases = append(pipelineReleases, item)
	}

	if len(pipelineReleases) > 0 {
		lastUID := (pipelineReleases)[len(pipelineReleases)-1].UID
		lastItem := &datamodel.PipelineRelease{}
		if result := r.db.Model(&datamodel.PipelineRelease{}).
			Where("pipeline_uid = ?", pipelineUid).
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

	return pipelineReleases, totalSize, nextPageToken, nil
}

func (r *repository) ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter) (pipelineReleases []*datamodel.PipelineRelease, totalSize int64, nextPageToken string, err error) {

	if result := r.db.Model(&datamodel.PipelineRelease{}).Count(&totalSize); result.Error != nil {
		return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := r.db.Model(&datamodel.PipelineRelease{}).Order("create_time DESC, uid DESC")

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
		var item *datamodel.PipelineRelease
		if err = r.db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", status.Error(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		pipelineReleases = append(pipelineReleases, item)
	}

	if len(pipelineReleases) > 0 {
		lastUID := (pipelineReleases)[len(pipelineReleases)-1].UID
		lastItem := &datamodel.PipelineRelease{}
		if result := r.db.Model(&datamodel.PipelineRelease{}).
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

	return pipelineReleases, totalSize, nextPageToken, nil
}

func (r *repository) GetUserPipelineReleaseByID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, id string, isBasicView bool) (*datamodel.PipelineRelease, error) {
	queryBuilder := r.db.Model(&datamodel.PipelineRelease{}).Where("id = ? AND pipeline_uid = ?", id, pipelineUid)
	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe")
	}
	var pipelineRelease datamodel.PipelineRelease
	if result := queryBuilder.First(&pipelineRelease); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetPipelineReleaseByID] The pipeline_release id %s you specified is not found", id)
	}
	return &pipelineRelease, nil
}

func (r *repository) GetUserPipelineReleaseByUID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error) {
	queryBuilder := r.db.Model(&datamodel.PipelineRelease{}).Where("uid = ? AND pipeline_uid = ?", uid, pipelineUid)
	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe")
	}
	var pipelineRelease datamodel.PipelineRelease
	if result := queryBuilder.First(&pipelineRelease); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetPipelineReleaseByUID] The pipeline_release uid %s you specified is not found", uid.String())
	}
	return &pipelineRelease, nil
}

func (r *repository) UpdateUserPipelineReleaseByID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, id string, pipelineRelease *datamodel.PipelineRelease) error {
	if result := r.db.Model(&datamodel.PipelineRelease{}).
		Where("id = ? AND pipeline_uid = ?", id, pipelineUid).
		Updates(pipelineRelease); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipelineRelease] The pipeline_release id %s you specified is not found", id)
	}
	return nil
}

func (r *repository) DeleteUserPipelineReleaseByID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, id string) error {
	result := r.db.Model(&datamodel.PipelineRelease{}).
		Where("id = ? AND pipeline_uid = ?", id, pipelineUid).
		Delete(&datamodel.PipelineRelease{})

	if result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[DeletePipelineRelease] The pipeline_release id %s you specified is not found", id)
	}

	return nil
}

func (r *repository) UpdateUserPipelineReleaseIDByID(ctx context.Context, ownerPermalink string, userPermalink string, pipelineUid uuid.UUID, id string, newID string) error {
	if result := r.db.Model(&datamodel.PipelineRelease{}).
		Where("id = ? AND pipeline_uid = ?", id, pipelineUid).
		Update("id", newID); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipelineReleaseID] The pipeline_release id %s you specified is not found", id)
	}
	return nil
}
