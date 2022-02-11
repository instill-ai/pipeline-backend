package model

import (
	"time"

	"gorm.io/gorm"
)

type Pipeline struct {
	Id uint64

	// the name of this Instill Pipeline
	Name string

	// the more detail of this Instill Pipeline
	Description string

	Active bool `gorm:"type:tinyint"`

	// the time when entity created
	CreatedAt time.Time `gorm:"type:timestamp"`

	// the time when entity has been updated
	UpdatedAt time.Time `gorm:"type:timestamp"`

	// the time when entity has been deleted
	DeletedAt gorm.DeletedAt

	Recipe *Recipe `gorm:"type:json"`

	Namespace string

	FullName string
}

type ListPipelineQuery struct {
	WithRecipe bool
	Namespace  string
	PageSize   int32
	Cursor     uint64
}

type TriggerPipeline struct {
	Name     string                    `json:"name,omitempty"`
	Contents []*TriggerPipelineContent `json:"contents,omitempty"`
}

type TriggerPipelineContent struct {
	Url    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
	Chunk  []byte `json:"chunk,omitempty"`
}
