//go:build dbtest
// +build dbtest

package repository

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/gofrs/uuid"
	"gopkg.in/guregu/null.v4"
	"gorm.io/gorm"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/config"
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"

	database "github.com/instill-ai/pipeline-backend/pkg/db"

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
	c.Cleanup(func() { tx.Commit() })

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

	pipelineRun := &datamodel.PipelineRun{
		PipelineTriggerUID: uuid.Must(uuid.NewV4()),
		PipelineUID:        p.UID,
		Status:             datamodel.RunStatus(runpb.RunStatus_RUN_STATUS_PROCESSING),
		Source:             datamodel.RunSource(runpb.RunSource_RUN_SOURCE_API),
		TriggeredBy:        ownerUID.String(),
		Inputs:             nil,
		Outputs:            nil,
		RecipeSnapshot:     nil,
		StartedTime:        time.Now(),
		TotalDuration:      null.IntFrom(42),
		Components:         nil,
	}

	err = repo.UpsertPipelineRun(ctx, pipelineRun)
	c.Assert(err, qt.IsNil)

	got1 := &datamodel.PipelineRun{PipelineTriggerUID: pipelineRun.PipelineTriggerUID}
	err = tx.First(got1).Error
	c.Assert(err, qt.IsNil)
	c.Check(got1.PipelineUID, qt.Equals, p.UID)
	c.Check(got1.Status, qt.Equals, pipelineRun.Status)
	c.Check(got1.Source, qt.Equals, pipelineRun.Source)

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
