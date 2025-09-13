package gen

import (
	"encoding/json"
)

type property struct {
	Description string `json:"description" validate:"required"`
	Title       string `json:"title" validate:"required"`
	Order       *int   `json:"uiOrder" validate:"required"`

	Type string `json:"type"`
	Ref  string `json:"$ref,omitempty"`

	// If Type is array, Items defines the element format.
	Items struct {
		Type        string              `json:"type"`
		Title       string              `json:"title"`
		Description string              `json:"description"`
		Ref         string              `json:"$ref,omitempty"`
		Properties  map[string]property `json:"properties" validate:"omitempty,dive"`
		OneOf       []objectSchema      `json:"oneOf" validate:"dive"`
		Enum        []string            `json:"enum,omitempty"`
	} `json:"items"`

	Properties map[string]property `json:"properties" validate:"omitempty,dive"`

	OneOf []objectSchema `json:"oneOf" validate:"dive"`
	Const string         `json:"const,omitempty"`

	Enum []string `json:"enum,omitempty"`

	Deprecated bool `json:"deprecated"`
}

type objectSchema struct {
	Description string              `json:"description"`
	Properties  map[string]property `json:"properties" validate:"dive"`
	Title       string              `json:"title" validate:"required"`
	Required    []string            `json:"required"`
	AllOf       []objectSchema      `json:"allOf"`
	OneOf       []objectSchema      `json:"oneOf" validate:"dive"`
}

func (t *objectSchema) MarshalJSON() ([]byte, error) {
	type Alias objectSchema
	return json.Marshal(&struct {
		*Alias
		ShortDescription string `json:"shortDescription,omitempty"`
	}{
		Alias:            (*Alias)(t),
		ShortDescription: t.Description,
	})
}

func (t *objectSchema) UnmarshalJSON(data []byte) error {
	type Alias objectSchema
	aux := &struct {
		ShortDescription string `json:"shortDescription"`
		Description      string `json:"description"`
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
	} else if aux.ShortDescription != "" {
		t.Description = aux.ShortDescription
	}

	return nil
}
