//go:build dbtest
// +build dbtest

package repository

import (
	"context"
	"testing"

	qt "github.com/frankban/quicktest"
	"github.com/gofrs/uuid"
	"github.com/instill-ai/pipeline-backend/config"
	database "github.com/instill-ai/pipeline-backend/pkg/db"
	pipelinepb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

func TestRepository_ComponentDefinitionUIDs(t *testing.T) {
	c := qt.New(t)
	ctx := context.Background()

	err := config.Init("../../config/config.yaml")
	c.Assert(err, qt.IsNil)

	db := database.GetSharedConnection()
	defer database.Close(db)

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

	err = repo.UpsertComponentDefinition(ctx, cd)
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
