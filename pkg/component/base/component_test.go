package base

import (
	"encoding/json"
	"testing"

	_ "embed"

	"google.golang.org/protobuf/encoding/protojson"

	qt "github.com/frankban/quicktest"
)

var (
	//go:embed testdata/connectorDef.json
	connectorDefJSON []byte
	//go:embed testdata/connectorTasks.json
	connectorTasksJSON []byte
	//go:embed testdata/connectorConfig.json
	connectorConfigJSON []byte
	//go:embed testdata/connectorAdditional.json
	connectorAdditionalJSON []byte
	//go:embed testdata/wantConnectorDefinition.json
	wantConnectorDefinitionJSON []byte
)

func TestComponent_ListConnectorDefinitions(t *testing.T) {
	c := qt.New(t)

	conn := new(Component)
	err := conn.LoadDefinition(
		connectorDefJSON,
		connectorConfigJSON,
		connectorTasksJSON,
		map[string][]byte{"additional.json": connectorAdditionalJSON})
	c.Assert(err, qt.IsNil)

	got, err := conn.GetDefinition(nil, nil)
	c.Assert(err, qt.IsNil)
	gotJSON, err := protojson.Marshal(got)
	c.Assert(err, qt.IsNil)

	wantConnectorDefinitionStruct := map[string]any{}
	err = json.Unmarshal(wantConnectorDefinitionJSON, &wantConnectorDefinitionStruct)
	c.Assert(err, qt.IsNil)
	c.Check(gotJSON, qt.JSONEquals, wantConnectorDefinitionStruct)
}
