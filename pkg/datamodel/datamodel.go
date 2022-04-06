package datamodel

import (
	"database/sql/driver"
	"errors"

	"gorm.io/gorm"
)

type Pipeline struct {
	gorm.Model

	// the name of this Instill Pipeline
	Name string

	// the more detail of this Instill Pipeline
	Description string

	Status ValidStatus `sql:"type:valid_status"`

	Recipe *Recipe `gorm:"type:json"`

	Namespace string

	FullName string
}

type ListPipelineQuery struct {
	WithRecipe bool
	Namespace  string
	PageSize   uint64
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

type ValidStatus string

const (
	StatusInactive ValidStatus = "STATUS_INACTIVE"
	StatusActive   ValidStatus = "STATUS_ACTIVE"
	StatusError    ValidStatus = "STATUS_ERROR"
)

func (p *ValidStatus) Scan(value interface{}) error {
	switch v := value.(type) {
	case string:
		*p = ValidStatus(v)
	case []byte:
		*p = ValidStatus(v)
	default:
		return errors.New("Incompatible type for ValidStatus")
	}
	return nil
}

func (p ValidStatus) Value() (driver.Value, error) {
	return string(p), nil
}
