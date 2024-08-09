package convert000022

import (
	"github.com/instill-ai/pipeline-backend/pkg/datamodel"
	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// const batchSize = 100

type ConvertWebsiteToWebConverter struct {
	DB     *gorm.DB
	Logger *zap.Logger
}

func (c *ConvertWebsiteToWebConverter) Migrate() error {
	if err := c.updateComponentDefinition(); err != nil {
		return err
	}

	return nil
}

func (c *ConvertWebsiteToWebConverter) updateComponentDefinition() error {
	var comDef datamodel.ComponentDefinition
	c.DB.Where("id = ?", "website").First(&comDef)
	c.DB.
		Model(&comDef).
		Updates(datamodel.ComponentDefinition{
			ID:            "web",
			Title:         "Web",
			ComponentType: datamodel.ComponentType(pb.ComponentType_COMPONENT_TYPE_OPERATOR),
			Version:       "0.2.0"})

	return nil
}
