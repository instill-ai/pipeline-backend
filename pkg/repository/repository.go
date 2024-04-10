package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
	"github.com/redis/go-redis/v9"
	"go.einride.tech/aip/filtering"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/plugin/dbresolver"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/internal/resource"
	"github.com/instill-ai/pipeline-backend/pkg/constant"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/x/paginate"

	pipelinePB "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

// TODO: in the repository, we'd better use uid as our function params

// DefaultPageSize is the default pagination page size when page size is not assigned
const DefaultPageSize = 10

// MaxPageSize is the maximum pagination page size if the assigned value is over this number
const MaxPageSize = 100

const VisibilityPublic = datamodel.ConnectorVisibility(pipelinePB.Connector_VISIBILITY_PUBLIC)

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

	ListConnectors(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByUID(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error)

	CreateNamespaceConnector(ctx context.Context, ownerPermalink string, connector *datamodel.Connector) error
	ListNamespaceConnectors(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool) ([]*datamodel.Connector, int64, string, error)
	GetNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool) (*datamodel.Connector, error)
	UpdateNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string, connector *datamodel.Connector) error
	DeleteNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string) error
	UpdateNamespaceConnectorIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error
	UpdateNamespaceConnectorStateByID(ctx context.Context, ownerPermalink string, id string, state datamodel.ConnectorState) error

	ListConnectorsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) ([]*datamodel.Connector, int64, string, error)
	GetConnectorByUIDAdmin(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error)

	ListComponentDefinitionUIDs(context.Context, ListComponentDefinitionsParams) (uids []*datamodel.ComponentDefinition, totalSize int64, err error)
	GetComponentDefinitionByUID(context.Context, uuid.UUID) (*datamodel.ComponentDefinition, error)
	UpsertComponentDefinition(context.Context, *pipelinePB.ComponentDefinition) error
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
	connectorUIDs := []uuid.UUID{}
	pipelineConnectorUIDs := map[uuid.UUID][]uuid.UUID{}
	for rows.Next() {
		var item datamodel.Pipeline
		if err = db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", err
		}
		createTime = item.CreateTime
		pipelines = append(pipelines, &item)
		pipelineUIDs = append(pipelineUIDs, item.UID)
		pipelineConnectorUIDs[item.UID] = []uuid.UUID{}
		if !isBasicView {
			for _, comp := range item.Recipe.Components {
				if comp.IsConnectorComponent() {
					if len(strings.Split(comp.ConnectorComponent.ConnectorName, "/")) == 2 {
						connectorUID := uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.ConnectorName, "/")[1])
						connectorUIDs = append(connectorUIDs, connectorUID)
						pipelineConnectorUIDs[item.UID] = append(pipelineConnectorUIDs[item.UID], connectorUID)
					}
				}
				if comp.IsIteratorComponent() {
					for _, nestedComp := range comp.IteratorComponent.Components {
						if nestedComp.IsConnectorComponent() {
							if len(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")) == 2 {
								connectorUID := uuid.FromStringOrNil(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")[1])
								connectorUIDs = append(connectorUIDs, connectorUID)
								pipelineConnectorUIDs[item.UID] = append(pipelineConnectorUIDs[item.UID], connectorUID)
							}
						}
					}
				}
			}
		}
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
			if !isBasicView {
				for _, comp := range pipelineRelease.Recipe.Components {

					if comp.IsConnectorComponent() {
						if len(strings.Split(comp.ConnectorComponent.ConnectorName, "/")) == 2 {
							connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.ConnectorName, "/")[1]))
						}

					}
					if comp.IsIteratorComponent() {
						for _, nestedComp := range comp.IteratorComponent.Components {
							if nestedComp.IsConnectorComponent() {
								if len(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")) == 2 {
									connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")[1]))
								}

							}
						}
					}
				}
			}
		}
		for idx := range pipelines {
			if releases, ok := releasesMap[pipelines[idx].UID]; ok {
				pipelines[idx].Releases = releases

			}
		}
	}

	if !isBasicView {
		connectorDB := r.db
		connectorsMap := map[uuid.UUID]*datamodel.Connector{}
		connectorRows, err := connectorDB.Model(&datamodel.Connector{}).Where("uid in ?", connectorUIDs).Order("create_time DESC, uid DESC").Rows()
		if err != nil {
			return nil, 0, "", err
		}

		defer connectorRows.Close()
		for connectorRows.Next() {
			var item datamodel.Connector
			if err = connectorDB.ScanRows(connectorRows, &item); err != nil {
				return nil, 0, "", err
			}
			connectorsMap[item.UID] = &item
		}

		for idx := range pipelines {
			pipelineConnectorsMap := map[uuid.UUID]*datamodel.Connector{}
			for _, connectorUID := range pipelineConnectorUIDs[pipelines[idx].UID] {
				pipelineConnectorsMap[connectorUID] = connectorsMap[connectorUID]
			}
			pipelines[idx].Connectors = pipelineConnectorsMap

			if embedReleases {
				for releaseIdx := range pipelines[idx].Releases {
					for _, comp := range pipelines[idx].Releases[releaseIdx].Recipe.Components {

						if comp.IsConnectorComponent() {
							if len(strings.Split(comp.ConnectorComponent.ConnectorName, "/")) == 2 {
								connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.ConnectorName, "/")[1]))
							}

						}
						if comp.IsIteratorComponent() {
							for _, nestedComp := range comp.IteratorComponent.Components {
								if nestedComp.IsConnectorComponent() {
									if len(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")) == 2 {
										connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")[1]))
									}

								}
							}
						}
					}

					connectorDB := r.db
					connectorsMap := map[uuid.UUID]*datamodel.Connector{}
					connectorRows, err := connectorDB.Model(&datamodel.Connector{}).Where("uid in ?", connectorUIDs).Order("create_time DESC, uid DESC").Rows()
					if err != nil {
						return nil, 0, "", err
					}
					defer connectorRows.Close()
					for connectorRows.Next() {
						var item datamodel.Connector
						if err = connectorDB.ScanRows(connectorRows, &item); err != nil {
							return nil, 0, "", err
						}
						connectorsMap[item.UID] = &item
					}
					pipelines[idx].Releases[releaseIdx].Connectors = connectorsMap
				}
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

	connectorUIDs := []uuid.UUID{}

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
			if !isBasicView {
				for _, comp := range pipelineRelease.Recipe.Components {

					if comp.IsConnectorComponent() {
						if len(strings.Split(comp.ConnectorComponent.ConnectorName, "/")) == 2 {
							connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.ConnectorName, "/")[1]))
						}

					}
					if comp.IsIteratorComponent() {
						for _, nestedComp := range comp.IteratorComponent.Components {
							if nestedComp.IsConnectorComponent() {
								if len(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")) == 2 {
									connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")[1]))
								}

							}
						}
					}
				}
			}
		}
	}

	if !isBasicView {
		for _, comp := range pipeline.Recipe.Components {

			if comp.IsConnectorComponent() {
				if len(strings.Split(comp.ConnectorComponent.ConnectorName, "/")) == 2 {
					connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.ConnectorName, "/")[1]))
				}

			}
			if comp.IsIteratorComponent() {
				for _, nestedComp := range comp.IteratorComponent.Components {
					if nestedComp.IsConnectorComponent() {
						if len(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")) == 2 {
							connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")[1]))
						}

					}
				}
			}
		}

		connectorDB := r.db
		connectorsMap := map[uuid.UUID]*datamodel.Connector{}
		connectorRows, err := connectorDB.Model(&datamodel.Connector{}).Where("uid in ?", connectorUIDs).Order("create_time DESC, uid DESC").Rows()
		if err != nil {
			return nil, err
		}
		defer connectorRows.Close()
		for connectorRows.Next() {
			var item datamodel.Connector
			if err = connectorDB.ScanRows(connectorRows, &item); err != nil {
				return nil, err
			}
			connectorsMap[item.UID] = &item
		}
		pipeline.Connectors = connectorsMap

		if embedReleases {
			for releaseIdx := range pipeline.Releases {
				for _, comp := range pipeline.Releases[releaseIdx].Recipe.Components {

					if comp.IsConnectorComponent() {
						if len(strings.Split(comp.ConnectorComponent.ConnectorName, "/")) == 2 {
							connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.ConnectorName, "/")[1]))
						}

					}
					if comp.IsIteratorComponent() {
						for _, nestedComp := range comp.IteratorComponent.Components {
							if nestedComp.IsConnectorComponent() {
								if len(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")) == 2 {
									connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")[1]))
								}

							}
						}
					}
				}

				connectorDB := r.db
				connectorsMap := map[uuid.UUID]*datamodel.Connector{}
				connectorRows, err := connectorDB.Model(&datamodel.Connector{}).Where("uid in ?", connectorUIDs).Order("create_time DESC, uid DESC").Rows()
				if err != nil {
					return nil, err
				}
				defer connectorRows.Close()
				for connectorRows.Next() {
					var item datamodel.Connector
					if err = connectorDB.ScanRows(connectorRows, &item); err != nil {
						return nil, err
					}
					connectorsMap[item.UID] = &item
				}
				pipeline.Releases[releaseIdx].Connectors = connectorsMap
			}
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

	connectorUIDs := []uuid.UUID{}
	releaseConnectorUIDs := map[uuid.UUID][]uuid.UUID{}

	for rows.Next() {
		var item *datamodel.PipelineRelease
		if err = db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", err
		}
		createTime = item.CreateTime
		pipelineReleases = append(pipelineReleases, item)
		releaseConnectorUIDs[item.UID] = []uuid.UUID{}
		if !isBasicView {
			for _, comp := range item.Recipe.Components {
				if comp.IsConnectorComponent() {
					if len(strings.Split(comp.ConnectorComponent.ConnectorName, "/")) == 2 {
						connectorUID := uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.ConnectorName, "/")[1])
						connectorUIDs = append(connectorUIDs, connectorUID)
						releaseConnectorUIDs[item.UID] = append(releaseConnectorUIDs[item.UID], connectorUID)
					}

				}
				if comp.IsIteratorComponent() {
					for _, nestedComp := range comp.IteratorComponent.Components {
						if nestedComp.IsConnectorComponent() {
							if len(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")) == 2 {
								connectorUID := uuid.FromStringOrNil(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")[1])
								connectorUIDs = append(connectorUIDs, connectorUID)
								releaseConnectorUIDs[item.UID] = append(releaseConnectorUIDs[item.UID], connectorUID)
							}
						}
					}
				}
			}
		}
	}

	if !isBasicView {
		connectorDB := r.db
		connectorsMap := map[uuid.UUID]*datamodel.Connector{}
		connectorRows, err := connectorDB.Model(&datamodel.Connector{}).Where("uid in ?", connectorUIDs).Order("create_time DESC, uid DESC").Rows()
		if err != nil {
			return nil, 0, "", err
		}
		defer connectorRows.Close()
		for connectorRows.Next() {
			var item datamodel.Connector
			if err = connectorDB.ScanRows(connectorRows, &item); err != nil {
				return nil, 0, "", err
			}
			connectorsMap[item.UID] = &item
		}
		for idx := range pipelineReleases {
			releaseConnectorsMap := map[uuid.UUID]*datamodel.Connector{}
			for _, connectorUID := range releaseConnectorUIDs[pipelineReleases[idx].UID] {
				releaseConnectorsMap[connectorUID] = connectorsMap[connectorUID]
			}
			pipelineReleases[idx].Connectors = releaseConnectorsMap
		}
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
	if !isBasicView {
		connectorUIDs := []uuid.UUID{}
		for _, comp := range pipelineRelease.Recipe.Components {

			if comp.IsConnectorComponent() {
				if len(strings.Split(comp.ConnectorComponent.ConnectorName, "/")) == 2 {
					connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(comp.ConnectorComponent.ConnectorName, "/")[1]))
				}

			}

			if comp.IsIteratorComponent() {
				for _, nestedComp := range comp.IteratorComponent.Components {
					if nestedComp.IsConnectorComponent() {
						if len(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")) == 2 {
							connectorUIDs = append(connectorUIDs, uuid.FromStringOrNil(strings.Split(nestedComp.ConnectorComponent.ConnectorName, "/")[1]))
						}
					}
				}
			}
		}

		connectorDB := r.db
		connectorsMap := map[uuid.UUID]*datamodel.Connector{}
		connectorRows, err := connectorDB.Model(&datamodel.Connector{}).Where("uid in ?", connectorUIDs).Order("create_time DESC, uid DESC").Rows()
		if err != nil {
			return nil, err
		}
		defer connectorRows.Close()
		for connectorRows.Next() {
			var item datamodel.Connector
			if err = connectorDB.ScanRows(connectorRows, &item); err != nil {
				return nil, err
			}
			connectorsMap[item.UID] = &item
		}
		pipelineRelease.Connectors = connectorsMap
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

func (r *repository) listConnectors(ctx context.Context, where string, whereArgs []interface{}, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

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
		db.Model(&datamodel.Connector{}).Where(where, whereArgs...).Where("uid in ?", uidAllowList).Count(&totalSize)
	} else {
		db.Model(&datamodel.Connector{}).Where(where, whereArgs...).Count(&totalSize)
	}

	queryBuilder := db.Model(&datamodel.Connector{}).Order("create_time DESC, uid DESC").Where(where, whereArgs...)
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
		queryBuilder.Omit("configuration")
	}

	var createTime time.Time // only using one for all loops, we only need the latest one in the end
	rows, err := queryBuilder.Rows()
	if err != nil {
		return nil, 0, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var item datamodel.Connector
		if err = db.ScanRows(rows, &item); err != nil {
			return nil, 0, "", err
		}
		createTime = item.CreateTime
		connectors = append(connectors, &item)
	}

	if len(connectors) > 0 {
		lastUID := (connectors)[len(connectors)-1].UID
		lastItem := &datamodel.Connector{}
		if uidAllowList != nil {
			if result := db.Model(&datamodel.Connector{}).
				Where(where, whereArgs...).
				Where("uid in ?", uidAllowList).
				Order("create_time ASC, uid ASC").Limit(1).Find(lastItem); result.Error != nil {
				return nil, 0, "", err
			}
		} else {
			if result := db.Model(&datamodel.Connector{}).
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

	return connectors, totalSize, nextPageToken, nil
}

func (r *repository) ListConnectorsAdmin(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, showDeleted bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {
	return r.listConnectors(ctx, "", []interface{}{}, pageSize, pageToken, isBasicView, filter, nil, showDeleted)
}

func (r *repository) ListConnectors(ctx context.Context, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

	userPermalink := fmt.Sprintf("users/%s", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
	return r.listConnectors(ctx,
		"(owner = ?)",
		[]interface{}{userPermalink},
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted)

}

func (r *repository) ListNamespaceConnectors(ctx context.Context, ownerPermalink string, pageSize int64, pageToken string, isBasicView bool, filter filtering.Filter, uidAllowList []uuid.UUID, showDeleted bool) (connectors []*datamodel.Connector, totalSize int64, nextPageToken string, err error) {

	return r.listConnectors(ctx,
		"(owner = ? )",
		[]interface{}{ownerPermalink},
		pageSize, pageToken, isBasicView, filter, uidAllowList, showDeleted)

}

func (r *repository) CreateNamespaceConnector(ctx context.Context, ownerPermalink string, connector *datamodel.Connector) error {

	r.pinUser(ctx, "connector")
	db := r.checkPinnedUser(ctx, r.db, "connector")

	if result := db.Model(&datamodel.Connector{}).Create(connector); result.Error != nil {
		return result.Error
	}
	return nil
}

func (r *repository) getNamespaceConnector(ctx context.Context, where string, whereArgs []interface{}, isBasicView bool) (*datamodel.Connector, error) {

	db := r.checkPinnedUser(ctx, r.db, "connector")

	var connector datamodel.Connector

	queryBuilder := db.Model(&datamodel.Connector{}).Where(where, whereArgs...)

	if isBasicView {
		queryBuilder.Omit("configuration")
	}

	if result := queryBuilder.First(&connector); result.Error != nil {
		return nil, result.Error
	}
	return &connector, nil
}

func (r *repository) GetNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string, isBasicView bool) (*datamodel.Connector, error) {

	return r.getNamespaceConnector(ctx,
		"(id = ? AND (owner = ?))",
		[]interface{}{id, ownerPermalink},
		isBasicView)
}

func (r *repository) GetConnectorByUID(ctx context.Context, uid uuid.UUID, isBasicView bool) (*datamodel.Connector, error) {

	userPermalink := fmt.Sprintf("users/%s", resource.GetRequestSingleHeader(ctx, constant.HeaderUserUIDKey))
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

	r.pinUser(ctx, "connector")
	db := r.checkPinnedUser(ctx, r.db, "connector")

	if result := db.Model(&datamodel.Connector{}).
		Where("(id = ? AND owner = ? )", id, ownerPermalink).
		Updates(connector); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) DeleteNamespaceConnectorByID(ctx context.Context, ownerPermalink string, id string) error {

	r.pinUser(ctx, "connector")
	db := r.checkPinnedUser(ctx, r.db, "connector")

	result := db.Model(&datamodel.Connector{}).
		Where("(id = ? AND owner = ? )", id, ownerPermalink).
		Delete(&datamodel.Connector{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrNoDataDeleted
	}

	return nil
}

func (r *repository) UpdateNamespaceConnectorIDByID(ctx context.Context, ownerPermalink string, id string, newID string) error {

	r.pinUser(ctx, "connector")
	db := r.checkPinnedUser(ctx, r.db, "connector")

	if result := db.Model(&datamodel.Connector{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Update("id", newID); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
}

func (r *repository) UpdateNamespaceConnectorStateByID(ctx context.Context, ownerPermalink string, id string, state datamodel.ConnectorState) error {

	r.pinUser(ctx, "connector")
	db := r.checkPinnedUser(ctx, r.db, "connector")

	if result := db.Model(&datamodel.Connector{}).
		Where("(id = ? AND owner = ?)", id, ownerPermalink).
		Update("state", state); result.Error != nil {
		return result.Error
	} else if result.RowsAffected == 0 {
		return ErrNoDataUpdated
	}
	return nil
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
func (r *repository) UpsertComponentDefinition(_ context.Context, cd *pipelinePB.ComponentDefinition) error {
	record := datamodel.ComponentDefinitionFromProto(cd)
	result := r.db.Clauses(clause.OnConflict{UpdateAll: true}).Create(record)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
