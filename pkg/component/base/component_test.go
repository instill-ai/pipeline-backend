package base

import (
	"testing"

	_ "embed"

	"google.golang.org/protobuf/encoding/protojson"
	"gopkg.in/yaml.v3"

	qt "github.com/frankban/quicktest"

	"github.com/instill-ai/pipeline-backend/pkg/component/resources/schemas"
)

var (
	//go:embed testdata/componentDef.yaml
	componentDefYAML []byte
	//go:embed testdata/componentTasks.yaml
	componentTasksYAML []byte
	//go:embed testdata/componentConfig.yaml
	componentConfigYAML []byte
	//go:embed testdata/componentAdditional.yaml
	componentAdditionalYAML []byte
	//go:embed testdata/wantComponentDefinition.yaml
	wantComponentDefinitionYAML []byte
)

func TestComponent_ListComponentDefinitions(t *testing.T) {
	c := qt.New(t)

	conn := new(Component)
	err := conn.LoadDefinition(
		componentDefYAML,
		componentConfigYAML,
		componentTasksYAML,
		nil,
		map[string][]byte{
			"additional.yaml": componentAdditionalYAML,
			"schema.yaml":     schemas.SchemaYAML,
		})
	c.Assert(err, qt.IsNil)

	got, err := conn.GetDefinition(nil, nil)
	c.Assert(err, qt.IsNil)
	gotJSON, err := protojson.Marshal(got)
	c.Assert(err, qt.IsNil)

	wantComponentDefinitionStruct := map[string]any{}
	err = yaml.Unmarshal(wantComponentDefinitionYAML, &wantComponentDefinitionStruct)
	c.Assert(err, qt.IsNil)
	c.Check(gotJSON, qt.JSONEquals, wantComponentDefinitionStruct)
}
