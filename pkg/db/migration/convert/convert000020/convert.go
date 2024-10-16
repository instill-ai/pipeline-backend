package convert000020

import (
	"context"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	"github.com/instill-ai/pipeline-backend/pkg/db/migration/convert"

	mgmtpb "github.com/instill-ai/protogen-go/core/mgmt/v1beta"
)

const batchSize = 100

// NamespaceIDMigrator migrate the existing owner field to new namespace_id and
// namespace_type field.
type NamespaceIDMigrator struct {
	convert.Basic
	MgmtClient mgmtpb.MgmtPrivateServiceClient
}

func (c *NamespaceIDMigrator) Migrate() error {
	c.Logger.Info("NamespaceIDMigrator start")
	if err := c.migratePipeline(); err != nil {
		return err
	}
	if err := c.migrateSecret(); err != nil {
		return err
	}
	return nil
}

func (c *NamespaceIDMigrator) migratePipeline() error {
	pipelines := make([]*datamodel.Pipeline, 0, batchSize)
	return c.DB.Select("uid", "owner").FindInBatches(&pipelines, batchSize, func(tx *gorm.DB, _ int) error {
		for _, p := range pipelines {

			l := c.Logger.With(zap.String("pipelineUID", p.UID.String()))
			ownerUID := strings.Split(p.Owner, "/")[1]

			ns, err := c.MgmtClient.CheckNamespaceByUIDAdmin(context.Background(), &mgmtpb.CheckNamespaceByUIDAdminRequest{
				Uid: ownerUID,
			})
			if err != nil {
				return err
			}

			nsType := ""
			if ns.Type == mgmtpb.CheckNamespaceByUIDAdminResponse_NAMESPACE_ORGANIZATION {
				nsType = "organizations"
			} else if ns.Type == mgmtpb.CheckNamespaceByUIDAdminResponse_NAMESPACE_USER {
				nsType = "users"
			}
			result := tx.Model(p).Where("uid = ?", p.UID).Update("namespace_id", ns.Id).Update("namespace_type", nsType)
			if result.Error != nil {
				l.Error("Failed to update pipeline.")
				return fmt.Errorf("updating pipeline namespace_id: %w", result.Error)
			}
		}

		return nil
	}).Error
}

func (c *NamespaceIDMigrator) migrateSecret() error {
	secrets := make([]*datamodel.Secret, 0, batchSize)
	return c.DB.Select("uid", "owner").FindInBatches(&secrets, batchSize, func(tx *gorm.DB, _ int) error {
		for _, s := range secrets {

			l := c.Logger.With(zap.String("secretUID", s.UID.String()))
			ownerUID := strings.Split(s.Owner, "/")[1]

			ns, err := c.MgmtClient.CheckNamespaceByUIDAdmin(context.Background(), &mgmtpb.CheckNamespaceByUIDAdminRequest{
				Uid: ownerUID,
			})
			if err != nil {
				return err
			}

			nsType := ""
			if ns.Type == mgmtpb.CheckNamespaceByUIDAdminResponse_NAMESPACE_ORGANIZATION {
				nsType = "organizations"
			} else if ns.Type == mgmtpb.CheckNamespaceByUIDAdminResponse_NAMESPACE_USER {
				nsType = "users"
			}
			result := tx.Model(s).Where("uid = ?", s.UID).Update("namespace_id", ns.Id).Update("namespace_type", nsType)
			if result.Error != nil {
				l.Error("Failed to update secret.")
				return fmt.Errorf("updating secret namespace_id: %w", result.Error)
			}
		}

		return nil
	}).Error
}
