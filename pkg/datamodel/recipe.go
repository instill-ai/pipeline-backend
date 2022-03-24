package datamodel

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

type Recipe struct {
	Source        *Source          `json:"source,omitempty"`
	Destination   *Destination     `json:"destination,omitempty"`
	Model         []*Model         `json:"model,omitempty"`
	LogicOperator []*LogicOperator `json:"logic_operator,omitempty"`
}

type Source struct {
	Type string `json:"type,omitempty"`
}

type Destination struct {
	Type string `json:"type,omitempty"`
}

type Model struct {
	Name    string `json:"model_name,omitempty"`
	Version uint64 `json:"version,omitempty"`
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
