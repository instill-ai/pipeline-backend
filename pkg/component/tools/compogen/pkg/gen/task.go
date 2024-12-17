package gen

import (
	"encoding/json"
)

// This struct is used to validate the tasks schema.
// type taskMap struct {
// 	Tasks map[string]task `validate:gt=0,dive`
// }

type task struct {
	Description string        `json:"description"`
	Title       string        `json:"title"`
	Input       *objectSchema `json:"input" validate:"omitnil"`
	Output      *objectSchema `json:"output" validate:"omitnil"`
}

func (t *task) MarshalJSON() ([]byte, error) {
	type Alias task
	return json.Marshal(&struct {
		*Alias
		ShortDescription string `json:"shortDescription,omitempty"`
	}{
		Alias:            (*Alias)(t),
		ShortDescription: t.Description,
	})
}

func (t *task) UnmarshalJSON(data []byte) error {
	type Alias task
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
