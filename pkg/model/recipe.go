package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type Recipe struct {
	DataSource         *DataSource           `json:"data_source,omitempty"`
	DataDestination    *DataDestination      `json:"data_destination,omitempty"`
	VisualDataOperator []*VisualDataOperator `json:"visual_data_operator,omitempty"`
	LogicOperator      []*LogicOperator      `json:"logic_operator,omitempty"`
}

type DataSource struct {
	Type string `json:"type,omitempty"`
}

type DataDestination struct {
	Type string `json:"type,omitempty"`
}

type VisualDataOperator struct {
	ModelId string `json:"model_id,omitempty"`
	Version int32  `json:"version,omitempty"`
}

type LogicOperator struct {
}

func (r *Recipe) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal Recipe value:", value))
	}

	if err := json.Unmarshal(bytes, &r); err != nil {
		return err
	}

	return nil
}

func (r *Recipe) Value() (driver.Value, error) {
	valueString, err := json.Marshal(r)
	return string(valueString), err
}
