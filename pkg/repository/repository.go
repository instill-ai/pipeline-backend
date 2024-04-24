package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgconn"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/x/errmsg"
	"github.com/instill-ai/x/paginate"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// TODO: in the repository, we'd better use uid as our function params

// DefaultPageSize is the default pagination page size when page size is not assigned
const DefaultPageSize = 10

// MaxPageSize is the maximum pagination page size if the assigned value is over this number
const MaxPageSize = 100

// Repository interface
type Repository interface {
	ListPipelines(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool) ([]*datamodel.Pipeline, int64, string, error)
	GetPipelineByUID(ctx context.Context, uid uuid.UUID, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error)

	CreateNamespacePipeline(ctx context.Context, ownerPermalink string, pipeline *datamodel.Pipeline) error
	ListNamespacePipelines(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool) ([]*datamodel.Pipeline, int64, string, error)
	GetNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error)

	UpdateNamespacePipelineByUID(ctx context.Context, uid uuid.UUID, pipeline *datamodel.Pipeline) error
	DeleteNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string) error
	UpdateNamespacePipelineIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error

	CreateNamespacePipelineRelease(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, pipelineRelease *datamodel.PipelineRelease) error
	ListNamespacePipelineReleases(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool, returnCount bool) ([]*datamodel.PipelineRelease, int64, string, error)
	GetNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string, isBasicView bool) (*datamodel.PipelineRelease, error)
	UpdateNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string, pipelineRelease *datamodel.PipelineRelease) error
	DeleteNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string) error
	UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string, newID string) error
	GetLatestNamespacePipelineRelease(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error)

	ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool, embedReleases bool) ([]*datamodel.Pipeline, int64, string, error)
	GetPipelineByIDAdmin(ctx context.Context, id string, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error)
	GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error)

	ListComponentDefinitionUIDs(context.Context, ListComponentDefinitionsParams) (uids []*datamodel.ComponentDefinition, totalSize int64, err error)
	GetComponentDefinitionByUID(context.Context, uuid.UUID) (*datamodel.ComponentDefinition, error)
	UpsertComponentDefinition(context.Context, *pb.ComponentDefinition) error

	CreateNamespaceSecret(ctx context.Context, ownerPermalink string, secret *datamodel.Secret) error
	ListNamespaceSecrets(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, filter filtering.Filter) ([]*datamodel.Secret, int64, string, error)
	GetNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string) (*datamodel.Secret, error)
	UpdateNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string, secret *datamodel.Secret) error
	DeleteNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string) error
}

type repository struct {
	db          *gorm.DB
	redisClient *redis.Client
}

// NewRepository initiates a repository instance
func NewRepository(db *gorm.DB, redisClient *redis.Client) Repository {
	return &repository{
		db:          db,
		redisClient: redisClient,
	}
}

func (r *repository) checkPinnedUser(ctx context.Context, db *gorm.DB, table string) *gorm.DB {
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	// If the user is pinned, we will use the primary database for querying.
	if !errors.Is(r.redisClient.Get(ctx, fmt.Sprintf("db_pin_user:%s:%s", userUID, table)).Err(), redis.Nil) {
		db = db.Clauses(dbresolver.Write)
	}
	return db
}

func (r *repository) pinUser(ctx context.Context, table string) {
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	// To solve the read-after-write inconsistency problem,
	// we will direct the user to read from the primary database for a certain time frame
	// to ensure that the data is synchronized from the primary DB to the replica DB.
	_ = r.redisClient.Set(ctx, fmt.Sprintf("db_pin_user:%s:%s", userUID, table), time.Now(), time.Duration(config.Config.Database.Replica.ReplicationTimeFrame)*time.Second)
}

func (r *repository) CreateNamespacePipeline(ctx context.Context, ownerPermalink string, pipeline *datamodel.Pipeline) error {
	r.pinUser(ctx, "pipeline")
	db := r.checkPinnedUser(ctx, r.db, "pipeline")

	if result := db.Model(&datamodel.Pipeline{}).Create(pipeline); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" || errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return errmsg.AddMessage(ErrNameExists, "Pipeline ID already exists")
		}
		return result.Error
	}
	return nil
}

func (r *repository) listPipelines(ctx context.Context, where string, whereArgs []interface{}, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool) (pipelines []*datamodel.Pipeline, totalSize int64, nextPageToken string, err error) {

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

	var expr *clause.Expr
	if expr, err = r.transpileFilter(filter); err != nil {
		return nil, 0, "", err
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

	if uidAllowList != nil {
		db.Model(&datamodel.Pipeline{}).Where(where, whereArgs...).Where("uid in ?", uidAllowList).Count(&totalSize)
	} else {
		db.Model(&datamodel.Pipeline{}).Where(where, whereArgs...).Count(&totalSize)
	}

	queryBuilder := db.Model(&datamodel.Pipeline{}).Order("create_time DESC, uid DESC").Where(where, whereArgs...)

	if uidAllowList != nil {
		queryBuilder = queryBuilder.Where("uid in ?", uidAllowList)
	}
	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createdAt, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, 0, "", ErrPageTokenDecode
		}

		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createdAt, uid)
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
	pipelineUIDs := []uuid.UUID{}

	for rows.Next() {
		var item datamodel.Pipeline
		if err = db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", err
		}
		createTime = item.CreateTime
		pipelines = append(pipelines, &item)
		pipelineUIDs = append(pipelineUIDs, item.UID)
	}

	if embedReleases {
		releaseDB := r.checkPinnedUser(ctx, r.db, "pipeline")
		releasesMap := map[uuid.UUID][]*datamodel.PipelineRelease{}
		releaseDBQueryBuilder := releaseDB.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid in ?", pipelineUIDs).Order("create_time DESC, uid DESC")
		if isBasicView {
			releaseDBQueryBuilder.Omit("pipeline_release.recipe")
		}
		releaseRows, err := releaseDBQueryBuilder.Rows()
		if err != nil {
			return nil, 0, "", err
		}
		defer releaseRows.Close()
		for releaseRows.Next() {
			var item datamodel.PipelineRelease
			if err = releaseDB.ScanRows(releaseRows, &item); err != nil {
				return nil, 0, "", err
			}
			pipelineUID := item.PipelineUID
			pipelineRelease := item
			if _, ok := releasesMap[pipelineUID]; !ok {
				releasesMap[pipelineUID] = []*datamodel.PipelineRelease{}
			}
			releasesMap[pipelineUID] = append(releasesMap[pipelineUID], &pipelineRelease)
		}
		for idx := range pipelines {
			if releases, ok := releasesMap[pipelines[idx].UID]; ok {
				pipelines[idx].Releases = releases

			}
		}
	}

	if len(pipelines) > 0 {
		lastUID := (pipelines)[len(pipelines)-1].UID
		lastItem := &datamodel.Pipeline{}

		if uidAllowList != nil {
			if result := db.Model(&datamodel.Pipeline{}).
				Where(where, whereArgs...).
				Where("uid in ?", uidAllowList).
				Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
				return nil, 0, "", err
			}
		} else {
			if result := db.Model(&datamodel.Pipeline{}).
				Where(where, whereArgs...).
				Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
				return nil, 0, "", err
			}
		}

		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return pipelines, totalSize, nextPageToken, nil
}

func (r *repository) ListPipelines(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx,
		"",
		[]interface{}{},
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted, embedReleases)
}
func (r *repository) ListNamespacePipelines(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx,
		"(owner = ?)",
		[]interface{}{ownerPermalink},
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted, embedReleases)
}

func (r *repository) ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool, embedReleases bool) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx, "", []interface{}{}, pageSize, pageToken, isBasicView, filter, nil, showDeleted, embedReleases)
}

func (r *repository) getNamespacePipeline(ctx context.Context, where string, whereArgs []interface{}, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error) {

	db := r.checkPinnedUser(ctx, r.db, "pipeline")

	var pipeline datamodel.Pipeline

	queryBuilder := db.Model(&datamodel.Pipeline{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("pipeline.recipe")
	}

	if result := queryBuilder.First(&pipeline); result.Error != nil {
		return nil, result.Error
	}

	if embedReleases {
		pipeline.Releases = []*datamodel.PipelineRelease{}

		releaseDB := r.checkPinnedUser(ctx, r.db, "pipeline")
		releaseDBQueryBuilder := releaseDB.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid = ?", pipeline.UID).Order("create_time DESC, uid DESC")
		if isBasicView {
			releaseDBQueryBuilder.Omit("pipeline_release.recipe")
		}

		releaseRows, err := releaseDBQueryBuilder.Rows()
		if err != nil {
			return nil, err
		}
		defer releaseRows.Close()
		for releaseRows.Next() {
			var item datamodel.PipelineRelease
			if err = releaseDB.ScanRows(releaseRows, &item); err != nil {
				return nil, err
			}
			pipelineRelease := item
			pipeline.Releases = append(pipeline.Releases, &pipelineRelease)

		}
	}

	return &pipeline, nil
}

func (r *repository) GetNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error) {
	return r.getNamespacePipeline(ctx,
		"(id = ? AND owner = ? )",
		[]interface{}{id, ownerPermalink},
		isBasicView,
		embedReleases,
	)
}

func (r *repository) GetPipelineByUID(ctx context.Context, uid uuid.UUID, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error) {
	// TODO: ACL
	return r.getNamespacePipeline(ctx,
		"(uid = ?)",
		[]interface{}{uid},
		isBasicView,
		embedReleases,
	)
}

func (r *repository) GetPipelineByIDAdmin(ctx context.Context, id string, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error) {
	return r.getNamespacePipeline(ctx,
		"(id = ?)",
		[]interface{}{id},
		isBasicView,
		embedReleases,
	)
}

func (r *repository) GetPipelineByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error) {
	return r.getNamespacePipeline(ctx,
		"(uid = ?)",
		[]interface{}{uid},
		isBasicView,
		embedReleases,
	)
}

func (r *repository) UpdateNamespacePipelineByUID(ctx context.Context, uid uuid.UUID, pipeline *datamodel.Pipeline) error {

	r.pinUser(ctx, "pipeline")
	db := r.checkPinnedUser(ctx, r.db, "pipeline")
	if result := db.Unscoped().Model(&datamodel.Pipeline{}).
		Where("(uid = ?)", uid).
		Updates(pipeline); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) DeleteNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string) error {

	r.pinUser(ctx, "pipeline")
	db := r.checkPinnedUser(ctx, r.db, "pipeline")

	result := db.Model(&datamodel.Pipeline{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Delete(&datamodel.Pipeline{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNoDataDeleted
	}

	return nil
}

func (r *repository) UpdateNamespacePipelineIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error {

	r.pinUser(ctx, "pipeline")
	db := r.checkPinnedUser(ctx, r.db, "pipeline")

	if result := db.Model(&datamodel.Pipeline{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Update("id", newID); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

// TranspileFilter transpiles a parsed AIP filter expression to GORM DB clauses
func (r *repository) transpileFilter(filter filtering.Filter) (*clause.Expr, error) {
	return (&Transpiler{
		filter: filter,
	}).Transpile()
}

func (r *repository) CreateNamespacePipelineRelease(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, pipelineRelease *datamodel.PipelineRelease) error {

	r.pinUser(ctx, "pipeline_release")
	db := r.checkPinnedUser(ctx, r.db, "pipeline_release")

	if result := db.Model(&datamodel.PipelineRelease{}).Create(pipelineRelease); result.Error != nil {
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" || errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return errmsg.AddMessage(ErrNameExists, "Release version already exists")
		}
		return result.Error
	}
	return nil
}

func (r *repository) ListNamespacePipelineReleases(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool, returnCount bool) (pipelineReleases []*datamodel.PipelineRelease, totalSize int64, nextPageToken string, err error) {

	db := r.checkPinnedUser(ctx, r.db, "pipeline_release")

	if showDeleted {
		db = db.Unscoped()
	}

	if returnCount {
		if result := db.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid = ?", pipelineUID).Count(&totalSize); result.Error != nil {
			return nil, 0, "", result.Error
		}
	}

	queryBuilder := db.Model(&datamodel.PipelineRelease{}).Order("create_time DESC, uid DESC").Where("pipeline_uid = ?", pipelineUID)

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createTime, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, 0, "", ErrPageTokenDecode
		}
		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createTime, uid)
	}

	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe")
	}

	if expr, err := r.transpileFilter(filter); err != nil {
		return nil, 0, "", err
	} else if expr != nil {
		queryBuilder.Where("(?)", expr)
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()

	for rows.Next() {
		var item *datamodel.PipelineRelease
		if err = db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", err
		}
		createTime = item.CreateTime
		pipelineReleases = append(pipelineReleases, item)
	}

	if len(pipelineReleases) > 0 {
		lastUID := (pipelineReleases)[len(pipelineReleases)-1].UID
		lastItem := &datamodel.PipelineRelease{}
		db := r.checkPinnedUser(ctx, r.db, "pipeline_release")
		if result := db.Model(&datamodel.PipelineRelease{}).
			Where("pipeline_uid = ?", pipelineUID).
			Order("create_time ASC, uid ASC").
			Limit(1).Find(lastItem); result.Error != nil {
			return nil, 0, "", err
		}
		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return pipelineReleases, totalSize, nextPageToken, nil
}

func (r *repository) GetNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string, isBasicView bool) (*datamodel.PipelineRelease, error) {

	db := r.checkPinnedUser(ctx, r.db, "pipeline_release")

	queryBuilder := db.Model(&datamodel.PipelineRelease{}).Where("id = ? AND pipeline_uid = ?", id, pipelineUID)
	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe")
	}
	var pipelineRelease datamodel.PipelineRelease
	if result := queryBuilder.First(&pipelineRelease); result.Error != nil {
		return nil, result.Error
	}

	return &pipelineRelease, nil
}

func (r *repository) UpdateNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string, pipelineRelease *datamodel.PipelineRelease) error {

	r.pinUser(ctx, "pipeline_release")
	db := r.checkPinnedUser(ctx, r.db, "pipeline_release")

	if result := db.Model(&datamodel.PipelineRelease{}).
		Where("id = ? AND pipeline_uid = ?", id, pipelineUID).
		Updates(pipelineRelease); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) DeleteNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string) error {

	r.pinUser(ctx, "pipeline_release")
	db := r.checkPinnedUser(ctx, r.db, "pipeline_release")

	result := db.Model(&datamodel.PipelineRelease{}).
		Where("id = ? AND pipeline_uid = ?", id, pipelineUID).
		Delete(&datamodel.PipelineRelease{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNoDataDeleted
	}

	return nil
}

func (r *repository) UpdateNamespacePipelineReleaseIDByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string, newID string) error {

	r.pinUser(ctx, "pipeline_release")
	db := r.checkPinnedUser(ctx, r.db, "pipeline_release")

	if result := db.Model(&datamodel.PipelineRelease{}).
		Where("id = ? AND pipeline_uid = ?", id, pipelineUID).
		Update("id", newID); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) GetLatestNamespacePipelineRelease(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error) {

	db := r.checkPinnedUser(ctx, r.db, "pipeline_release")

	queryBuilder := db.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid = ?", pipelineUID).Order("id DESC")
	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe")
	}
	var pipelineRelease datamodel.PipelineRelease
	if result := queryBuilder.First(&pipelineRelease); result.Error != nil {
		return nil, result.Error
	}
	return &pipelineRelease, nil
}

// ListComponentDefinitionsParams allows clients to request a page of component
// definitions.
type ListComponentDefinitionsParams struct {
	Offset int
	Limit  int
	Filter filtering.Filter
}

// ListComponentDefinitionUIDs returns the UIDs of a page of component
// definitions.
//
// The source of truth for a compnent definition is its JSON
// specification. These are loaded in memory, but we hold a table that allows
// us to quiclky transpile query filters and to have unified filtering and
// pagination.
//
// Since the component definitions might take different shapes, we need to know
// the component type in order to cast the definition to the right type.
// Therefore, the whole datamodel is returned (some fields won't be needed by
// the receiver but this solution is more compact than adding yet another type
// with no methods).
func (r *repository) ListComponentDefinitionUIDs(_ context.Context, p ListComponentDefinitionsParams) (defs []*datamodel.ComponentDefinition, totalSize int64, err error) {
	db := r.db
	where := ""
	whereArgs := []any{}

	expr, err := r.transpileFilter(p.Filter)
	if err != nil {
		return nil, 0, err
	}

	if expr != nil {
		where = "(?)"
		whereArgs = []any{expr}
	}

	queryBuilder := db.Model(&datamodel.ComponentDefinition{}).
		Where(where, whereArgs...).
		Where("is_visible IS TRUE")

	queryBuilder.Count(&totalSize)

	// Several results might have the same score and release stage. We need to
	// sort by at least one unique field so the pagination results aren't
	// arbitrary.
	orderBy := "feature_score DESC, release_stage DESC, uid DESC"
	rows, err := queryBuilder.Order(orderBy).Limit(p.Limit).Offset(p.Offset).Rows()
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	defs = make([]*datamodel.ComponentDefinition, 0, p.Limit)
	for rows.Next() {
		item := new(datamodel.ComponentDefinition)
		if err = db.ScanRows(rows, item); err != nil {
			return nil, 0, err
		}

		defs = append(defs, item)
	}

	return defs, totalSize, nil
}

// GetComponentDefinition fetches the component definition datamodel given its
// UID. Note that the repository only stores an index of the component
// definition fields that the clients need to filter by and that the source of
// truth for component definition info is always the definitions.json
// configuration in each component.
func (r *repository) GetComponentDefinitionByUID(_ context.Context, uid uuid.UUID) (*datamodel.ComponentDefinition, error) {
	record := new(datamodel.ComponentDefinition)

	if result := r.db.Model(record).Where("uid = ?", uid.String()).First(record); result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}

		return nil, result.Error
	}

	return record, nil
}

// UpsertComponentDefinition transforms a domain component definition into its
// datamodel (i.e. the fields used for filtering) and stores it in the
// database. If the record already exists, it will be updated with the provided
// fields.
func (r *repository) UpsertComponentDefinition(_ context.Context, cd *pb.ComponentDefinition) error {
	record := datamodel.ComponentDefinitionFromProto(cd)
	result := r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(record)
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *repository) CreateNamespaceSecret(ctx context.Context, ownerPermalink string, secret *datamodel.Secret) error {
	r.pinUser(ctx, "secret")
	db := r.checkPinnedUser(ctx, r.db, "secret")

	logger, _ := logger.GetZapLogger(ctx)
	if result := db.Model(&datamodel.Secret{}).Create(secret); result.Error != nil {
		logger.Error(result.Error.Error())
		var pgErr *pgconn.PgError
		if errors.As(result.Error, &pgErr) && pgErr.Code == "23505" || errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return errmsg.AddMessage(ErrNameExists, "Secret ID already exists")
		}
		return result.Error
	}
	return nil
}

func (r *repository) ListNamespaceSecrets(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, filter filtering.Filter) (secrets []*datamodel.Secret, totalSize int64, nextPageToken string, err error) {
	db := r.checkPinnedUser(ctx, r.db, "secret")

	if result := db.Model(&datamodel.Secret{}).Where("owner = ?", ownerPermalink).Count(&totalSize); result.Error != nil {
		return nil, 0, "", err
	}

	queryBuilder := db.Model(&datamodel.Secret{}).Order("create_time DESC, uid DESC").Where("owner = ?", ownerPermalink)

	if pageSize == 0 {
		pageSize = DefaultPageSize
	} else if pageSize > MaxPageSize {
		pageSize = MaxPageSize
	}

	queryBuilder = queryBuilder.Limit(int(pageSize))

	if pageToken != "" {
		createTime, uid, err := paginate.DecodeToken(pageToken)
		if err != nil {
			return nil, 0, "", err
		}
		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createTime, uid)
	}

	var createTime time.Time
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Secret
		if err = db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", err
		}
		createTime = item.CreateTime
		secrets = append(secrets, &item)
	}

	if len(secrets) > 0 {
		lastUID := (secrets)[len(secrets)-1].UID
		lastItem := &datamodel.Secret{}
		if result := db.Model(&datamodel.Secret{}).
			Where("owner = ?", ownerPermalink).
			Order("create_time ASC, uid ASC").
			Limit(1).Find(lastItem); result.Error != nil {
			return nil, 0, "", err
		}
		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken = paginate.EncodeToken(createTime, lastUID.String())
		}
	}

	return secrets, totalSize, nextPageToken, nil
}

func (r *repository) GetNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string) (*datamodel.Secret, error) {
	db := r.checkPinnedUser(ctx, r.db, "secret")

	queryBuilder := db.Model(&datamodel.Secret{}).Where("id = ? AND owner = ?", id, ownerPermalink)
	var secret datamodel.Secret
	if result := queryBuilder.First(&secret); result.Error != nil {
		return nil, result.Error
	}
	return &secret, nil
}

func (r *repository) UpdateNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string, secret *datamodel.Secret) error {
	r.pinUser(ctx, "secret")
	db := r.checkPinnedUser(ctx, r.db, "secret")

	logger, _ := logger.GetZapLogger(ctx)
	if result := db.Select("*").Omit("UID").Model(&datamodel.Secret{}).Where("id = ? AND owner = ?", id, ownerPermalink).Updates(secret); result.Error != nil {
		logger.Error(result.Error.Error())
		return result.Error
	}
	return nil
}

func (r *repository) DeleteNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string) error {
	r.pinUser(ctx, "secret")
	db := r.checkPinnedUser(ctx, r.db, "secret")

	result := db.Model(&datamodel.Secret{}).
		Where("id = ? AND owner = ?", id, ownerPermalink).
		Delete(&datamodel.Secret{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNoDataDeleted
	}

	return nil
}
