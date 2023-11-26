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

const VisibilityPublic = datamodel.ConnectorVisibility(pipelinePB.Connector_VISIBILITY_PUBLIC)

// Repository interface
type Repository interface {
	ListPipelines(ctx context.Context, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Pipeline, int64, string, error)
	GetPipelineByUID(ctx context.Context, userPermalink string, uid uuid.UUID, isBasicView bool, code string) (*datamodel.Pipeline, error)

	CreateNamespacePipeline(ctx context.Context, ownerPermalink string, pipeline *datamodel.Pipeline) error
	ListNamespacePipelines(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Pipeline, int64, string, error)
	GetNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool, code string) (*datamodel.Pipeline, error)

	UpdateNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string, pipeline *datamodel.Pipeline) error
	DeleteNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string) error
	UpdateNamespacePipelineIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error

	CreateNamespacePipelineRelease(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, pipelineRelease *datamodel.PipelineRelease) error
	ListNamespacePipelineReleases(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.PipelineRelease, int64, string, error)
	GetNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, id string, isBasicView bool) (*datamodel.PipelineRelease, error)
	GetNamespacePipelineReleaseByUID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error)
	UpdateNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, id string, pipelineRelease *datamodel.PipelineRelease) error
	DeleteNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, id string) error
	UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, id string, newID string) error
	GetLatestNamespacePipelineRelease(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Pipeline, int64, string, error)
	GetPipelineByIDAdmin(ctx context.Context, id string, isBasicView bool) (*datamodel.Pipeline, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error)
	ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.PipelineRelease, int64, string, error)

	ListConnectors(ctx context.Context, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByUID(ctx context.Context, userPermalink string, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error)

	CreateNamespaceConnector(ctx context.Context, ownerPermalink string, connector *datamodel.Connector) error
	ListNamespaceConnectors(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Connector, int64, string, error)
	GetNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool) (*datamodel.Connector, error)
	UpdateNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string, connector *datamodel.Connector) error
	DeleteNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string) error
	UpdateNamespaceConnectorIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error
	UpdateNamespaceConnectorStateByID(ctx context.Context, ownerPermalink string, id string, state datamodel.ConnectorState) error

	ListConnectorsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error)
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

func (r *repository) CreateNamespacePipeline(ctx context.Context, ownerPermalink string, pipeline *datamodel.Pipeline) error {
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

func (r *repository) listPipelines(ctx context.Context, where string, whereArgs []interface{}, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) (pipelines []*datamodel.Pipeline, totalSize int64, nextPageToken string, err error) {

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

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

	db.Model(&datamodel.Pipeline{}).Where(where, whereArgs...).Count(&totalSize)

	queryBuilder := db.Model(&datamodel.Pipeline{}).Order("create_time DESC, uid DESC").Where(where, whereArgs...)

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
		if err = db.ScanRows(rows, &item); err != nil {
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

		if result := db.Model(&datamodel.Pipeline{}).
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

func (r *repository) ListPipelines(ctx context.Context, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx,
		"(owner = ? OR (permission @> '{\"users\":{\"users/*\":{\"role\": \"ROLE_VIEWER\", \"enabled\": true}}}'))",
		[]interface{}{userPermalink},
		pageSize, pageToken, isBasicView, filter, showDeleted)
}
func (r *repository) ListNamespacePipelines(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx,
		"(owner = ? AND ((permission @> '{\"users\":{\"users/*\":{\"role\": \"ROLE_VIEWER\", \"enabled\": true}}}')))",
		[]interface{}{ownerPermalink},
		pageSize, pageToken, isBasicView, filter, showDeleted)
}

func (r *repository) ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx, "", []interface{}{}, pageSize, pageToken, isBasicView, filter, showDeleted)
}

func (r *repository) getNamespacePipeline(ctx context.Context, where string, whereArgs []interface{}, isBasicView bool) (*datamodel.Pipeline, error) {
	logger, _ := logger.GetZapLogger(ctx)

	var pipeline datamodel.Pipeline

	queryBuilder := r.db.Model(&datamodel.Pipeline{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}

	if result := queryBuilder.First(&pipeline); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] getNamespacePipeline error: %s", result.Error.Error()),
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

func (r *repository) GetNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool, code string) (*datamodel.Pipeline, error) {
	return r.getNamespacePipeline(ctx,
		"(id = ? AND owner = ? AND ((permission @> '{\"users\":{\"users/*\":{\"role\": \"ROLE_VIEWER\", \"enabled\": true}}}') OR (permission @> '{\"share_code\":{\"user\":\"users/*\", \"role\": \"ROLE_VIEWER\", \"enabled\": true}}' AND share_code = ?)))",
		[]interface{}{id, ownerPermalink, code},
		isBasicView)
}

func (r *repository) GetPipelineByUID(ctx context.Context, userPermalink string, uid uuid.UUID, isBasicView bool, code string) (*datamodel.Pipeline, error) {
	// TODO: ACL
	return r.getNamespacePipeline(ctx,
		"(uid = ? AND ((permission @> '{\"users\":{\"users/*\":{\"role\": \"ROLE_VIEWER\", \"enabled\": true}}}') OR (permission @> '{\"share_code\":{\"user\":\"users/*\", \"role\": \"ROLE_VIEWER\", \"enabled\": true}}' AND share_code = ?) OR owner = ?))",
		[]interface{}{uid, code, userPermalink},
		isBasicView)
}

func (r *repository) GetPipelineByIDAdmin(ctx context.Context, id string, isBasicView bool) (*datamodel.Pipeline, error) {
	return r.getNamespacePipeline(ctx,
		"(id = ?)",
		[]interface{}{id},
		isBasicView)
}

func (r *repository) GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Pipeline, error) {
	return r.getNamespacePipeline(ctx,
		"(uid = ?)",
		[]interface{}{uid},
		isBasicView)
}

func (r *repository) UpdateNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string, pipeline *datamodel.Pipeline) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Updates(pipeline); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipeline] The pipeline id %s you specified is not found", id)
	}
	return nil
}

func (r *repository) DeleteNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string) error {
	result := r.db.Model(&datamodel.Pipeline{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Delete(&datamodel.Pipeline{})

	if result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	}

	if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[DeletePipeline] The pipeline id %s you specified is not found", id)
	}

	return nil
}

func (r *repository) UpdateNamespacePipelineIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error {
	if result := r.db.Model(&datamodel.Pipeline{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
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

func (r *repository) CreateNamespacePipelineRelease(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, pipelineRelease *datamodel.PipelineRelease) error {
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

func (r *repository) ListNamespacePipelineReleases(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) (pipelineReleases []*datamodel.PipelineRelease, totalSize int64, nextPageToken string, err error) {

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

	if result := db.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid = ?", pipelineUid).Count(&totalSize); result.Error != nil {
		return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := db.Model(&datamodel.PipelineRelease{}).Order("create_time DESC, uid DESC").Where("pipeline_uid = ?", pipelineUid)

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
		if err = db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", status.Error(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		pipelineReleases = append(pipelineReleases, item)
	}

	if len(pipelineReleases) > 0 {
		lastUID := (pipelineReleases)[len(pipelineReleases)-1].UID
		lastItem := &datamodel.PipelineRelease{}
		if result := db.Model(&datamodel.PipelineRelease{}).
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

func (r *repository) ListPipelineReleasesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) (pipelineReleases []*datamodel.PipelineRelease, totalSize int64, nextPageToken string, err error) {

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

	if result := db.Model(&datamodel.PipelineRelease{}).Count(&totalSize); result.Error != nil {
		return nil, 0, "", status.Errorf(codes.Internal, result.Error.Error())
	}

	queryBuilder := db.Model(&datamodel.PipelineRelease{}).Order("create_time DESC, uid DESC")

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
		if err = db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", status.Error(codes.Internal, err.Error())
		}
		createTime = item.CreateTime
		pipelineReleases = append(pipelineReleases, item)
	}

	if len(pipelineReleases) > 0 {
		lastUID := (pipelineReleases)[len(pipelineReleases)-1].UID
		lastItem := &datamodel.PipelineRelease{}
		if result := db.Model(&datamodel.PipelineRelease{}).
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

func (r *repository) GetNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, id string, isBasicView bool) (*datamodel.PipelineRelease, error) {
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

func (r *repository) GetNamespacePipelineReleaseByUID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, uid uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error) {
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

func (r *repository) UpdateNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, id string, pipelineRelease *datamodel.PipelineRelease) error {
	if result := r.db.Model(&datamodel.PipelineRelease{}).
		Where("id = ? AND pipeline_uid = ?", id, pipelineUid).
		Updates(pipelineRelease); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipelineRelease] The pipeline_release id %s you specified is not found", id)
	}
	return nil
}

func (r *repository) DeleteNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, id string) error {
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

func (r *repository) UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, id string, newID string) error {
	if result := r.db.Model(&datamodel.PipelineRelease{}).
		Where("id = ? AND pipeline_uid = ?", id, pipelineUid).
		Update("id", newID); result.Error != nil {
		return status.Error(codes.Internal, result.Error.Error())
	} else if result.RowsAffected == 0 {
		return status.Errorf(codes.NotFound, "[UpdatePipelineReleaseID] The pipeline_release id %s you specified is not found", id)
	}
	return nil
}

func (r *repository) GetLatestNamespacePipelineRelease(ctx context.Context, ownerPermalink string, pipelineUid uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error) {
	queryBuilder := r.db.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid = ?", pipelineUid).Order("id DESC")
	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe")
	}
	var pipelineRelease datamodel.PipelineRelease
	if result := queryBuilder.First(&pipelineRelease); result.Error != nil {
		return nil, status.Errorf(codes.NotFound, "[GetPipelineReleaseByUID] no release")
	}
	return &pipelineRelease, nil
}

func (r *repository) listConnectors(ctx context.Context, where string, whereArgs []interface{}, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

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

	db.Model(&datamodel.Connector{}).Where(where, whereArgs...).Count(&totalSize)

	queryBuilder := db.Model(&datamodel.Connector{}).Order("create_time DESC, uid DESC").Where(where, whereArgs...)

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
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
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
		queryBuilder.Omit("configuration")
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] list connector error: %s", err.Error()),
			"connector",
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
		var item datamodel.Connector
		if err = db.ScanRows(rows, &item); err != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] list connector error: %s", err.Error()),
				"connector",
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
		connectors = append(connectors, &item)
	}

	if len(connectors) > 0 {
		lastUID := (connectors)[len(connectors)-1].UID
		lastItem := &datamodel.Connector{}

		if result := db.Model(&datamodel.Connector{}).
			Where(where, whereArgs...).
			Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
			st, err := sterr.CreateErrorResourceInfo(
				codes.Internal,
				fmt.Sprintf("[db] listConnectors: %s", err.Error()),
				"connector",
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

	return connectors, totalSize, nextPageToken, nil
}

func (r *repository) ListConnectorsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {
	return r.listConnectors(ctx, "", []interface{}{}, pageSize, pageToken, isBasicView, filter, showDeleted)
}

func (r *repository) ListConnectors(ctx context.Context, userPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

	return r.listConnectors(ctx,
		"(owner = ?)",
		[]interface{}{userPermalink},
		pageSize, pageToken, isBasicView, filter, showDeleted)

}

func (r *repository) ListNamespaceConnectors(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

	return r.listConnectors(ctx,
		"(owner = ? )",
		[]interface{}{ownerPermalink},
		pageSize, pageToken, isBasicView, filter, showDeleted)

}

func (r *repository) CreateNamespaceConnector(ctx context.Context, ownerPermalink string, connector *datamodel.Connector) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.Connector{}).Create(connector); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) {
			if pgErr.Code == "23505" {
				st, err := sterr.CreateErrorResourceInfo(
					codes.AlreadyExists,
					fmt.Sprintf("[db] create connector error: %s", pgErr.Message),
					"connector",
					fmt.Sprintf("id %s", connector.ID),
					connector.Owner,
					pgErr.Message,
				)
				if err != nil {
					logger.Error(err.Error())
				}
				return st.Err()
			}
		}
	}
	return nil
}

func (r *repository) getNamespaceConnector(ctx context.Context, where string, whereArgs []interface{}, isBasicView bool) (*datamodel.Connector, error) {
	logger, _ := logger.GetZapLogger(ctx)

	var connector datamodel.Connector

	queryBuilder := r.db.Model(&datamodel.Connector{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] getNamespaceConnector error: %s", result.Error.Error()),
			"connector",
			"",
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return nil, st.Err()
	}
	return &connector, nil
}

func (r *repository) GetNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool) (*datamodel.Connector, error) {

	return r.getNamespaceConnector(ctx,
		"(id = ? AND (owner = ?))",
		[]interface{}{id, ownerPermalink},
		isBasicView)
}

func (r *repository) GetConnectorByUID(ctx context.Context, userPermalink string, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error) {

	// TODO: ACL
	return r.getNamespaceConnector(ctx,
		"(uid = ? AND (owner = ?))",
		[]interface{}{uid, userPermalink},
		isBasicView)

}

func (r *repository) GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error) {
	return r.getNamespaceConnector(ctx,
		"(uid = ?)",
		[]interface{}{uid},
		isBasicView)
}

func (r *repository) UpdateNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string, connector *datamodel.Connector) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.Connector{}).
		Where("(id = ? AND owner = ? )", id, ownerPermalink).
		Updates(connector); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector error: %s", result.Error.Error()),
			"connector",
			"",
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	} else if result.RowsAffected == 0 {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] update connector error: %s", "Not found"),
			"connector",
			"",
			"",
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}
	return nil
}

func (r *repository) DeleteNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string) error {

	logger, _ := logger.GetZapLogger(ctx)

	result := r.db.Model(&datamodel.Connector{}).
		Where("(id = ? AND owner = ? )", id, ownerPermalink).
		Delete(&datamodel.Connector{})

	if result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] delete connector error: %s", result.Error.Error()),
			"connector",
			"",
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	if result.RowsAffected == 0 {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] delete connector error: %s", "Not found"),
			"connector",
			"",
			"",
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}

	return nil
}

func (r *repository) UpdateNamespaceConnectorIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.Connector{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Update("id", newID); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector id error: %s", result.Error.Error()),
			"connector",
			"",
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	} else if result.RowsAffected == 0 {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] update connector id error: %s", "Not found"),
			"connector",
			"",
			"",
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}
	return nil
}

func (r *repository) UpdateNamespaceConnectorStateByID(ctx context.Context, ownerPermalink string, id string, state datamodel.ConnectorState) error {

	logger, _ := logger.GetZapLogger(ctx)

	if result := r.db.Model(&datamodel.Connector{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Update("state", state); result.Error != nil {
		st, err := sterr.CreateErrorResourceInfo(
			codes.Internal,
			fmt.Sprintf("[db] update connector state by id error: %s", result.Error.Error()),
			"connector",
			"",
			"",
			result.Error.Error(),
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	} else if result.RowsAffected == 0 {
		st, err := sterr.CreateErrorResourceInfo(
			codes.NotFound,
			fmt.Sprintf("[db] update connector state by id error: %s", "Not found"),
			"connector",
			"",
			"",
			"Not found",
		)
		if err != nil {
			logger.Error(err.Error())
		}
		return st.Err()
	}
	return nil
}
