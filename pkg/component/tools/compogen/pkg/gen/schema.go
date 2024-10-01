package gen

import (
	"encoding/json"
)

type property struct {
	Description string `json:"description" validate:"required"`
	Title       string `json:"title" validate:"required"`
	Order       *int   `json:"instillUIOrder" validate:"required"`

	Type string `json:"type"`

	// If Type is array, Items defines the element type.
	Items struct {
		Type       string              `json:"type"`
		Properties map[string]property `json:"properties" validate:"omitempty,dive"`
	} `json:"items"`

	Properties map[string]property `json:"properties" validate:"omitempty,dive"`

	OneOf []objectSchema `json:"oneOf" validate:"dive"`
	Const string         `json:"const,omitempty"`

	Enum []string `json:"enum,omitempty"`

	Deprecated bool `json:"deprecated"`
}

type objectSchema struct {
	Description string              `json:"description"`
	Properties  map[string]property `json:"properties" validate:"gt=0,dive"`
	Title       string              `json:"title" validate:"required"`
	Required    []string            `json:"required"`
}

func (t *objectSchema) MarshalJSON() ([]byte, error) {
	type Alias objectSchema
	return json.Marshal(&struct {
		*Alias
		InstillShortDescription string `json:"instillShortDescription,omitempty"`
	}{
		Alias:                   (*Alias)(t),
		InstillShortDescription: t.Description,
	})
}

func (t *objectSchema) UnmarshalJSON(data []byte) error {
	type Alias objectSchema
	aux := &struct {
		InstillShortDescription string `json:"instillShortDescription"`
		Description             string `json:"description"`
		*Alias
	}{
		Alias: (*Alias)(t),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Set Description based on the presence of the fields
	if aux.Description != "" {
		t.Description = aux.Description
	} else if aux.InstillShortDescription != "" {
		t.Description = aux.InstillShortDescription
	}

	return nil
}
