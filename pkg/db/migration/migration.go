package migration

import (
	"context"

	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000013"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000015"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000016"
	"github.com/instill-ai/pipeline-backend/pkg/logger"

	database "github.com/instill-ai/pipeline-backend/pkg/db"
)

type migration interface {
	Migrate() error
}

// Migrate executes custom code as part of a database migration. This code is
// intended to be run only once and typically goes along a change
// in the database schemas. Some use cases might be backfilling a table or
// updating some existing records according to the schema changes.
//
// Note that the changes in the database schemas shouldn't be run here, only
// code accompanying them.
func Migrate(version uint) error {
	var m migration
	ctx := context.Background()
	l, _ := logger.GetZapLogger(ctx)

	db := database.GetConnection().WithContext(ctx)
	defer database.Close(db)

	switch version {
	case 13:
		m = new(convert000013.Migration)
	case 15:
		m = new(convert000015.Migration)
	case 16:
		m = new(convert000016.Migration)
	case 19:
		m = &jqInputToKebabCaseConverter{db: db, logger: l}
	default:
		return nil
	}

	return m.Migrate()
}
