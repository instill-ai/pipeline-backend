package base

import (
	"encoding/json"
	"testing"

	_ "embed"

	"google.golang.org/protobuf/encoding/protojson"

	qt "github.com/frankban/quicktest"
)

var (
	//go:embed testdata/componentDef.json
	componentDefJSON []byte
	//go:embed testdata/componentTasks.json
	componentTasksJSON []byte
	//go:embed testdata/componentConfig.json
	componentConfigJSON []byte
	//go:embed testdata/componentAdditional.json
	componentAdditionalJSON []byte
	//go:embed testdata/wantComponentDefinition.json
	wantComponentDefinitionJSON []byte
)

func TestComponent_ListComponentDefinitions(t *testing.T) {
	c := qt.New(t)

	conn := new(Component)
	err := conn.LoadDefinition(
		componentDefJSON,
		componentConfigJSON,
		componentTasksJSON,
		nil,
		map[string][]byte{"additional.json": componentAdditionalJSON})
	c.Assert(err, qt.IsNil)

	got, err := conn.GetDefinition(nil, nil)
	c.Assert(err, qt.IsNil)
	gotJSON, err := protojson.Marshal(got)
	c.Assert(err, qt.IsNil)

	wantComponentDefinitionStruct := map[string]any{}
	err = json.Unmarshal(wantComponentDefinitionJSON, &wantComponentDefinitionStruct)
	c.Assert(err, qt.IsNil)
	c.Check(gotJSON, qt.JSONEquals, wantComponentDefinitionStruct)
}
