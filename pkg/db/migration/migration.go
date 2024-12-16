package migration

import (
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000013"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000015"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000016"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000019"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000020"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000021"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000022"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000024"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000029"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000031"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000032"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000033"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000034"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000036"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert/convert000039"
	"github.com/instill-ai/pipeline-backend/pkg/service"

	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

type migration interface {
	Migrate() error
}

// CodeMigrator orchestrates the execution of the code associated with the
// different database migrations and holds their dependencies.
type CodeMigrator struct {
	Logger *zap.Logger

	DB                       *gorm.DB
	RetentionHandler         service.MetadataRetentionHandler
	MGMTPrivateServiceClient mgmtpb.MgmtPrivateServiceClient
}

// Migrate executes custom code as part of a database migration. This code is
// intended to be run only once and typically goes along a change
// in the database schemas. Some use cases might be backfilling a table or
// updating some existing records according to the schema changes.
//
// Note that the changes in the database schemas shouldn't be run here, only
// code accompanying them.
func (cm *CodeMigrator) Migrate(version uint) error {
	var m migration

	bc := convert.Basic{
		DB:     cm.DB,
		Logger: cm.Logger,
	}

	switch version {
	case 13:
		m = new(convert000013.Migration)
	case 15:
		m = new(convert000015.Migration)
	case 16:
		m = new(convert000016.Migration)
	case 19:
		m = &convert000019.JQInputToKebabCaseConverter{Basic: bc}
	case 20:
		m = &convert000020.NamespaceIDMigrator{
			Basic:      bc,
			MgmtClient: cm.MGMTPrivateServiceClient,
		}
	case 21:
		m = &convert000021.ConvertToTextTaskConverter{Basic: bc}
	case 22:
		m = &convert000022.ConvertWebsiteToWebConverter{Basic: bc}
	case 24:
		m = &convert000024.ConvertToTextTaskConverter{Basic: bc}
	case 29:
		m = &convert000029.ConvertToArtifactType{Basic: bc}
	case 31:
		m = &convert000031.SlackSetupConverter{Basic: bc}
	case 32:
		m = &convert000032.ConvertToWeb{Basic: bc}
	case 33:
		m = &convert000033.ConvertWebFields{Basic: bc}
	case 34:
		m = &convert000034.RenameHTTPComponent{Basic: bc}
	case 36:
		m = &convert000036.RenameInstillFormat{Basic: bc}
	case 39:
		m = &convert000039.AddExpirationDateToRuns{
			Basic:            bc,
			RetentionHandler: cm.RetentionHandler,
		}
	default:
		return nil
	}

	return m.Migrate()
}
