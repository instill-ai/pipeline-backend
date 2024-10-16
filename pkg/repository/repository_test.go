//go:build dbtest
// +build dbtest

package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/gofrs/uuid"
	"github.com/google/go-cmp/cmp/cmpopts"
	"go.einride.tech/aip/filtering"
	"go.einride.tech/aip/ordering"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
	"gopkg.in/guregu/null.v4"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	componentstore "github.com/instill-ai/pipeline-backend/pkg/component/store"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	errdomain "github.com/instill-ai/pipeline-backend/pkg/errors"
	runpb "github.com/instill-ai/protogen-go/common/run/v1alpha"
	pipelinepb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

var db *gorm.DB

func TestMain(m *testing.M) {
	if err := config.Init("../../config/config.yaml"); err != nil {
		panic(err)
	}

	db = database.GetSharedConnection()
	defer database.Close(db)

	os.Exit(m.Run())
}

func TestRepository_ComponentDefinitionUIDs(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	tx := db.Begin()
	c.Cleanup(func() { tx.Rollback() })

	repo := NewRepository(tx, nil)
	uid := uuid.Must(uuid.NewV4())
	id := "json"
	cd := &pipelinepb.ComponentDefinition{
		Type:   pipelinepb.ComponentType_COMPONENT_TYPE_OPERATOR,
		Uid:    uid.String(),
		Id:     id,
		Public: true,
	}

	err := repo.UpsertComponentDefinition(ctx, cd)
	c.Check(err, qt.IsNil)

	dbDef, err := repo.GetDefinitionByUID(ctx, uid)
	c.Check(err, qt.IsNil)
	c.Check(dbDef.ID, qt.Equals, id)

	p := ListComponentDefinitionsParams{Limit: 10}
	dbDefs, size, err := repo.ListComponentDefinitionUIDs(ctx, p)
	c.Check(err, qt.IsNil)
	c.Check(size, qt.Equals, int64(1))
	c.Check(dbDefs, qt.HasLen, 1)
	c.Check(dbDefs[0].UID.String(), qt.Equals, uid.String())
}

func TestRepository_Cursor(t *testing.T) {
	c := qt.New(t)

	c.Run("ok - integration cursor", func(c *qt.C) {
		cursor := integrationCursor{
			Score: 30,
			UID:   uuid.Must(uuid.NewV4()),
		}

		token, err := encodeCursor[integrationCursor](cursor)
		c.Assert(err, qt.IsNil)

		got, err := decodeCursor[integrationCursor](token)
		c.Assert(err, qt.IsNil)
		c.Check(got, qt.ContentEquals, cursor)
	})

	c.Run("ok - connection cursor", func(c *qt.C) {
		cursor := connectionCursor{
			Score:          30,
			IntegrationUID: uuid.Must(uuid.NewV4()),
			CreateTime:     time.Now(),
		}

		token, err := encodeCursor[connectionCursor](cursor)
		c.Assert(err, qt.IsNil)

		got, err := decodeCursor[connectionCursor](token)
		c.Assert(err, qt.IsNil)
		c.Check(got, qt.ContentEquals, cursor)
	})
}

func TestRepository_Integrations(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	tx := db.Begin()
	c.Cleanup(func() { tx.Rollback() })

	repo := NewRepository(tx, nil)

	// IDs define the score
	ids := []string{
		"instill-model", // 50,
		"pinecone",      // 30,
		"numbers",       // 30,
		"foo",           // 0,
		"bar",           // 0,
	}
	uidStrings := []string{
		"1deff56a-0510-43fe-90d4-1c8d0cd44db2",
		"349f1c92-6f73-4a80-889d-ffea793fa057",
		"5fb69d62-b92c-4916-9460-604c45863736",
		"2c5793d6-8b6d-451e-9e0e-f2a0d248a465",
		"e6d1f275-6b5e-408f-9e23-baa4faca004b",
	}

	// Score precedes UID when sorting.
	wantUIDsOrdered := []string{
		uidStrings[0], uidStrings[2], uidStrings[1], uidStrings[4], uidStrings[3],
	}

	// Public but without integration.
	err := repo.UpsertComponentDefinition(ctx, &pipelinepb.ComponentDefinition{
		Type:   pipelinepb.ComponentType_COMPONENT_TYPE_OPERATOR,
		Uid:    uuid.Must(uuid.NewV4()).String(),
		Id:     "json",
		Public: true,
	})
	c.Assert(err, qt.IsNil)

	// With integration but hidden.
	compSpec, err := structpb.NewStruct(map[string]any{
		"properties": map[string]any{
			"setup": map[string]any{},
		},
	})
	c.Assert(err, qt.IsNil)
	spec := &pipelinepb.ComponentDefinition_Spec{ComponentSpecification: compSpec}

	err = repo.UpsertComponentDefinition(ctx, &pipelinepb.ComponentDefinition{
		Type: pipelinepb.ComponentType_COMPONENT_TYPE_AI,
		Uid:  uuid.Must(uuid.NewV4()).String(),
		Id:   "weaviate",
		Spec: spec,
	})
	c.Assert(err, qt.IsNil)

	for i := range uidStrings {
		err = repo.UpsertComponentDefinition(ctx, &pipelinepb.ComponentDefinition{
			Type:   pipelinepb.ComponentType_COMPONENT_TYPE_AI,
			Uid:    uidStrings[i],
			Id:     ids[i],
			Spec:   spec,
			Public: true,
		})
		c.Assert(err, qt.IsNil)
	}

	// Page one
	p := ListIntegrationsParams{
		Limit: 2,
	}
	page, err := repo.ListIntegrations(ctx, p)
	c.Check(err, qt.IsNil)
	c.Check(page.TotalSize, qt.Equals, int32(5))
	c.Check(page.ComponentDefinitions, qt.HasLen, 2)
	c.Check(page.NextPageToken, qt.Not(qt.HasLen), 0)
	for i, want := range wantUIDsOrdered[:2] {
		c.Check(page.ComponentDefinitions[i].UID.String(), qt.Equals, want)
	}

	// Page two
	p.PageToken = page.NextPageToken
	page, err = repo.ListIntegrations(ctx, p)
	c.Check(err, qt.IsNil)
	c.Check(page.TotalSize, qt.Equals, int32(5))
	c.Check(page.ComponentDefinitions, qt.HasLen, 2)
	c.Check(page.NextPageToken, qt.Not(qt.HasLen), 0)
	for i, want := range wantUIDsOrdered[2:4] {
		c.Check(page.ComponentDefinitions[i].UID.String(), qt.Equals, want)
	}

	// Page three
	p.PageToken = page.NextPageToken
	page, err = repo.ListIntegrations(ctx, p)
	c.Check(err, qt.IsNil)
	c.Check(page.TotalSize, qt.Equals, int32(5))
	c.Check(page.ComponentDefinitions, qt.HasLen, 1)
	c.Check(page.NextPageToken, qt.HasLen, 0)
	for i, want := range wantUIDsOrdered[4:] {
		c.Check(page.ComponentDefinitions[i].UID.String(), qt.Equals, want)
	}
}

func TestRepository_Connection(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()
	logger := zap.NewNop()

	// Need to load and store component definitions as they're referenced by
	// connections.
	cds := componentstore.Init(logger, nil, nil)
	openAI, err := cds.GetDefinitionByID("openai", nil, nil)
	c.Assert(err, qt.IsNil)

	pinecone, err := cds.GetDefinitionByID("pinecone", nil, nil)
	c.Assert(err, qt.IsNil)

	email, err := cds.GetDefinitionByID("email", nil, nil)
	c.Assert(err, qt.IsNil)

	newRepo := func(c *qt.C) Repository {
		tx := db.Begin()
		c.Cleanup(func() { tx.Rollback() })

		repo := NewRepository(tx, nil)

		c.Assert(repo.UpsertComponentDefinition(ctx, openAI), qt.IsNil)
		c.Assert(repo.UpsertComponentDefinition(ctx, pinecone), qt.IsNil)
		c.Assert(repo.UpsertComponentDefinition(ctx, email), qt.IsNil)

		return repo
	}

	nsUID := uuid.Must(uuid.NewV4())
	method := datamodel.ConnectionMethod(pipelinepb.Connection_METHOD_DICTIONARY)
	newConn := func() *datamodel.Connection {
		return &datamodel.Connection{
			NamespaceUID: nsUID,
			Method:       method,
			Setup:        datatypes.JSON(`{}`),
		}

	}

	c.Run("nok - connection not found", func(c *qt.C) {
		_, err := newRepo(c).GetNamespaceConnectionByID(ctx, uuid.Must(uuid.NewV4()), "foo")
		c.Check(errors.Is(err, errdomain.ErrNotFound), qt.IsTrue)
	})

	c.Run("nok - missing integration reference", func(c *qt.C) {
		conn := newConn()
		conn.ID = "invalid-integration-uid"
		conn.IntegrationUID = uuid.Must(uuid.NewV4())

		_, err := newRepo(c).CreateNamespaceConnection(ctx, conn)
		c.Check(err, qt.ErrorMatches, ".*foreign key.*integration_uid.*")
	})

	c.Run("nok - double creation", func(c *qt.C) {
		repo := newRepo(c)
		conn := newConn()
		conn.ID = "foo"
		conn.IntegrationUID = uuid.FromStringOrNil(email.GetUid())

		_, err := repo.CreateNamespaceConnection(ctx, conn)
		c.Check(err, qt.IsNil)
		_, err = repo.CreateNamespaceConnection(ctx, conn)
		c.Check(errors.Is(err, errdomain.ErrAlreadyExists), qt.IsTrue)
	})

	c.Run("nok - update not found", func(c *qt.C) {
		repo := newRepo(c)
		conn := newConn()
		_, err := repo.UpdateNamespaceConnectionByUID(ctx, uuid.Must(uuid.NewV4()), conn)
		c.Check(errors.Is(err, errdomain.ErrNotFound), qt.IsTrue)
	})

	c.Run("nok - deletion not found", func(c *qt.C) {
		err := newRepo(c).DeleteNamespaceConnectionByID(ctx, uuid.Must(uuid.NewV4()), "foo")
		c.Check(errors.Is(err, errdomain.ErrNotFound), qt.IsTrue)
	})

	c.Run("ok - create, get, list", func(c *qt.C) {
		repo := newRepo(c)

		connectionIDs := []string{"2nd", "3rd", "4th", "1st"}
		integrations := []*pipelinepb.ComponentDefinition{openAI, pinecone, email, openAI}

		for i, id := range connectionIDs {
			integration := integrations[i]

			conn := newConn()
			conn.ID = id
			conn.IntegrationUID = uuid.FromStringOrNil(integration.GetUid())

			inserted, err := repo.CreateNamespaceConnection(ctx, conn)
			c.Check(err, qt.IsNil, qt.Commentf(conn.UID.String()))
			c.Check(inserted.UID, qt.IsNotNil)
			c.Check(inserted.CreateTime.IsZero(), qt.IsFalse)
			c.Check(inserted.UpdateTime.IsZero(), qt.IsFalse)
			c.Check(inserted.DeleteTime.Valid, qt.IsFalse)
			// Testing proto scan & write.
			c.Check(inserted.Method, qt.ContentEquals, method)
			c.Check(inserted.Integration.Title, qt.Not(qt.HasLen), 0)

			fetched, err := repo.GetNamespaceConnectionByID(ctx, conn.NamespaceUID, conn.ID)
			c.Check(err, qt.IsNil)

			cmp := qt.CmpEquals(
				cmpopts.EquateApproxTime(time.Millisecond),
				cmpopts.IgnoreFields(datamodel.Connection{}, "Integration"),
			)
			c.Check(fetched, cmp, inserted)

			// Query should preload Integration to avoid fetching it later in order
			// to build the integration title and ID.
			c.Check(fetched.Integration.Title, qt.Equals, integration.GetTitle())
			c.Check(fetched.Integration.ID, qt.Equals, integration.GetId())
		}

		// Page one
		p := ListNamespaceConnectionsParams{
			NamespaceUID: nsUID,
			Limit:        2,
		}
		connList, err := repo.ListNamespaceConnections(ctx, p)
		c.Check(err, qt.IsNil)
		c.Check(connList.TotalSize, qt.Equals, int32(4))
		c.Check(connList.Connections, qt.HasLen, 2)
		c.Check(connList.Connections[0].ID, qt.Equals, "1st")
		c.Check(connList.Connections[1].ID, qt.Equals, "2nd")
		c.Check(connList.NextPageToken, qt.Not(qt.HasLen), 0)

		// Page two
		p.PageToken = connList.NextPageToken
		connList, err = repo.ListNamespaceConnections(ctx, p)
		c.Check(err, qt.IsNil)
		c.Check(connList.Connections, qt.HasLen, 2)
		c.Check(connList.Connections[0].ID, qt.Equals, "3rd")
		c.Check(connList.Connections[1].ID, qt.Equals, "4th")
		c.Check(connList.NextPageToken, qt.HasLen, 0)

		// Check Integration preload
		c.Check(connList.Connections[0].Integration.Title, qt.Not(qt.HasLen), 0)

		preUpdateConn := connList.Connections[0]
		conn, err := repo.UpdateNamespaceConnectionByUID(ctx, preUpdateConn.UID, &datamodel.Connection{
			ID:             "testytest",
			NamespaceUID:   uuid.Must(uuid.NewV4()),
			IntegrationUID: uuid.Must(uuid.NewV4()),
			Setup:          datatypes.JSON(`{"foo":"bar"}`),
		})
		c.Check(err, qt.IsNil)
		c.Check(conn.ID, qt.Equals, "testytest")
		c.Check(conn.UID, qt.Equals, preUpdateConn.UID)
		c.Check(conn.NamespaceUID, qt.Equals, preUpdateConn.NamespaceUID)
		c.Check(conn.IntegrationUID, qt.Equals, preUpdateConn.IntegrationUID)
		c.Check([]byte(conn.Setup), qt.JSONEquals, json.RawMessage(`{"foo":"bar"}`))

		// Delete & fetch
		err = repo.DeleteNamespaceConnectionByID(ctx, nsUID, "1st")
		c.Check(err, qt.IsNil)
		connList, err = repo.ListNamespaceConnections(ctx, p)
		c.Check(err, qt.IsNil)
		c.Check(connList.TotalSize, qt.Equals, int32(3))
	})
}

func TestRepository_AddPipelineRuns(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	tx := db.Begin()
	c.Cleanup(func() { tx.Commit() })

	cache, _ := redismock.NewClientMock()
	repo := NewRepository(tx, cache)

	t0 := time.Now().UTC()
	pipelineUID, ownerUID := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	ownerPermalink := "users/" + ownerUID.String()

	p := &datamodel.Pipeline{
		Owner: ownerPermalink,
		ID:    "test",
		BaseDynamic: datamodel.BaseDynamic{
			UID:        pipelineUID,
			CreateTime: t0,
			UpdateTime: t0,
		},
	}
	err := repo.CreateNamespacePipeline(ctx, p)
	c.Assert(err, qt.IsNil)

	got, err := repo.GetNamespacePipelineByID(ctx, ownerPermalink, "test", true, false)
	c.Assert(err, qt.IsNil)
	c.Check(got.NumberOfRuns, qt.Equals, 0)
	c.Check(got.LastRunTime.IsZero(), qt.IsTrue)

	err = repo.AddPipelineRuns(ctx, got.UID)
	c.Check(err, qt.IsNil)

	got, err = repo.GetNamespacePipelineByID(ctx, ownerPermalink, "test", true, false)
	c.Assert(err, qt.IsNil)
	c.Check(got.NumberOfRuns, qt.Equals, 1)
	c.Check(got.LastRunTime.After(t0), qt.IsTrue)
}

func TestRepository_UpsertPipelineRun(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	tx := db.Begin()
	c.Cleanup(func() { tx.Rollback() })

	cache, _ := redismock.NewClientMock()
	repo := NewRepository(tx, cache)

	t0 := time.Now().UTC()
	pipelineUID, ownerUID := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	ownerPermalink := "users/" + ownerUID.String()

	pipelineID := "test"
	p := &datamodel.Pipeline{
		Owner: ownerPermalink,
		ID:    pipelineID,
		BaseDynamic: datamodel.BaseDynamic{
			UID:        pipelineUID,
			CreateTime: t0,
			UpdateTime: t0,
		},
	}
	err := repo.CreateNamespacePipeline(ctx, p)
	c.Assert(err, qt.IsNil)

	got, err := repo.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineID, true, false)
	c.Assert(err, qt.IsNil)
	c.Check(got.NumberOfRuns, qt.Equals, 0)
	c.Check(got.LastRunTime.IsZero(), qt.IsTrue)

	minioURL := `http://localhost:19000/instill-ai-vdp/e9ee5c7e-23a4-4910-b3be-afe1d3ca5254.recipe.json?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=minioadmin%2F20240816%2Fus-east-1%2Fs3%2Faws4_request\u0026X-Amz-Date=20240816T030849Z\u0026X-Amz-Expires=604800\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=f25a30c82e067b8da32c01a17452977082309c873d4a3bd72767ffe1118d695c`
	minioURL = url.QueryEscape(minioURL)
	c.Assert(err, qt.IsNil)

	pipelineRun := &datamodel.PipelineRun{
		PipelineTriggerUID: uuid.Must(uuid.NewV4()),
		PipelineUID:        p.UID,
		Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
		Source:             datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API),
		TriggeredBy:        ownerUID.String(),
		RecipeSnapshot: datamodel.JSONB{{
			URL: minioURL,
		}},
		StartedTime:   time.Now(),
		TotalDuration: null.IntFrom(42),
		Components:    nil,
	}

	err = repo.UpsertPipelineRun(ctx, pipelineRun)
	c.Assert(err, qt.IsNil)

	got1, err := repo.GetPipelineRunByUID(ctx, pipelineRun.PipelineTriggerUID)
	c.Assert(err, qt.IsNil)
	c.Check(got1.PipelineUID, qt.Equals, p.UID)
	c.Check(got1.Status, qt.Equals, pipelineRun.Status)
	c.Check(got1.Source, qt.Equals, pipelineRun.Source)
	c.Check(got1.Pipeline.UID, qt.Equals, p.UID)

	componentRun := &datamodel.ComponentRun{
		PipelineTriggerUID: pipelineRun.PipelineTriggerUID,
		ComponentID:        uuid.Must(uuid.NewV4()).String(),
		Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_FAILED),
		TotalDuration:      null.IntFrom(10),
		StartedTime:        time.Now(),
		Inputs:             nil,
		Outputs:            nil,
	}

	err = repo.UpsertComponentRun(ctx, componentRun)
	c.Assert(err, qt.IsNil)

	got2 := &datamodel.ComponentRun{PipelineTriggerUID: pipelineRun.PipelineTriggerUID, ComponentID: componentRun.ComponentID}
	err = tx.First(got2).Error
	c.Assert(err, qt.IsNil)
	c.Check(got2.Status, qt.Equals, componentRun.Status)
	c.Check(got2.TotalDuration.Valid, qt.IsTrue)
	c.Check(got2.TotalDuration.Int64, qt.Equals, componentRun.TotalDuration.Int64)

}

func TestRepository_GetPaginatedPipelineRunsWithPermissions(t *testing.T) {
	t.SkipNow()
	c := qt.New(t)
	ctx := context.Background()

	minioURL := `http://localhost:19000/instill-ai-vdp/e9ee5c7e-23a4-4910-b3be-afe1d3ca5254.recipe.json?X-Amz-Algorithm=AWS4-HMAC-SHA256\u0026X-Amz-Credential=minioadmin%2F20240816%2Fus-east-1%2Fs3%2Faws4_request\u0026X-Amz-Date=20240816T030849Z\u0026X-Amz-Expires=604800\u0026X-Amz-SignedHeaders=host\u0026X-Amz-Signature=f25a30c82e067b8da32c01a17452977082309c873d4a3bd72767ffe1118d695c`
	minioURL = url.QueryEscape(minioURL)

	mockUIDs := make([]uuid.UUID, 4)
	for i := range len(mockUIDs) {
		mockUIDs[i] = uuid.Must(uuid.NewV4())
	}
	user1 := mockUIDs[0].String()
	user2 := mockUIDs[1].String()
	namespace1 := mockUIDs[2].String()

	cache, _ := redismock.NewClientMock()

	t0 := time.Now().UTC()
	pipelineUID, ownerUID := uuid.Must(uuid.NewV4()), uuid.Must(uuid.NewV4())
	ownerPermalink := "users/" + ownerUID.String()
	pipelineID := "test"

	testCases := []struct {
		runner        string
		runNamespace  string
		viewer        string
		viewNamespace string
		canView       bool
	}{
		{
			runner:        user1,
			runNamespace:  namespace1,
			viewer:        user1,
			viewNamespace: namespace1,
			canView:       true,
		},
		{
			runner:        user1,
			runNamespace:  user1,
			viewer:        user1,
			viewNamespace: user1,
			canView:       true,
		},
		{
			runner:        user1,
			runNamespace:  namespace1,
			viewer:        user1,
			viewNamespace: user1,
			canView:       true,
		},
		{
			runner:        user2,
			runNamespace:  user2,
			viewer:        user1,
			viewNamespace: user1,
			canView:       false,
		},
	}

	for i, testCase := range testCases {
		c.Run(fmt.Sprintf("get pipeline run with permissions test case %d", i), func(c *qt.C) {
			c.Log(testCase)

			tx := db.Begin()
			c.Cleanup(func() { tx.Rollback() })

			repo := NewRepository(tx, cache)

			p := &datamodel.Pipeline{
				Owner: ownerPermalink,
				ID:    pipelineID,
				BaseDynamic: datamodel.BaseDynamic{
					UID:        pipelineUID,
					CreateTime: t0,
					UpdateTime: t0,
				},
			}
			err := repo.CreateNamespacePipeline(ctx, p)
			c.Assert(err, qt.IsNil)

			got, err := repo.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineID, true, false)
			c.Assert(err, qt.IsNil)
			c.Check(got.NumberOfRuns, qt.Equals, 0)
			c.Check(got.LastRunTime.IsZero(), qt.IsTrue)

			pipelineRun := &datamodel.PipelineRun{
				PipelineTriggerUID: uuid.Must(uuid.NewV4()),
				PipelineUID:        p.UID,
				Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
				Source:             datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API),
				TriggeredBy:        testCase.runner,
				Namespace:          testCase.runNamespace,
				RecipeSnapshot: datamodel.JSONB{{
					URL: minioURL,
				}},
				StartedTime:   time.Now(),
				TotalDuration: null.IntFrom(42),
				Components:    nil,
			}

			err = repo.UpsertPipelineRun(ctx, pipelineRun)
			c.Assert(err, qt.IsNil)

			response, _, err := repo.GetPaginatedPipelineRunsWithPermissions(ctx, testCase.viewNamespace, p.UID.String(), 0, 10, filtering.Filter{}, ordering.OrderBy{}, false)
			c.Assert(err, qt.IsNil)
			if testCase.canView {
				c.Check(len(response), qt.Equals, 1)
			} else {
				c.Check(len(response), qt.Equals, 0)
			}
		})
	}
}

func TestRepository_GetPaginatedPipelineRunsByCreditOwner(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	cache, _ := redismock.NewClientMock()

	t0 := time.Now().UTC()

	mockUIDs := make([]uuid.UUID, 5)
	for i := range len(mockUIDs) {
		mockUIDs[i] = uuid.Must(uuid.NewV4())
	}
	user1 := mockUIDs[0].String()
	namespace1 := mockUIDs[1].String()
	now := time.Now()

	pipelineUID, ownerUID := mockUIDs[2], mockUIDs[3]
	pipelineUID2 := mockUIDs[4]
	ownerPermalink := "users/" + ownerUID.String()
	pipelineID := "test"
	pipelineID2 := "test2"

	tx := db.Begin()
	c.Cleanup(func() { tx.Rollback() })

	repo := NewRepository(tx, cache)

	p := &datamodel.Pipeline{
		Owner: ownerPermalink,
		ID:    pipelineID,
		BaseDynamic: datamodel.BaseDynamic{
			UID:        pipelineUID,
			CreateTime: t0,
			UpdateTime: t0,
		},
	}
	err := repo.CreateNamespacePipeline(ctx, p)
	c.Check(err, qt.IsNil)

	p2 := &datamodel.Pipeline{
		Owner: ownerPermalink,
		ID:    pipelineID2,
		BaseDynamic: datamodel.BaseDynamic{
			UID:        pipelineUID2,
			CreateTime: t0,
			UpdateTime: t0,
		},
	}
	err = repo.CreateNamespacePipeline(ctx, p2)
	c.Check(err, qt.IsNil)

	got, err := repo.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineID, true, false)
	c.Check(err, qt.IsNil)
	c.Check(got.NumberOfRuns, qt.Equals, 0)
	c.Check(got.LastRunTime.IsZero(), qt.IsTrue)

	got, err = repo.GetNamespacePipelineByID(ctx, ownerPermalink, pipelineID2, true, false)
	c.Check(err, qt.IsNil)
	c.Check(got.NumberOfRuns, qt.Equals, 0)
	c.Check(got.LastRunTime.IsZero(), qt.IsTrue)

	pipelineRun := &datamodel.PipelineRun{
		PipelineTriggerUID: uuid.Must(uuid.NewV4()),
		PipelineUID:        p.UID,
		Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
		Source:             datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API),
		TriggeredBy:        user1,
		Namespace:          namespace1,
		StartedTime:        now.Add(-1 * time.Hour),
		TotalDuration:      null.IntFrom(42),
		Components:         nil,
	}

	err = repo.UpsertPipelineRun(ctx, pipelineRun)
	c.Check(err, qt.IsNil)

	resp, _, err := repo.GetPaginatedPipelineRunsByRequester(ctx, GetPipelineRunsByRequesterParams{
		RequesterUID:   namespace1,
		StartTimeBegin: now.Add(-3 * time.Hour),
		StartTimeEnd:   now.Add(-2 * time.Hour),
		Page:           0,
		PageSize:       10,
		Filter:         filtering.Filter{},
		Order:          ordering.OrderBy{},
	})
	c.Check(err, qt.IsNil)
	c.Check(resp, qt.HasLen, 0)

	resp, _, err = repo.GetPaginatedPipelineRunsByRequester(ctx, GetPipelineRunsByRequesterParams{
		RequesterUID:   namespace1,
		StartTimeBegin: now.Add(-2 * time.Hour),
		StartTimeEnd:   now,
		Page:           0,
		PageSize:       10,
		Filter:         filtering.Filter{},
		Order:          ordering.OrderBy{},
	})
	c.Check(err, qt.IsNil)
	c.Check(resp, qt.HasLen, 1)
	c.Check(resp[0].PipelineTriggerUID, qt.Equals, pipelineRun.PipelineTriggerUID)
	c.Check(resp[0].Pipeline.ID, qt.Equals, p.ID)

	pipelineRun2 := &datamodel.PipelineRun{
		PipelineTriggerUID: uuid.Must(uuid.NewV4()),
		PipelineUID:        p2.UID,
		Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
		Source:             datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API),
		TriggeredBy:        user1,
		Namespace:          namespace1,
		StartedTime:        now.Add(-1 * time.Hour),
		TotalDuration:      null.IntFrom(42),
		Components:         nil,
	}

	err = repo.UpsertPipelineRun(ctx, pipelineRun2)
	c.Check(err, qt.IsNil)

	resp, _, err = repo.GetPaginatedPipelineRunsByRequester(ctx, GetPipelineRunsByRequesterParams{
		RequesterUID:   namespace1,
		StartTimeBegin: now.Add(-2 * time.Hour),
		StartTimeEnd:   now,
		Page:           0,
		PageSize:       10,
		Filter:         filtering.Filter{},
		Order:          ordering.OrderBy{},
	})
	c.Check(err, qt.IsNil)
	c.Check(resp, qt.HasLen, 2)
	c.Check(resp[0].PipelineTriggerUID, qt.Equals, pipelineRun.PipelineTriggerUID)
	c.Check(resp[0].Pipeline.ID, qt.Equals, p.ID)
	c.Check(resp[1].PipelineTriggerUID, qt.Equals, pipelineRun2.PipelineTriggerUID)
	c.Check(resp[1].Pipeline.ID, qt.Equals, p2.ID)
}
