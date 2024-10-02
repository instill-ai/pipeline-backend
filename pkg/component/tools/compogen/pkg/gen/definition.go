package gen

import (
	"encoding/json"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	pb "github.com/instill-ai/protogen-go/vdp/pipeline/v1beta"
)

type releaseStage pb.ComponentDefinition_ReleaseStage

func (rs *releaseStage) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	*rs = releaseStage(pb.ComponentDefinition_ReleaseStage_value[s])
	return nil
}

func (rs releaseStage) String() string {
	pbRS := pb.ComponentDefinition_ReleaseStage(rs)
	if pbRS == pb.ComponentDefinition_RELEASE_STAGE_GA {
		return "GA"
	}

	upperSnake, _ := strings.CutPrefix(pbRS.String(), "RELEASE_STAGE_")
	return cases.Title(language.English).String(strings.ReplaceAll(upperSnake, "_", " "))
}

type definition struct {
	ID             string       `json:"id" validate:"required"`
	Title          string       `json:"title" validate:"required"`
	Description    string       `json:"description" validate:"required"`
	ReleaseStage   releaseStage `json:"releaseStage" validate:"required"`
	AvailableTasks []string     `json:"availableTasks" validate:"gt=0"`
	SourceURL      string       `json:"sourceUrl" validate:"url"`

	Public        bool   `json:"public"`
	Type          string `json:"type"`
	Prerequisites string `json:"prerequisites"`
	Vendor        string `json:"vendor"`
}
