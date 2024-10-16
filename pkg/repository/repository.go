package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/iancoleman/strcase"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/logger"
	"github.com/instill-ai/pipeline-backend/pkg/resource"
	"github.com/instill-ai/x/paginate"

	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// TODO: in the repository, we'd better use uid as our function params

// DefaultPageSize is the default pagination page size when page size is not assigned
const DefaultPageSize = 10

// MaxPageSize is the maximum pagination page size if the assigned value is over this number
const MaxPageSize = 100

// Repository interface
type Repository interface {
	PinUser(_ context.Context, table string)
	CheckPinnedUser(_ context.Context, _ *gorm.DB, table string) *gorm.DB

	GetHubStats(uidAllowList []uuid.UUID) (*datamodel.HubStats, error)
	ListPipelines(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool, order ordering.OrderBy, presetNamespaceUID uuid.UUID) ([]*datamodel.Pipeline, int64, string, error)
	GetPipelineByUID(ctx context.Context, uid uuid.UUID, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error)

	CreateNamespacePipeline(ctx context.Context, pipeline *datamodel.Pipeline) error
	ListNamespacePipelines(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool, order ordering.OrderBy) ([]*datamodel.Pipeline, int64, string, error)
	GetNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error)

	UpdateNamespacePipelineByUID(ctx context.Context, uid uuid.UUID, pipeline *datamodel.Pipeline) error
	DeleteNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string) error
	UpdateNamespacePipelineIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error

	AddPipelineRuns(ctx context.Context, uid uuid.UUID) error
	AddPipelineClones(ctx context.Context, uid uuid.UUID) error

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
	GetPipelineReleaseByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error)

	ListComponentDefinitionUIDs(context.Context, ListComponentDefinitionsParams) (uids []*datamodel.ComponentDefinition, totalSize int64, err error)
	GetDefinitionByUID(context.Context, uuid.UUID) (*datamodel.ComponentDefinition, error)
	UpsertComponentDefinition(context.Context, *pb.ComponentDefinition) error
	ListIntegrations(context.Context, ListIntegrationsParams) (IntegrationList, error)

	CreateNamespaceConnection(context.Context, *datamodel.Connection) (*datamodel.Connection, error)
	UpdateNamespaceConnectionByUID(context.Context, uuid.UUID, *datamodel.Connection) (*datamodel.Connection, error)
	DeleteNamespaceConnectionByID(_ context.Context, nsUID uuid.UUID, id string) error
	GetNamespaceConnectionByID(_ context.Context, nsUID uuid.UUID, id string) (*datamodel.Connection, error)
	ListNamespaceConnections(context.Context, ListNamespaceConnectionsParams) (ConnectionList, error)
	ListPipelineIDsByConnectionID(context.Context, ListPipelineIDsByConnectionIDParams) (PipelinesByConnectionList, error)

	CreateNamespaceSecret(ctx context.Context, ownerPermalink string, secret *datamodel.Secret) error
	ListNamespaceSecrets(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, filter filtering.Filter) ([]*datamodel.Secret, int64, string, error)
	GetNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string) (*datamodel.Secret, error)
	UpdateNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string, secret *datamodel.Secret) error
	DeleteNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string) error
	CreatePipelineTags(ctx context.Context, pipelineUID uuid.UUID, tagNames []string) error
	DeletePipelineTags(ctx context.Context, pipelineUID uuid.UUID, tagNames []string) error
	ListPipelineTags(ctx context.Context, pipelineUID uuid.UUID) ([]datamodel.Tag, error)

	// TODO this function can remain unexported once connector and operator
	// definition lists are removed.
	TranspileFilter(filtering.Filter) (*clause.Expr, error)

	GetPipelineRunByUID(context.Context, uuid.UUID) (*datamodel.PipelineRun, error)
	UpsertPipelineRun(ctx context.Context, pipelineRun *datamodel.PipelineRun) error
	UpdatePipelineRun(ctx context.Context, pipelineTriggerUID string, pipelineRun *datamodel.PipelineRun) error
	UpsertComponentRun(ctx context.Context, componentRun *datamodel.ComponentRun) error
	UpdateComponentRun(ctx context.Context, pipelineTriggerUID, componentID string, componentRun *datamodel.ComponentRun) error

	GetPaginatedPipelineRunsWithPermissions(ctx context.Context, requesterUID, pipelineUID string, page, pageSize int, filter filtering.Filter, order ordering.OrderBy, isOwner bool) ([]datamodel.PipelineRun, int64, error)
	GetPaginatedComponentRunsByPipelineRunIDWithPermissions(ctx context.Context, pipelineRunID string, page, pageSize int, filter filtering.Filter, order ordering.OrderBy) ([]datamodel.ComponentRun, int64, error)
	GetPaginatedPipelineRunsByRequester(ctx context.Context, params GetPipelineRunsByRequesterParams) ([]datamodel.PipelineRun, int64, error)
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

func (r *repository) toDomainErr(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" || errors.Is(err, gorm.ErrDuplicatedKey) {
		return errdomain.ErrAlreadyExists
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errdomain.ErrNotFound
	}

	return err
}

func decodeCursor[T any](token string) (cursor T, err error) {
	b, err := base64.StdEncoding.DecodeString(token)
	if err != nil {
		return cursor, fmt.Errorf("decoding token: %w", err)
	}

	if err := json.Unmarshal(b, &cursor); err != nil {
		return cursor, fmt.Errorf("unmarshalling token: %w", err)
	}

	return cursor, nil
}

func encodeCursor[T any](cursor T) (string, error) {
	b, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("marshalling cursor: %w", err)
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

func (r *repository) GetHubStats(uidAllowList []uuid.UUID) (*datamodel.HubStats, error) {

	db := r.db

	var totalSize int64
	var totalFeaturedSize int64

	db.Model(&datamodel.Pipeline{}).Where("uid in ?", uidAllowList).Count(&totalSize)
	db.Model(&datamodel.Pipeline{}).Joins("left join tag on tag.pipeline_uid = pipeline.uid").
		Where("uid in ?", uidAllowList).
		Where("tag.tag_name = ?", "featured").
		Count(&totalFeaturedSize)

	return &datamodel.HubStats{
		NumberOfPublicPipelines:   int32(totalSize),
		NumberOfFeaturedPipelines: int32(totalFeaturedSize),
	}, nil
}

// CheckPinnedUser uses the primary database for querying if the user is
// pinned. This is used to solve read-after-write inconsistency problems on
// multi-region setups.
func (r *repository) CheckPinnedUser(ctx context.Context, db *gorm.DB, table string) *gorm.DB {
	db = db.WithContext(ctx)

	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if userUID == "" {
		return db
	}

	if !errors.Is(r.redisClient.Get(ctx, fmt.Sprintf("db_pin_user:%s:%s", userUID, table)).Err(), redis.Nil) {
		return db.Clauses(dbresolver.Write)
	}

	return db
}

// PinUser sets the primary database as the read of the user for a certain
// period to ensure that the data is synchronized from the primary DB to the
// replica DB.
func (r *repository) PinUser(ctx context.Context, table string) {
	userUID := resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey)
	if userUID == "" {
		return
	}

	_ = r.redisClient.Set(ctx, fmt.Sprintf("db_pin_user:%s:%s", userUID, table), time.Now(), time.Duration(config.Config.Database.Replica.ReplicationTimeFrame)*time.Second)
}

func (r *repository) CreateNamespacePipeline(ctx context.Context, pipeline *datamodel.Pipeline) error {
	r.PinUser(ctx, "pipeline")
	db := r.CheckPinnedUser(ctx, r.db, "pipeline")

	err := db.Model(&datamodel.Pipeline{}).Create(pipeline).Error
	return r.toDomainErr(err)
}

func (r *repository) listPipelines(ctx context.Context, where string, whereArgs []interface{}, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool, order ordering.OrderBy) (pipelines []*datamodel.Pipeline, totalSize int64, nextPageToken string, err error) {

	db := r.db
	if showDeleted {
		db = db.Unscoped()
	}

	var expr *clause.Expr
	if expr, err = r.TranspileFilter(filter); err != nil {
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

	joinStr := "left join tag on tag.pipeline_uid = pipeline.uid"

	countBuilder := db.Distinct("pipeline.uid").Model(&datamodel.Pipeline{}).Where(where, whereArgs...).Joins(joinStr)
	if uidAllowList != nil {
		countBuilder = countBuilder.Where("uid in ?", uidAllowList).Count(&totalSize)
	}

	countBuilder.Count(&totalSize)

	queryBuilder := db.Distinct().Model(&datamodel.Pipeline{}).Joins(joinStr).Where(where, whereArgs...)
	if len(order.Fields) == 0 {
		order.Fields = append(order.Fields, ordering.Field{
			Path: "create_time",
			Desc: true,
		})
	}

	for _, field := range order.Fields {
		// TODO: We should implement a shared `orderBy` parser.
		orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(field.Desc)
		queryBuilder.Order(orderString)
	}
	queryBuilder.Order("uid DESC")

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
		tokens, err := DecodeToken(pageToken)
		if err != nil {
			return nil, 0, "", newPageTokenErr(err)
		}

		for _, o := range order.Fields {

			p := strcase.ToSnake(o.Path)
			if v, ok := tokens[p]; ok {

				switch p {
				case "create_time", "update_time", "last_run_time":
					// Add "pipeline." prefix to prevent ambiguous since tag table also has the two columns.
					if o.Desc {
						queryBuilder = queryBuilder.Where("pipeline."+p+" < ?::timestamp", v)
					} else {
						queryBuilder = queryBuilder.Where("pipeline."+p+" > ?::timestamp", v)
					}
				default:
					if o.Desc {
						queryBuilder = queryBuilder.Where(p+" < ?", v)
					} else {
						queryBuilder = queryBuilder.Where(p+" > ?", v)
					}
				}

			}
		}

	}

	if isBasicView {
		queryBuilder.Omit("pipeline.recipe_yaml")
	}

	result := queryBuilder.Preload("Tags").Find(&pipelines)
	if result.Error != nil {
		return nil, 0, "", result.Error
	}
	pipelineUIDs := []uuid.UUID{}

	for _, p := range pipelines {
		pipelineUIDs = append(pipelineUIDs, p.UID)
	}

	if embedReleases {
		releaseDB := r.CheckPinnedUser(ctx, r.db, "pipeline")
		releasesMap := map[uuid.UUID][]*datamodel.PipelineRelease{}
		releaseDBQueryBuilder := releaseDB.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid in ?", pipelineUIDs).Order("create_time DESC, uid DESC")
		if isBasicView {
			releaseDBQueryBuilder.Omit("pipeline_release.recipe_yaml")
		}
		pipelineReleases := []*datamodel.PipelineRelease{}
		result := releaseDBQueryBuilder.Find(&pipelineReleases)
		if result.Error != nil {
			return nil, 0, "", result.Error
		}
		for idx := range pipelineReleases {

			pipelineUID := pipelineReleases[idx].PipelineUID
			if _, ok := releasesMap[pipelineUID]; !ok {
				releasesMap[pipelineUID] = []*datamodel.PipelineRelease{}
			}
			releasesMap[pipelineUID] = append(releasesMap[pipelineUID], pipelineReleases[idx])
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

		tokens := map[string]any{}

		lastItemQueryBuilder := db.Distinct().Model(&datamodel.Pipeline{}).Joins(joinStr).Where(where, whereArgs...)
		if uidAllowList != nil {
			lastItemQueryBuilder = lastItemQueryBuilder.Where("uid in ?", uidAllowList)

		}

		for _, field := range order.Fields {
			orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(!field.Desc)
			lastItemQueryBuilder.Order(orderString)
			switch p := strcase.ToSnake(field.Path); p {
			case "id":
				tokens[p] = (pipelines)[len(pipelines)-1].ID
			case "create_time":
				tokens[p] = (pipelines)[len(pipelines)-1].CreateTime.Format(time.RFC3339Nano)
			case "update_time":
				tokens[p] = (pipelines)[len(pipelines)-1].UpdateTime.Format(time.RFC3339Nano)
			case "last_run_time":
				tokens[p] = (pipelines)[len(pipelines)-1].LastRunTime.Format(time.RFC3339Nano)
			case "number_of_runs":
				tokens[p] = (pipelines)[len(pipelines)-1].NumberOfRuns
			case "number_of_clones":
				tokens[p] = (pipelines)[len(pipelines)-1].NumberOfClones
			}

		}
		lastItemQueryBuilder.Order("uid ASC")
		tokens["uid"] = lastUID.String()

		if result := lastItemQueryBuilder.Limit(1).Find(lastItem); result.Error != nil {
			return nil, 0, "", err
		}

		if lastItem.UID.String() == lastUID.String() {
			nextPageToken = ""
		} else {
			nextPageToken, err = EncodeToken(tokens)
			if err != nil {
				return nil, 0, "", err
			}
		}
	}

	return pipelines, totalSize, nextPageToken, nil
}

func (r *repository) ListPipelines(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool, order ordering.OrderBy, presetNamespaceUID uuid.UUID) ([]*datamodel.Pipeline, int64, string, error) {
	// Note: Preset pipelines are ignored in `GET /pipelines` requests. These
	// pipelines are used by the artifact backend and should not be directly
	// exposed to users.
	return r.listPipelines(ctx,
		"(owner != ?)",
		[]interface{}{fmt.Sprintf("organizations/%s", presetNamespaceUID)},
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted, embedReleases, order)
}
func (r *repository) ListNamespacePipelines(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool, embedReleases bool, order ordering.OrderBy) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx,
		"(owner = ?)",
		[]interface{}{ownerPermalink},
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted, embedReleases, order)
}

func (r *repository) ListPipelinesAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool, embedReleases bool) ([]*datamodel.Pipeline, int64, string, error) {
	return r.listPipelines(ctx, "", []interface{}{}, pageSize, pageToken, isBasicView, filter, nil, showDeleted, embedReleases, ordering.OrderBy{})
}

func (r *repository) getNamespacePipeline(ctx context.Context, where string, whereArgs []interface{}, isBasicView bool, embedReleases bool) (*datamodel.Pipeline, error) {

	db := r.CheckPinnedUser(ctx, r.db, "pipeline")

	var pipeline datamodel.Pipeline

	queryBuilder := db.Model(&datamodel.Pipeline{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("pipeline.recipe_yaml")
	}

	if result := queryBuilder.First(&pipeline); result.Error != nil {
		return nil, result.Error
	}

	if embedReleases {
		pipeline.Releases = []*datamodel.PipelineRelease{}

		releaseDB := r.CheckPinnedUser(ctx, r.db, "pipeline")
		releaseDBQueryBuilder := releaseDB.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid = ?", pipeline.UID).Order("create_time DESC, uid DESC")
		if isBasicView {
			releaseDBQueryBuilder.Omit("pipeline_release.recipe_yaml")
		}

		pipelineReleases := []*datamodel.PipelineRelease{}
		result := releaseDBQueryBuilder.Find(&pipelineReleases)
		if result.Error != nil {
			return nil, result.Error
		}
		pipeline.Releases = pipelineReleases
	}

	pipeline.Tags = []*datamodel.Tag{}
	tagDB := r.CheckPinnedUser(ctx, r.db, "tag")
	tagDBQueryBuilder := tagDB.Model(&datamodel.Tag{}).Where("pipeline_uid = ?", pipeline.UID)
	tagDBQueryBuilder.Find(&pipeline.Tags)

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
func (r *repository) GetPipelineReleaseByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.PipelineRelease, error) {
	db := r.CheckPinnedUser(ctx, r.db, "pipeline_release")

	queryBuilder := db.Model(&datamodel.PipelineRelease{}).Where("uid = ?", uid)
	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe_yaml")
	}
	var pipelineRelease datamodel.PipelineRelease
	if result := queryBuilder.First(&pipelineRelease); result.Error != nil {
		return nil, result.Error
	}

	return &pipelineRelease, nil
}

func (r *repository) UpdateNamespacePipelineByUID(ctx context.Context, uid uuid.UUID, pipeline *datamodel.Pipeline) error {

	r.PinUser(ctx, "pipeline")
	db := r.CheckPinnedUser(ctx, r.db, "pipeline")

	// Note: To make the BeforeUpdate hook work, we need to use
	// `Model(pipeline)` instead of `Model(&datamodel.Pipeline{})`.
	if result := db.Unscoped().Model(pipeline).
		Where("(uid = ?)", uid).
		Updates(pipeline); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) DeleteNamespacePipelineByID(ctx context.Context, ownerPermalink string, id string) error {

	r.PinUser(ctx, "pipeline")
	db := r.CheckPinnedUser(ctx, r.db, "pipeline")

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

	r.PinUser(ctx, "pipeline")
	db := r.CheckPinnedUser(ctx, r.db, "pipeline")

	if result := db.Model(&datamodel.Pipeline{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Update("id", newID); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

// TranspileFilter transpiles a parsed AIP filter expression to GORM DB clauses.
func (r *repository) TranspileFilter(filter filtering.Filter) (*clause.Expr, error) {
	return (&transpiler{filter: filter}).Transpile()
}

func (r *repository) CreateNamespacePipelineRelease(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, pipelineRelease *datamodel.PipelineRelease) error {

	r.PinUser(ctx, "pipeline_release")
	db := r.CheckPinnedUser(ctx, r.db, "pipeline_release")

	err := db.Model(&datamodel.PipelineRelease{}).Create(pipelineRelease).Error
	return r.toDomainErr(err)
}

func (r *repository) ListNamespacePipelineReleases(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool, returnCount bool) (pipelineReleases []*datamodel.PipelineRelease, totalSize int64, nextPageToken string, err error) {

	db := r.CheckPinnedUser(ctx, r.db, "pipeline_release")

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
			return nil, 0, "", newPageTokenErr(err)
		}
		queryBuilder = queryBuilder.Where("(create_time,uid) < (?::timestamp, ?)", createTime, uid)
	}

	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe_yaml")
	}

	if expr, err := r.TranspileFilter(filter); err != nil {
		return nil, 0, "", err
	} else if expr != nil {
		queryBuilder.Where("(?)", expr)
	}

	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()

	result := queryBuilder.Find(&pipelineReleases)
	if result.Error != nil {
		return nil, 0, "", result.Error
	}

	if len(pipelineReleases) > 0 {
		createTime := pipelineReleases[len(pipelineReleases)-1].CreateTime
		lastUID := (pipelineReleases)[len(pipelineReleases)-1].UID
		lastItem := &datamodel.PipelineRelease{}
		db := r.CheckPinnedUser(ctx, r.db, "pipeline_release")
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

	db := r.CheckPinnedUser(ctx, r.db, "pipeline_release")

	queryBuilder := db.Model(&datamodel.PipelineRelease{}).Where("id = ? AND pipeline_uid = ?", id, pipelineUID)
	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe_yaml")
	}
	var pipelineRelease datamodel.PipelineRelease
	if result := queryBuilder.First(&pipelineRelease); result.Error != nil {
		return nil, result.Error
	}

	return &pipelineRelease, nil
}

func (r *repository) UpdateNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string, pipelineRelease *datamodel.PipelineRelease) error {

	r.PinUser(ctx, "pipeline_release")
	db := r.CheckPinnedUser(ctx, r.db, "pipeline_release")

	if result := db.Model(pipelineRelease).
		Where("id = ? AND pipeline_uid = ?", id, pipelineUID).
		Updates(pipelineRelease); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) DeleteNamespacePipelineReleaseByID(ctx context.Context, ownerPermalink string, pipelineUID uuid.UUID, id string) error {

	r.PinUser(ctx, "pipeline_release")
	db := r.CheckPinnedUser(ctx, r.db, "pipeline_release")

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

	r.PinUser(ctx, "pipeline_release")
	db := r.CheckPinnedUser(ctx, r.db, "pipeline_release")

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

	db := r.CheckPinnedUser(ctx, r.db, "pipeline_release")

	queryBuilder := db.Model(&datamodel.PipelineRelease{}).Where("pipeline_uid = ?", pipelineUID).Order("id DESC")
	if isBasicView {
		queryBuilder.Omit("pipeline_release.recipe_yaml")
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
// The source of truth for a component definition is its JSON
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

	expr, err := r.TranspileFilter(p.Filter)
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
func (r *repository) GetDefinitionByUID(_ context.Context, uid uuid.UUID) (*datamodel.ComponentDefinition, error) {
	record := new(datamodel.ComponentDefinition)

	if err := r.db.Model(record).Where("uid = ?", uid.String()).First(record).Error; err != nil {
		return nil, r.toDomainErr(err)
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

// ListIntegrationsParams allows clients to request a page of integrations.
type ListIntegrationsParams struct {
	PageToken string
	Limit     int
	Filter    filtering.Filter
}

// IntegrationList returns a page of integrations.
type IntegrationList struct {
	ComponentDefinitions []*datamodel.ComponentDefinition
	NextPageToken        string
	TotalSize            int32
}

type integrationCursor struct {
	Score int       `json:"score"`
	UID   uuid.UUID `json:"uid"`
}

// ListIntegrations returns the UIDs and indexed information of a page of
// integrations.
//
// The source of truth for integration is the JSON specification of its
// component definition. The database only holds the indexed information of
// component definitions in order to filter and paginate them.
func (r *repository) ListIntegrations(ctx context.Context, p ListIntegrationsParams) (IntegrationList, error) {
	var resp IntegrationList

	db := r.db.WithContext(ctx)
	where := ""
	whereArgs := []any{}

	// Get item count.
	expr, err := r.TranspileFilter(p.Filter)
	if err != nil {
		return resp, fmt.Errorf("transpiling filter: %w", err)
	}

	if expr != nil {
		where = "(?)"
		whereArgs = []any{expr}
	}

	queryBuilder := db.Model(&datamodel.ComponentDefinition{}).
		Where(where, whereArgs...).
		Where("is_visible IS TRUE AND has_integration IS TRUE")

	var count int64
	queryBuilder.Count(&count)
	resp.TotalSize = int32(count)

	// Get definitions matching criteria.
	if p.PageToken != "" {
		cursor, err := decodeCursor[integrationCursor](p.PageToken)
		if err != nil {
			return resp, err
		}
		queryBuilder = queryBuilder.Where("(feature_score,uid) < (?, ?)", cursor.Score, cursor.UID)
	}

	// From here we'll apply different search criteria.
	queryBuilder = queryBuilder.Session(&gorm.Session{})

	// Several results might have the same score and release stage. We need to
	// sort by at least one unique field so the pagination results aren't
	// arbitrary.
	resp.ComponentDefinitions = make([]*datamodel.ComponentDefinition, 0, p.Limit)
	err = queryBuilder.
		Limit(p.Limit).
		Order("feature_score DESC, uid DESC").
		Find(&resp.ComponentDefinitions).Error
	if err != nil {
		return resp, fmt.Errorf("querying database rows: %w", err)
	}

	if len(resp.ComponentDefinitions) == 0 {
		return resp, nil
	}

	lastInPage := resp.ComponentDefinitions[len(resp.ComponentDefinitions)-1]
	lastInDB := new(datamodel.ComponentDefinition)

	orderInverse := "feature_score ASC, uid ASC"
	if result := queryBuilder.Order(orderInverse).Limit(1).Find(lastInDB); result.Error != nil {
		return resp, fmt.Errorf("finding last item: %w", err)
	}

	if lastInDB.UID.String() == lastInPage.UID.String() {
		return resp, nil
	}

	nextCursor := integrationCursor{
		Score: lastInPage.FeatureScore,
		UID:   lastInPage.UID,
	}
	resp.NextPageToken, err = encodeCursor[integrationCursor](nextCursor)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

func (r *repository) CreateNamespaceSecret(ctx context.Context, ownerPermalink string, secret *datamodel.Secret) error {
	r.PinUser(ctx, "secret")
	db := r.CheckPinnedUser(ctx, r.db, "secret")

	err := db.Model(&datamodel.Secret{}).Create(secret).Error
	return r.toDomainErr(err)
}

func (r *repository) ListNamespaceSecrets(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, filter filtering.Filter) (secrets []*datamodel.Secret, totalSize int64, nextPageToken string, err error) {
	db := r.CheckPinnedUser(ctx, r.db, "secret")

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
	db := r.CheckPinnedUser(ctx, r.db, "secret")

	queryBuilder := db.Model(&datamodel.Secret{}).Where("id = ? AND owner = ?", id, ownerPermalink)
	var secret datamodel.Secret
	if result := queryBuilder.First(&secret); result.Error != nil {
		return nil, result.Error
	}
	return &secret, nil
}

func (r *repository) UpdateNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string, secret *datamodel.Secret) error {
	r.PinUser(ctx, "secret")
	db := r.CheckPinnedUser(ctx, r.db, "secret")

	logger, _ := logger.GetZapLogger(ctx)
	if result := db.Select("*").Omit("UID").Model(&datamodel.Secret{}).Where("id = ? AND owner = ?", id, ownerPermalink).Updates(secret); result.Error != nil {
		logger.Error(result.Error.Error())
		return result.Error
	}
	return nil
}

func (r *repository) DeleteNamespaceSecretByID(ctx context.Context, ownerPermalink string, id string) error {
	r.PinUser(ctx, "secret")
	db := r.CheckPinnedUser(ctx, r.db, "secret")

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

func (r *repository) CreatePipelineTags(ctx context.Context, pipelineUID uuid.UUID, tagNames []string) error {

	r.PinUser(ctx, "tag")

	db := r.CheckPinnedUser(ctx, r.db, "tag")

	tags := []datamodel.Tag{}
	for _, tagName := range tagNames {
		tag := datamodel.Tag{
			PipelineUID: pipelineUID,
			TagName:     tagName,
			CreateTime:  time.Now(),
			UpdateTime:  time.Now(),
		}
		tags = append(tags, tag)
	}

	err := db.Model(&datamodel.Tag{}).Create(&tags).Error
	return r.toDomainErr(err)
}

func (r *repository) DeletePipelineTags(ctx context.Context, pipelineUID uuid.UUID, tagNames []string) error {

	r.PinUser(ctx, "tag")

	db := r.CheckPinnedUser(ctx, r.db, "tag")

	result := db.Model(&datamodel.Tag{}).Where("pipeline_uid = ? and tag_name in ?", pipelineUID, tagNames).Delete(&datamodel.Tag{})

	if result.Error != nil {

		return result.Error

	}

	if result.RowsAffected == 0 {

		return ErrNoDataDeleted

	}

	return nil

}

func (r *repository) ListPipelineTags(ctx context.Context, pipelineUID uuid.UUID) ([]datamodel.Tag, error) {

	db := r.db

	var tags []datamodel.Tag

	result := db.Model(&datamodel.Tag{}).Where("pipeline_uid = ?", pipelineUID).Find(&tags)

	if result.Error != nil {

		return nil, result.Error

	}

	return tags, nil

}

func (r *repository) AddPipelineRuns(ctx context.Context, pipelineUID uuid.UUID) error {
	db := r.db.WithContext(ctx)

	result := db.Model(&datamodel.Pipeline{}).
		Where("uid = ?", pipelineUID).
		UpdateColumns(map[string]any{
			"last_run_time":  time.Now(),
			"number_of_runs": gorm.Expr("number_of_runs + 1"),
		})
	if result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *repository) AddPipelineClones(ctx context.Context, pipelineUID uuid.UUID) error {

	db := r.db

	if result := db.Model(&datamodel.Pipeline{}).
		Where("uid = ?", pipelineUID).
		UpdateColumn("number_of_clones", gorm.Expr("number_of_clones + 1")); result.Error != nil {
		return result.Error
	}

	return nil
}

func (r *repository) GetPipelineRunByUID(ctx context.Context, pipelineTriggerUID uuid.UUID) (*datamodel.PipelineRun, error) {
	pipelineRun := &datamodel.PipelineRun{PipelineTriggerUID: pipelineTriggerUID}
	err := r.db.Preload(clause.Associations).First(pipelineRun).Error
	if err != nil {
		return nil, err
	}

	return pipelineRun, nil
}

func (r *repository) UpsertPipelineRun(ctx context.Context, pipelineRun *datamodel.PipelineRun) error {
	return r.db.Save(pipelineRun).Error
}

func (r *repository) UpdatePipelineRun(ctx context.Context, pipelineTriggerUID string, pipelineRun *datamodel.PipelineRun) error {
	uid := uuid.FromStringOrNil(pipelineTriggerUID)
	return r.db.Model(&datamodel.PipelineRun{}).Where(&datamodel.PipelineRun{PipelineTriggerUID: uid}).Updates(&pipelineRun).Error
}

func (r *repository) UpsertComponentRun(ctx context.Context, componentRun *datamodel.ComponentRun) error {
	return r.db.Save(componentRun).Error
}

func (r *repository) UpdateComponentRun(ctx context.Context, pipelineTriggerUID, componentID string, componentRun *datamodel.ComponentRun) error {
	uid := uuid.FromStringOrNil(pipelineTriggerUID)
	return r.db.Model(&datamodel.ComponentRun{}).Where(&datamodel.ComponentRun{PipelineTriggerUID: uid, ComponentID: componentID}).Updates(componentRun).Error
}

func (r *repository) GetPaginatedPipelineRunsWithPermissions(ctx context.Context, requesterUID, pipelineUID string, page, pageSize int, filter filtering.Filter, order ordering.OrderBy, isOwner bool) ([]datamodel.PipelineRun, int64, error) {
	var pipelineRuns []datamodel.PipelineRun
	var totalRows int64

	whereConditions := []string{"pipeline_uid = ?"}
	whereArgs := []any{pipelineUID}

	var expr *clause.Expr
	var err error
	if expr, err = r.TranspileFilter(filter); err != nil {
		return nil, 0, err
	}
	if expr != nil {
		whereConditions = append(whereConditions, "(?)")
		whereArgs = append(whereArgs, expr)
	}

	if !isOwner { // for a view ns without ownership, they could only view the logs in same ns
		whereConditions = append(whereConditions, "namespace = ?")
		whereArgs = append(whereArgs, requesterUID)
	}

	var where string
	if len(whereConditions) > 0 {
		where = strings.Join(whereConditions, " and ")
	}

	// Count total rows with permissions
	err = r.db.Model(&datamodel.PipelineRun{}).
		Where(where, whereArgs...).
		Count(&totalRows).Error
	if err != nil {
		return nil, 0, err
	}

	queryBuilder := r.db.Where(where, whereArgs...)

	if len(order.Fields) == 0 {
		order.Fields = append(order.Fields, ordering.Field{
			Path: "started_time",
			Desc: true,
		})
	}

	for _, field := range order.Fields {
		orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(field.Desc)
		queryBuilder.Order(orderString)
	}

	// Retrieve paginated results with permissions
	err = queryBuilder.
		Offset(page * pageSize).Limit(pageSize).
		Find(&pipelineRuns).Error
	if err != nil {
		return nil, 0, err
	}

	return pipelineRuns, totalRows, nil
}

func (r *repository) GetPaginatedComponentRunsByPipelineRunIDWithPermissions(ctx context.Context, pipelineRunID string, page, pageSize int, filter filtering.Filter, order ordering.OrderBy) ([]datamodel.ComponentRun, int64, error) {
	var componentRuns []datamodel.ComponentRun
	var totalRows int64

	whereConditions := []string{"pipeline_trigger_uid = ?"}
	whereArgs := []any{pipelineRunID}

	var expr *clause.Expr
	var err error
	if expr, err = r.TranspileFilter(filter); err != nil {
		return nil, 0, err
	}
	if expr != nil {
		whereConditions = append(whereConditions, "(?)")
		whereArgs = append(whereArgs, expr)
	}

	var where string
	if len(whereConditions) > 0 {
		where = strings.Join(whereConditions, " and ")
	}

	// Count total rows
	err = r.db.Model(&datamodel.ComponentRun{}).
		Where(where, whereArgs...).
		Count(&totalRows).Error
	if err != nil {
		return nil, 0, err
	}

	queryBuilder := r.db.Where(where, whereArgs...)

	if len(order.Fields) == 0 {
		order.Fields = append(order.Fields, ordering.Field{
			Path: "started_time",
			Desc: true,
		})
	}

	for _, field := range order.Fields {
		orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(field.Desc)
		queryBuilder.Order(orderString)
	}

	// Retrieve paginated results
	err = queryBuilder.
		Offset(page * pageSize).Limit(pageSize).
		Find(&componentRuns).Error
	if err != nil {
		return nil, 0, err
	}

	return componentRuns, totalRows, nil
}

type GetPipelineRunsByRequesterParams struct {
	RequesterUID   string
	StartTimeBegin time.Time
	StartTimeEnd   time.Time
	Page           int
	PageSize       int
	Filter         filtering.Filter
	Order          ordering.OrderBy
}

func (r *repository) GetPaginatedPipelineRunsByRequester(ctx context.Context, params GetPipelineRunsByRequesterParams) ([]datamodel.PipelineRun, int64, error) {
	var pipelineRuns []datamodel.PipelineRun
	var totalRows int64

	whereConditions := []string{"namespace = ? and started_time >= ? and started_time <= ?"}
	whereArgs := []any{params.RequesterUID, params.StartTimeBegin, params.StartTimeEnd}

	var expr *clause.Expr
	var err error
	if expr, err = r.TranspileFilter(params.Filter); err != nil {
		return nil, 0, err
	}
	if expr != nil {
		whereConditions = append(whereConditions, "(?)")
		whereArgs = append(whereArgs, expr)
	}

	var where string
	if len(whereConditions) > 0 {
		where = strings.Join(whereConditions, " and ")
	}

	err = r.db.Model(&datamodel.PipelineRun{}).
		Where(where, whereArgs...).
		Count(&totalRows).Error
	if err != nil {
		return nil, 0, err
	}

	queryBuilder := r.db.Preload(clause.Associations).Where(where, whereArgs...)

	order := params.Order
	if len(order.Fields) == 0 {
		order.Fields = append(order.Fields, ordering.Field{
			Path: "started_time",
			Desc: true,
		})
	}

	for _, field := range order.Fields {
		orderString := strcase.ToSnake(field.Path) + transformBoolToDescString(field.Desc)
		queryBuilder.Order(orderString)
	}

	// Retrieve paginated results with permissions
	err = queryBuilder.
		Offset(params.Page * params.PageSize).Limit(params.PageSize).
		Find(&pipelineRuns).Error
	if err != nil {
		return nil, 0, err
	}

	return pipelineRuns, totalRows, nil
}

func (r *repository) CreateNamespaceConnection(ctx context.Context, conn *datamodel.Connection) (*datamodel.Connection, error) {
	db := r.db.WithContext(ctx)

	err := db.Create(conn).Error
	if err != nil {
		return nil, r.toDomainErr(err)
	}

	// Extra query is used to return the associated integration.
	return r.GetNamespaceConnectionByID(ctx, conn.NamespaceUID, conn.ID)
}

func (r *repository) UpdateNamespaceConnectionByUID(ctx context.Context, uid uuid.UUID, conn *datamodel.Connection) (*datamodel.Connection, error) {
	db := r.db.WithContext(ctx)

	result := db.Where("uid = ?", uid).
		Omit("UID", "NamespaceUID", "IntegrationUID"). // Immutable fields
		Clauses(clause.Returning{}).
		Updates(conn)
	if result.Error != nil {
		return nil, r.toDomainErr(result.Error)
	}

	if result.RowsAffected == 0 {
		return nil, errdomain.ErrNotFound
	}

	// Extra query is used to return the associated integration.
	return r.GetNamespaceConnectionByID(ctx, conn.NamespaceUID, conn.ID)
}

func (r *repository) DeleteNamespaceConnectionByID(ctx context.Context, nsUID uuid.UUID, id string) error {
	db := r.db.WithContext(ctx)

	result := db.Where("(id = ? AND namespace_uid = ?)", id, nsUID).Delete(&datamodel.Connection{})
	if result.Error != nil {
		return r.toDomainErr(result.Error)
	}

	if result.RowsAffected == 0 {
		return errdomain.ErrNotFound
	}

	return nil
}
func (r *repository) GetNamespaceConnectionByID(ctx context.Context, nsUID uuid.UUID, id string) (*datamodel.Connection, error) {
	db := r.db.WithContext(ctx)

	q := db.Preload("Integration").Where("namespace_uid = ? AND id = ?", nsUID, id)
	conn := new(datamodel.Connection)
	if err := q.First(&conn).Error; err != nil {
		return nil, r.toDomainErr(err)
	}

	return conn, nil
}

// ListNamespaceConnectionsParams allows clients to request a page of
// connections.
type ListNamespaceConnectionsParams struct {
	NamespaceUID uuid.UUID
	PageToken    string
	Limit        int
	Filter       filtering.Filter
}

// ConnectionList contains a page of connections.
type ConnectionList struct {
	Connections   []*datamodel.Connection
	NextPageToken string
	TotalSize     int32
}

type connectionCursor struct {
	Score          int       `json:"score"`
	IntegrationUID uuid.UUID `json:"integration_uid"`
	CreateTime     time.Time `json:"create_time"`
}

func (r *repository) ListNamespaceConnections(ctx context.Context, p ListNamespaceConnectionsParams) (ConnectionList, error) {
	var resp ConnectionList

	db := r.db.WithContext(ctx)
	where := ""
	whereArgs := []any{}

	// Get item count.
	expr, err := r.TranspileFilter(p.Filter)
	if err != nil {
		return resp, fmt.Errorf("transpiling filter: %w", err)
	}

	if expr != nil {
		where = "(?)"
		whereArgs = []any{expr}
	}

	queryBuilder := db.Model(&datamodel.Connection{}).
		Where(where, whereArgs...).
		Where("namespace_uid = ?", p.NamespaceUID).
		Joins("LEFT JOIN component_definition_index ON component_definition_index.uid = connection.integration_uid")

	var count int64
	queryBuilder.Count(&count)
	resp.TotalSize = int32(count)

	// Get definitions matching criteria.
	if p.PageToken != "" {
		cursor, err := decodeCursor[connectionCursor](p.PageToken)
		if err != nil {
			return resp, err
		}
		queryBuilder = queryBuilder.Where(
			"(feature_score,integration_uid,create_time) < (?, ?, ?)",
			cursor.Score,
			cursor.IntegrationUID,
			cursor.CreateTime,
		)
	}

	// From here we'll apply different search criteria.
	queryBuilder = queryBuilder.
		Select("connection.*, component_definition_index.feature_score").
		Session(&gorm.Session{})

	resp.Connections = make([]*datamodel.Connection, 0, p.Limit)
	err = queryBuilder.Preload("Integration").
		Limit(p.Limit).
		Order("feature_score DESC, integration_uid DESC, create_time DESC").
		Find(&resp.Connections).Error
	if err != nil {
		return resp, fmt.Errorf("querying database rows: %w", err)
	}

	if len(resp.Connections) == 0 {
		return resp, nil
	}

	lastInPage := resp.Connections[len(resp.Connections)-1]
	lastInDB := new(datamodel.Connection)
	orderInverse := "feature_score ASC, integration_uid ASC, create_time ASC"
	if result := queryBuilder.Order(orderInverse).Limit(1).Find(lastInDB); result.Error != nil {
		return resp, fmt.Errorf("finding last item: %w", err)
	}

	if lastInDB.UID.String() == lastInPage.UID.String() {
		return resp, nil
	}

	nextCursor := connectionCursor{
		Score:          lastInPage.Integration.FeatureScore,
		IntegrationUID: lastInPage.IntegrationUID,
		CreateTime:     lastInPage.CreateTime,
	}
	resp.NextPageToken, err = encodeCursor[connectionCursor](nextCursor)
	if err != nil {
		return resp, err
	}

	return resp, nil
}

// ListPipelineIDsByConnectionIDParams allows clients to request a page of
// pipeline IDs that reference a connection.
type ListPipelineIDsByConnectionIDParams struct {
	Owner        resource.Namespace
	ConnectionID string
	PageToken    string
	Limit        int
	Filter       filtering.Filter
}

// PipelinesByConnectionList contains a page of pipeline IDs that reference a
// given connection.
type PipelinesByConnectionList struct {
	PipelineIDs   []string
	NextPageToken string
	TotalSize     int32
}

// pipelineByConnection can be used both for fetching the pipeline ID and as a
// cursor for pagination.
type pipelineByConnection struct {
	CreateTime time.Time `json:"create_time"`
	UID        uuid.UUID `json:"uid"`
	ID         string    `json:"-"`
}

func (r *repository) ListPipelineIDsByConnectionID(
	ctx context.Context,
	p ListPipelineIDsByConnectionIDParams,
) (page PipelinesByConnectionList, err error) {
	db := r.db.WithContext(ctx)
	where := ""
	whereArgs := []any{}

	// Get item count.
	expr, err := r.TranspileFilter(p.Filter)
	if err != nil {
		return page, fmt.Errorf("transpiling filter: %w", err)
	}

	if expr != nil {
		where = "(?)"
		whereArgs = []any{expr}
	}

	// This is an inefficient search and won't scale.
	// TODO jvallesm INS-6191: store a JSON copy of the recipe and use it to
	// search by its properties.
	reference := fmt.Sprintf("%%${%s.%s}%%", constant.SegConnection, p.ConnectionID)
	queryBuilder := db.Table("pipeline").
		Where(where, whereArgs...).
		Where("recipe_yaml LIKE ?", reference).
		Where("delete_time IS NULL").
		Where("owner = ?", p.Owner.Permalink())

	var count int64
	queryBuilder.Count(&count)
	page.TotalSize = int32(count)

	// Get definitions matching criteria.
	if p.PageToken != "" {
		cursor, err := decodeCursor[pipelineByConnection](p.PageToken)
		if err != nil {
			return page, err
		}
		queryBuilder = queryBuilder.Where(
			"(create_time,uid) < (?, ?)",
			cursor.CreateTime,
			cursor.UID,
		)
	}

	// From here we'll apply different search criteria.
	queryBuilder = queryBuilder.Session(&gorm.Session{})

	rows, err := queryBuilder.Limit(p.Limit).Order("create_time DESC, uid DESC").Rows()
	if err != nil {
		return page, fmt.Errorf("querying database rows: %w", err)
	}
	defer rows.Close()

	page.PipelineIDs = make([]string, 0, p.Limit)
	lastItem := new(pipelineByConnection)
	for rows.Next() {
		if err = db.ScanRows(rows, lastItem); err != nil {
			return page, fmt.Errorf("scanning pipeline ID: %w", err)
		}

		page.PipelineIDs = append(page.PipelineIDs, lastItem.ID)
	}

	lastInDB := new(pipelineByConnection)
	orderInverse := "create_time ASC, uid ASC"
	if result := queryBuilder.Order(orderInverse).Limit(1).Find(lastInDB); result.Error != nil {
		return page, fmt.Errorf("finding last item: %w", err)
	}

	if lastInDB.UID.String() == lastItem.UID.String() {
		return page, nil
	}

	page.NextPageToken, err = encodeCursor[pipelineByConnection](*lastItem)
	if err != nil {
		return page, err
	}

	return page, nil
}
